
package prediction

import (
	"fmt"
	"math"
	"math/big"
	"encoding/hex"
	"encoding/binary"

	uint256 "github.com/holiman/uint256"

	common  "github.com/ledgerwatch/erigon/common"
	crypto  "github.com/ledgerwatch/erigon/crypto"
	stateDB "github.com/ledgerwatch/erigon/core/state"
	kv      "github.com/ledgerwatch/erigon-lib/kv"

	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"

	// bench "github.com/ledgerwatch/erigon/bench"
)

// The following must be kept the same as in encode_predictors.py
const BLOCK_ID_SHIFTS = 0
const BLOCK_ID_MAX    = uint64(65536 << BLOCK_ID_SHIFTS) - 1
const INVALID_REG     = 65000

const INVALID_TARGET  = int(math.MaxInt64)

const ZEROS32         = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

var UNKNOWN_U256 = uint256.Int{
	random_u256_part_0,
	random_u256_part_0,
	random_u256_part_0,
	random_u256_part_0,
}

var JumpTable [256]func(*State)

func init() { JumpTable = jumpTable } // go doesn't like circle with opCallCommon

type Mem struct {
	data  [65536]byte
	msize uint64
	modified bool
}
func (mem *Mem) Init() {
	mem.msize    = 0
	mem.modified = false
}
func (mem *Mem) Msize() uint64 {
	return mem.msize
}
func (mem *Mem) debug() {
	if mem.modified {
		mem.modified = false
		for j := uint64(0); j < mem.msize; j += 0x20 {
			w := mem.data[j:j+0x20]
			if string(w) != ZEROS32 {
				fmt.Println(fmt.Sprintf("  mem %4x  %s", j, hex.EncodeToString(w)))
			}
		}
	}
}
func (mem *Mem) updateMsize(i1 uint64) {
	m1 := mem.msize
	m2 := (i1 + 31) & ^uint64(31)
	if m2 > m1 {
		t := mem.data[m1:m2]
		for i := range t { t[i] = 0 }
		mem.msize = m2
	}
}
func (mem *Mem) get(i0, s uint64) []byte {
	i1 := i0 + s
	if i1 > uint64(len(mem.data)) || i0 > i1 { return nil }
	mem.updateMsize(i1)
	return mem.data[i0:i1]
}
func (mem *Mem) set(i0, s uint64, data []byte) {
	i1 := i0 + s
	if i1 > uint64(len(mem.data)) || i0 > i1 { return }
	mem.updateMsize(i1)
	mem.modified = true
	copy(mem.data[i0:i1], data)
}
func (mem *Mem) setUnknown(i0, s uint64) {
	i1 := i0 + s
	if i1 > uint64(len(mem.data)) || i0 > i1 { return }
	mem.updateMsize(i1)
	mem.modified = true
	copy(mem.data[i0:i1], random_byte_string)
	// copy(mem.data[i0:i1], random_byte_string[i0&0x3:]) // tiny 0.001% worse
}
func (mem *Mem) setByte(i uint64, b byte) {
	if i >= uint64(len(mem.data))            { return }
	mem.updateMsize(i + 1)
	mem.modified = true
	mem.data[i]  = b
}
func (mem *Mem) getByte(i uint64) byte {
	if i >= uint64(len(mem.data))            { return 0 }
	mem.updateMsize(i + 1)
	return mem.data[i]
}
func (mem *Mem) get32(i0 uint64)        []byte  { return mem.get(i0, 32)          }
func (mem *Mem) set32(i0 uint64, data [32]byte) {        mem.set(i0, 32, data[:]) }


// ibs.Empty(a)
// ibs.GetBalance(a)
// ibs.GetCode(a)
// ibs.GetCodeHash(a)
// ibs.GetCodeSize(a)
// ibs.GetState(a, k, d)
// ibs.PrefetchState(a, k)
// ibs.SetDirtyState(a, k, v)

// Ctx can't change* during execution of a TX, only between TXs, should not be copied and is unique to each thread
// *except for returnData/Size and Predicted
type Ctx struct {
	sp          StatePool
	hasher      crypto.KeccakState
	buf         [32]byte
	ibs         *stateDB.IntraBlockState
	//
	returnData  Mem    // needed here to keep alive after FreeState()
	returnSize  uint64
	//
	Coinbase    common.Address
	Difficulty  *big.Int
	BlockNumber uint64
	Timestamp   uint64
	GasLimit    uint64
	//
	Origin      common.Address
	GasPrice    *uint256.Int
	//
	Predicted   []common.Hash
	Debug       bool
}
func NewCtx(db kv.Getter) *Ctx {
	ctx := &Ctx{
		hasher: crypto.NewKeccakState(),
		ibs:    stateDB.New(stateDB.NewPlainStateReader(db)),
	}
	ctx.sp.Init(ctx)
	return ctx
}
func (ctx *Ctx) BlockEnded() {
	ctx.ibs.Reset()
}
func (ctx *Ctx) SHA3(data []byte) []byte {
	ctx.hasher.Reset()
	ctx.hasher.Write(data)
	ctx.hasher.Read(ctx.buf[:])
	//
	if common.DEBUG_TX && ctx.Debug {
		fmt.Print(hex.EncodeToString(data), " ", hex.EncodeToString(ctx.buf[:]))
	}
	//
	return ctx.buf[:]
}
func (ctx *Ctx) getHashBytes(i uint64) []byte {
	copy(                      ctx.buf[ 0:24], "BLOCKHASH_abcdef01234567")
	binary.BigEndian.PutUint64(ctx.buf[24:32], i)
	return ctx.SHA3(ctx.buf[:])
}

func PredictTX(
	ctx       *Ctx,
	address   common.Address,
	callvalue *uint256.Int,
	calldata  []byte,
	gaz       int,
) {
	state := ctx.sp.NewState()
	if state == nil { return }
	state.address   = address
	state.caller    = ctx.Origin
	state.callvalue.Set(callvalue)
	state.calldata  = calldata
	state.gaz       = gaz
	predictCall(state, address)
	ctx.sp.FreeState(state)
}

// var inside = false

func predictCall(state *State, codeAddress common.Address) (byte, bool) {
	if common.DEBUG_TX && state.ctx.Debug {
		fmt.Println("predictCall",
			codeAddress,
			state.address,
			state.caller,
			state.callvalue,
			hex.EncodeToString(state.calldata),
			state.gaz,
		)
	}
	//
	if isPrecompile(codeAddress) { return 1, true }
	//
	// bench.Tick(210)
	ch := state.ctx.ibs.GetCodeHash(codeAddress)
	// bench.Tick(211)
	p  := predictorDB.GetPredictor(ch)
	// bench.Tick(212)
	if     common.DEBUG_TX && state.ctx.Debug { fmt.Println("  code hash", ch, "have?", p.Code != nil) }
	if p.Code == nil { return 0, false }
	//
	// bench.Tick(215)
	state.ctx.returnData.Init()
	state.ctx.returnSize = 0 // TODO maybe set to random_u256_part_0
	state.blockTbl = p.BlockTbl
	state.code     = p.Code
	state.curBlock = 0
	state.i        = 0
	state.mem.Init()
	code          := state.code
	i_max         := len(code)
	//
	// me := false
	// if !inside {
	// 	inside = true
	// 	me = true
	// 	bench.Tick(220)
	// 	if common.DEBUG_TX && state.ctx.Debug { fmt.Println("\n\n---- tx @", state.ctx.BlockNumber) }
	// }
	if     common.DEBUG_TX && state.ctx.Debug { fmt.Println("  call", codeAddress) }
	//
	for state.i < i_max && state.gaz > 0 {
		state.gaz -= 1
		op := code[state.i]
		//
		var rd uint16
		if common.DEBUG_TX && state.ctx.Debug { _, rd = getArg(code, state.i + 1) }
		if common.DEBUG_TX && state.ctx.Debug { fmt.Print(fmt.Sprintf(  "%5d| %4d = %20s ", state.gaz, rd, jumpTableNames[op])) }
		//
		JumpTable[op](state)
		//
		if common.DEBUG_TX && state.ctx.Debug { fmt.Print(fmt.Sprintf("\n%5d| %4d = %20s \n", state.gaz, rd, _enc_reg(state, rd))) }
		if common.DEBUG_TX && state.ctx.Debug { state.mem.debug() }
	}
	// if me {
	// 	bench.Tick(221)
	// 	inside = false
	// 	if common.DEBUG_TX && state.ctx.Debug { fmt.Println("\n\n---- end") }
	// }
	if state.gaz <= 0 {
		if common.DEBUG_TX && state.ctx.Debug { fmt.Println("Call out of gaz, ca:", codeAddress) }
	}
	// bench.Tick(216)
	return 1, true
}
// Not exact, only for prediction
func isPrecompile(codeAddress common.Address) bool {
	last := codeAddress[common.AddressLength-1]
	if 1 <= last && last < 10 {
		ok := true
		for i := 0; i < common.AddressLength-1; i += 1 {
			ok = ok && codeAddress[i] == 0
		}
		return ok
	}
	return false
}
