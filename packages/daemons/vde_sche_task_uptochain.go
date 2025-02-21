/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package daemons

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	chain_api "github.com/IBAX-io/go-ibax/packages/chain_sdk"

	"path/filepath"

	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/model"

	log "github.com/sirupsen/logrus"
)

//Scheduling task information up the chain
func VDEScheTaskUpToChain(ctx context.Context, d *daemon) error {
	var (
	m := &model.VDEScheTaskChainStatus{}
	ScheTask, err := m.GetAllByContractStateAndChainState(1, 1, 0) //0 contract not install，1 contract installd，2 fail； 0not up to chain
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("getting all untreated task data")
		return err
	}
	if len(ScheTask) == 0 {
		//log.Info("Sche task not found")
		time.Sleep(time.Millisecond * 100)
		return nil
	}
	chaininfo := &model.VDEScheChainInfo{}
	ScheChainInfo, err := chaininfo.Get()
	if err != nil {
		//log.WithFields(log.Fields{"error": err}).Error("VDE Sche uptochain getting chain info")
		log.Info("Sche chain info not found")
		time.Sleep(time.Second * 2)
		return err
	}
	if ScheChainInfo == nil {
		log.Info("Sche chain info not found")
		//fmt.Println("Src chain info not found")
		time.Sleep(time.Second * 2)
		return nil
	}
	blockchain_http = ScheChainInfo.BlockchainHttp
	blockchain_ecosystem = ScheChainInfo.BlockchainEcosystem
	//fmt.Println("ScheChainInfo:", blockchain_http, blockchain_ecosystem)

	// deal with task data
	for _, item := range ScheTask {
		fmt.Println("ScheTask:", item.TaskUUID)

		ecosystemID, err := strconv.Atoi(blockchain_ecosystem)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("VDEScheTaskUpToChain encode error")
			time.Sleep(2 * time.Second)
			continue
		}
		chain_apiAddress := blockchain_http
		chain_apiEcosystemID := int64(ecosystemID)

		src := filepath.Join(conf.Config.KeysDir, "PrivateKey")
		// Login
		gAuth_chain, _, gPrivate_chain, _, _, err := chain_api.KeyLogin(chain_apiAddress, src, chain_apiEcosystemID)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Login chain failure")
			time.Sleep(2 * time.Second)
			continue
		}
		//fmt.Println("Login OK!")

		form := url.Values{
			"TaskUUID":     {item.TaskUUID},
			"TaskName":     {item.TaskName},
			"TaskSender":   {item.TaskSender},
			"TaskReceiver": {item.TaskReceiver},
			"Comment":      {item.Comment},
			"Parms":        {item.Parms},
			"TaskType":     {converter.Int64ToStr(item.TaskType)},
			"TaskState":    {converter.Int64ToStr(item.TaskState)},

			"ContractSrcName":     {item.ContractSrcName},
			"ContractSrcGet":      {item.ContractSrcGet},
			"ContractSrcGetHash":  {item.ContractSrcGetHash},
			"ContractDestName":    {item.ContractDestName},
			"ContractDestGet":     {item.ContractDestGet},
			"ContractDestGetHash": {item.ContractDestGetHash},

			"ContractRunHttp":      {item.ContractRunHttp},
			"ContractRunEcosystem": {item.ContractRunEcosystem},
			"ContractRunParms":     {item.ContractRunParms},

			"ContractMode": {converter.Int64ToStr(item.ContractMode)},
			`CreateTime`:   {converter.Int64ToStr(time.Now().Unix())},
		}

		ContractName := `@1VDEShareTaskCreate`
		_, txHash, _, err := chain_api.VDEPostTxResult(chain_apiAddress, chain_apiEcosystemID, gAuth_chain, gPrivate_chain, ContractName, &form)
		if err != nil {
			fmt.Println("Send VDEScheTask to chain err: ", err)
			log.WithFields(log.Fields{"error": err}).Error("Send VDEScheTask to chain!")
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Println("Send chain Contract to run, ContractName:", ContractName)

		item.ChainState = 1
		item.TxHash = txHash
		item.BlockId = 0
		item.ChainErr = ""
		item.UpdateTime = time.Now().Unix()
		err = item.Updates()
		if err != nil {
			fmt.Println("Update VDEScheTask table err: ", err)
			log.WithFields(log.Fields{"error": err}).Error("Update VDEScheTask table!")
			time.Sleep(time.Millisecond * 100)
			continue
		}
	} //for
	time.Sleep(time.Millisecond * 100)
	return nil
}

//Query the status of the chain on the scheduling task information
func VDEScheTaskUpToChainState(ctx context.Context, d *daemon) error {
	var (
		blockchain_http      string
		blockchain_ecosystem string
		err                  error
	)

	m := &model.VDEScheTaskChainStatus{}
	ScheTask, err := m.GetAllByContractStateAndChainState(1, 1, 1) //0 not install contract，1 installed，2 fail； 1up to chain
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("getting all untreated task data")
		time.Sleep(time.Millisecond * 2)
		return err
	}
	if len(ScheTask) == 0 {
		//log.Info("Sche task not found")
		time.Sleep(time.Millisecond * 2)
		return nil
	}
	chaininfo := &model.VDEScheChainInfo{}
	ScheChainInfo, err := chaininfo.Get()
	if err != nil {
		//log.WithFields(log.Fields{"error": err}).Error("VDE Sche uptochain getting chain info")
		log.Info("Sche chain info not found")
		time.Sleep(time.Millisecond * 100)
		return err
	}
	if ScheChainInfo == nil {
		log.Info("Sche chain info not found")
		//fmt.Println("Src chain info not found")
		time.Sleep(time.Millisecond * 100)
		return nil
	}
	blockchain_http = ScheChainInfo.BlockchainHttp
	blockchain_ecosystem = ScheChainInfo.BlockchainEcosystem
	//fmt.Println("ScheChainInfo:", blockchain_http, blockchain_ecosystem)

	// deal with task data
	for _, item := range ScheTask {
		fmt.Println("ScheTask:", item.TaskUUID)

		ecosystemID, err := strconv.Atoi(blockchain_ecosystem)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("encode error")
			time.Sleep(time.Millisecond * 2)
			continue
		}
		chain_apiAddress := blockchain_http
		chain_apiEcosystemID := int64(ecosystemID)

		src := filepath.Join(conf.Config.KeysDir, "PrivateKey")
		// Login
		gAuth_chain, _, _, _, _, err := chain_api.KeyLogin(chain_apiAddress, src, chain_apiEcosystemID)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Login chain failure")
			time.Sleep(time.Millisecond * 2)
			continue
		}
		//fmt.Println("Login OK!")

		blockId, err := chain_api.VDEWaitTx(chain_apiAddress, gAuth_chain, string(item.TxHash))
		if blockId > 0 {
			item.BlockId = blockId
			item.ChainId = converter.StrToInt64(err.Error())
			item.ChainState = 2
			item.ChainErr = ""

		} else if blockId == 0 {
			//item.ChainState = 3
			item.ChainState = 1 //
			item.ChainErr = err.Error()
		} else {
			//fmt.Println("VDEWaitTx! err: ", err)
			time.Sleep(time.Millisecond * 2)
			continue
		}
		err = item.Updates()
		if err != nil {
			fmt.Println("Update VDEScheTask table err: ", err)
			log.WithFields(log.Fields{"error": err}).Error("Update VDEScheTask table!")
			time.Sleep(time.Millisecond * 2)
			continue
		}
		fmt.Println("VDE Sche Run chain Contract ok, TxHash:", string(item.TxHash))
	} //for
	return nil
}
