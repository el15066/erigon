
package prediction

import (
	"fmt"
	"os"
	"bufio"
	"math/big"

	common "github.com/ledgerwatch/erigon/common"

	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"
)

var blockVars BlockVars
var statePool *StatePool

var tracefile *bufio.Writer

func Init() {
	err := predictorDB.Init(); if err != nil { panic(err) }
	//
	statePool = new(StatePool)
	statePool.Init()
	//
	if common.TRACE_PREDICTED {
		if common.PREFETCH_WORKERS_COUNT > 1 {
			panic("TRACE_PREDICTED specified with more than 1 worker (PREFETCH_WORKERS_COUNT > 1)")
		}
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

func SetBlockVars(
	coinbase    common.Address,
	difficulty  *big.Int,
	blockNumber uint64,
	timestamp   uint64,
	gasLimit    uint64,
) {
	blockVars.Coinbase    = coinbase
	blockVars.Difficulty  = difficulty
	blockVars.BlockNumber = blockNumber
	blockVars.Timestamp   = timestamp
	blockVars.GasLimit    = gasLimit
	//
	if common.DEBUG_TX && blockNumber == common.DEBUG_TX_BLOCK {
		fmt.Println("SetBlockVars", blockVars)
	}
}
