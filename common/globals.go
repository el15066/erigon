
package common

import (
	"sync/atomic"
)

// const MAX_BLOCK = 7_500_000 - 1 + 4546
const MAX_BLOCK = 10_500_000 - 1 + 5_000
// const MAX_BLOCK = 10_500_000 - 1 + 10_000

const  STORAGE_TRACING = false
const PREFETCH_TRACING = false
const       TX_DUMPING = false
const     CODE_DUMPING = false
const     JUMP_TRACING = false

const PREFETCH_BLOCKS           = true
const BLOCK_READAHEAD           = 1 + 1
const PREFETCH_ACCOUNTS         = true
const PREFETCH_CODE             = true
const USE_PREDICTORS            = true
const USE_STORAGE_PREFETCH_FILE = false

const PREFETCH_WORKERS_COUNT = 1

const TRACE_PREDICTED = true

const DEBUG_TX       = false
const DEBUG_TX_BLOCK = 10_500_005
const DEBUG_TX_INDEX = 81

const PREDICTOR_CACHE_SIZE      = 1024
// const PREDICTOR_INITIAL_GAZ     = 10000
const PREDICTOR_GAS_TO_GAZ_RATE = 64 // div 1024
const PREDICTOR_CALL_GAZ_BONUS  = 0
const PREDICTOR_RESERVE_GAZ_DIV = 2
const PREDICTOR_DB_PATH         = "dbdir/predictorDB_new"

// if CODE_DUMPING {
var CONTRACT_CODE       = map[Hash][]byte{}
var CONTRACT_CODE_COUNT = map[Hash]uint{}
var CONTRACT_CODE_ALIAS = map[Address]Hash{}
// }

var _tx_index = int64(-1) // DEBUG_TX || TRACE_PREDICTED

func GetTxIndex() int {
	return int(atomic.LoadInt64(&_tx_index))
}
func SetTxIndex(i int) {
	atomic.StoreInt64(&_tx_index, int64(i))
}

var CALLID         = -1 // JUMP_TRACING
var CALLID_COUNTER =  0 // JUMP_TRACING

// if JUMP_TRACING {
var JUMP_COUNT         = map[Hash]uint{}
var JUMP_CALLS         = map[Hash]map[int]struct{}{}
// var JUMP_EDGE_COUNT    = map[Hash]map[uint64]uint{}
var JUMP_DST_CALLCOUNT = map[Hash]map[uint32]map[uint32]map[int]uint{}
// var JUMP_DST_CALLS     = map[Hash]map[uint32]map[uint32]uint{} // len of the above map
// var JUMP_DST_COUNT     = map[Hash]map[uint32]map[uint32]uint{} // sum of the above map's items
// }
