/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package daemons

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBAX-io/go-ibax/packages/model"

	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/statsd"
	"github.com/IBAX-io/go-ibax/packages/utils"

	log "github.com/sirupsen/logrus"
)

var (
	// MonitorDaemonCh is monitor daemon channel
	MonitorDaemonCh = make(chan []string, 100)
	NtpDriftFlag    = false
)

type daemon struct {
	goRoutineName string
	sleepTime     time.Duration
	logger        *log.Entry
	atomic        uint32
}

var daemonsList = map[string]func(context.Context, *daemon) error{
	"BlocksCollection":  BlocksCollection,
	"BlockGenerator":    BlockGenerator,
	"Disseminator":      Disseminator,
	"QueueParserTx":     QueueParserTx,
	"QueueParserBlocks": QueueParserBlocks,
	"Confirmations":     Confirmations,
	"Scheduler":         Scheduler,
	"ExternalNetwork":   ExternalNetwork,

	"SubNodeSrcTaskInstallChannel": SubNodeSrcTaskInstallChannel,
	"SubNodeSrcData":               SubNodeSrcData,
	"SubNodeSrcDataStatus":         SubNodeSrcDataStatus,
	"SubNodeSrcDataStatusAgent":    SubNodeSrcDataStatusAgent,
	"SubNodeAgentData":             SubNodeAgentData,
	"SubNodeSrcDataUpToChain":      SubNodeSrcDataUpToChain,
	"SubNodeSrcHashUpToChainState": SubNodeSrcHashUpToChainState,
	"SubNodeDestData":              SubNodeDestData,

	"VDESrcDataStatus":                      VDESrcDataStatus,
	"VDESrcDataStatusAgent":                 VDESrcDataStatusAgent,
	"VDEAgentData":                          VDEAgentData,
	"VDESrcData":                            VDESrcData,
	"VDEScheTaskUpToChain":                  VDEScheTaskUpToChain,
	"VDEScheTaskUpToChainState":             VDEScheTaskUpToChainState,
	"VDESrcTaskUpToChain":                   VDESrcTaskUpToChain,
	"VDESrcTaskUpToChainState":              VDESrcTaskUpToChainState,
	"VDEDestTaskSrcGetFromChain":            VDEDestTaskSrcGetFromChain,
	"VDEDestTaskScheGetFromChain":           VDEDestTaskScheGetFromChain,
	"VDESrcTaskScheGetFromChain":            VDESrcTaskScheGetFromChain,
	"VDEScheTaskInstallContractSrc":         VDEScheTaskInstallContractSrc,
	"VDEScheTaskInstallContractDest":        VDEScheTaskInstallContractDest,
	"VDESrcTaskInstallContractSrc":          VDESrcTaskInstallContractSrc,
	"VDEDestTaskInstallContractDest":        VDEDestTaskInstallContractDest,
	"VDEDestData":                           VDEDestData,
	"VDEDestDataStatus":                     VDEDestDataStatus,
	"VDESrcHashUpToChain":                   VDESrcHashUpToChain,
	"VDESrcHashUpToChainState":              VDESrcHashUpToChainState,
	"VDESrcLogUpToChain":                    VDESrcLogUpToChain,
	"VDESrcLogUpToChainState":               VDESrcLogUpToChainState,
	"VDEDestLogUpToChain":                   VDEDestLogUpToChain,
	"VDEDestLogUpToChainState":              VDEDestLogUpToChainState,
	"VDEDestDataHashGetFromChain":           VDEDestDataHashGetFromChain,
	"VDESrcTaskStatus":                      VDESrcTaskStatus,
	"VDESrcTaskStatusRun":                   VDESrcTaskStatusRun,
	"VDESrcTaskStatusRunState":              VDESrcTaskStatusRunState,
	"VDESrcTaskFromScheStatus":              VDESrcTaskFromScheStatus,
	"VDESrcTaskFromScheStatusRun":           VDESrcTaskFromScheStatusRun,
	"VDESrcTaskFromScheStatusRunState":      VDESrcTaskFromScheStatusRunState,
	"VDEAgentLogUpToChain":                  VDEAgentLogUpToChain,
	"VDEScheTaskChainStatus":                VDEScheTaskChainStatus,
	"VDEScheTaskChainStatusState":           VDEScheTaskChainStatusState,
	"VDESrcTaskChainStatus":                 VDESrcTaskChainStatus,
	"VDESrcTaskChainStatusState":            VDESrcTaskChainStatusState,
	"VDESrcTaskAuthChainStatus":             VDESrcTaskAuthChainStatus,
	"VDESrcTaskAuthChainStatusState":        VDESrcTaskAuthChainStatusState,
	"VDEScheTaskSrcGetFromChain":            VDEScheTaskSrcGetFromChain,
	"VDEScheTaskFromSrcInstallContractSrc":  VDEScheTaskFromSrcInstallContractSrc,
	"VDEScheTaskFromSrcInstallContractDest": VDEScheTaskFromSrcInstallContractDest,
}

var rollbackList = []string{
	"BlocksCollection",
	"Confirmations",
}

func daemonLoop(ctx context.Context, goRoutineName string, handler func(context.Context, *daemon) error, retCh chan string) {
	logger := log.WithFields(log.Fields{"daemon_name": goRoutineName})
	defer func() {
		if r := recover(); r != nil {
			logger.WithFields(log.Fields{"type": consts.PanicRecoveredError, "error": r}).Error("panic in daemon")
			panic(r)
		}
	}()

	err := WaitDB(ctx)
	if err != nil {
		return
			retCh <- goRoutineName
			return
		case <-idleDelay.C:
			MonitorDaemonCh <- []string{d.goRoutineName, converter.Int64ToStr(time.Now().Unix())}
			startTime := time.Now()
			counterName := statsd.DaemonCounterName(goRoutineName)
			handler(ctx, d)
			statsd.Client.TimingDuration(counterName+statsd.Time, time.Now().Sub(startTime), 1.0)
		}
	}
}

// StartDaemons starts daemons
func StartDaemons(ctx context.Context, daemonsToStart []string) {
	go WaitStopTime()

	daemonsTable := make(map[string]string)
	go func() {
		for {
			daemonNameAndTime := <-MonitorDaemonCh
			daemonsTable[daemonNameAndTime[0]] = daemonNameAndTime[1]
			if time.Now().Unix()%10 == 0 {
				log.Debug("daemonsTable: %v\n", daemonsTable)
			}
		}
	}()

	//go Ntp_Work(ctx)
	// ctx, cancel := context.WithCancel(context.Background())
	// utils.CancelFunc = cancel
	// utils.ReturnCh = make(chan string)

	if conf.Config.TestRollBack {
		daemonsToStart = rollbackList
	}

	log.WithFields(log.Fields{"daemons_to_start": daemonsToStart}).Info("starting daemons")

	for _, name := range daemonsToStart {
		handler, ok := daemonsList[name]
		if ok {
			go daemonLoop(ctx, name, handler, utils.ReturnCh)
			log.WithFields(log.Fields{"daemon_name": name}).Info("started")
			utils.DaemonsCount++
			continue
		}

		log.WithFields(log.Fields{"daemon_name": name}).Warning("unknown daemon name")
	}
}

func getHostPort(h string) string {
	if strings.Contains(h, ":") {
		return h
	}
	return fmt.Sprintf("%s:%d", h, consts.DEFAULT_TCP_PORT)
}

//ntp
func Ntp_Work(ctx context.Context) {
	var count = 0
	for {
		select {
		case <-ctx.Done():
			log.Error("Ntp_Work done his work")
			return
		case <-time.After(time.Second * 4):
			b, err := utils.CheckClockDrift()
			if err != nil {
				log.WithFields(log.Fields{"daemon_name Ntp_Work err": err.Error()}).Error("Ntp_Work")
			} else {
				if b {
					NtpDriftFlag = true
					count = 0
				} else {
					count++
				}
				if count > 10 {
					var sp model.SystemParameter
					count, err := sp.GetNumberOfHonorNodes()
					if err != nil {
						log.WithFields(log.Fields{"Ntp_Work GetNumberOfHonorNodes  err": err.Error()}).Error("GetNumberOfHonorNodes")
					} else {
						if NtpDriftFlag && count > 1 {
							NtpDriftFlag = false
						}
					}

				}
			}

		}
	}
}
