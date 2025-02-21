/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
package notificator

import (
	"encoding/json"

	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/model"
	"github.com/IBAX-io/go-ibax/packages/publisher"

	log "github.com/sirupsen/logrus"
)

type notificationRecord struct {
	EcosystemID  string `json:"ecosystem"`
	RoleID       string `json:"role_id"`
	RecordsCount int64  `json:"count"`
}

// UpdateNotifications send stats about unreaded messages to centrifugo for ecosystem
func UpdateNotifications(ecosystemID int64, accounts []string) {
	notificationsStats, err := getEcosystemNotificationStats(ecosystemID, accounts)
	if err != nil {
		return
	}

	for account, n := range notificationsStats {
		sendUserStats(account, *n)
	}
}

// UpdateRolesNotifications send stats about unreaded messages to centrifugo for ecosystem
func UpdateRolesNotifications(ecosystemID int64, roles []int64) {
	members, _ := model.GetRoleMembers(nil, ecosystemID, roles)
	UpdateNotifications(ecosystemID, members)
}

func getEcosystemNotificationStats(ecosystemID int64, users []string) (map[string]*[]notificationRecord, error) {
	result, err := model.GetNotificationsCount(ecosystemID, users)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting notification count")
		return nil, err
	}

	return parseRecipientNotification(result, ecosystemID), nil
}

func parseRecipientNotification(rows []model.NotificationsCount, systemID int64) map[string]*[]notificationRecord {
	recipientNotifications := make(map[string]*[]notificationRecord)

	for _, r := range rows {
		if r.RecipientID == 0 {
			continue
		}

		}

		recipientNotifications[r.Account] = &records
	}

	return recipientNotifications
}

func sendUserStats(account string, stats []notificationRecord) {
	rawStats, err := json.Marshal(stats)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.JSONMarshallError, "error": err}).Error("notification statistic")
	}

	err = publisher.Write(account, string(rawStats))
	if err != nil {
		log.WithFields(log.Fields{"type": consts.IOError, "error": err}).Debug("writing to centrifugo")
	}
}
