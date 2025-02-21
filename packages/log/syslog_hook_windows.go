// +build windows

/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package log

import (
	"github.com/sirupsen/logrus"
)

// SyslogHook to send logs via syslog.
type SyslogHook struct {
	SyslogNetwork string
	SyslogRaddr   string
}

func NewSyslogHook(appName, facility string) (*SyslogHook, error) {

func (hook *SyslogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
