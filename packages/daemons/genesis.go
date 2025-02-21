package daemons

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/IBAX-io/go-ibax/packages/network/tcpclient"

	log "github.com/sirupsen/logrus"
	"github.com/IBAX-io/go-ibax/packages/block"
	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/model"
)

func InitialLoad(logger *log.Entry) error {
			return err
		}

		if err := model.UpdateSchema(); err != nil {
			return err
		}
	}

	return nil
}

// init first block from file or from embedded value
func loadFirstBlock(logger *log.Entry) error {
	newBlock, err := os.ReadFile(conf.Config.FirstBlockPath)
	if err != nil && len(conf.Config.NodesAddr) == 0 {
		return errors.Wrap(err, "reading first block from file path")
	}
	if len(conf.Config.NodesAddr) > 0 {
		ctxDone, cancel := context.WithCancel(context.Background())
		defer func() {
			cancel()
		}()
		host, _, err := getHostWithMaxID(ctxDone, logger)
		if err != nil {
			return errors.Wrap(err, "reading host")
		}
		rawBlocksChan, err := tcpclient.GetBlocksBodies(ctxDone, host, 1, true)
		if err != nil {
			return err
		}
		for rawBlock := range rawBlocksChan {
			newBlock = rawBlock
		}
	}
	if err = block.InsertBlockWOForks(newBlock, false, true); err != nil {
		logger.WithFields(log.Fields{"type": consts.ParserError, "error": err}).Error("inserting new block")
		return err
	}

	return nil
}

func firstLoad(logger *log.Entry) error {
	DBLock()
	defer DBUnlock()

	return loadFirstBlock(logger)
}

func needLoad(logger *log.Entry) (bool, error) {
	infoBlock := &model.InfoBlock{}
	_, err := infoBlock.Get()
	if err != nil {
		logger.WithFields(log.Fields{"error": err, "type": consts.DBError}).Error("getting info block")
		return false, err
	}
	// we have empty blockchain, we need to load blockchain from file or other source
	if infoBlock.BlockID == 0 {
		logger.Debug("blockchain should be loaded")
		return true, nil
	}
	return false, nil
}
