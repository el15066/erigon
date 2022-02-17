
package prediction

import (
	"fmt"
	"math"
	"encoding/hex"

	uint256 "github.com/holiman/uint256"

	common  "github.com/ledgerwatch/erigon/common"

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

var jumpTable [256]func(*State)

func init() { jumpTable = _jumpTable } // go doesn't like circle with opCallCommon

// ibs.Empty(a)
// ibs.GetBalance(a)
// ibs.GetCode(a)
// ibs.GetCodeHash(a)
// ibs.GetCodeSize(a)
// ibs.GetState(a, k, d)
// ibs.PrefetchState(a, k)
// ibs.SetDirtyState(a, k, v)

func internalPredictTX(
	ctx       *Ctx,
	address   common.Address,
	callvalue *uint256.Int,
	calldata  []byte,
	gaz       int,
) {
	state := statePool.NewState(ctx)
	if state == nil { return }
	state.address   = address
	state.caller    = ctx.Origin
	state.callvalue.Set(callvalue)
	state.calldata  = calldata
	state.gaz       = gaz
	state.predictCall(address)
	statePool.FreeState(state)
}

// var inside = false

func (state *State) predictCall(codeAddress common.Address) (byte, bool) {
	//
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
		jumpTable[op](state)
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
