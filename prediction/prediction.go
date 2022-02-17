
package prediction

import (
	"fmt"
	"os"
	"sort"
	"bytes"
	"bufio"
	"math/big"

	uint256 "github.com/holiman/uint256"
	common  "github.com/ledgerwatch/erigon/common"
	kv      "github.com/ledgerwatch/erigon-lib/kv"

	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"

	bench "github.com/ledgerwatch/erigon/bench"
)

var statePool *StatePool

var ctx       *Ctx
var tracefile *bufio.Writer

func Init() {
	err := predictorDB.Init(); if err != nil { panic(err) }
	//
	statePool = new(StatePool)
	statePool.Init()
	//
	if common.TRACE_PREDICTED {
		_f, _err := os.OpenFile("logz/predicted.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if  _err == nil {
			tracefile = bufio.NewWriterSize(_f, 1024*1024)
		} else {
			Close()
			panic(_err)
		}
	}
}
func Close() {
	predictorDB.Close()
	if common.TRACE_PREDICTED && tracefile != nil {
		tracefile.Flush()
	}
}

func InitCtx(db kv.Getter) {
	ctx = NewCtx(db)
}

func SetBlockVars(
	coinbase    common.Address,
	difficulty  *big.Int,
	blockNumber uint64,
	timestamp   uint64,
	gasLimit    uint64,
) {
	ctx.bvs = BlockVars{
		Coinbase:    coinbase,
		Difficulty:  difficulty,
		BlockNumber: blockNumber,
		Timestamp:   timestamp,
		GasLimit:    gasLimit,
	}
	//
	if common.DEBUG_TX && blockNumber == common.DEBUG_TX_BLOCK {
		fmt.Println("SetBlockVars", ctx.bvs)
	}
}
func BlockEnded() { ctx.BlockEnded() }

func PredictTX(
	txIndex   int,
	to_addr   common.Address,
	//
	origin    common.Address,
	gasPrice  *uint256.Int,
	//
	callvalue *uint256.Int,
	calldata  []byte,
	//
	gas       uint64,
) {
	ctx.Origin   = origin
	ctx.GasPrice = gasPrice
	if common.DEBUG_TX {
		if ctx.bvs.BlockNumber == common.DEBUG_TX_BLOCK && txIndex == common.DEBUG_TX_INDEX {
			fmt.Println("PredictTX",
				txIndex,
				ctx.Origin,
				ctx.GasPrice,
			)
			ctx.Debug = true
		} else {
			ctx.Debug = false
		}
	}

	gaz := int(gas * common.PREDICTOR_GAS_TO_GAZ_RATE / 1024)

	bench.Tick(150)
	ctx.PredictTX(to_addr, callvalue, calldata, gaz)
	bench.Tick(151)

	if common.TRACE_PREDICTED && tracefile != nil {
		//
		tracefile.WriteString(fmt.Sprintf("Tx %8d %3d %x\n", ctx.bvs.BlockNumber, txIndex, to_addr))
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
				tracefile.WriteString(fmt.Sprintf("%x\n", ctx.Predicted[i])) // can't use p which is a slice
			}
		}
		ctx.Predicted = ctx.Predicted[:0]
		//
		if cap(ctx.Predicted) != PREDICTED_CAP {
			fmt.Println("Note", ctx.bvs.BlockNumber, txIndex, "ctx.Predicted len cap", len(ctx.Predicted), cap(ctx.Predicted))
			ctx.Predicted = make([]common.Hash, 0, PREDICTED_CAP)
		}
		bench.Tick(152)
	}
}
