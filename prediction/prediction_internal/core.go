
package prediction_internal

import (
	"math"
	"encoding/binary"

	"github.com/holiman/uint256"

	common  "github.com/ledgerwatch/erigon/common"
	crypto  "github.com/ledgerwatch/erigon/crypto"
	stateDB "github.com/ledgerwatch/erigon/core/state"
	kvDB    "github.com/ledgerwatch/erigon-lib/kv"
)

const BLOCK_ID_SHIFTS = 1
const BLOCK_ID_MAX    = uint64(65536 << BLOCK_ID_SHIFTS) - 1
const INVALID_TARGET  = uint(-1)

type Regs  [65536]uint256.Int
type Known [65536]bool
type Mem   [65536]byte

type BlockTableEntry struct {
	index uint
	edges []uint16 // allow in-edges
}
type BlockTable map[uint]BlockTableEntry

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
	blockTbl  *BlockTable
	code      []byte
	calldata  []byte
	gaz       uint
	regs      Regs
	known     Known
	mem       Mem
	curBlock  uint
	phiindex  uint
	philen    uint
	i         uint
}

func newState(ctx *Ctx) *State {
	// TODO: create a pool and allocate without clearing any fields
	return &State{ ctx }
}
func freeState(state *State) {
	// TODO: allow to be reused by newState
}

func run(address *common.Address) {
	ctx := &Ctx{
		hasher: sha3.NewLegacyKeccak256().(keccakState)
	}
	predictTX(ctx, address)
}

func predictTX(ctx *Ctx, address *common.Address) {
	state := newState(ctx)
	state.address = address
	state.caller  = ctx.origin
	state.gaz     = 10000
	predictCall()
	freeState(state)
}

func (state *State) bidToIndex(bid64 uint64) uint {
	if bid64 <= 0xFFFF {
		bid := uint16(bid64)
		if b, ok := state.blockTbl[bid]; ok {
			return b.index
		}
	}
	return INVALID_TARGET
}

func (state *State) changeBlock(bid uint16) {
	b, ok := state.blockTbl[bid]
	if ok {
		ok = false
		for i, e := range b.edges {
			if e == state.curBlock {
				state.phiindex = i
				ok = true
				break
			}
		}
	}
	if ok {
		state.philen   = len(b.edges)
		state.curBlock = bid
	} else {
		state.i = INVALID_TARGET
	}
}

func predictCall(state *State, codeAddress *common.Address) (byte, bool) {
	isPrecompile := 1 <= codeAddress < 10
	if isPrecompile { return 1, true }

	find the predictor
	load blocktable, code
	not found ? return 0, false
	state.blockTbl = blockTbl
	state.code     = code
	state.curBlock = 0
	state.i        = 0
	i_max         := len(code)
	//
	for state.i < i_max && state.gaz > 0 {
		state.gaz -= 1
		op := code[state.i]
		jt[op](&state)
	}
}
