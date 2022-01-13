
package common

const  STORAGE_TRACING = false
const PREFETCH_TRACING = false
const       TX_DUMPING = false
const     CODE_DUMPING = true

const USE_STORAGE_PREFETCH_FILE = true

var CONTRACT_CODE       = map[Hash][]byte{}
var CONTRACT_CODE_COUNT = map[Hash]uint{}
var CONTRACT_CODE_ALIAS = map[Address]Hash{}
