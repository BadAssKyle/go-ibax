package model

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/IBAX-io/go-ibax/packages/conf"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/crypto"
	"github.com/IBAX-io/go-ibax/packages/migration"
	"github.com/IBAX-io/go-ibax/packages/migration/obs"
)

// ExecSchemaEcosystem is executing ecosystem schema
func ExecSchemaEcosystem(db *DbTransaction, id int, wallet int64, name string, founder, appID int64) error {
	if id == 1 {
		q, err := migration.GetCommonEcosystemScript()
		if err != nil {
			return err
		}
		if err := GetDB(db).Exec(q).Error; err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("executing comma ecosystem schema")
			return err
		}
	}
	q, err := migration.GetEcosystemScript(id, wallet, name, founder, appID)
	if err != nil {
		return err
	}
	if err := GetDB(db).Exec(q).Error; err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("executing ecosystem schema")
		return err
	}
	if id == 1 {
		q, err = migration.GetFirstEcosystemScript(wallet)
		if err != nil {
			return err
		}
		if err := GetDB(db).Exec(q).Error; err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("executing first ecosystem schema")
		}
		q, err = migration.GetFirstTableScript(id)
		if err != nil {
			return err
		}
		if err := GetDB(db).Exec(q).Error; err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("executing first tables schema")
		}
	}
	return nil
}

func ExecSubSchema() error {
	if conf.Config.IsSubNode() {
		if err := migration.InitMigrate(&MigrationHistory{}); err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("on executing obs script")
			return err
		}
	}
	return nil
}

// ExecOBSSchema is executing schema for off blockchainService
func ExecOBSSchema(id int, wallet int64) error {

	if conf.Config.IsSupportingOBS() {
		if err := migration.InitMigrate(&MigrationHistory{}); err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("on executing obs script")
			return err
		}

		query := fmt.Sprintf(obs.GetOBSScript(), id, wallet, converter.AddressToString(wallet))
		if err := DBConn.Exec(query).Error; err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("on executing obs script")
			return err
		}

		pubfunc := func(privateKeyFilename string) ([]byte, error) {
			var (
				privkey, privKey, pubKey []byte
				err                      error
			)
			privkey, err = os.ReadFile(filepath.Join(conf.Config.KeysDir, privateKeyFilename))
			if err != nil {
				log.WithFields(log.Fields{"type": consts.IOError, "error": err}).Error("reading private key from file")
				return nil, err
			}
			privKey, err = hex.DecodeString(string(privkey))
			if err != nil {
				log.WithFields(log.Fields{"type": consts.ConversionError, "error": err}).Error("decoding private key from hex")
				return nil, err
			}
			pubKey, err = crypto.PrivateToPublic(privKey)
			if err != nil {
				log.WithFields(log.Fields{"type": consts.CryptoError, "error": err}).Error("converting private key to public")
				return nil, err
			}
			return pubKey, nil
		}

		amount := decimal.New(consts.FounderAmount, int32(consts.MoneyDigits)).String()
		if err = GetDB(nil).Exec(`insert into "1_keys" (account,pub,amount) values (?,?,?,?),(?,?,?,?)`,
			keyID, converter.AddressToString(keyID), PubKey, amount, nodeKeyID, converter.AddressToString(nodeKeyID), nodePubKey, 0).Error; err != nil {
			return err
		}
	}
	return nil
}

// ExecSchema is executing schema
func ExecSchema() error {
	return migration.InitMigrate(&MigrationHistory{})
}

// UpdateSchema run update migrations
func UpdateSchema() error {
	if !conf.Config.IsOBSMaster() {
		b := &Block{}
		if found, err := b.GetMaxBlock(); !found {
			return err
		}
	}
	return migration.UpdateMigrate(&MigrationHistory{})
}
