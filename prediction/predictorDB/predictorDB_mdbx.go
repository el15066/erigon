
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

var DB PredictorDB

var _mdbx_getChan chan []byte
var _mdbx_resChan chan []byte

func openPredictorDB() error {
	errChan := make(chan error)
	go _mdbx_helperThread(errChan)
	return <- errChan
}

func (db PredictorDB) Get(k []byte) ([]byte, []byte) {
	//
	_mdbx_getChan <- k
	data   := <- _mdbx_resChan;               if len(data) < 10 { return nil, nil }
	//
	s := uint(data[0]) | (uint(data[1]) << 8)
	blocks := data[  2:s+2]
	code   := data[s+2:]
	//
	return blocks, code
}

func (db PredictorDB) Close() {
	close(_mdbx_getChan)
	<- _mdbx_resChan
}


//// Below are internal stuff


func (db PredictorDB) _close() {
	db.t.Rollback()
	db.u.Close()
}

func (db PredictorDB) _get(k []byte) []byte {
	data, err := db.t.GetOne("p", k);             if err != nil { return nil }
	return data
}

func _mdbx_helperThread(errChan chan error) {
	// MDBX requires all requests to come from the same thread
	// locking to thread is done in NewMDBX().Open()
	defer close(_mdbx_resChan)
	{
		err := _mdbx_openPredictorDB()
		errChan <- err
		if err != nil { return }
	}
	_mdbx_printDBinfo()
	//
	for k := range _mdbx_getChan {
		_mdbx_resChan <- DB._get(k)
	}
	DB._close()
}

func _mdbx_openPredictorDB() error {
	db, err := _mdbx_open(common.PREDICTOR_DB_PATH)
	if err != nil {
		return err
	}
	tx, err := db.BeginRo(context.Background())
	if err != nil {
		db.Close()
		return err
	}
	DB = PredictorDB{ u: db, t: tx }
	return nil
}

func _mdbx_printDBinfo() {
	_info := DB._get(([]byte)("info"))
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
}

// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/tables.go#L460
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go#L42
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go#L232
func _mdbx_myTables(_ignore kv.TableCfg) kv.TableCfg {
	return kv.TableCfg{
		"p": {},
	}
}

// node/node.go:521
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/util.go
// https://github.com/ledgerwatch/erigon-lib/blob/main/kv/mdbx/kv_mdbx.go
func _mdbx_open(fileName string) (kv.RwDB, error) {
	db, err := mdbx.
		NewMDBX(log).
		WithTablessCfg(_mdbx_myTables).
		Path(fileName).
		Label(kv.Label(77)).
		DBVerbosity(kv.DBVerbosityLvl(2)). // for c code, 2 is warning, >2 gives info
		Exclusive().
		Readonly().
		Open()
	if err != nil { return nil, err }
	return db, err
}
