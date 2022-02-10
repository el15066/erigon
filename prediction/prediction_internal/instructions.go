
package prediction_internal

import (
	"fmt"
	"encoding/hex"

	"github.com/holiman/uint256"

	"github.com/ledgerwatch/erigon/common"
)

func is_known(v *uint256.Int) bool {
	return v.Uint64() != random_u256_part_0
	// return uint32(v.Uint64()) != random_u256_lsu32 // tiny 0.001% worse
}

func _enc(v *uint256.Int) string {
	if v == nil || !is_known(v) { return " ?"
	} else                      { return " #" + hex.EncodeToString(v.Bytes()) }
}
func _enc_reg(state *State, r uint16) string {
	return _enc(&state.regs[r])
}

// from core/vm/common.go
func getData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return common.RightPadBytes(data[start:end], int(size))
}

func isValidTarget(target int) bool { return target != INVALID_TARGET }

func getArg(data []byte, i int) (int, uint16) { return i+2, uint16(data[i]) | (uint16(data[i+1]) << 8) }

func opStop(state *State) {
	state.i = INVALID_TARGET
	return
}
func opConstant(state *State) {
	// Equivalent of PUSH, a constant is given as immediate value
	i      := state.i
	size   := int(state.code[i]) - OP_CONSTANT_OFFSET + 1
	i, rd  := getArg(state.code, i + 1)
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
	i      += 2 * state.philen
	state.i = i
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
	_bid   := &state.regs[rb]
	// ok     := is_known(_bid)
	if _bid.GtUint64(BLOCK_ID_MAX) {
		state.i = INVALID_TARGET
		return
	}
	bid64  := _bid.Uint64() >> BLOCK_ID_SHIFTS
	state.i = state.bidToIndex(bid64)
	state.changeBlock(uint16(bid64))
	return
}
func opJumpi(state *State) {
	i      := state.i + 1
	i, rb  := getArg(state.code, i)
	i, rc  := getArg(state.code, i)
	_bid   := &state.regs[rb]
	cond   := &state.regs[rc]
	// ok     := is_known(_bid)
	if _bid.GtUint64(BLOCK_ID_MAX) {
		state.i = INVALID_TARGET
		return
	}
	bid64  := _bid.Uint64() >> BLOCK_ID_SHIFTS
	target := state.bidToIndex(bid64)
	var taken bool
	if is_known(cond) { taken = !cond.IsZero()
	} else            { taken = isValidTarget(target) }
	if taken {
		state.i = target
		state.changeBlock(uint16(bid64))
	} else  {
		state.i = i
	}
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(_enc_reg(state, rc)) }
	return
}


func zerOpArgVs(state *State) *uint256.Int {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	state.i = i
	d      := &state.regs[rd]
	return d
}
func opReturnDataSize(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.returnSize)
	return
}
func opCodeSize(state *State) {
	d := zerOpArgVs(state)
	d.Set(&UNKNOWN_U256)
	// a := state.address // not always same as code address, maybe keep a ref to contract's code in state, also see CODECOPY
	// d.SetUint64(uint64(state.ctx.ibs.GetCodeSize(a)))
	return
}
func opAddress(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.address.Bytes())
	return
}
func opOrigin(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.ctx.Origin.Bytes())
	return
}
func opCaller(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.caller.Bytes())
	return
}
func opCallValue(state *State) {
	d := zerOpArgVs(state)
	d.Set(&state.callvalue)
	return
}
func opCallDataSize(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(uint64(len(state.calldata)))
	return
}
func opGasprice(state *State) {
	d := zerOpArgVs(state)
	// d.Set(state.ctx.GasPrice)
	d.Set(&UNKNOWN_U256) // set to unknown, to continue predicting even if it wouldn't
	return
}
func opCoinbase(state *State) {
	d := zerOpArgVs(state)
	d.SetBytes(state.ctx.Coinbase.Bytes())
	return
}
func opTimestamp(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.Timestamp)
	return
}
func opNumber(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.BlockNumber)
	return
}
func opDifficulty(state *State) {
	d := zerOpArgVs(state)
	d.SetFromBig(state.ctx.Difficulty)
	return
}
func opMsize(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.mem.Msize())
	return
}
func opGas(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.GasLimit) // we don't track gas usage, so return max
	return
}
func opGasLimit(state *State) {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.GasLimit)
	return
}

func uniOp(state *State, op func (*uint256.Int, *uint256.Int) *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	d      := &state.regs[rd]
	v0     := &state.regs[r0]
	ok     := is_known(v0)
	if ok  { op(d, v0)
	} else { d.Set(&UNKNOWN_U256) }
	return
}
func opNot(state *State) { uniOp(state, (*uint256.Int).Not)  }

func _iszero(d, v0 *uint256.Int) *uint256.Int { if v0.IsZero() { return d.SetOne() } else { return d.Clear() } }
func opIszero(state *State) { uniOp(state, _iszero) }

func uniOpArgVs(state *State) (*uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	d      := &state.regs[rd]
	v0     := &state.regs[r0]
	ok     := is_known(v0)
	if !ok {
		d.Set(&UNKNOWN_U256)
		d, v0 = nil, nil
	}
	return d, v0
}
func opBalance(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	a := common.Address(v0.Bytes20())
	d.Set(state.ctx.ibs.GetBalance(a))
	// d.Set(&UNKNOWN_U256) // TEMP
	return
}
func opExtCodeSize(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	d.Set(&UNKNOWN_U256) // set to unknown, to continue predicting even if it wouldn't
	_ = v0
	// a := common.Address(v0.Bytes20())
	// d.SetUint64(uint64(state.ctx.ibs.GetCodeSize(a)))
	return
}
func opExtCodeHash(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	d.Set(&UNKNOWN_U256) // set to unknown, to continue predicting even if it wouldn't
	_ = v0
	// a := common.Address(v0.Bytes20())
	// if state.ctx.ibs.Empty(a) { // TODO: maybe speculatively skip check ?
	// 	d.Clear()
	// } else {
	// 	d.SetBytes(state.ctx.ibs.GetCodeHash(a).Bytes())
	// }
	return
}
func opSload(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	a := state.address
	// k := (*common.Hash)(v0) // hash is [32]byte :(
	k := (*common.Hash)(&state.ctx.buf)
	*k = v0.Bytes32()
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print("SLOAD ", k) }
	if common.TRACE_PREDICTED { state.ctx.Predicted = append(state.ctx.Predicted, *k) }
	state.ctx.ibs.GetState(a, k, d)
	// d.Set(&UNKNOWN_U256) // TEMP
	return
}
func opCallDataLoad(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	if v0.GtUint64(65535) {
		d.Set(&UNKNOWN_U256)
		return
	}
	i := v0.Uint64()
	d.SetBytes(getData(state.calldata, i, 32))
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(_enc(v0)) }
	return
}
func opBlockhash(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	if !v0.IsUint64() {
		d.Set(&UNKNOWN_U256)
		return
	}
	i     := v0.Uint64()
	delta := state.ctx.BlockNumber - i - 1
	if delta < 256 { d.SetBytes(state.ctx.getHashBytes(i))
	} else         { d.Clear() }
	return
}
func opMload(state *State) {
	d, v0 := uniOpArgVs(state)
	if d == nil { return }
	if !v0.IsUint64() {
		d.Set(&UNKNOWN_U256)
		return
	}
	i    := v0.Uint64()
	data := state.mem.get32(i)
	if data == nil {
		d.Set(&UNKNOWN_U256)
		return
	}
	d.SetBytes(data)
	return
}

func uniNVOpArgVs(state *State) (*uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	state.i = i
	v0     := &state.regs[r0]
	ok     := is_known(v0)
	if !ok { v0 = nil }
	return   v0
}
func opStouch(state *State) {
	v0 := uniNVOpArgVs(state)
	if v0 == nil { return }
	a := state.address
	k := (*common.Hash)(&state.ctx.buf)
	*k = v0.Bytes32()
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print("STOUCH ", k) }
	if common.TRACE_PREDICTED { state.ctx.Predicted = append(state.ctx.Predicted, *k) }
	// state.ctx.ibs.PrefetchState(a, k)
	state.ctx.ibs.SetDirtyState(a, k, UNKNOWN_U256)
	return
}

func binOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	d      := &state.regs[rd]
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	ok     := is_known(v0) && is_known(v1)
	if ok  { op(d, v0, v1)
	} else { d.Set(&UNKNOWN_U256) }
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(_enc(v0), _enc(v1), " [", r0, r1, "]") }
	return
}
func opAdd( state *State) { binOp(state, (*uint256.Int).Add)  }
func opSub( state *State) { binOp(state, (*uint256.Int).Sub)  }
func opMul( state *State) { binOp(state, (*uint256.Int).Mul)  }
func opDiv( state *State) { binOp(state, (*uint256.Int).Div)  }
func opSdiv(state *State) { binOp(state, (*uint256.Int).SDiv) }
func opMod( state *State) { binOp(state, (*uint256.Int).Mod)  }
func opSmod(state *State) { binOp(state, (*uint256.Int).SMod) }
func opAnd( state *State) { binOp(state, (*uint256.Int).And)  }
func opOr(  state *State) { binOp(state, (*uint256.Int).Or)   }
func opXor( state *State) { binOp(state, (*uint256.Int).Xor)  }

func _eq( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Eq( v1) { return d.SetOne() } else { return d.Clear() } }
func _lt( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Lt( v1) { return d.SetOne() } else { return d.Clear() } }
func _gt( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Gt( v1) { return d.SetOne() } else { return d.Clear() } }
func _slt(d, v0, v1 *uint256.Int) *uint256.Int { if v0.Slt(v1) { return d.SetOne() } else { return d.Clear() } }
func _sgt(d, v0, v1 *uint256.Int) *uint256.Int { if v0.Sgt(v1) { return d.SetOne() } else { return d.Clear() } }
func opEq( state *State) { binOp(state, _eq)  }
func opLt( state *State) { binOp(state, _lt)  }
func opGt( state *State) { binOp(state, _gt)  }
func opSlt(state *State) { binOp(state, _slt) }
func opSgt(state *State) { binOp(state, _sgt) }

func binOpArgVs(state *State) (*uint256.Int, *uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	d      := &state.regs[rd]
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	ok     := is_known(v0) && is_known(v1)
	if !ok {
		d.Set(&UNKNOWN_U256)
		d, v0, v1 = nil, nil, nil
	}
	return d, v0, v1
}
func opExp(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	b := v0
	e := v1
	switch {
		case e.IsZero():      d.SetOne()
		case b.IsZero():      d.Clear()
		case b.LtUint64(2):   d.SetOne()
		case e.LtUint64(2):   d.Set(b)
		case e.GtUint64(256): d.Set(&UNKNOWN_U256) // prob due to random data for unknowns
		case b.LtUint64(3):   d.Lsh(d.SetOne(), uint(e.Uint64()))
		default:              d.Exp(b, e)
	}
	return
}
func opSignExtend(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	i := v0
	x := v1
	if i.GtUint64(31) {
		d.Set(&UNKNOWN_U256) // same as EXP
		return
	}
	d.ExtendSign(x, i)
	return
}
func opByte(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	if v0.GtUint64(31) {
		d.Set(&UNKNOWN_U256) // same as EXP
		return
	}
	i := v0.Uint64()
	x := v1
	d.SetUint64((x[3 - i / 8] >> ((7 - i % 8) * 8)) & 0xFF)
	return
}
func opSHL(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	i := v0
	x := v1
	if i.GtUint64(255) {
		d.Set(&UNKNOWN_U256) // same as EXP
		return
	}
	d.Lsh(x, uint(i.Uint64()))
	return
}
func opSHR(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	i := v0
	x := v1
	if i.GtUint64(255) {
		d.Set(&UNKNOWN_U256) // same as EXP
		return
	}
	d.Rsh(x, uint(i.Uint64()))
	return
}
func opSAR(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	i := v0
	x := v1
	if i.GtUint64(255) {
		d.Set(&UNKNOWN_U256) // same as EXP
		return
	}
	d.SRsh(x, uint(i.Uint64()))
	return
}
func opSha3(state *State) {
	d, v0, v1 := binOpArgVs(state)
	if d == nil { return }
	i := v0.Uint64()
	s := v1.Uint64()
	data := state.mem.get(i, s)
	if data == nil {
		d.Set(&UNKNOWN_U256)
		return
	}
	d.SetBytes(state.ctx.SHA3(data))
	return
}

func binNVOpOptArgVs(state *State) (*uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(_enc(v0), _enc(v1), " [", r0, r1, "]") }
	return v0, v1
}
func opMstore(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if !is_known(v0) { return }
	i := v0.Uint64()
	state.mem.set32(i, v1.Bytes32())
	return
}
func opMstore8(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if !is_known(v0) { return }
	i := v0.Uint64()
	state.mem.setByte(i, byte(v1.Uint64()))
	return
}
func opSstore(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if !is_known(v0) { return }
	a := state.address
	k := (*common.Hash)(&state.ctx.buf)
	*k = v0.Bytes32()
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print("SSTORE ", k) }
	if common.TRACE_PREDICTED { state.ctx.Predicted = append(state.ctx.Predicted, *k) }
	state.ctx.ibs.SetDirtyState(a, k, *v1)
	return
}
func opReturn(state *State) {
	v0, v1 := binNVOpOptArgVs(state)
	if !is_known(v0) || !is_known(v1) { return }
	i := v0.Uint64()
	s := v1.Uint64()
	data := state.mem.get(i, s)
	if data != nil {
		state.ctx.returnData.set(0, s, data)
		state.ctx.returnSize = s
	} else {
		state.ctx.returnData.setUnknown(0, s)
		state.ctx.returnSize = random_u256_part_0
	}
	return
}

func triOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	d      := &state.regs[rd]
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	v2     := &state.regs[r2]
	ok     := is_known(v0) && is_known(v1) && is_known(v2)
	if ok  { op(d, v0, v1, v2)
	} else { d.Set(&UNKNOWN_U256) }
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(_enc(v0), _enc(v1), _enc(v2), " [", r0, r1, r2, "]") }
	return
}
func opAddmod(state *State) { triOp(state, (*uint256.Int).AddMod) }
func opMulmod(state *State) { triOp(state, (*uint256.Int).MulMod) }

func triNVOpArgVs(state *State) (*uint256.Int, *uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	v2     := &state.regs[r2]
	ok     := is_known(v0) && is_known(v1) && is_known(v2)
	if !ok { v0, v1, v2 = nil, nil, nil }
	return   v0, v1, v2
}
func opCallDataCopy(state *State) {
	v0, v1, v2 := triNVOpArgVs(state)
	if v0 == nil { return }
	if v0.GtUint64(65535) || v2.GtUint64(65535) { return }
	o := v0.Uint64()
	i := v1.Uint64()
	s := v2.Uint64()
	if v1.GtUint64(65535) {
		state.mem.setUnknown(o, s)
		return
	}
	data := getData(state.calldata, i, s)
	state.mem.set(o, s, data)
	return
}
func opReturnDataCopy(state *State) {
	v0, v1, v2 := triNVOpArgVs(state)
	if v0 == nil { return }
	o := v0.Uint64()
	i := v1.Uint64()
	s := v2.Uint64()
	data := state.ctx.returnData.get(i, s)
	if data != nil { state.mem.set(o, s, data)
	} else         { state.mem.setUnknown(o, s) }
	return
}
func opCodeCopy(state *State) {
	// simillar to CODESIZE, these should really have been optimized out by the analyzer
	v0, v1, v2 := triNVOpArgVs(state)
	if v0 == nil { return }
	o := v0.Uint64()
	// i := v1.Uint64()
	_ = v1
	s := v2.Uint64()
	// if i < 1024 * 1024 {
	// a    := state.address
	// data := getData(state.ctx.ibs.GetCode(a), i, s)
	// state.mem.set(o, s, data)
	// } else {
	state.mem.setUnknown(o, s)
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
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	v2     := &state.regs[r2]
	v3     := &state.regs[r3]
	ok     := is_known(v0) && is_known(v1) && is_known(v2) && is_known(v3)
	if !ok { v0, v1, v2, v3 = nil, nil, nil, nil }
	return   v0, v1, v2, v3
}
func opExtCodeCopy(state *State) {
	v0, v1, v2, v3 := tetNVOpArgVs(state)
	if v0 == nil { return }
	a := common.Address(v0.Bytes20())
	o := v1.Uint64()
	i := v2.Uint64()
	s := v3.Uint64()
	// if i < 1024 * 1024 {
	data := getData(state.ctx.ibs.GetCode(a), i, s)
	state.mem.set(o, s, data)
	// } else {
	// 	state.mem.setUnknown(o, s)
	// }
	return
}


type CallOpType byte
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
		i, r2 = getArg(state.code, i) // value
	}
	i, r3  := getArg(state.code, i) // i0
	i, r4  := getArg(state.code, i) // i1-i0
	i, r5  := getArg(state.code, i) // o0
	i, r6  := getArg(state.code, i) // o1-o0
	state.i = i
	d      := &state.regs[rd]
	v0     := &state.regs[r0]
	v1     := &state.regs[r1]
	v2     := &state.regs[r2]
	v3     := &state.regs[r3]
	v4     := &state.regs[r4]
	v5     := &state.regs[r5]
	v6     := &state.regs[r6]
	//
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print("CALL type ", t, " r ", r0,r1,r2,r3,r4,r5,r6, " v ", _enc(v0),_enc(v1),_enc(v2),_enc(v3),_enc(v4),_enc(v5),_enc(v6)) }
	//
	_ = v0 // ignore gas
	//
	ca    := common.Address(v1.Bytes20())
	i0    := v3.Uint64()
	iS    := v4.Uint64()
	idata := state.mem.get(i0, iS)
	if !(is_known(v1) && v3.IsUint64() && v4.IsUint64() && idata != nil) {
		d.Set(&UNKNOWN_U256)
		return
	}
	//
	ns := state.ctx.sp.NewState() // Careful: we can't return without FreeState() from now on
	                              //          we won't use defer for now
	if ns == nil {                // this must be checked on its own
		d.Set(&UNKNOWN_U256)
		return
	}
	ns.calldata = idata
	//
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(" calldata ", hex.EncodeToString(ns.calldata)) }
	//
	if t == CALL_REGULAR  { ns.address = ca
	} else                { ns.address = state.address }
	//
	if t == CALL_DELEGATE { ns.caller  = state.caller
	} else                { ns.caller  = state.address }
	//
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(" ns.address ", ns.address, " ns.caller ", ns.caller) }
	//
	switch {
		case t == CALL_DELEGATE: ns.callvalue.Set(&state.callvalue)
		case t == CALL_STATIC:   ns.callvalue.Clear()
		default:                 ns.callvalue.Set(v2)
	}
	//
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print(" ns.callvalue", _enc(&ns.callvalue)) }
	//
	reservedGaz := state.gaz /               common.PREDICTOR_RESERVE_GAZ_DIV
	ns.gaz       = state.gaz - reservedGaz + common.PREDICTOR_CALL_GAZ_BONUS
	//
	if common.DEBUG_TX && state.ctx.Debug { fmt.Println(" ns.gaz ", ns.gaz) }
	if common.DEBUG_TX && state.ctx.Debug { fmt.Println("---> call start") }
	//
	res, known  := predictCall(ns, ca)
	//
	if common.DEBUG_TX && state.ctx.Debug { fmt.Print("<---  call result ", res, known) }
	//
	state.gaz    = ns.gaz + reservedGaz
	state.ctx.sp.FreeState(ns)
	//
	if v5.IsUint64() && v6.IsUint64() {
		o0 := v5.Uint64()
		oS := v6.Uint64()
		if  oS > state.ctx.returnSize {
			oS = state.ctx.returnSize
		}
		odata := state.ctx.returnData.get(0, oS)
		if odata != nil {
			state.mem.set(o0, oS, odata)
			if common.DEBUG_TX && state.ctx.Debug { fmt.Print(" returnData ", hex.EncodeToString(odata)) }
		} else {
			state.mem.setUnknown(o0, oS)
			if common.DEBUG_TX && state.ctx.Debug { fmt.Print(" returnData unknown, size", oS) }
		}
	}
	//
	if known { d.SetUint64(uint64(res))
	} else   { d.Set(&UNKNOWN_U256) }
	return
}
func opCall(        state *State) { opCallCommon(state, CALL_REGULAR)  }
func opCallCode(    state *State) { opCallCommon(state, CALL_CODE)     }
func opDelegateCall(state *State) { opCallCommon(state, CALL_DELEGATE) }
func opStaticCall(  state *State) { opCallCommon(state, CALL_STATIC)   }

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
