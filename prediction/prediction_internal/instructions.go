
package prediction

import (
	"fmt"

	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/log/v3"
)

func isValidTarget(target uint) { return target != INVALID_TARGET }

func getArg(data []byte, i uint) (uint, uint16) { return i+2, uint16(data[i]) | (uint16(data[i+1]) << 8) }

func opStop(state *State) {
	state.i = INVALID_TARGET
	return
}
func opConstant(state *State) {
	// Equivalent of PUSH, a constant is given as immediate value
	i      := state.i
	size   := state.code[i] - OP_CONSTANT_OFFSET
	i, rd  := getArg(state.code, i + 1)
	state.known[rd] = true
	d      := &state.regs[rd]
	d.SetBytes(state.code[i : i + size])
	state.i = i + size
	return
}
func opPhi(state *State) {
	if state.phiindex >= state.philen { panic("state.phiindex >= state.philen") }
	// Artifact of SSA, should be removed with proper reg allocation
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	_, rs  := getArg(state.code, i + 2 * state.phiindex)
	state.i = i + 2 * state.philen
	state.known[rd] = state.known[rs]
	d      := &state.regs[rd]
	s      := &state.regs[rs]
	d.Set(s)
	return
}
func opBlockId(state *State) {
	// Artifact of SSA, should be removed with proper reg allocation
	// Similar to JUMPDEST
	i      := state.i + 1
	i, bid := getArg(state.code, i)
	state.i = i
	state.changeBlock(bid)
	return
}
func opJump(state *State) {
	i      := state.i + 1
	i, rb  := getArg(state.code, i)
	ok     := state.known[rb]
	_bid   := &state.regs[rb]
	if !ok || _bid.GtUint64(BLOCK_ID_MAX) {
		state.i = INVALID_TARGET
		return
	}
	bid    := _bid.Uint64() >> BLOCK_ID_SHIFTS
	state.i = state.bidToIndex(bid)
	state.changeBlock(uint16(bid))
	return
}
func opJumpi(state *State) {
	i      := state.i + 1
	i, rb  := getArg(state.code, i)
	i, rc  := getArg(state.code, i)
	ok     := state.known[rb]
	_bid   := &state.regs[rb]
	cond   := &state.regs[rc]
	if !ok || _bid.GtUint64(BLOCK_ID_MAX) {
		state.i = INVALID_TARGET
		return
	}
	bid    := _bid.Uint64() >> BLOCK_ID_SHIFTS
	target := state.bidToIndex(bid)
	var taken bool
	if state.known[rc] { taken = !cond.IsZero()
	} else             { taken = isValidTarget(target) }
	if taken {
		state.i = target
		state.changeBlock(uint16(bid))
	}
	else{
		state.i = i
	}
	return
}

func zerOpArgs(state *State) int {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	state.i = i
	state.known[rd] = true
	return rd
}
func opReturnDataSize(state *State) {
	rd := zerOpArgs(state)
	state.known[rd] = false
	return
}
func opCodeSize(state *State) {
	rd := zerOpArgs(state)
	state.known[rd] = false
	// a := state.address // not always same as code address, maybe keep a ref to contract's code in state, also see CODECOPY
	// d.SetUint64(uint64(state.ctx.ibs.GetCodeSize(a)))
	return
}

func zerOpArgVs(state *State) *uint256.Int {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	state.i = i
	state.known[rd] = true
	d  := &state.regs[rd]
	return d
}
func opAddress(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.address.Bytes())
	return
}
func opOrigin(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.ctx.origin.Bytes())
	return
}
func opCaller(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.caller.Bytes())
	return
}
func opCallValue(state *State) {
	d := zerOpArgVs(state)
	d.Set(state.callvalue)
	return
}
func opCallDataSize(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(len(state.calldata))
	return
}
func opGasprice(state *State) {
	d := zerOpArgVs(state)
	d.Set(state.ctx.gasPrice)
	return
}
func opCoinbase(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.ctx.coinbase.Bytes())
	return
}
func opTimestamp(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.timestamp)
	return
}
func opNumber(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.blockNumber)
	return
}
func opDifficulty(state *State) {
	d := zerOpArgVs(state)
	d.Set(state.ctx.difficulty)
	return
}
func opMsize(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.mem.msize())
	return
}
func opGas(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.gasLimit) // we don't track gas usage, so return max
	return
}
func opGasLimit(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.gasLimit)
	return
}

func uniOp(state *State, op func (*uint256.Int, *uint256.Int) *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0]
	state.known[rd] = ok
	if !ok { return }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	op(d, v0)
	return
}
func opNot(state *State) { return uniOp(state, (uint256.Int).Not)  }

func _iszero(d, v0 *uint256.Int) *uint256.Int { if v0.IsZero() { return d.SetOne() } else { return d.Clear() } }
func opIszero(state *State) { return uniOp(state, _iszero) }

func uniOpArgVs(state *State) (*uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0]
	state.known[rd] = ok
	if !ok { return, nil }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	return d, v0
}
func opBalance(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	a := common.Address(v0.Bytes20())
	d.Set(state.ctx.ibs.GetBalance(a))
	return
}
func opExtCodeSize(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	a := common.Address(v0.Bytes20())
	d.SetUint64(uint64(state.ctx.ibs.GetCodeSize(a)))
	return
}
func opExtCodeHash(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	a := common.Address(v0.Bytes20())
	if state.ctx.ibs.Empty(a) { // TODO: maybe speculatively skip check ?
		d.Clear()
	} else {
		d.SetBytes(state.ctx.ibs.GetCodeHash(a).Bytes())
	}
	return
}
func opSload(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	a := state.address
	// k := (*common.Hash)(v0) // hash is []byte :(
	k := &state.ctx.hasherBuf
	*k = v0.Bytes32()
	state.ctx.ibs.GetState(a, k, d)
	return
}

func uniNVOpArgVs(state *State) (*uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0]
	if !ok { return }
	v0 := &state.regs[r0]
	return v0
}
func opStouch(state *State) {
	v0 := uniNVOpArgVs(state)
	if v0 == nil { return }
	a := state.address
	k := &state.ctx.hasherBuf
	*k = v0.Bytes32()
	state.ctx.ibs.PrefetchState(a, k)
	return
}

func uniOpArgs(state *State) (int, int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0]
	state.known[rd] = ok
	if !ok { rd = -1 }
	return rd, r0
}
func opCallDataLoad(state *State) {
	rd, r0 := uniOpArgs(state)
	if rd < 0 { return }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	if _i.GtUint64(65535) {
		state.known[rd] = false
		return
	}
	i := _i.Uint64()
	d.SetBytes(getData(state.calldata, i, 32))
	return
}
func opBlockhash(state *State) {
	rd, r0 := uniOpArgs(state)
	if rd < 0 { return }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	if !_i.IsUint64() {
		state.known[rd] = false
		return
	}
	i     := _i.Uint64()
	delta := state.ctx.blockNumber - i - 1
	if delta < 256 { d.SetBytes(state.ctx.GetHash(i).Bytes())
	} else         { d.Clear() }
	return
}
func opMload(state *State) {
	rd, r0 := uniOpArgs(state)
	if rd < 0 { return }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	if _i.GtUint64(65535) {
		state.known[rd] = false
		return
	}
	i := _i.Uint64()
	d.SetBytes(state.mem.get(i, 32))
	return
}

func binOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1]
	state.known[rd] = ok
	if !ok { return }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	op(d, v0, v1)
	return
}
func opAdd( state *State) { return binOp(state, (uint256.Int).Add)  }
func opSub( state *State) { return binOp(state, (uint256.Int).Sub)  }
func opMul( state *State) { return binOp(state, (uint256.Int).Mul)  }
func opDiv( state *State) { return binOp(state, (uint256.Int).Div)  }
func opSdiv(state *State) { return binOp(state, (uint256.Int).SDiv) }
func opMod( state *State) { return binOp(state, (uint256.Int).Mod)  }
func opSmod(state *State) { return binOp(state, (uint256.Int).SMod) }
func opAnd( state *State) { return binOp(state, (uint256.Int).And)  }
func opOr(  state *State) { return binOp(state, (uint256.Int).Or)   }
func opXor( state *State) { return binOp(state, (uint256.Int).Xor)  }

func _eq( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Eq( v1) { return d.SetOne() } else { return d.Clear() } }
func _lt( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Lt( v1) { return d.SetOne() } else { return d.Clear() } }
func _gt( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Gt( v1) { return d.SetOne() } else { return d.Clear() } }
func _slt(d, v0, v1 *uint256.Int) *uint256.Int { if v0.Slt(v1) { return d.SetOne() } else { return d.Clear() } }
func _sgt(d, v0, v1 *uint256.Int) *uint256.Int { if v0.Sgt(v1) { return d.SetOne() } else { return d.Clear() } }
func opEq( state *State) { return binOp(state, _eq)  }
func opLt( state *State) { return binOp(state, _lt)  }
func opGt( state *State) { return binOp(state, _gt)  }
func opSlt(state *State) { return binOp(state, _slt) }
func opSgt(state *State) { return binOp(state, _sgt) }

func binOpArgs(state *State) (int, int, int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1]
	state.known[rd] = ok
	if !ok { rd = -1 }
	return rd, r0, r1
}
func opExp(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d := &state.regs[rd]
	b := &state.regs[r0]
	e := &state.regs[r1]
	switch {
		case e.IsZero():      d.SetOne()
		case b.IsZero():      d.Clear()
		case b.LtUint64(2):   d.SetOne()
		case e.LtUint64(2):   d.Set(b)
		case e.GtUint64(256): state.known[rd] = false // prob due to random data for unknowns
		case b.LtUint64(3):   d.Lsh(d.SetOne(), uint(e.Uint64()))
		default:              d.Exp(b, e)
	}
	return
}
func opSignExtend(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(31) {
		state.known[rd] = false // same as EXP
		return
	}
	d.ExtendSign(x, i)
	return
}
func opByte(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	x  := &state.regs[r1]
	if _i.GtUint64(31) {
		state.known[rd] = false // same as EXP
		return
	}
	i := _i.Uint64()
	d.SetUint64((x[3 - i / 8] >> ((7 - i % 8) * 8)) & 0xFF)
	return
}
func opSHL(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(255) {
		state.known[rd] = false // same as EXP
		return
	}
	d.Lsh(x, uint(i.Uint64()))
	return
}
func opSHR(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(255) {
		state.known[rd] = false // same as EXP
		return
	}
	d.Rsh(x, uint(i.Uint64()))
	return
}
func opSAR(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(255) {
		state.known[rd] = false // same as EXP
		return
	}
	d.SRsh(x, uint(i.Uint64()))
	return
}
func opSha3(state *State) {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return }
	d := &state.regs[rd]
	i := &state.regs[r0]
	s := &state.regs[r1]
	data, ok := state.mem.get(i.Uint64(), i.Uint64() + s.Uint64())
	if !ok {
		state.known[rd] = false
		return
	}
	state.ctx.hasher.Reset()
	state.ctx.hasher.Write(data)
	if _, err := state.ctx.hasher.Read(state.ctx.hasherBuf[:]); err != nil { return err }
	d.SetBytes(state.ctx.hasherBuf[:])
	return
}

func binNVOpOptArgVs(state *State) (*uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	var v0, v1 *uint256.Int
	if state.known[r0] { v0 = &state.regs[r0] }
	if state.known[r1] { v1 = &state.regs[r1] }
	return v0, v1
}
func opMstore(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if v0 == nil { return }
	i := v0.Uint64()
	if v1 == nil { state.mem.SetUnknown32(i)
	} else       { state.mem.SetU256(i, v1) }
	return
}
func opMstore8(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if v0 == nil { return }
	i := v0.Uint64()
	if v1 == nil { state.mem.SetUnknown1(i)
	} else       { state.mem.SetByte(i, byte(v1.Uint64())) }
	return
}
func opSstore(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if v0 == nil { return }
	a := state.address
	k := &state.ctx.hasherBuf
	*k = v0.Bytes32()
	// state.ctx.ibs.PrefetchState(a, k)
	if v1 == nil { state.ctx.ibs.PrefetchState(a, k)
	} else       { state.ctx.ibs.SetDirtyState(a, k, v1) }
	return
}

func triOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2]
	state.known[rd] = ok
	if !ok { return }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	v2 := &state.regs[r2]
	op(d, v0, v1, v2)
	return
}
func opAddmod(state *State) { return triOp(state, (uint256.Int).AddMod) }
func opMulmod(state *State) { return triOp(state, (uint256.Int).MulMod) }

func triNVOpArgVs(state *State) (*uint256.Int, *uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2]
	if !ok { return, nil, nil }
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	v2 := &state.regs[r2]
	return v0, v1, v2
}
func opCallDataCopy(state *State) {
	v0, v1, v2 := triNVOpArgs(state)
	if v0 == nil { return }
	o := v0.Uint64()
	i := v1.Uint64()
	s := v2.Uint64()
	// if i < 65536 {
	data := getData(state.calldata, i, s)
	state.mem.Set(o, s, data)
	// } else {
	// 	state.mem.SetUnknown(o, s)
	// }
	return
}
func opReturnDataCopy(state *State) {
	v0, v1, v2 := triNVOpArgs(state)
	if v0 == nil { return }
	o := v0.Uint64()
	// i := v1.Uint64()
	s := v2.Uint64()
	state.mem.SetUnknown(o, s)
	return
}
func opCodeCopy(state *State) {
	// simillar to CODESIZE, these should really have been optimized out by the analyzer
	v0, v1, v2 := triNVOpArgs(state)
	if v0 == nil { return }
	o := v0.Uint64()
	// i := v1.Uint64()
	s := v2.Uint64()
	// if i < 1024 * 1024 {
	// a    := state.address
	// data := getData(state.ctx.ibs.GetCode(a), i, s)
	// state.mem.Set(o, s, data)
	// } else {
	state.mem.SetUnknown(o, s)
	// }
	return
}

func tetNVOpArgVs(state *State) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	i, r3  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2] && state.known[r3]
	if !ok { return, nil, nil, nil }
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	v2 := &state.regs[r2]
	v3 := &state.regs[r3]
	return v0, v1, v2, v3
}
func opExtCodeCopy(state *State) {
	v0, v1, v2, v3 := tetNVOpArgs(state)
	if v0 == nil { return }
	a := common.Address(v0.Bytes20())
	o := v1.Uint64()
	i := v2.Uint64()
	s := v3.Uint64()
	// if i < 1024 * 1024 {
	data := getData(state.ctx.ibs.GetCode(a), i, s)
	state.mem.Set(o, s, data)
	// } else {
	// 	state.mem.SetUnknown(o, s)
	// }
	return
}



const (
	CALL_REGULAR  CallOpType = iota
	CALL_CODE
	CALL_DELEGATE
	CALL_STATIC
)
func opCallCommon(state *State, t CallOpType) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i) // gas
	i, r1  := getArg(state.code, i) // address
	var r2 uint16
	if t == CALL_REGULAR ||
	   t == CALL_CODE {
		i, r2  := getArg(state.code, i) // value
	}
	i, r3  := getArg(state.code, i) // i0
	i, r4  := getArg(state.code, i) // i1-i0
	i, r5  := getArg(state.code, i) // o0
	i, r6  := getArg(state.code, i) // o1-o0
	state.i = i
	ok := state.known[r1] && state.known[r3] && state.known[r4]
	state.known[rd] = ok
	if !ok { return }
	d  := &state.regs[rd]
	var v0, v1, v2, v3, v4, v5, v6 *uint256.Int
	if state.known[r0] { v0 = &state.regs[r0] }
	if state.known[r1] { v1 = &state.regs[r1] }
	if state.known[r2] { v2 = &state.regs[r2] }
	if state.known[r3] { v3 = &state.regs[r3] }
	if state.known[r4] { v4 = &state.regs[r4] }
	if state.known[r5] { v5 = &state.regs[r5] }
	if state.known[r6] { v6 = &state.regs[r6] }
	//
	ca := &common.Address(v1.Bytes20())
	//
	if v3.GtUint64(65535) || v4.GtUint64(65535) {
		state.known[rd] = false
		return
	}
	i0 := v3.Uint64()
	i1 := v4.Uint64() + i0
	//
	if !(v5 == nil || v6 == nil) {
		if !(v5.GtUint64(65535) || v6.GtUint64(65535)) {
			o0 := v5.Uint64()
			o1 := v6.Uint64() + o0
    		// clear resulting mem, since we don't currently support return value
			state.mem.SetUnknown(o0, o1)
		}
	}
	//
	ns := newState(state.ctx)
	ns.calldata = state.mem.getRaw(i0, i1)
	//
	if t == CALL_REGULAR  { ns.address = ca
	} else                { ns.address = state.address }
	//
	if t == CALL_DELEGATE { ns.caller  = state.caller
	} else                { ns.caller  = state.address }
	//
	switch {
		case t == CALL_DELEGATE: ns.callvalue.Set(state.callvalue)
		case t == CALL_STATIC:   ns.callvalue.Clear()
		case v2 != nil:          ns.callvalue.Set(v2)
		default:                 ns.callvalue.Clear()
	}
	//
	reservedGaz := state.gaz / 4
	ns.gaz       = state.gaz - reserved_gaz
	//
	res, known  := predictCall(ns, ca)
	//
	state.gaz    =    ns.gaz + reserved_gaz
	freeState(ns)
	//
	d.SetUint64(uint64(res))
	state.known[rd] = known
	return
}
func opCall(        state *State) { return opCallCommon(state, CALL_REGULAR)  }
func opCallCode(    state *State) { return opCallCommon(state, CALL_CODE)     }
func opDelegateCall(state *State) { return opCallCommon(state, CALL_DELEGATE) }
func opStaticCall(  state *State) { return opCallCommon(state, CALL_STATIC)   }

// func opConstant01(state *State) { return opConstant(state,  1) }
// func opConstant02(state *State) { return opConstant(state,  2) }
// func opConstant03(state *State) { return opConstant(state,  3) }
// func opConstant04(state *State) { return opConstant(state,  4) }
// func opConstant05(state *State) { return opConstant(state,  5) }
// func opConstant06(state *State) { return opConstant(state,  6) }
// func opConstant07(state *State) { return opConstant(state,  7) }
// func opConstant08(state *State) { return opConstant(state,  8) }
// func opConstant09(state *State) { return opConstant(state,  9) }
// func opConstant10(state *State) { return opConstant(state, 10) }
// func opConstant11(state *State) { return opConstant(state, 11) }
// func opConstant12(state *State) { return opConstant(state, 12) }
// func opConstant13(state *State) { return opConstant(state, 13) }
// func opConstant14(state *State) { return opConstant(state, 14) }
// func opConstant15(state *State) { return opConstant(state, 15) }
// func opConstant16(state *State) { return opConstant(state, 16) }
// func opConstant17(state *State) { return opConstant(state, 17) }
// func opConstant18(state *State) { return opConstant(state, 18) }
// func opConstant19(state *State) { return opConstant(state, 19) }
// func opConstant20(state *State) { return opConstant(state, 20) }
// func opConstant21(state *State) { return opConstant(state, 21) }
// func opConstant22(state *State) { return opConstant(state, 22) }
// func opConstant23(state *State) { return opConstant(state, 23) }
// func opConstant24(state *State) { return opConstant(state, 24) }
// func opConstant25(state *State) { return opConstant(state, 25) }
// func opConstant26(state *State) { return opConstant(state, 26) }
// func opConstant27(state *State) { return opConstant(state, 27) }
// func opConstant28(state *State) { return opConstant(state, 28) }
// func opConstant29(state *State) { return opConstant(state, 29) }
// func opConstant30(state *State) { return opConstant(state, 30) }
// func opConstant31(state *State) { return opConstant(state, 31) }
// func opConstant32(state *State) { return opConstant(state, 32) }
