
package common

// const MAX_BLOCK = 7_500_000 - 1 + 4546
// const MAX_BLOCK = 10_500_000 - 1 + 60_000
const MAX_BLOCK = 10_500_000 - 1 + 5_000

const  STORAGE_TRACING = false
const PREFETCH_TRACING = false
const       TX_DUMPING = false
const     CODE_DUMPING = false
const     JUMP_TRACING = false

const PREFETCH_BLOCKS           = true
const BLOCK_READAHEAD           = 1 + 1
const PREFETCH_ACCOUNTS         = false
const PREFETCH_CODE             = false
const PREFETCH_ACCOUNTS         = true
const USE_PREDICTORS            = true
const USE_STORAGE_PREFETCH_FILE = false

const TRACE_PREDICTED = true

const PREDICTOR_CACHE_SIZE      = 256
const PREDICTOR_INITIAL_GAZ     = 10000
const PREDICTOR_RESERVE_GAZ_DIV = 4
const PREDICTOR_CALL_GAZ_BONUS  = 0

const DEBUG_TX       = false
const DEBUG_TX_BLOCK = 7504466
const DEBUG_TX_INDEX = 62

const PREDICTOR_DB_PATH = "dbdir/predictorDB_new"

// if CODE_DUMPING {
var CONTRACT_CODE       = map[Hash][]byte{}
var CONTRACT_CODE_COUNT = map[Hash]uint{}
var CONTRACT_CODE_ALIAS = map[Address]Hash{}
// }

var CALLID         = uint32(0)
var CALLID_COUNTER = uint32(0)

// if JUMP_TRACING {
var JUMP_COUNT         = map[Hash]uint{}
var JUMP_CALLS         = map[Hash]map[uint32]struct{}{}
// var JUMP_EDGE_COUNT    = map[Hash]map[uint64]uint{}
var JUMP_DST_CALLCOUNT = map[Hash]map[uint32]map[uint32]map[uint32]uint{}
// var JUMP_DST_CALLS     = map[Hash]map[uint32]map[uint32]uint{} // size of the above map
// var JUMP_DST_COUNT     = map[Hash]map[uint32]map[uint32]uint{} // sum  of the above map's items
// }
