/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package syspar

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/crypto"
	"github.com/IBAX-io/go-ibax/packages/model"

	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// NumberNodes is the number of nodes
	NumberNodes = `number_of_nodes`
	// FuelRate is the rate
	FuelRate = `fuel_rate`
	// HonorNodes is the list of nodes
	HonorNodes = `honor_nodes`
	// GapsBetweenBlocks is the time between blocks
	MaxColumns = `max_columns`
	// MaxIndexes is the maximum indexes in tables
	MaxIndexes = `max_indexes`
	// MaxBlockUserTx is the maximum number of user's transactions in one block
	MaxBlockUserTx = `max_tx_block_per_user`
	// SizeFuel is the fuel cost of 1024 bytes of the transaction data
	SizeFuel = `price_tx_data`
	// TaxesWallet is the address for taxess
	TaxesWallet = `taxes_wallet`
	// RbBlocks1 rollback from queue_bocks
	RbBlocks1 = `rollback_blocks`
	// BlockReward value of reward, which is chrged on block generation
	BlockReward = "block_reward"
	// IncorrectBlocksPerDay is value of incorrect blocks per day before global ban
	IncorrectBlocksPerDay = `incorrect_blocks_per_day`
	// NodeBanTime is value of ban time for bad nodes (in ms)
	NodeBanTime = `node_ban_time`
	// LocalNodeBanTime is value of local ban time for bad nodes (in ms)
	LocalNodeBanTime = `local_node_ban_time`
	// TaxesSize is the value of the taxes
	TaxesSize = `taxes_size`
	// PriceTxSizeWallet is the size of a user's resource in the database
	PriceTxSizeWallet = `price_tx_size_wallet`
	// PriceCreateRate is new element rate, include table、contract、column、ecosystem、page、menu
	PriceCreateRate = `price_create_rate`
	// Test equals true or 1 if we have a test blockchain
	Test = `test`
	// PrivateBlockchain is value defining blockchain mode
	PrivateBlockchain = `private_blockchain`

	// CostDefault is the default maximum cost of F
	CostDefault = int64(20000000)

	PriceExec       = "price_exec_"
	AccessExec      = "access_exec_"
	PayFreeContract = "pay_free_contract"
)

var (
	cache             = map[string]string{}
	nodes             = make(map[string]*HonorNode)
	nodesByPosition   = make([]*HonorNode, 0)
	fuels             = make(map[int64]string)
	wallets           = make(map[int64]string)
	mutex             = &sync.RWMutex{}
	firstBlockData    *consts.FirstBlock
	errFirstBlockData = errors.New("Failed to get data of the first block")
	errNodeDisabled   = errors.New("node is disabled")
	nodePubKey        []byte
	nodePrivKey       []byte
	cacheTableColType = make([]map[string]string, 0)
)

func ReadNodeKeys() (err error) {
	var (
		nprivkey []byte
	)
	nprivkey, err = os.ReadFile(filepath.Join(conf.Config.KeysDir, consts.NodePrivateKeyFilename))
	if err != nil {
		log.WithFields(log.Fields{"type": consts.IOError, "error": err}).Error("reading node private key from file")
		return
	}
	nodePrivKey, err = hex.DecodeString(string(nprivkey))
	if err != nil {
		log.WithFields(log.Fields{"type": consts.ConversionError, "error": err}).Error("decoding private key from hex")
		return
	}
	nodePubKey, err = crypto.PrivateToPublic(nodePrivKey)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.CryptoError, "error": err}).Error("converting node private key to public")
		return
	}
	return
}

func GetNodePubKey() []byte {
	return nodePubKey
}

func GetNodePrivKey() []byte {
	return nodePrivKey
}

// SysUpdate reloads/updates values of system parameters
func SysUpdate(dbTransaction *model.DbTransaction) error {
	var err error
	systemParameters, err := model.GetAllSystemParameters(dbTransaction)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting all system parameters")
		return err
	}
	mutex.Lock()
	defer mutex.Unlock()
	for _, param := range systemParameters {
		cache[param.Name] = param.Value
	}
	if len(cache[HonorNodes]) > 0 {
		if err = updateNodes(); err != nil {
			return err
		}
	}
	getParams := func(name string) (map[int64]string, error) {
		res := make(map[int64]string)
		if len(cache[name]) > 0 {
			ifuels := make([][]string, 0)
			err = json.Unmarshal([]byte(cache[name]), &ifuels)
			if err != nil {
				log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling params from json")
				return res, err
			}
			for _, item := range ifuels {
				if len(item) < 2 {
					continue
				}
				res[converter.StrToInt64(item[0])] = item[1]
			}
		}
		return res, nil
	}
	fuels, err = getParams(FuelRate)
	wallets, err = getParams(TaxesWallet)

	return err
}

func updateNodes() (err error) {
	items := make([]*HonorNode, 0)
	if len(cache[HonorNodes]) > 0 {
		err = json.Unmarshal([]byte(cache[HonorNodes]), &items)

		if err != nil {
			log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err, "v": cache[HonorNodes]}).Error("unmarshalling honor nodes from json")
			return err
		}
	}
	if len(items) > 1 {
		if err = DuplicateHonorNode(items); err != nil {
			return err
		}
	}
	nodes = make(map[string]*HonorNode)
	nodesByPosition = []*HonorNode{}
	for i := 0; i < len(items); i++ {
		nodes[hex.EncodeToString(items[i].PublicKey)] = items[i]

		if !items[i].Stopped {
			nodesByPosition = append(nodesByPosition, items[i])
		}
	}

	return nil
}

// addHonorNodeKeys adds node by keys to list of nodes
func addHonorNodeKeys(publicKey []byte) {
	nodesByPosition = append(nodesByPosition, &HonorNode{
		PublicKey: publicKey,
	})
}

func GetNodes() []HonorNode {
	mutex.RLock()
	defer mutex.RUnlock()

	result := make([]HonorNode, 0, len(nodesByPosition))
	for _, node := range nodesByPosition {
		result = append(result, *node)
	}

	return result
}

func GetThisNodePosition() (int64, error) {
	return GetNodePositionByPublicKey(GetNodePubKey())
}

func GetHonorNodeType() bool {
	d, err := GetNodePositionByPublicKey(GetNodePubKey())
	if err == nil {
		return true
	}
	if d == 0 && err != nil {
		return false
	}
	return false
}

// GetNodePositionByKeyID is returning node position by key id
func GetNodePositionByPublicKey(publicKey []byte) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	for i, item := range nodesByPosition {
		if item.Stopped {
			if bytes.Equal(item.PublicKey, publicKey) {
				return 0, errNodeDisabled
			}
			continue
		}

		if bytes.Equal(item.PublicKey, publicKey) {
			return int64(i), nil
		}
	}

	return 0, fmt.Errorf("Incorrect public key")
}

// GetCountOfActiveNodes is count of nodes with stopped = false
func GetCountOfActiveNodes() int64 {
	return int64(len(nodesByPosition))
}

// GetNumberOfNodes is count number of nodes
func GetNumberOfNodes() int64 {
	return int64(len(nodesByPosition))
}

func GetNumberOfNodesFromDB(transaction *model.DbTransaction) int64 {
	sp := &model.SystemParameter{}
	sp.GetTransaction(transaction, HonorNodes)
	var honorNodes []map[string]interface{}
	if len(sp.Value) > 0 {
		if err := json.Unmarshal([]byte(sp.Value), &honorNodes); err != nil {
			log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err, "value": sp.Value}).Error("unmarshalling honor nodes from JSON")
		}
	}
	if len(honorNodes) == 0 {
		return 1
	}
	return int64(len(honorNodes))
}

// GetNodeByPosition is retrieving node by position
func GetNodeByPosition(position int64) (*HonorNode, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	if int64(len(nodesByPosition)) <= position {
		return nil, fmt.Errorf("incorrect position")
	}
	return nodesByPosition[position], nil
}

func GetNodeByHost(host string) (HonorNode, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	for _, n := range nodes {
		if n.TCPAddress == host {
			return *n, nil
		}
	}

	return HonorNode{}, fmt.Errorf("incorrect host")
}

// GetNodeHostByPosition is retrieving node host by position
func GetNodeHostByPosition(position int64) (string, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	nodeData, err := GetNodeByPosition(position)
	if err != nil {
		return "", err
	}
	return nodeData.TCPAddress, nil
}

// GetNodePublicKeyByPosition is retrieving node public key by position
func GetNodePublicKeyByPosition(position int64) ([]byte, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	if int64(len(nodesByPosition)) <= position {
		return nil, fmt.Errorf("incorrect position")
	}
	nodeData, err := GetNodeByPosition(position)
	if err != nil {
		return nil, err
	}

	return nodeData.PublicKey, nil
}

// SysInt64 is converting sys string to int64
func SysInt64(name string) int64 {
	return converter.StrToInt64(SysString(name))
}

// SysInt is converting sys string to int
func SysInt(name string) int {
	return converter.StrToInt(SysString(name))
}

// GetSizeFuel is returns fuel size
func GetSizeFuel() int64 {
	return SysInt64(SizeFuel)
}

// GetFuelRate is returning fuel rate
func GetFuelRate(ecosystem int64) string {
	mutex.RLock()
	defer mutex.RUnlock()
	if ret, ok := fuels[ecosystem]; ok {
		return ret
	}
	return fuels[1]
}

func IsFuelRate(ecosystem int64) (string, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	if ret, ok := fuels[ecosystem]; ok {
		return ret, ok
	}
	return "", false
}

// GetTaxesWallet is returns taxes wallet
func GetTaxesWallet(ecosystem int64) string {
	mutex.RLock()
	defer mutex.RUnlock()
	if ret, ok := wallets[ecosystem]; ok {
		return ret
	}
	return wallets[1]
}

// IsTaxesWallet is returns taxes exist
func IsTaxesWallet(ecosystem int64) (string, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	if ret, ok := wallets[ecosystem]; ok {
		return ret, ok
	}
	return "", false
}

// GetMaxBlockSize is returns max block size
func GetMaxBlockSize() int64 {
	return converter.StrToInt64(SysString(MaxBlockSize))
}

// GetMaxBlockFuel is returns max block fuel
func GetMaxBlockFuel() int64 {
	return converter.StrToInt64(SysString(MaxBlockFuel))
}

// GetMaxTxFuel is returns max tx fuel
func GetMaxTxFuel() int64 {
	return converter.StrToInt64(SysString(MaxTxFuel))
}

// GetMaxBlockGenerationTime is returns max block generation time (in ms)
func GetMaxBlockGenerationTime() int64 {
	return converter.StrToInt64(SysString(MaxBlockGenerationTime))
}

// GetMaxTxSize is returns max tx size
func GetMaxTxSize() int64 {
	return converter.StrToInt64(SysString(MaxTxSize))
}

// GetMaxTxTextSize is returns max tx text size
func GetMaxForsignSize() int64 {
	return converter.StrToInt64(SysString(MaxForsignSize))
}

// GetGapsBetweenBlocks is returns gaps between blocks
func GetGapsBetweenBlocks() int64 {
	return converter.StrToInt64(SysString(GapsBetweenBlocks))
}

// GetMaxTxCount is returns max tx count
func GetMaxTxCount() int {
	return converter.StrToInt(SysString(MaxTxCount))
}

// GetMaxColumns is returns max columns
func GetMaxColumns() int {
	return converter.StrToInt(SysString(MaxColumns))
}

// GetMaxIndexes is returns max indexes
func GetMaxIndexes() int {
	return converter.StrToInt(SysString(MaxIndexes))
}

// GetMaxBlockUserTx is returns max tx block user
func GetMaxBlockUserTx() int {
	return converter.StrToInt(SysString(MaxBlockUserTx))
}

func IsTestMode() bool {
	return SysString(Test) == `true` || SysString(Test) == `1`
}

func GetIncorrectBlocksPerDay() int {
	return converter.StrToInt(SysString(IncorrectBlocksPerDay))
}

func GetNodeBanTime() time.Duration {
	return time.Millisecond * time.Duration(converter.StrToInt64(SysString(NodeBanTime)))
}

func GetLocalNodeBanTime() time.Duration {
	return time.Millisecond * time.Duration(converter.StrToInt64(SysString(LocalNodeBanTime)))
}

// GetRemoteHosts returns array of hostnames excluding myself
func GetDefaultRemoteHosts() []string {
	ret := make([]string, 0)

	mutex.RLock()
	defer mutex.RUnlock()

	nodeKey := hex.EncodeToString(GetNodePubKey())
	for pubKey, item := range nodes {
		if pubKey != nodeKey && !item.Stopped {
			ret = append(ret, item.TCPAddress)
		}
	}
	if len(ret) == 0 && len(conf.Config.NodesAddr) > 0 {
		ret = append(ret, conf.Config.NodesAddr[0])
	}
	return ret
}

// GetRemoteHosts returns array of hostnames excluding myself
func GetRemoteHosts() []string {
	ret := make([]string, 0)

	mutex.RLock()
	defer mutex.RUnlock()

	nodeKey := hex.EncodeToString(GetNodePubKey())
	for pubKey, item := range nodes {
		if pubKey != nodeKey && !item.Stopped {
			ret = append(ret, item.TCPAddress)
		}
	}
	return ret
}

// SysString returns string value of the system parameter
func SysString(name string) string {
	mutex.RLock()
	ret := cache[name]
	mutex.RUnlock()
	return ret
}

// GetRbBlocks1 is returns RbBlocks1
func GetRbBlocks1() int64 {
	return SysInt64(RbBlocks1)
}

// HasSys returns boolean whether this system parameter exists
func HasSys(name string) bool {
	mutex.RLock()
	_, ok := cache[name]
	mutex.RUnlock()
	return ok
}

// SetFirstBlockData sets data of first block to global variable
func SetFirstBlockData(data *consts.FirstBlock) {
	mutex.Lock()
	defer mutex.Unlock()

	firstBlockData = data

	// If list of nodes is empty, then used node from the first block
	if len(nodesByPosition) == 0 {
		addHonorNodeKeys(firstBlockData.NodePublicKey)

		nodesByPosition = []*HonorNode{&HonorNode{
			PublicKey: firstBlockData.NodePublicKey,
			Stopped:   false,
		}}
	}
}

// GetFirstBlockData gets data of first block from global variable
func GetFirstBlockData() (*consts.FirstBlock, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	if firstBlockData == nil {
		return nil, errFirstBlockData
	}

	return firstBlockData, nil
}

// IsPrivateBlockchain returns the value of private_blockchain system parameter or true
func IsPrivateBlockchain() bool {
	par := SysString(PrivateBlockchain)
	return len(par) > 0 && par != `0` && par != `false`
}

func GetMaxCost() int64 {
	cost := GetMaxTxFuel()
	if cost == 0 {
		cost = CostDefault
	}
	return cost
}

func GetAccessExec(s string) string {
	return SysString(AccessExec + s)
}

func GetPriceExec(s string) (price int64, ok bool) {
	if ok = HasSys(PriceExec + s); !ok {
		return
	}
	price = SysInt64(PriceExec + s)
	return
}

// SysTableColType reloads/updates values of all ecosystem table column data type
func SysTableColType(dbTransaction *model.DbTransaction) error {
	var err error
	mutex.RLock()
	defer mutex.RUnlock()
	cacheTableColType, err = model.GetAllTransaction(dbTransaction, `
		SELECT table_name,column_name,data_type,character_maximum_length
		FROM information_schema.columns Where table_schema NOT IN ('pg_catalog', 'information_schema') AND table_name ~ '[\d]' AND data_type = 'bytea' ORDER BY ordinal_position ASC;`, -1)
	if err != nil {
		return err
	}
	return nil
}

func GetTableColType() []map[string]string {
	mutex.RLock()
	defer mutex.RUnlock()
	return cacheTableColType
}

func IsByteColumn(table, column string) bool {
	for _, row := range GetTableColType() {
		if row["table_name"] == table && row["column_name"] == column {
			return true
		}
	}
	return false
}
