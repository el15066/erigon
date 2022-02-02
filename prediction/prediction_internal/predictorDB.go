
package prediction_internal

import (
	"sync"

	// lru       "github.com/hashicorp/golang-lru/simplelru" //  thread-safe
	simplelru "github.com/hashicorp/golang-lru/simplelru" // not thread-safe

	common    "github.com/ledgerwatch/erigon/common"
)

var pdb predictorCache

type predictorCache struct {
	var mu    sync.Mutex
	var cache *simplelru.LRU
	var db    predictorDB
}

type Predictor struct {
	blockTbl  BlockTable
	code      []byte
}

func InitPredictorDB() error {
	pdb.db, err := openPredictorDB()
	if err != nil { return err }
	pdb.cache, _ = simplelru.NewLRU(2048)
	return nil
}
func ClosePredictorDB() error {
	pdb.mu.Lock()
	defer pdb.mu.Unlock()
	//
	pdb.cache.Purge()
	pdb.db.closeDB()
}

func GetPredictor(h common.Hash) Predictor {
	pdb.mu.Lock()
	defer pdb.mu.Unlock()
	//
	p, ok := pdb.cache.Get(h)
	if !ok {
		b, c, ok := pdb.db.get(h)
		if !ok { return nil }
		p,     ok = decodePredictor(b, c)
		if !ok { return nil }
		pdb.cache.Add(h, p)
	}
	return p
}

func decodePredictor(blocks []byte, code []byte) (Predictor, bool) {
	res := Predictor{ code: code }
	// for ??? {
	// 	res.blockTbl[k] = v
	// }
	return res, true
}
