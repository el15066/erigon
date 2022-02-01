
package prediction

import (
	internal "github.com/ledgerwatch/erigon/prediction/prediction_internal"
	common   "github.com/ledgerwatch/erigon/common"
)

func PredictTX(ctx *internal.Ctx, address *common.Address) {
	internal.PredictTX(ctx, address)
}
