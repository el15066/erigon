
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

func getArg(data []byte, i uint) (int, int) { return i+2, int(data[i]) | (int(data[i+1]) << 8) }

func zerOpArgs(state *State) int {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	state.i = i
	state.known[rd] = true
	return rd
}
func opReturnDataSize(state *State) error {
	rd := zerOpArgs(state)
	state.known[rd] = false
	return nil
}
func opCodeSize(state *State) error {
	rd := zerOpArgs(state)
	state.known[rd] = false
	// a := state.address // not always same as code address, maybe keep a ref to contract's code in state, also see CODECOPY
	// d.SetUint64(uint64(state.ctx.ibs.GetCodeSize(a)))
	return nil
}

func zerOpArgVs(state *State) *uint256.Int {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	state.i = i
	state.known[rd] = true
	d  := &state.regs[rd]
	return d
}
func opAddress(state *State) error {
	d := zerOpArgVs(state)
	d.SetBytes(state.address.Bytes())
	return nil
}
func opOrigin(state *State) error {
	d := zerOpArgVs(state)
	d.SetBytes(state.ctx.origin.Bytes())
	return nil
}
func opCaller(state *State) error {
	d := zerOpArgVs(state)
	d.SetBytes(state.caller.Bytes())
	return nil
}
func opCallValue(state *State) error {
	d := zerOpArgVs(state)
	d.Set(state.callvalue)
	return nil
}
func opCallDataSize(state *State) error {
	d := zerOpArgVs(state)
	d.SetUint64(len(state.calldata))
	return nil
}
func opGasprice(state *State) error {
	d := zerOpArgVs(state)
	d.Set(state.ctx.gasPrice)
	return nil
}
func opCoinbase(state *State) error {
	d := zerOpArgVs(state)
	d.SetBytes(state.ctx.coinbase.Bytes())
	return nil
}
func opTimestamp(state *State) error {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.timestamp)
	return nil
}
func opNumber(state *State) error {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.blockNumber)
	return nil
}
func opDifficulty(state *State) error {
	d := zerOpArgVs(state)
	d.Set(state.ctx.difficulty)
	return nil
}
func opMsize(state *State) error {
	d := zerOpArgVs(state)
	d.SetUint64(state.mem.msize())
	return nil
}
func opGas(state *State) error {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.gasLimit) // we don't track gas usage, so return max
	return nil
}
func opGasLimit(state *State) error {
	d := zerOpArgVs(state)
	d.SetUint64(state.ctx.gasLimit)
	return nil
}

func uniOp(state *State, op func (*uint256.Int, *uint256.Int) *uint256.Int) error {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0]
	state.known[rd] = ok
	if !ok { return nil }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	op(d, v0)
	return nil
}
func opNot(state *State) error { return uniOp(state, (uint256.Int).Not)  }

func _iszero(d, v0 *uint256.Int) *uint256.Int { if v0.IsZero() { return d.SetOne() } else { return d.Clear() } }
func opIszero(state *State) error { return uniOp(state, _iszero) }

func uniOpArgVs(state *State) (*uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0]
	state.known[rd] = ok
	if !ok { return nil, nil }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	return d, v0
}
func opBalance(state *State) error {
	d, v0 := uniOpArgVs(state)
	if d == nil { return nil }
	a := common.Address(v0.Bytes20())
	d.Set(state.ctx.ibs.GetBalance(a))
	return nil
}
func opExtCodeSize(state *State) error {
	d, v0 := uniOpArgVs(state)
	if d == nil { return nil }
	a := common.Address(v0.Bytes20())
	d.SetUint64(uint64(state.ctx.ibs.GetCodeSize(a)))
	return nil
}
func opExtCodeHash(state *State) error {
	d, v0 := uniOpArgVs(state)
	if d == nil { return nil }
	a := common.Address(v0.Bytes20())
	if state.ctx.ibs.Empty(a) { // TODO: maybe speculatively skip check ?
		d.Clear()
	} else {
		d.SetBytes(state.ctx.ibs.GetCodeHash(a).Bytes())
	}
	return nil
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
func opCallDataLoad(state *State) error {
	rd, r0 := uniOpArgs(state)
	if rd < 0 { return nil }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	if _i.GtUint64(65536) {
		state.known[rd] = false
		return nil
	}
	i := _i.Uint64()
	d.SetBytes(getData(state.calldata, i, 32))
	return nil
}
func opBlockhash(state *State) error {
	rd, r0 := uniOpArgs(state)
	if rd < 0 { return nil }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	if !_i.IsUint64() {
		state.known[rd] = false
		return nil
	}
	i     := _i.Uint64()
	delta := state.ctx.blockNumber - i - 1
	if delta < 256 { d.SetBytes(state.ctx.GetHash(i).Bytes())
	} else         { d.Clear() }
	return nil
}
func opMload(state *State) error {
	rd, r0 := uniOpArgs(state)
	if rd < 0 { return nil }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	if _i.GtUint64(65536) {
		state.known[rd] = false
		return nil
	}
	i := _i.Uint64()
	d.SetBytes(state.mem.get(i, 32))
	return nil
}

func binOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) error {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1]
	state.known[rd] = ok
	if !ok { return nil }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	op(d, v0, v1)
	return nil
}
func opAdd( state *State) error { return binOp(state, (uint256.Int).Add)  }
func opSub( state *State) error { return binOp(state, (uint256.Int).Sub)  }
func opMul( state *State) error { return binOp(state, (uint256.Int).Mul)  }
func opDiv( state *State) error { return binOp(state, (uint256.Int).Div)  }
func opSdiv(state *State) error { return binOp(state, (uint256.Int).SDiv) }
func opMod( state *State) error { return binOp(state, (uint256.Int).Mod)  }
func opSmod(state *State) error { return binOp(state, (uint256.Int).SMod) }
func opAnd( state *State) error { return binOp(state, (uint256.Int).And)  }
func opOr(  state *State) error { return binOp(state, (uint256.Int).Or)   }
func opXor( state *State) error { return binOp(state, (uint256.Int).Xor)  }

func _eq( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Eq( v1) { return d.SetOne() } else { return d.Clear() } }
func _lt( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Lt( v1) { return d.SetOne() } else { return d.Clear() } }
func _gt( d, v0, v1 *uint256.Int) *uint256.Int { if v0.Gt( v1) { return d.SetOne() } else { return d.Clear() } }
func _slt(d, v0, v1 *uint256.Int) *uint256.Int { if v0.Slt(v1) { return d.SetOne() } else { return d.Clear() } }
func _sgt(d, v0, v1 *uint256.Int) *uint256.Int { if v0.Sgt(v1) { return d.SetOne() } else { return d.Clear() } }
func opEq( state *State) error { return binOp(state, _eq)  }
func opLt( state *State) error { return binOp(state, _lt)  }
func opGt( state *State) error { return binOp(state, _gt)  }
func opSlt(state *State) error { return binOp(state, _slt) }
func opSgt(state *State) error { return binOp(state, _sgt) }

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
func opExp(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
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
	return nil
}
func opSignExtend(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(31) {
		state.known[rd] = false // same as EXP
		return nil
	}
	d.ExtendSign(x, i)
	return nil
}
func opByte(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
	d  := &state.regs[rd]
	_i := &state.regs[r0]
	x  := &state.regs[r1]
	if _i.GtUint64(31) {
		state.known[rd] = false // same as EXP
		return nil
	}
	i := _i.Uint64()
	d.SetUint64((x[3 - i / 8] >> ((7 - i % 8) * 8)) & 0xFF)
	return nil
}
func opSHL(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(255) {
		state.known[rd] = false // same as EXP
		return nil
	}
	d.Lsh(x, uint(i.Uint64()))
	return nil
}
func opSHR(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(255) {
		state.known[rd] = false // same as EXP
		return nil
	}
	d.Rsh(x, uint(i.Uint64()))
	return nil
}
func opSAR(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
	d := &state.regs[rd]
	i := &state.regs[r0]
	x := &state.regs[r1]
	if i.GtUint64(255) {
		state.known[rd] = false // same as EXP
		return nil
	}
	d.SRsh(x, uint(i.Uint64()))
	return nil
}
func opSha3(state *State) error {
	rd, r0, r1 := binOpArgs(state)
	if rd < 0 { return nil }
	d := &state.regs[rd]
	i := &state.regs[r0]
	s := &state.regs[r1]
	data, ok := state.mem.get(i.Uint64(), i.Uint64() + s.Uint64())
	if !ok {
		state.known[rd] = false
		return nil
	}
	state.ctx.hasher.Reset()
	state.ctx.hasher.Write(data)
	if _, err := state.ctx.hasher.Read(state.ctx.hasherBuf[:]); err != nil { return err }
	d.SetBytes(state.ctx.hasherBuf[:])
	return nil
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
func opMstore(state *State) error {
	v0, v1 := binNVOpOptArgVs(state)
	if v0 == nil { return nil }
	i := v0.Uint64()
	if v1 == nil { state.mem.SetUnknown32(i)
	} else       { state.mem.SetU256(i, v1) }
	return nil
}
func opMstore8(state *State) error {
	v0, v1 := binNVOpOptArgVs(state)
	if v0 == nil { return nil }
	i := v0.Uint64()
	if v1 == nil { state.mem.SetUnknown1(i)
	} else       { state.mem.SetByte(i, byte(v1.Uint64())) }
	return nil
}

func triOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) error {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2]
	state.known[rd] = ok
	if !ok { return nil }
	d  := &state.regs[rd]
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	v2 := &state.regs[r2]
	op(d, v0, v1, v2)
	return nil
}
func opAddmod(state *State) error { return triOp(state, (uint256.Int).AddMod) }
func opMulmod(state *State) error { return triOp(state, (uint256.Int).MulMod) }

func triOpArgs(state *State) (int, int, int, int) {
	i      := state.i + 1
	i, rd  := getArg(state.code, i)
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2]
	state.known[rd] = ok
	if !ok { rd = -1 }
	return rd, r0, r1, r2
}

func triNVOpArgVs(state *State) (*uint256.Int, *uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2]
	if !ok { return nil, nil, nil }
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	v2 := &state.regs[r2]
	return v0, v1, v2
}
func opCallDataCopy(state *State) error {
	v0, v1, v2 := triNVOpArgs(state)
	if v0 == nil { return nil }
	o := v0.Uint64()
	i := v1.Uint64()
	s := v2.Uint64()
	// if i < 65536 {
	data := getData(state.calldata, i, s)
	state.mem.Set(o, s, data)
	// } else {
	// 	state.mem.SetUnknown(o, s)
	// }
	return nil
}
func opReturnDataCopy(state *State) error {
	v0, v1, v2 := triNVOpArgs(state)
	if v0 == nil { return nil }
	o := v0.Uint64()
	// i := v1.Uint64()
	s := v2.Uint64()
	state.mem.SetUnknown(o, s)
	return nil
}
func opCodeCopy(state *State) error {
	// simillar to CODESIZE, these should really have been optimized out by the analyzer
	v0, v1, v2 := triNVOpArgs(state)
	if v0 == nil { return nil }
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
	return nil
}


func tetNVOpArgVs(state *State) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int) {
	i      := state.i + 1
	i, r0  := getArg(state.code, i)
	i, r1  := getArg(state.code, i)
	i, r2  := getArg(state.code, i)
	i, r3  := getArg(state.code, i)
	state.i = i
	ok := state.known[r0] && state.known[r1] && state.known[r2] && state.known[r3]
	if !ok { return nil, nil, nil, nil }
	v0 := &state.regs[r0]
	v1 := &state.regs[r1]
	v2 := &state.regs[r2]
	v3 := &state.regs[r3]
	return v0, v1, v2, v3
}
func opExtCodeCopy(state *State) error {
	v0, v1, v2, v3 := tetNVOpArgs(state)
	if v0 == nil { return nil }
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
	return nil
}











func opSload(state *State) error {
	loc := callContext.stack.Peek()
	interpreter.hasherBuf = loc.Bytes32()
	interpreter.evm.IntraBlockState.GetState(callContext.contract.Address(), &interpreter.hasherBuf, loc)
	return nil
}

func opSstore(state *State) error {
	loc := callContext.stack.Pop()
	val := callContext.stack.Pop()
	interpreter.hasherBuf = loc.Bytes32()
	interpreter.evm.IntraBlockState.SetState(callContext.contract.Address(), &interpreter.hasherBuf, val)
	return nil
}

func opJump(state *State) error {
	pos := callContext.stack.Pop()
	if valid, usedBitmap := callContext.contract.validJumpdest(&pos); !valid {
		if usedBitmap && interpreter.cfg.TraceJumpDest {
			log.Warn("Code Bitmap used for detecting invalid jump",
				"tx", fmt.Sprintf("0x%x", interpreter.evm.TxContext.TxHash),
				"block number", interpreter.evm.Context.BlockNumber,
			)
		}
		return nil, ErrInvalidJump
	}
	*pc = pos.Uint64()
	return nil
}

func opJumpi(state *State) error {
	pos, cond := callContext.stack.Pop(), callContext.stack.Pop()
	if !cond.IsZero() {
		if valid, usedBitmap := callContext.contract.validJumpdest(&pos); !valid {
			if usedBitmap && interpreter.cfg.TraceJumpDest {
				log.Warn("Code Bitmap used for detecting invalid jump",
					"tx", fmt.Sprintf("0x%x", interpreter.evm.TxContext.TxHash),
					"block number", interpreter.evm.Context.BlockNumber,
				)
			}
			return nil, ErrInvalidJump
		}
		*pc = pos.Uint64()
	} else {
		*pc++
	}
	return nil
}

func opCreate(state *State) error {
	var (
		value  = callContext.stack.Pop()
		offset = callContext.stack.Pop()
		size   = callContext.stack.Peek()
		input  = callContext.memory.GetCopy(offset.Uint64(), size.Uint64())
		gas    = callContext.contract.Gas
	)
	if interpreter.evm.ChainRules.IsEIP150 {
		gas -= gas / 64
	}
	// reuse size int for stackvalue
	stackvalue := size

	callContext.contract.UseGas(gas)

	res, addr, returnGas, suberr := interpreter.evm.Create(callContext.contract, input, gas, &value)

	// Push item on the stack based on the returned error. If the ruleset is
	// homestead we must check for CodeStoreOutOfGasError (homestead only
	// rule) and treat as an error, if the ruleset is frontier we must
	// ignore this error and pretend the operation was successful.
	if interpreter.evm.ChainRules.IsHomestead && suberr == ErrCodeStoreOutOfGas {
		stackvalue.Clear()
	} else if suberr != nil && suberr != ErrCodeStoreOutOfGas {
		stackvalue.Clear()
	} else {
		stackvalue.SetBytes(addr.Bytes())
	}
	callContext.contract.Gas += returnGas

	if suberr == ErrExecutionReverted {
		return res, nil
	}
	return nil
}

func opCreate2(state *State) error {
	var (
		endowment    = callContext.stack.Pop()
		offset, size = callContext.stack.Pop(), callContext.stack.Pop()
		salt         = callContext.stack.Pop()
		input        = callContext.memory.GetCopy(offset.Uint64(), size.Uint64())
		gas          = callContext.contract.Gas
	)

	// Apply EIP150
	gas -= gas / 64
	callContext.contract.UseGas(gas)
	// reuse size int for stackvalue
	stackValue := size
	res, addr, returnGas, suberr := interpreter.evm.Create2(callContext.contract, input, gas, &endowment, &salt)

	// Push item on the stack based on the returned error.
	if suberr != nil {
		stackValue.Clear()
	} else {
		stackValue.SetBytes(addr.Bytes())
	}

	callContext.stack.Push(&stackValue)
	callContext.contract.Gas += returnGas

	if suberr == ErrExecutionReverted {
		return res, nil
	}
	return nil
}

func opCall(state *State) error {
	stack := callContext.stack
	// Pop gas. The actual gas in interpreter.evm.callGasTemp.
	// We can use this as a temporary value
	temp := stack.Pop()
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.Address(addr.Bytes20())
	// Get the arguments from the memory.
	args := callContext.memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	if !value.IsZero() {
		gas += params.CallStipend
	}

	ret, returnGas, err := interpreter.evm.Call(callContext.contract, toAddr, args, gas, &value, false /* bailout */)

	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.Push(&temp)
	if err == nil || err == ErrExecutionReverted {
		ret = common.CopyBytes(ret)
		callContext.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}

	callContext.contract.Gas += returnGas

	return ret, nil
}

func opCallCode(state *State) error {
	// Pop gas. The actual gas is in interpreter.evm.callGasTemp.
	stack := callContext.stack
	// We use it as a temporary value
	temp := stack.Pop()
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := callContext.memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	//TODO: use uint256.Int instead of converting with toBig()
	if !value.IsZero() {
		gas += params.CallStipend
	}

	ret, returnGas, err := interpreter.evm.CallCode(callContext.contract, toAddr, args, gas, &value)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.Push(&temp)
	if err == nil || err == ErrExecutionReverted {
		ret = common.CopyBytes(ret)
		callContext.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}

	callContext.contract.Gas += returnGas

	return ret, nil
}

func opDelegateCall(state *State) error {
	stack := callContext.stack
	// Pop gas. The actual gas is in interpreter.evm.callGasTemp.
	// We use it as a temporary value
	temp := stack.Pop()
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := callContext.memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	// fmt.Println("opDelegateCall", toAddr)
	ret, returnGas, err := interpreter.evm.DelegateCall(callContext.contract, toAddr, args, gas)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.Push(&temp)
	if err == nil || err == ErrExecutionReverted {
		ret = common.CopyBytes(ret)
		callContext.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}

	callContext.contract.Gas += returnGas

	return ret, nil
}

func opStaticCall(state *State) error {
	// Pop gas. The actual gas is in interpreter.evm.callGasTemp.
	stack := callContext.stack
	// We use it as a temporary value
	temp := stack.Pop()
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop(), stack.Pop()
	toAddr := common.Address(addr.Bytes20())
	// Get arguments from the memory.
	args := callContext.memory.GetPtr(inOffset.Uint64(), inSize.Uint64())

	ret, returnGas, err := interpreter.evm.StaticCall(callContext.contract, toAddr, args, gas)
	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.Push(&temp)
	if err == nil || err == ErrExecutionReverted {
		ret = common.CopyBytes(ret)
		callContext.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}

	callContext.contract.Gas += returnGas

	return ret, nil
}

func opReturn(state *State) error {
	offset, size := callContext.stack.Pop(), callContext.stack.Pop()
	ret := callContext.memory.GetPtr(offset.Uint64(), size.Uint64())
	return ret, nil
}

func opRevert(state *State) error {
	offset, size := callContext.stack.Pop(), callContext.stack.Pop()
	ret := callContext.memory.GetPtr(offset.Uint64(), size.Uint64())
	return ret, nil
}

func opStop(state *State) error {
	return nil
}

func opSuicide(state *State) error {
	beneficiary := callContext.stack.Pop()
	callerAddr := callContext.contract.Address()
	beneficiaryAddr := common.Address(beneficiary.Bytes20())
	balance := interpreter.evm.IntraBlockState.GetBalance(callerAddr)
	interpreter.evm.IntraBlockState.AddBalance(beneficiaryAddr, balance)
	if interpreter.evm.Config.Debug {
		interpreter.evm.Config.Tracer.CaptureSelfDestruct(callerAddr, beneficiaryAddr, balance.ToBig())
	}
	interpreter.evm.IntraBlockState.Suicide(callerAddr)
	return nil
}

// following functions are used by the instruction jump  table

// make log instruction function
func makeLog(size int) executionFunc {
	return func(state *State) error {
		topics := make([]common.Hash, size)
		stack := callContext.stack
		mStart, mSize := stack.Pop(), stack.Pop()
		for i := 0; i < size; i++ {
			addr := stack.Pop()
			topics[i] = common.Hash(addr.Bytes32())
		}

		d := callContext.memory.GetCopy(mStart.Uint64(), mSize.Uint64())
		interpreter.evm.IntraBlockState.AddLog(&types.Log{
			Address: callContext.contract.Address(),
			Topics:  topics,
			Data:    d,
			// This is a non-consensus field, but assigned here because
			// core/state doesn't know the current block number.
			BlockNumber: interpreter.evm.Context.BlockNumber,
		})

		return nil
	}
}

// opPush1 is a specialized version of pushN
func opPush1(state *State) error {
	var (
		codeLen = uint64(len(callContext.contract.Code))
		integer = new(uint256.Int)
	)
	*pc++
	if *pc < codeLen {
		callContext.stack.Push(integer.SetUint64(uint64(callContext.contract.Code[*pc])))
	} else {
		callContext.stack.Push(integer.Clear())
	}
	return nil
}

// make push instruction function
func makePush(size uint64, pushByteSize int) executionFunc {
	return func(state *State) error {
		codeLen := len(callContext.contract.Code)

		startMin := int(*pc + 1)
		if startMin >= codeLen {
			startMin = codeLen
		}
		endMin := startMin + pushByteSize
		if startMin+pushByteSize >= codeLen {
			endMin = codeLen
		}

		integer := new(uint256.Int)
		callContext.stack.Push(integer.SetBytes(common.RightPadBytes(
			// So it doesn't matter what we push onto the stack.
			callContext.contract.Code[startMin:endMin], pushByteSize)))

		*pc += size
		return nil
	}
}

// make dup instruction function
func makeDup(size int64) executionFunc {
	return func(state *State) error {
		callContext.stack.Dup(int(size))
		return nil
	}
}

// make swap instruction function
func makeSwap(size int64) executionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size++
	return func(state *State) error {
		callContext.stack.Swap(int(size))
		return nil
	}
}

