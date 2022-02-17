
package prediction

import (
	"math/big"

	common  "github.com/ledgerwatch/erigon/common"
)

type BlockVars struct {
	Coinbase    common.Address
	Difficulty  *big.Int
	BlockNumber uint64
	Timestamp   uint64
	GasLimit    uint64
}

// Not exact, only for prediction
func isPrecompile(codeAddress common.Address) bool {
	last := codeAddress[common.AddressLength-1]
	if 1 <= last && last < 10 {
		ok := true
		for i := 0; i < common.AddressLength-1; i += 1 {
			ok = ok && codeAddress[i] == 0
		}
		return ok
	}
	return false
}
