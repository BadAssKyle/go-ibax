/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
package model

type VDEDestHashTime struct {
	ID         int64 `gorm:"primary_key; not null" json:"id"`
	UpdateTime int64 `gorm:"not null" json:"update_time"`
	CreateTime int64 `gorm:"not null" json:"create_time"`
}

func (VDEDestHashTime) TableName() string {
	return "vde_dest_hash_time"
}

func (m *VDEDestHashTime) Create() error {
	return DBConn.Create(&m).Error
}

func (m *VDEDestHashTime) Updates() error {
	return DBConn.Model(m).Updates(m).Error
}

func (m *VDEDestHashTime) Delete() error {
	return DBConn.Delete(m).Error
}

func (m *VDEDestHashTime) Get() (*VDEDestHashTime, error) {
	err := DBConn.First(&m).Error
	return m, err
