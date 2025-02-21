/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package migration

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/migration/updates"

	log "github.com/sirupsen/logrus"
)

const (
	eVer = `Wrong version %s`
)

var migrations = []*migration{
	// Inital migration
	&migration{"0.0.1", migrationInitial, true},

	// Initial schema
	&migration{"0.1.5", migrationInitialTables, true},
	&migration{"0.1.6", migrationInitialSchema, false},
}

var migrationsSub = &migration{"0.1.7", migrationInitialTablesSub, true}

var migrationsCLB = &migration{"0.1.8", migrationInitialTablesCLB, true}

var updateMigrations = []*migration{
	&migration{"3.1.0", updates.M310, false},
	&migration{"3.2.0", updates.M320, false},

type database interface {
	CurrentVersion() (string, error)
	ApplyMigration(string, string) error
}

func compareVer(a, b string) (int, error) {
	var (
		av, bv []string
		ai, bi int
		err    error
	)
	if av = strings.Split(a, `.`); len(av) != 3 {
		return 0, fmt.Errorf(eVer, a)
	}
	if bv = strings.Split(b, `.`); len(bv) != 3 {
		return 0, fmt.Errorf(eVer, b)
	}
	for i, v := range av {
		if ai, err = strconv.Atoi(v); err != nil {
			return 0, fmt.Errorf(eVer, a)
		}
		if bi, err = strconv.Atoi(bv[i]); err != nil {
			return 0, fmt.Errorf(eVer, b)
		}
		if ai < bi {
			return -1, nil
		}
		if ai > bi {
			return 1, nil
		}
	}
	return 0, nil
}

func migrate(db database, appVer string, migrations []*migration) error {
	dbVerString, err := db.CurrentVersion()
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "err": err}).Errorf("parse version")
		return err
	}

	if cmp, err := compareVer(dbVerString, appVer); err != nil {
		log.WithFields(log.Fields{"type": consts.MigrationError, "err": err}).Errorf("parse version")
		return err
	} else if cmp >= 0 {
		return nil
	}

	for _, m := range migrations {
		if cmp, err := compareVer(dbVerString, m.version); err != nil {
			log.WithFields(log.Fields{"type": consts.MigrationError, "err": err}).Errorf("parse version")
			return err
		} else if cmp >= 0 {
			continue
		}
		if m.template {
			m.data, err = sqlConvert([]string{m.data})
			if err != nil {
				return err
			}
		}
		err = db.ApplyMigration(m.version, m.data)
		if err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "err": err, "version": m.version}).Errorf("apply migration")
			return err
		}

		log.WithFields(log.Fields{"version": m.version}).Debug("apply migration")
	}

	return nil
}

func runMigrations(db database, migrationList []*migration) error {
	return migrate(db, consts.VERSION, migrationList)
}

// InitMigrate applies initial migrations
func InitMigrate(db database) error {
	mig := migrations
	if conf.Config.IsSubNode() {
		mig = append(mig, migrationsSub)
	}
	if conf.Config.IsSupportingOBS() {
		mig = append(mig, migrationsCLB)
	}
	return runMigrations(db, mig)
}

// UpdateMigrate applies update migrations
func UpdateMigrate(db database) error {
	return runMigrations(db, updateMigrations)
}
