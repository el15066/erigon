
package prediction

import (
	"fmt"
	"encoding/hex"
	"encoding/binary"

	uint256 "github.com/holiman/uint256"

	common  "github.com/ledgerwatch/erigon/common"
	crypto  "github.com/ledgerwatch/erigon/crypto"
	stateDB "github.com/ledgerwatch/erigon/core/state"
	kv      "github.com/ledgerwatch/erigon-lib/kv"
)

const PREDICTED_CAP = 16384 // If TRACE_PREDICTED, warn if the capacity of ctx.Predicted exceeds this

// Ctx can't change* during execution of a TX, only between TXs, should not be copied and is unique to each thread
// *except for returnData/Size and Predicted
type Ctx struct {
	hasher      crypto.KeccakState
	buf         [32]byte
	ibs         *stateDB.IntraBlockState
	//
	returnData  Mem    // needed here to keep alive after FreeState()
	returnSize  uint64
	//
	bvs         BlockVars
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
	if common.TRACE_PREDICTED {
		ctx.Predicted = make([]common.Hash, 0, PREDICTED_CAP)
	}
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
	return ctx.buf[:]
}

func (ctx *Ctx) getHashBytes(i uint64) []byte {
	copy(                      ctx.buf[ 0:24], "BLOCKHASH_abcdef01234567")
	binary.BigEndian.PutUint64(ctx.buf[24:32], i)
	return ctx.SHA3(ctx.buf[:])
}

func (ctx *Ctx) PredictTX(
	//
	address   common.Address,
	callvalue *uint256.Int,
	calldata  []byte,
	gaz       int,
	//
) {
	state := statePool.NewState(ctx)
	if state == nil { return }

	state.address   = address
	state.caller    = ctx.Origin
	state.callvalue.Set(callvalue)  // TODO: maybe pointer instead ?
	state.calldata  = calldata
	state.gaz       = gaz

	state.predictCall(address)

	statePool.FreeState(state)
}
