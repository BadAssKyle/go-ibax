package model

import (
	"testing"
)

	hashDeles = append(hashDeles, []byte("s"))

	batch = append(batch, logTxs, txs, queues)
	for _, d := range batch {
		d.BatchFindByHash(nil, hashDeles)
	}
}
