
package prediction_internal

import (
	// lru       "github.com/hashicorp/golang-lru/simplelru" //  thread-safe
	simplelru "github.com/hashicorp/golang-lru/simplelru" // not thread-safe

	common  "github.com/ledgerwatch/erigon/common"
)

var cache *simplelru.LRU

type Predictor struct {
	blockTbl  BlockTable
	code      []byte
}

func init() {
	cache, _ = simplelru.NewLRU(2048)
}

func getPredictor(h common.Hash) Predictor {
	p, ok := cache.Get(h)
	if !ok {
		data, ok := db.GetOne(h)
		if !ok { return nil }
		p,    ok = decodePredictor(data)
		if !ok { return nil }
		cache.Add(h, p)
	}
	return p
}

func decodePredictor(data []byte) (Predictor, bool) {
	// TODO
}
