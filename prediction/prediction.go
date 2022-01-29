
package prediction

import (
	"golang.org/x/crypto/sha3"
)

type Regs  [65536]uint256.Int
type Known [65536]bool
type Mem   [65536]byte

// Ctx can't change during execution of a TX, only between TXs, should not be copied and is unique to each thread
type Ctx struct {
	var hasher    keccakState
	var hasherBuf common.Hash
	origin        *common.Address
}

// State changes during the execution of a TX, and is 'copied' during calls
type State struct {
	ctx       *Ctx
	address   *common.Address
	// codeaddr  *common.Address
	caller    *common.Address
	callvalue uint256.Int
	calldata  []byte
	gaz       uint
	regs      Regs
	known     Known
	mem       Mem
	phiindex  int
	code      []byte
	i         uint
}

func run() {
	state := &State{}
	state.ctx.hasher = sha3.NewLegacyKeccak256().(keccakState)
	// i  := uint(0)
	// op := data[i]
	// jt[op](&i, &data, &state)
	jt[op](&state)
}


