
package prediction

import (
	"fmt"
	"math/big"

	common "github.com/ledgerwatch/erigon/common"

	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"
)

var blockVars BlockVars
var statePool *StatePool

func Init() {
	err := predictorDB.Init(); if err != nil { panic(err) }
	//
	statePool = new(StatePool)
	statePool.Init()
}

func Close() {
	predictorDB.Close()
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
