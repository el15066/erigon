
package prediction

import (
	"fmt"
	"os"
	"bufio"
	"math/big"

	uint256 "github.com/holiman/uint256"
	common  "github.com/ledgerwatch/erigon/common"
	kv      "github.com/ledgerwatch/erigon-lib/kv"

	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"
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
	ctx.PredictTX(txIndex, origin, gasPrice, to_addr, callvalue, calldata, gas)
}
