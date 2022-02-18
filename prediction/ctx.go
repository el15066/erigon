
package prediction

import (
	"fmt"
	"os"
	"bufio"
	"sort"
	"bytes"
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
	tracefile   *bufio.Writer
	Predicted   []common.Hash
	Debug       bool
}

func NewCtx(myID int, db kv.Getter) *Ctx {
	ctx := &Ctx{
		hasher: crypto.NewKeccakState(),
		ibs:    stateDB.New(stateDB.NewPlainStateReader(db)),
	}
	if common.TRACE_PREDICTED {
		fname    := fmt.Sprintf("logz/predicted_%02d.txt", myID)
		_f, _err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if  _err == nil {
			ctx.tracefile = bufio.NewWriterSize(_f, 1024*1024)
		} else {
			Close() // prediction.go
			panic(_err)
		}
		ctx.Predicted = make([]common.Hash, 0, PREDICTED_CAP)
	}
	return ctx
}

func (ctx *Ctx) Close() {
	if common.TRACE_PREDICTED && ctx.tracefile != nil {
		ctx.tracefile.Flush()
	}
}

func (ctx *Ctx) StartingNewBlock() {
	ctx.bvs = blockVars
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
	txIndex   uint64,
	//
	origin    common.Address,
	gasPrice  *uint256.Int,
	//
	address   common.Address,
	callvalue *uint256.Int,
	calldata  []byte,
	gas       uint64,
	//
) {
	if common.DEBUG_TX {
		if ctx.bvs.BlockNumber == common.DEBUG_TX_BLOCK && txIndex == common.DEBUG_TX_INDEX {
			fmt.Println("PredictTX",
				ctx.bvs.BlockNumber,
				txIndex,
				origin,
				gasPrice,
			)
			ctx.Debug = true
		} else {
			ctx.Debug = false
		}
	}

	ctx.Origin   = origin
	ctx.GasPrice = gasPrice

	state := statePool.NewState(ctx)
	if state == nil { return }

	state.address   = address
	state.caller    = ctx.Origin
	state.callvalue.Set(callvalue)  // TODO: maybe pointer instead ?
	state.calldata  = calldata
	state.gaz       = int(gas * common.PREDICTOR_GAS_TO_GAZ_RATE / 1024)

	state.predictCall(address)

	statePool.FreeState(state)

	if common.TRACE_PREDICTED && ctx.tracefile != nil {
		//
		ctx.tracefile.WriteString(fmt.Sprintf("Tx %8d %3d %x\n", ctx.bvs.BlockNumber, txIndex, address))
		//
		if len(ctx.Predicted) > 0 {
			sort.Slice(ctx.Predicted, func(a, b int) bool {
				return bytes.Compare(ctx.Predicted[a][:], ctx.Predicted[b][:]) < 0
			})
			var prev []byte // nil is not equal to the zero hash (32 zeros)
			for i := range ctx.Predicted {
				p :=       ctx.Predicted[i][:]
				if bytes.Equal(p, prev) { continue }
				prev = p
				ctx.tracefile.WriteString(fmt.Sprintf("%x\n", ctx.Predicted[i])) // can't use p which is a slice
			}
		}
		//
		if cap(ctx.Predicted) != PREDICTED_CAP {
			fmt.Println("Note: transaction", ctx.bvs.BlockNumber, txIndex, "ctx.Predicted len is", len(ctx.Predicted))
			ctx.Predicted = make([]common.Hash, 0, PREDICTED_CAP)
		}
		//
		ctx.Predicted = ctx.Predicted[:0]
	}
}
