
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
const INVALID_TARGET  = int(math.MaxInt64)

type Regs  [65536]uint256.Int
type Known [65536]bool
type Mem   [65536]byte

func (mem *Mem) msize() uint64 {
	return 2048 // TODO: keep track of this
}
func (mem *Mem) get(i0, s uint64) []byte {
	i1 := i0 + s
	if i1 > uint64(len(mem)) || i0 > i1 { return nil }
	return mem[i0:i1]
}
func (mem *Mem) set(i0, s uint64, data []byte) {
	i1 := i0 + s
	if i1 > uint64(len(mem)) || i0 > i1 { return }
	copy(mem[i0:i1], data)
}
func (mem *Mem) setUnknown(i0, s uint64) {
	i1 := i0 + s
	if i1 > uint64(len(mem)) || i0 > i1 { return }
	copy(mem[i0:i1], random_byte_string)
}
func (mem *Mem) setByte(i uint64, b byte) {
	if i >= uint64(len(mem)) { return }
	mem[i] = b
}
func (mem *Mem) get32(i0 uint64)        []byte  { return mem.get(i0, 32)          }
func (mem *Mem) set32(i0 uint64, data [32]byte) {        mem.set(i0, 32, data[:]) }
func (mem *Mem) setUnknown1( i uint64) { mem.setUnknown(i, 1 ) }
func (mem *Mem) setUnknown32(i uint64) { mem.setUnknown(i, 32) }

type BlockTableEntry struct {
	index int
	edges []uint16 // allow in-edges
}
type BlockTable map[uint16]BlockTableEntry

// Ctx can't change during execution of a TX, only between TXs, should not be copied and is unique to each thread
type Ctx struct {
	hasher      crypto.KeccakState
	buf         [32]byte
	ibs         *stateDB.IntraBlockState
	origin      *common.Address
	coinbase    *common.Address
	gasPrice    *uint256.Int
	difficulty  *uint256.Int
	timestamp   uint64
	blockNumber uint64
	gasLimit    uint64
}
func newCtx(db kvDB.Getter) *Ctx {
	return &Ctx{
		hasher: crypto.NewKeccakState(),
		ibs:    stateDB.New(stateDB.NewPlainStateReader(db)),
	}
}
func (ctx *Ctx) SHA3(data []byte) []byte {
	ctx.hasher.Reset()
	ctx.hasher.Write(data)
	ctx.hasher.Read(ctx.buf[:])
	return ctx.buf[:]
}
func (ctx *Ctx) getHashBytes(i uint64) []byte {
	copy(                      ctx.buf[ 0:24], "BLOCKHASH_abcdef01234567")
	binary.BigEndian.PutUint64(ctx.buf[24:32], i)
	return ctx.SHA3(ctx.buf[:])
}

// State changes during the execution of a TX, and is 'copied' during calls
type State struct {
	ctx       *Ctx
	address   *common.Address
	// codeaddr  *common.Address
	caller    *common.Address
	callvalue *uint256.Int
	blockTbl  BlockTable
	code      []byte
	calldata  []byte
	gaz       uint
	regs      Regs
	known     Known
	mem       Mem
	curBlock  uint16
	phiindex  int
	philen    int
	i         int
}

func newState(ctx *Ctx) *State {
	// TODO: create a pool and allocate without clearing any fields
	return &State{ ctx: ctx }
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

func (state *State) bidToIndex(bid64 uint64) int {
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

// Not exact, only for prediction
func isPrecompile(codeAddress *common.Address) bool {
	last := codeAddress[common.AddressLength-1]
	if 1 <= last && last < 10 {
		ok := true
		for i := 0; i < common.AddressLength - 1; i += 1 {
			ok = ok && codeAddress[i] == 0
		}
		return ok
	}
	return false
}

func predictCall(state *State, codeAddress *common.Address) (byte, bool) {
	if isPrecompile(codeAddress) { return 1, true }

	// find the predictor
	// load blocktable, code
	// not found ? return 0, false
	// state.blockTbl = blockTbl
	code := []byte{}
	state.code     = code
	state.curBlock = 0
	state.i        = 0
	i_max         := len(code)
	//
	for state.i < i_max && state.gaz > 0 {
		state.gaz -= 1
		op := code[state.i]
		jumpTable[op](state)
	}
	return 1, true
}