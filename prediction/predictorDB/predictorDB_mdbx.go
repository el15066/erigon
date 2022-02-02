
package prediction_internal

import (
	kv   "github.com/ledgerwatch/erigon-lib/kv"
	mdbx "github.com/ledgerwatch/erigon-lib/kv/mdbx"
	log  "github.com/ledgerwatch/log/v3"
)

// node/node.go:521
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go
func openMDBX() (kv.RwDB, error) {
	db, err := mdbx.NewMDBX(log.New()).Path("predictors.dat").Readonly().Exclusive().Label(kv.Label(77)).DBVerbosity(kv.DBVerbosityLvl(2)).Open()
	return db, err
}
