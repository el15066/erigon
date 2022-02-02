
package prediction

import (
	"math/big"

	uint256 "github.com/holiman/uint256"
	common  "github.com/ledgerwatch/erigon/common"
	kvDB    "github.com/ledgerwatch/erigon-lib/kv"

	internal    "github.com/ledgerwatch/erigon/prediction/prediction_internal"
	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"
)

var ctx internal.Ctx

func Init() {
	var err error
	err = predictorDB.Init(); if err != nil { panic(err) }
}
func Close() {
	predictorDB.Close()
}

func InitCtx(db kvDB.Getter) {
	ctx = *internal.NewCtx(db)
}

func SetBlockVars(
	coinbase    common.Address,
	difficulty  *big.Int,
	blockNumber uint64,
	timestamp   uint64,
	gasLimit    uint64,
) {
	ctx.Coinbase    = coinbase
	ctx.Difficulty  = difficulty
	ctx.BlockNumber = blockNumber
	ctx.Timestamp   = timestamp
	ctx.GasLimit    = gasLimit
}

func PredictTX(
	to_addr   common.Address,
	//
	origin    common.Address,
	gasPrice  *uint256.Int,
	//
	callvalue *uint256.Int,
	calldata  []byte,
) {
	ctx.Origin   = origin
	ctx.GasPrice = gasPrice
	internal.PredictTX(&ctx, to_addr, callvalue, calldata)
}
