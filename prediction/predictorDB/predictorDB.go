
package predictorDB

import (
	// "fmt"
	"sync"

	// simplelru "github.com/hashicorp/golang-lru" //  thread-safe
	simplelru "github.com/hashicorp/golang-lru/simplelru" // not thread-safe

	logging   "github.com/ledgerwatch/log/v3"
	common    "github.com/ledgerwatch/erigon/common"

	types     "github.com/ledgerwatch/erigon/prediction/types"
)

var log logging.Logger

var pdb predictorCache

type predictorCache struct {
	Mu    common.Spinlock // not RW because simplelru is not ok with concurrent reads
	MuDB  sync.RWMutex
	Cache *simplelru.LRU
}

func Init() error {
	log = logging.New("package", "predictorDB")
	//
	err := openPredictorDB()
	if err != nil { return err }
	//
	pdb.Cache, _ = simplelru.NewLRU(common.PREDICTOR_CACHE_SIZE, nil)
	return nil
}

func Close() {
	pdb.MuDB.Lock()
	DB.Close()
	pdb.MuDB.Unlock()
	// pdb.Mu.Lock()
	// pdb.Cache.Purge()
	// pdb.Mu.Unlock()
}

func GetPredictor(h common.Hash) types.Predictor {
	var p types.Predictor
	//
	pdb.Mu.Lock()
	_p, ok := pdb.Cache.Get(h)
	pdb.Mu.Unlock()
	//
	if ok {
		p = _p.(types.Predictor)
	} else {
		//
		pdb.MuDB.Lock()
		b, c := DB.Get(h.Bytes())
		pdb.MuDB.Unlock()
		//
		if b != nil {
			p = decodePredictor(b, c)
			if p.Code == nil {
				log.Warn("Bad predictor", "codehash", h)
				// fmt.Println("Predictor for", h, p)
			} else {
				// log.Info("Good predictor", "codehash", h)
			}
		}
		//
		pdb.Mu.Lock()
		pdb.Cache.Add(h, p)
		pdb.Mu.Unlock()
	}
	return p
}

// _padding needs to hold at least the `CONSTANT_32` instruction's size
const _padding = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" // 40 B

func decodePredictor(bt []byte, code []byte) types.Predictor {
	res  := types.Predictor{
		BlockTbl: types.BlockTable{},
		Code:     append(code, _padding...), // to prevent index panic
	}
	ok   := true
	i    := 0
	imax := len(bt)
	for i < imax {
		// fmt.Println("i 6", i, imax)
		//
		if imax - i < 6     { ok = false; break }
		bid   := uint16(bt[i]) | (uint16(bt[i+1]) << 8);                        i += 2
		pos   :=    int(bt[i]) | (   int(bt[i+1]) << 8) | (int(bt[i+2]) << 16); i += 3
		c     :=    int(bt[i]);                                                 i += 1
		//
		// fmt.Println("i 2c", i, c, imax)
		if imax - i < 2 * c { ok = false; break }
		edges := make([]uint16, c)
		for j := 0; j < c; j += 1 {
			e := uint16(bt[i]) | (uint16(bt[i+1]) << 8);                        i += 2
			edges[j] = e
		}
		// fmt.Println("i ok", i, imax)
		//
		res.BlockTbl[bid] = types.BlockTableEntry{
			Index: pos,
			Edges: edges,
		}
	}
	//
	if !ok {
		res.Code     = nil
		res.BlockTbl = nil
	}
	return res
}
