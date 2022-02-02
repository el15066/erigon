
package prediction_internal

import (
	"sync"

	// lru       "github.com/hashicorp/golang-lru/simplelru" //  thread-safe
	simplelru "github.com/hashicorp/golang-lru/simplelru" // not thread-safe

	common    "github.com/ledgerwatch/erigon/common"
)

var pdb predictorCache

type predictorCache struct {
	Mu    sync.Mutex
	Cache *simplelru.LRU
	DB    *predictorDB
}

type Predictor struct {
	BlockTbl BlockTable
	Code     []byte
}

func InitPredictorDB() error {
	var err error
	pdb.DB, err  = openPredictorDB()
	if err != nil { return err }
	pdb.Cache, _ = simplelru.NewLRU(2048, nil)
	return nil
}
func ClosePredictorDB() {
	pdb.Mu.Lock()
	defer pdb.Mu.Unlock()
	//
	pdb.Cache.Purge()
	pdb.DB.CloseDB()
}

func GetPredictor(h common.Hash) Predictor {
	pdb.Mu.Lock()
	defer pdb.Mu.Unlock()
	//
	var p Predictor
	//
	_p, ok := pdb.Cache.Get(h)
	if ok {
		p = _p.(Predictor)
	} else {
		b, c, ok := pdb.DB.Get(h.Bytes())
		if !ok { return p }
		p,    ok  = decodePredictor(b, c)
		if !ok { return p }
		pdb.Cache.Add(h, p)
	}
	return p
}

func decodePredictor(blocks []byte, code []byte) (Predictor, bool) {
	res := Predictor{ Code: code }
	// for ??? {
	// 	res.BlockTbl[k] = v
	// }
	return res, true
}
