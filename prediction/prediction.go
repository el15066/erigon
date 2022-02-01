
package prediction

import (
	"math/big"

	"github.com/holiman/uint256"

	internal "github.com/ledgerwatch/erigon/prediction/prediction_internal"
	common   "github.com/ledgerwatch/erigon/common"
	kvDB     "github.com/ledgerwatch/erigon-lib/kv"
)

var ctx internal.Ctx

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
