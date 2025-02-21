/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package model

// QueueBlock is model
type QueueBlock struct {
	Hash        []byte `gorm:"primary_key;not null"`
	BlockID     int64  `gorm:"not null"`
	HonorNodeID int64  `gorm:"not null"`
// Delete is deleting queue
func (qb *QueueBlock) Delete() error {
	return DBConn.Delete(qb).Error
}

// DeleteQueueBlockByHash is deleting queue by hash
func (qb *QueueBlock) DeleteQueueBlockByHash() error {
	query := DBConn.Exec("DELETE FROM queue_blocks WHERE hash = ?", qb.Hash)
	return query.Error
}

// DeleteOldBlocks is deleting old blocks
func (qb *QueueBlock) DeleteOldBlocks() error {
	query := DBConn.Exec("DELETE FROM queue_blocks WHERE block_id <= ?", qb.BlockID)
	return query.Error
}

// Create is creating record of model
func (qb *QueueBlock) Create() error {
	return DBConn.Create(qb).Error
}
