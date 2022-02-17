
package prediction

import (
	"fmt"
	"sync"
	"math/bits"

	uint256 "github.com/holiman/uint256"

	common  "github.com/ledgerwatch/erigon/common"

	types   "github.com/ledgerwatch/erigon/prediction/types"
)

type Regs [65536]uint256.Int

// State changes during the execution of a TX, and is 'copied' during calls
type State struct {
	spIndex   byte              // readonly
	ctx       *Ctx
	address   common.Address
	// codeaddr  common.Address
	caller    common.Address
	blockTbl  types.BlockTable
	code      []byte
	callvalue uint256.Int
	calldata  []byte
	gaz       int
	regs      Regs
	mem       Mem
	curBlock  uint16
	phiindex  int
	philen    int
	i         int
}
func (state *State) bidToIndex(bid64 uint64) int {
	if bid64 <= 0xFFFF {
		bid := uint16(bid64)
		if b, ok := state.blockTbl[bid]; ok {
			return b.Index
		}
	}
	return INVALID_TARGET
}
func (state *State) changeBlock(bid uint16) {
	b, ok := state.blockTbl[bid]
	if ok {
		ok = false
		for i, e := range b.Edges {
			if e == state.curBlock {
				state.phiindex = i
				ok = true
				break
			}
		}
	}
	if ok {
		state.philen   = len(b.Edges)
		state.curBlock = bid
	} else {
		state.i = INVALID_TARGET
	}
	if common.DEBUG_TX && state.ctx.Debug { fmt.Printf(" ~%x", bid) }
}

// To avoid actual state copying (can be 100x more expensive than actual call)
// we keep all the states pre-allocated and reuse them without zeroing.
type StatePool struct {
	mu        sync.Mutex
	states    [32]State // Can be dynamic in the future (up to threads * maximum call depth, =1024 yellowpaper p.36 CALL, but will abort at reasonable limit)
	available uint64    // make sure num of bits >= len(states)
}
func (sp *StatePool) Init(ctx *Ctx) {
	c := len(sp.states)
	if c > 64 { panic("Too many states in StatePool") }
	//
	for i := 0; i < c; i += 1 {
		sp.states[i].spIndex = byte(i)
		sp.states[i].ctx     = ctx
		sp.states[i].regs[INVALID_REG].Set(&UNKNOWN_U256)
	}
	sp.available = (uint64(1) << c) - 1
}
func (sp *StatePool) NewState() *State {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	//
	av := sp.available
	if av == 0 { return nil }
	//
	i := bits.TrailingZeros64(av)
	//
	sp.available &= ^(uint64(1) << i)
	//
	return &sp.states[i]
}
func (sp *StatePool) FreeState(state *State) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	//
	i := state.spIndex
	sp.available |= uint64(1) << i
}
