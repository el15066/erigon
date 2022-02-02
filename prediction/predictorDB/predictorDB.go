
package predictorDB

import (
	// "fmt"
	"sync"

	// lru       "github.com/hashicorp/golang-lru/simplelru" //  thread-safe
	simplelru "github.com/hashicorp/golang-lru/simplelru" // not thread-safe

	logging   "github.com/ledgerwatch/log/v3"
	common    "github.com/ledgerwatch/erigon/common"

	types     "github.com/ledgerwatch/erigon/prediction/types"
)

var log logging.Logger

var pdb predictorCache

type predictorCache struct {
	Mu    sync.Mutex
	Cache *simplelru.LRU
	DB    PredictorDB
}

func Init() error {
	log = logging.New("package", "predictorDB")
	//
	var err error
	pdb.DB, err  = openPredictorDB()
	if err != nil { return err }
	pdb.Cache, _ = simplelru.NewLRU(2048, nil)
	return nil
}
func Close() {
	pdb.Mu.Lock()
	defer pdb.Mu.Unlock()
	//
	// pdb.Cache.Purge()
	pdb.DB.CloseDB()
}

func GetPredictor(h common.Hash) types.Predictor {
	pdb.Mu.Lock()
	defer pdb.Mu.Unlock()
	//
	var p types.Predictor
	//
	_p, ok := pdb.Cache.Get(h)
	if ok {
		p = _p.(types.Predictor)
	} else {
		b, c := pdb.DB.Get(h.Bytes())
		if b != nil {
			p = decodePredictor(b, c)
			if p.Code == nil {
				log.Warn("Bad predictor", "codehash", h)
				// fmt.Println("Predictor for", h, p)
			} else {
				// log.Info("Good predictor", "codehash", h)
			}
		}
		pdb.Cache.Add(h, p)
	}
	return p
}

func decodePredictor(blocks []byte, code []byte) types.Predictor {
	res := types.Predictor{ Code: code }
	// for ??? {
	// 	res.BlockTbl[k] = v
	// }
	return res
}
