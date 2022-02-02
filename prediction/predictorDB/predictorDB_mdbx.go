
package predictorDB

import (
	"context"

	kv   "github.com/ledgerwatch/erigon-lib/kv"
	mdbx "github.com/ledgerwatch/erigon-lib/kv/mdbx"
	log  "github.com/ledgerwatch/log/v3"
)

type PredictorDB struct {
	u kv.RwDB
}

func (db PredictorDB) CloseDB() {
	db.u.Close()
}
func (db PredictorDB) get(k []byte) []byte {
	tx,   err := db.u.BeginRo(context.Background()); if err != nil { return nil }
	data, err := tx.GetOne("p", k);                  if err != nil { return nil }
	return data
}
func (db PredictorDB) Get(k []byte) ([]byte, []byte) {
	data      := db.get(k);                          if len(data) < 10 { return nil, nil }
	//
	s := uint(data[0]) | (uint(data[1]) << 8)
	blocks := data[  2:s+2]
	code   := data[s+2:]
	//
	return blocks, code
}

func openPredictorDB() (PredictorDB, error) {
	db, err := openMDBX("predictors.sqlite3")
	return PredictorDB{ u: db }, err
}

// node/node.go:521
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go
func openMDBX(fileName string) (kv.RwDB, error) {
	db, err := mdbx.NewMDBX(log.New()).Path(fileName).Readonly().Exclusive().Label(kv.Label(77)).DBVerbosity(kv.DBVerbosityLvl(2)).Open()
	if err != nil { return nil, err }
	return db, err
}
