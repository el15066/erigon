
package predictorDB

import (
	"context"
	"strings"
	"unicode"

	common "github.com/ledgerwatch/erigon/common"

	kv   "github.com/ledgerwatch/erigon-lib/kv"
	mdbx "github.com/ledgerwatch/erigon-lib/kv/mdbx"
)

type PredictorDB struct {
	u kv.RwDB
	t kv.Tx
}

func (db PredictorDB) CloseDB() {
	db.t.Rollback()
	db.u.Close()
}
func (db PredictorDB) get(k []byte) []byte {
	// tx,   err := db.u.BeginRo(context.Background()); if err != nil { return nil }
	// defer tx.Rollback()
	// data, err := tx.GetOne("p", k);                  if err != nil { return nil }
	data, err := db.t.GetOne("p", k);                   if err != nil { return nil }
	return data
}
func (db PredictorDB) Get(k []byte) ([]byte, []byte) {
	data   := db.get(k);                                if len(data) < 10 { return nil, nil }
	//
	s := uint(data[0]) | (uint(data[1]) << 8)
	blocks := data[  2:s+2]
	code   := data[s+2:]
	//
	return blocks, code
}

func openPredictorDB() (PredictorDB, error) {
	db, err := openMDBX(common.PREDICTOR_DB_PATH)
	if err != nil {
		return PredictorDB{}, nil
	}
	tx, err := db.BeginRo(context.Background())
	if err != nil {
		db.Close()
		return PredictorDB{}, nil
	}
	res   := PredictorDB{ u: db, t: tx }
	_info := res.get(([]byte)("info"))
	var info string
	if _info == nil {
		info = "nil"
	} else {
		info = strings.Map(func(r rune) rune {
			if unicode.IsPrint(r) {
				return r
			}
			return '?'
		}, string(_info))
	}
	log.Info("Opened database", "info", string(info))
	return res, nil
}

// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/tables.go#L460
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go#L42
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go#L232
var tablesCfg = kv.TableCfg{
	"p": {},
}
func myTables(_ignore kv.TableCfg) kv.TableCfg { return tablesCfg }

// node/node.go:521
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/util.go
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go
func openMDBX(fileName string) (kv.RwDB, error) {
	db, err := mdbx.
		NewMDBX(log).
		WithTablessCfg(myTables).
		Path(fileName).
		Label(kv.Label(77)).
		DBVerbosity(kv.DBVerbosityLvl(2)). // for c code, 2 is warning, >2 gives info
		Exclusive().
		Readonly().
		Open()
	if err != nil { return nil, err }
	return db, err
}
