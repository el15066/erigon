
package common

const  STORAGE_TRACING = false
const PREFETCH_TRACING = false
const       TX_DUMPING = false
const     CODE_DUMPING = false
const     JUMP_TRACING = false

const PREFETCH_BLOCKS           = false
const BLOCK_READAHEAD           = 10 + 1
const PREFETCH_ACCOUNTS         = false
const PREFETCH_CODE             = false
const USE_PREDICTORS            = false
const USE_STORAGE_PREFETCH_FILE = false

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
