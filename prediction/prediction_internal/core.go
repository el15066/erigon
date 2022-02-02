
package prediction_internal

import (
	"fmt"
	"math"
	"math/big"
	"encoding/binary"

	uint256 "github.com/holiman/uint256"

	common  "github.com/ledgerwatch/erigon/common"
	crypto  "github.com/ledgerwatch/erigon/crypto"
	stateDB "github.com/ledgerwatch/erigon/core/state"
	kvDB    "github.com/ledgerwatch/erigon-lib/kv"

	predictorDB "github.com/ledgerwatch/erigon/prediction/predictorDB"
)

const BLOCK_ID_SHIFTS = 0
const BLOCK_ID_MAX    = uint64(65536 << BLOCK_ID_SHIFTS) - 1
const INVALID_TARGET  = int(math.MaxInt64)

var JumpTable [256]func(*State)

func init() { JumpTable = jumpTable } // go doesn't like circle with opCallCommon

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
	if i >= uint64(len(mem))            { return }
	mem[i] = b
}
func (mem *Mem) get32(i0 uint64)        []byte  { return mem.get(i0, 32)          }
func (mem *Mem) set32(i0 uint64, data [32]byte) {        mem.set(i0, 32, data[:]) }
func (mem *Mem) setUnknown1( i uint64) { mem.setUnknown(i, 1 ) }
func (mem *Mem) setUnknown32(i uint64) { mem.setUnknown(i, 32) }


// ibs.Empty(a)
// ibs.GetBalance(a)
// ibs.GetCode(a)
// ibs.GetCodeHash(a)
// ibs.GetCodeSize(a)
// ibs.GetState(a, k, d)
// ibs.PrefetchState(a, k)
// ibs.SetDirtyState(a, k, v)

// Ctx can't change during execution of a TX, only between TXs, should not be copied and is unique to each thread
type Ctx struct {
	sp          StatePool
	hasher      crypto.KeccakState
	buf         [32]byte
	ibs         *stateDB.IntraBlockState
	//
	Coinbase    common.Address
	Difficulty  *big.Int
	BlockNumber uint64
	Timestamp   uint64
	GasLimit    uint64
	//
	Origin      common.Address
	GasPrice    *uint256.Int
}
func NewCtx(db kvDB.Getter) *Ctx {
	ctx := &Ctx{
		hasher: crypto.NewKeccakState(),
		ibs:    stateDB.New(stateDB.NewPlainStateReader(db)),
	}
	ctx.sp.Init(ctx)
	return ctx
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

func PredictTX(
	ctx       *Ctx,
	address   common.Address,
	callvalue *uint256.Int,
	calldata  []byte,
) {
	state := ctx.sp.NewState()
	state.address   = address
	state.caller    = ctx.Origin
	state.callvalue = callvalue
	state.calldata  = calldata
	state.gaz       = 10000
	predictCall(state, address)
	ctx.sp.FreeState(state)
}
func predictCall(state *State, codeAddress common.Address) (byte, bool) {
	if isPrecompile(codeAddress) { return 1, true }
	//
	ch := state.ctx.ibs.GetCodeHash(codeAddress)
	p  := predictorDB.GetPredictor(ch)
	if p.Code == nil { return 0, false }
	state.blockTbl = p.BlockTbl
	state.code     = p.Code
	state.curBlock = 0
	state.i        = 0
	i_max         := len(state.code)
	//
	for state.i < i_max && state.gaz > 0 {
		state.gaz -= 1
		op := state.code[state.i]
		JumpTable[op](state)
	}
	if state.gaz <= 0 {
		fmt.Println("Call out of gaz, ca:", codeAddress)
	}
	return 1, true
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
