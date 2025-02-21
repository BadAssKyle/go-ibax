/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package block

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/IBAX-io/go-ibax/packages/converter"

	"github.com/IBAX-io/go-ibax/packages/notificator"

	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/conf/syspar"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/crypto"
	"github.com/IBAX-io/go-ibax/packages/model"
	"github.com/IBAX-io/go-ibax/packages/protocols"
	"github.com/IBAX-io/go-ibax/packages/script"
	"github.com/IBAX-io/go-ibax/packages/smart"
	"github.com/IBAX-io/go-ibax/packages/transaction"
	"github.com/IBAX-io/go-ibax/packages/transaction/custom"
	"github.com/IBAX-io/go-ibax/packages/types"
	"github.com/IBAX-io/go-ibax/packages/utils"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	ErrIncorrectRollbackHash = errors.New("Rollback hash doesn't match")
	ErrEmptyBlock            = errors.New("Block doesn't contain transactions")
	ErrIncorrectBlockTime    = utils.WithBan(errors.New("Incorrect block time"))
)

// Block is storing block data
type Block struct {
	Header            utils.BlockData
	PrevHeader        *utils.BlockData
	PrevRollbacksHash []byte
	MrklRoot          []byte
	BinData           []byte
	Transactions      []*transaction.Transaction
	SysUpdate         bool
	GenBlock          bool // it equals true when we are generating a new block
	Notifications     []types.Notifications
}

func (b Block) String() string {
	return fmt.Sprintf("header: %s, prevHeader: %s", b.Header, b.PrevHeader)
}

// GetLogger is returns logger
func (b Block) GetLogger() *log.Entry {
	return log.WithFields(log.Fields{"block_id": b.Header.BlockID, "block_time": b.Header.Time, "block_wallet_id": b.Header.KeyID,
		"block_state_id": b.Header.EcosystemID, "block_hash": b.Header.Hash, "block_version": b.Header.Version})
}
func (b *Block) IsGenesis() bool {
	return b.Header.BlockID == 1
}

// PlaySafe is inserting block safely
func (b *Block) PlaySafe() error {
	logger := b.GetLogger()
	dbTransaction, err := model.StartTransaction()
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("starting db transaction")
		return err
	}

	inputTx := b.Transactions[:]
	err = b.Play(dbTransaction)
	if err != nil {
		dbTransaction.Rollback()
		if b.GenBlock && len(b.Transactions) == 0 {
			if err == ErrLimitStop {
				err = ErrLimitTime
			}
			if inputTx[0].TxHeader != nil {
				BadTxForBan(inputTx[0].TxHeader.KeyID)
			}
			if err := transaction.MarkTransactionBad(dbTransaction, inputTx[0].TxHash, err.Error()); err != nil {
				return err
			}
		}
		return err
	}

	if b.GenBlock {
		if len(b.Transactions) == 0 {
			dbTransaction.Commit()
			return ErrEmptyBlock
		} else if len(inputTx) != len(b.Transactions) {
			if err = b.repeatMarshallBlock(); err != nil {
				dbTransaction.Rollback()
				return err
			}
		}
	}

	if err := UpdBlockInfo(dbTransaction, b); err != nil {
		dbTransaction.Rollback()
		return err
	}

	if err := InsertIntoBlockchain(dbTransaction, b); err != nil {
		dbTransaction.Rollback()
		return err
	}

	if b.SysUpdate {
		b.SysUpdate = false
		//if err = syspar.SysUpdate(dbTransaction); err != nil {
		//	log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("updating syspar")
		//	dbTransaction.Rollback()
		//	return err
		//}
	}

	err = dbTransaction.Commit()
	if err != nil {
		return err
	}

	for _, q := range b.Notifications {
		q.Send()
	}
	return nil
}

func (b *Block) repeatMarshallBlock() error {
	trData := make([][]byte, 0, len(b.Transactions))
	for _, tr := range b.Transactions {
		trData = append(trData, tr.TxFullData)
	}
	NodePrivateKey, _ := utils.GetNodeKeys()
	if len(NodePrivateKey) < 1 {
		err := errors.New(`empty private node key`)
		log.WithFields(log.Fields{"type": consts.NodePrivateKeyFilename, "error": err}).Error("reading node private key")
		return err
	}

	newBlockData, err := MarshallBlock(&b.Header, trData, b.PrevHeader, NodePrivateKey)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("marshalling new block")
		return err
	}

	nb, err := UnmarshallBlock(bytes.NewBuffer(newBlockData), true)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("parsing new block")
		return err
	}
	b.BinData = newBlockData
	b.Transactions = nb.Transactions
	b.MrklRoot = nb.MrklRoot
	b.SysUpdate = nb.SysUpdate
	return nil
}
	b.PrevHeader, err = GetBlockDataFromBlockChain(b.Header.BlockID - 1)
	if err != nil {
		return errors.Wrapf(err, "Can't get block %d", b.Header.BlockID-1)
	}
	return nil
}

func (b *Block) Play(dbTransaction *model.DbTransaction) (batchErr error) {
	var (
		playTxs model.AfterTxs
	)
	logger := b.GetLogger()
	limits := NewLimits(b)
	rand := utils.NewRand(b.Header.Time)
	var timeLimit int64
	if b.GenBlock {
		timeLimit = syspar.GetMaxBlockGenerationTime()
	}
	proccessedTx := make([]*transaction.Transaction, 0, len(b.Transactions))
	defer func() {
		if b.GenBlock {
			b.Transactions = proccessedTx
		}
		if err := model.AfterPlayTxs(dbTransaction, b.Header.BlockID, playTxs, logger); err != nil {
			batchErr = err
			return
		}
	}()

	for curTx, t := range b.Transactions {
		var (
			msg string
			err error
		)
		t.DbTransaction = dbTransaction
		t.Rand = rand.BytesSeed(t.TxHash)
		t.Notifications = notificator.NewQueue()

		err = dbTransaction.Savepoint(consts.SetSavePointMarkBlock(curTx))
		if err != nil {
			logger.WithFields(log.Fields{"type": consts.DBError, "error": err, "tx_hash": t.TxHash}).Error("using savepoint")
			return err
		}
		var flush []smart.FlushInfo
		t.GenBlock = b.GenBlock
		t.TimeLimit = timeLimit
		t.PrevBlock = b.PrevHeader
		msg, flush, err = t.Play(curTx)
		if err == nil && t.TxSmart != nil {
			err = limits.CheckLimit(t)
		}
		if err != nil {
			if flush != nil {
				for i := len(flush) - 1; i >= 0; i-- {
					finfo := flush[i]
					if finfo.Prev == nil {
						if finfo.ID != uint32(len(smart.GetVM().Children)-1) {
							logger.WithFields(log.Fields{"type": consts.ContractError, "value": finfo.ID,
								"len": len(smart.GetVM().Children) - 1}).Error("flush rollback")
						} else {
							smart.GetVM().Children = smart.GetVM().Children[:len(smart.GetVM().Children)-1]
							delete(smart.GetVM().Objects, finfo.Name)
						}
					} else {
						smart.GetVM().Children[finfo.ID] = finfo.Prev
						smart.GetVM().Objects[finfo.Name] = finfo.Info
					}
				}
			}
			if err == custom.ErrNetworkStopping {
				return err
			}

			errRoll := dbTransaction.RollbackSavepoint(consts.SetSavePointMarkBlock(curTx))
			if errRoll != nil {
				logger.WithFields(log.Fields{"type": consts.DBError, "error": err, "tx_hash": t.TxHash}).Error("rolling back to previous savepoint")
				return errRoll
			}
			if b.GenBlock {
				if err == ErrLimitStop {
					if curTx == 0 {
						return err
					}
					break
				}
				if strings.Contains(err.Error(), script.ErrVMTimeLimit.Error()) { // very heavy tx
					err = ErrLimitTime
				}
			}
			// skip this transaction
			transaction.MarkTransactionBad(t.DbTransaction, t.TxHash, err.Error())
			if t.SysUpdate {
				if err := syspar.SysUpdate(t.DbTransaction); err != nil {
					log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("updating syspar")
				}
				t.SysUpdate = false
			}

			if b.GenBlock {
				continue
			}

			return err
		}
		err = dbTransaction.ReleaseSavepoint(curTx, consts.SavePointMarkBlock)
		if err != nil {
			logger.WithFields(log.Fields{"type": consts.DBError, "error": err, "tx_hash": t.TxHash}).Error("releasing savepoint")
		}

		if t.SysUpdate {
			b.SysUpdate = true
			t.SysUpdate = false
		}

		if err := model.SetTransactionStatusBlockMsg(t.DbTransaction, b.Header.BlockID, msg, t.TxHash); err != nil {
			logger.WithFields(log.Fields{"type": consts.DBError, "error": err, "tx_hash": t.TxHash}).Error("updating transaction status block id")
			return err
		}
		if t.Notifications.Size() > 0 {
			b.Notifications = append(b.Notifications, t.Notifications)
		}
		playTxs.UsedTx = append(playTxs.UsedTx, t.TxHash)
		playTxs.Lts = append(playTxs.Lts, &model.LogTransaction{Block: b.Header.BlockID, Hash: t.TxHash})
		playTxs.Rts = append(playTxs.Rts, t.RollBackTx...)
		proccessedTx = append(proccessedTx, t)
	}

	return nil
}

// Check is checking block
func (b *Block) Check() error {
	if b.IsGenesis() {
		return nil
	}
	logger := b.GetLogger()
	if b.PrevHeader == nil || b.PrevHeader.BlockID != b.Header.BlockID-1 {
		if err := b.readPreviousBlockFromBlockchainTable(); err != nil {
			logger.WithFields(log.Fields{"type": consts.InvalidObject}).Error("block id is larger then previous more than on 1")
			return err
		}
	}
	if b.Header.Time > time.Now().Unix() {
		logger.WithFields(log.Fields{"type": consts.ParameterExceeded}).Error("block time is larger than now")
		return ErrIncorrectBlockTime
	}

	// is this block too early? Allowable error = error_time
	if b.PrevHeader != nil {
		// skip time validation for first block
		exists, err := protocols.NewBlockTimeCounter().BlockForTimeExists(time.Unix(b.Header.Time, 0), int(b.Header.NodePosition))
		if err != nil {
			logger.WithFields(log.Fields{"type": consts.BlockError, "error": err}).Error("calculating block time")
			return err
		}

		if exists {
			logger.WithFields(log.Fields{"type": consts.BlockError, "error": err}).Warn("incorrect block time")
			return utils.WithBan(fmt.Errorf("%s %d", ErrIncorrectBlockTime, b.PrevHeader.Time))
		}
	}

	// check each transaction
	txCounter := make(map[int64]int)
	txHashes := make(map[string]struct{})
	for _, t := range b.Transactions {
		hexHash := string(converter.BinToHex(t.TxHash))
		// check for duplicate transactions
		if _, ok := txHashes[hexHash]; ok {
			logger.WithFields(log.Fields{"tx_hash": hexHash, "type": consts.DuplicateObject}).Error("duplicate transaction")
			return utils.ErrInfo(fmt.Errorf("duplicate transaction %s", hexHash))
		}
		txHashes[hexHash] = struct{}{}

		// check for max transaction per user in one block
		//txCounter[t.TxKeyID]++
		if txCounter[t.TxKeyID] > syspar.GetMaxBlockUserTx() {
			return utils.WithBan(utils.ErrInfo(fmt.Errorf("max_block_user_transactions")))
		}

		if err := t.CheckTime(b.Header.Time); err != nil {
			return errors.Wrap(err, "check transaction")
		}
	}

	// hash compare could be failed in the case of fork
	_, err := b.CheckHash()
	if err != nil {
		transaction.CleanCache()
		return err
	}
	return nil
}

// CheckHash is checking hash
func (b *Block) CheckHash() (bool, error) {
	logger := b.GetLogger()
	if b.IsGenesis() {
		return true, nil
	}
	if conf.Config.IsSubNode() {
		return true, nil
	}
	// check block signature
	if b.PrevHeader != nil {
		nodePublicKey, err := syspar.GetNodePublicKeyByPosition(b.Header.NodePosition)
		if err != nil {
			return false, utils.ErrInfo(err)
		}
		if len(nodePublicKey) == 0 {
			logger.WithFields(log.Fields{"type": consts.EmptyObject}).Error("node public key is empty")
			return false, utils.ErrInfo(fmt.Errorf("empty nodePublicKey"))
		}

		signSource := b.Header.ForSign(b.PrevHeader, b.MrklRoot)

		resultCheckSign, err := utils.CheckSign(
			[][]byte{nodePublicKey},
			[]byte(signSource),
			b.Header.Sign,
			true)

		if err != nil {
			if err == crypto.ErrIncorrectSign {
				if !bytes.Equal(b.PrevRollbacksHash, b.PrevHeader.RollbacksHash) {
					return false, ErrIncorrectRollbackHash
				}
			}
			logger.WithFields(log.Fields{"error": err, "type": consts.CryptoError}).Error("checking block header sign")
			return false, utils.ErrInfo(fmt.Errorf("err: %v / block.PrevHeader.BlockID: %d /  block.PrevHeader.Hash: %x / ", err, b.PrevHeader.BlockID, b.PrevHeader.Hash))
		}

		return resultCheckSign, nil
	}

	return true, nil
}

// InsertBlockWOForks is inserting blocks
func InsertBlockWOForks(data []byte, genBlock, firstBlock bool) error {
	block, err := ProcessBlockWherePrevFromBlockchainTable(data, !firstBlock)
	if err != nil {
		return err
	}
	block.GenBlock = genBlock
	if err := block.Check(); err != nil {
		return err
	}

	err = block.PlaySafe()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"block_id": block.Header.BlockID}).Debug("block was inserted successfully")
	return nil
}

var (
	ErrMaxBlockSize    = utils.WithBan(errors.New("Block size exceeds maximum limit"))
	ErrZeroBlockSize   = utils.WithBan(errors.New("Block size is zero"))
	ErrUnmarshallBlock = utils.WithBan(errors.New("Unmarshall block"))
)

// ProcessBlockWherePrevFromBlockchainTable is processing block with in table previous block
func ProcessBlockWherePrevFromBlockchainTable(data []byte, checkSize bool) (*Block, error) {
	if checkSize && int64(len(data)) > syspar.GetMaxBlockSize() {
		log.WithFields(log.Fields{"check_size": checkSize, "size": len(data), "max_size": syspar.GetMaxBlockSize(), "type": consts.ParameterExceeded}).Error("binary block size exceeds max block size")
		return nil, ErrMaxBlockSize
	}

	buf := bytes.NewBuffer(data)
	if buf.Len() == 0 {
		log.WithFields(log.Fields{"type": consts.EmptyObject}).Error("buffer is empty")
		return nil, ErrZeroBlockSize
	}

	block, err := UnmarshallBlock(buf, true)
	if err != nil {
		return nil, errors.Wrap(ErrUnmarshallBlock, err.Error())
	}
	block.BinData = data

	if err := block.readPreviousBlockFromBlockchainTable(); err != nil {
		return nil, err
	}

	return block, nil
}
