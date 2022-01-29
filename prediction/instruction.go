
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


func arg2B(data []byte, i uint) int { return uint(data[i]) | (uint(data[i+1]) << 8) }

func uniOp(state *State, op func (*uint256.Int, *uint256.Int) *uint256.Int) error {
	i  := state.i + 1
	rd := arg2B(state.code, i)
	r0 := arg2B(state.code, i + 2)
	state.i = i + 4
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

func binOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) error {
	i  := state.i + 1
	rd := arg2B(state.code, i)
	r0 := arg2B(state.code, i + 2)
	r1 := arg2B(state.code, i + 4)
	state.i = i + 6
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
	i  := state.i + 1
	rd := arg2B(state.code, i)
	r0 := arg2B(state.code, i + 2)
	r1 := arg2B(state.code, i + 4)
	state.i = i + 6
	ok := state.known[r0] && state.known[r1]
	state.known[rd] = ok
	if !ok { return -1, -1, -1 }
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

func triOp(state *State, op func (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int) *uint256.Int) error {
	i  := state.i + 1
	rd := arg2B(state.code, i)
	r0 := arg2B(state.code, i + 2)
	r1 := arg2B(state.code, i + 4)
	r2 := arg2B(state.code, i + 6)
	state.i = i + 8
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

func opAddress(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetBytes(callContext.contract.Address().Bytes()))
	return nil, nil
}

func opBalance(state *State) error {
	slot := callContext.stack.Peek()
	address := common.Address(slot.Bytes20())
	slot.Set(interpreter.evm.IntraBlockState.GetBalance(address))
	return nil, nil
}

func opOrigin(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetBytes(interpreter.evm.Origin.Bytes()))
	return nil, nil
}
func opCaller(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetBytes(callContext.contract.Caller().Bytes()))
	return nil, nil
}

func opCallValue(state *State) error {
	callContext.stack.Push(callContext.contract.value)
	return nil, nil
}

func opCallDataLoad(state *State) error {
	x := callContext.stack.Peek()
	if offset, overflow := x.Uint64WithOverflow(); !overflow {
		data := getData(callContext.contract.Input, offset, 32)
		x.SetBytes(data)
	} else {
		x.Clear()
	}
	return nil, nil
}

func opCallDataSize(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetUint64(uint64(len(callContext.contract.Input))))
	return nil, nil
}

func opCallDataCopy(state *State) error {
	var (
		memOffset  = callContext.stack.Pop()
		dataOffset = callContext.stack.Pop()
		length     = callContext.stack.Pop()
	)
	dataOffset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		dataOffset64 = 0xffffffffffffffff
	}
	// These values are checked for overflow during gas cost calculation
	memOffset64 := memOffset.Uint64()
	length64 := length.Uint64()
	callContext.memory.Set(memOffset64, length64, getData(callContext.contract.Input, dataOffset64, length64))
	return nil, nil
}

func opReturnDataSize(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetUint64(uint64(len(interpreter.returnData))))
	return nil, nil
}

func opReturnDataCopy(state *State) error {
	var (
		memOffset  = callContext.stack.Pop()
		dataOffset = callContext.stack.Pop()
		length     = callContext.stack.Pop()
	)

	offset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		return nil, ErrReturnDataOutOfBounds
	}
	// we can reuse dataOffset now (aliasing it for clarity)
	end := dataOffset
	_, overflow = end.AddOverflow(&dataOffset, &length)
	if overflow {
		return nil, ErrReturnDataOutOfBounds
	}

	end64, overflow := end.Uint64WithOverflow()
	if overflow || uint64(len(interpreter.returnData)) < end64 {
		return nil, ErrReturnDataOutOfBounds
	}
	callContext.memory.Set(memOffset.Uint64(), length.Uint64(), interpreter.returnData[offset64:end64])
	return nil, nil
}

func opExtCodeSize(state *State) error {
	slot := callContext.stack.Peek()
	slot.SetUint64(uint64(interpreter.evm.IntraBlockState.GetCodeSize(common.Address(slot.Bytes20()))))
	return nil, nil
}

func opCodeSize(state *State) error {
	l := new(uint256.Int)
	l.SetUint64(uint64(len(callContext.contract.Code)))
	callContext.stack.Push(l)
	return nil, nil
}

func opCodeCopy(state *State) error {
	var (
		memOffset  = callContext.stack.Pop()
		codeOffset = callContext.stack.Pop()
		length     = callContext.stack.Pop()
	)
	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = 0xffffffffffffffff
	}
	codeCopy := getData(callContext.contract.Code, uint64CodeOffset, length.Uint64())
	callContext.memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)
	return nil, nil
}

func opExtCodeCopy(state *State) error {
	var (
		stack      = callContext.stack
		a          = stack.Pop()
		memOffset  = stack.Pop()
		codeOffset = stack.Pop()
		length     = stack.Pop()
	)
	addr := common.Address(a.Bytes20())
	len64 := length.Uint64()
	codeCopy := getDataBig(interpreter.evm.IntraBlockState.GetCode(addr), &codeOffset, len64)
	callContext.memory.Set(memOffset.Uint64(), len64, codeCopy)
	return nil, nil
}

// opExtCodeHash returns the code hash of a specified account.
// There are several cases when the function is called, while we can relay everything
// to `state.GetCodeHash` function to ensure the correctness.
//   (1) Caller tries to get the code hash of a normal contract account, state
// should return the relative code hash and set it as the result.
//
//   (2) Caller tries to get the code hash of a non-existent account, state should
// return common.Hash{} and zero will be set as the result.
//
//   (3) Caller tries to get the code hash for an account without contract code,
// state should return emptyCodeHash(0xc5d246...) as the result.
//
//   (4) Caller tries to get the code hash of a precompiled account, the result
// should be zero or emptyCodeHash.
//
// It is worth noting that in order to avoid unnecessary create and clean,
// all precompile accounts on mainnet have been transferred 1 wei, so the return
// here should be emptyCodeHash.
// If the precompile account is not transferred any amount on a private or
// customized chain, the return value will be zero.
//
//   (5) Caller tries to get the code hash for an account which is marked as suicided
// in the current transaction, the code hash of this account should be returned.
//
//   (6) Caller tries to get the code hash for an account which is marked as deleted,
// this account should be regarded as a non-existent account and zero should be returned.
func opExtCodeHash(state *State) error {
	slot := callContext.stack.Peek()
	address := common.Address(slot.Bytes20())
	if interpreter.evm.IntraBlockState.Empty(address) {
		slot.Clear()
	} else {
		slot.SetBytes(interpreter.evm.IntraBlockState.GetCodeHash(address).Bytes())
	}
	return nil, nil
}

func opGasprice(state *State) error {
	v, overflow := uint256.FromBig(interpreter.evm.GasPrice)
	if overflow {
		return nil, fmt.Errorf("interpreter.evm.GasPrice higher than 2^256-1")
	}
	callContext.stack.Push(v)
	return nil, nil
}

func opBlockhash(state *State) error {
	num := callContext.stack.Peek()
	num64, overflow := num.Uint64WithOverflow()
	if overflow {
		num.Clear()
		return nil, nil
	}
	var upper, lower uint64
	upper = interpreter.evm.Context.BlockNumber
	if upper < 257 {
		lower = 0
	} else {
		lower = upper - 256
	}
	if num64 >= lower && num64 < upper {
		num.SetBytes(interpreter.evm.Context.GetHash(num64).Bytes())
	} else {
		num.Clear()
	}
	return nil, nil
}

func opCoinbase(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetBytes(interpreter.evm.Context.Coinbase.Bytes()))
	return nil, nil
}

func opTimestamp(state *State) error {
	v := new(uint256.Int).SetUint64(interpreter.evm.Context.Time)
	callContext.stack.Push(v)
	return nil, nil
}

func opNumber(state *State) error {
	v := new(uint256.Int).SetUint64(interpreter.evm.Context.BlockNumber)
	callContext.stack.Push(v)
	return nil, nil
}

func opDifficulty(state *State) error {
	v, overflow := uint256.FromBig(interpreter.evm.Context.Difficulty)
	if overflow {
		return nil, fmt.Errorf("interpreter.evm.Context.Difficulty higher than 2^256-1")
	}
	callContext.stack.Push(v)
	return nil, nil
}

func opGasLimit(state *State) error {
	if interpreter.evm.Context.MaxGasLimit {
		callContext.stack.Push(new(uint256.Int).SetAllOne())
	} else {
		callContext.stack.Push(new(uint256.Int).SetUint64(interpreter.evm.Context.GasLimit))
	}
	return nil, nil
}

func opPop(state *State) error {
	callContext.stack.Pop()
	return nil, nil
}

func opMload(state *State) error {
	v := callContext.stack.Peek()
	offset := v.Uint64()
	v.SetBytes(callContext.memory.GetPtr(offset, 32))
	return nil, nil
}

func opMstore(state *State) error {
	mStart, val := callContext.stack.Pop(), callContext.stack.Pop()
	callContext.memory.Set32(mStart.Uint64(), &val)
	return nil, nil
}

func opMstore8(state *State) error {
	off, val := callContext.stack.Pop(), callContext.stack.Pop()
	callContext.memory.store[off.Uint64()] = byte(val.Uint64())
	return nil, nil
}

func opSload(state *State) error {
	loc := callContext.stack.Peek()
	interpreter.hasherBuf = loc.Bytes32()
	interpreter.evm.IntraBlockState.GetState(callContext.contract.Address(), &interpreter.hasherBuf, loc)
	return nil, nil
}

func opSstore(state *State) error {
	loc := callContext.stack.Pop()
	val := callContext.stack.Pop()
	interpreter.hasherBuf = loc.Bytes32()
	interpreter.evm.IntraBlockState.SetState(callContext.contract.Address(), &interpreter.hasherBuf, val)
	return nil, nil
}

func traceJump(h common.Hash, src uint64, dst uint64) {
	if h == (common.Hash{}) { return }

	common.JUMP_COUNT[h] += 1
	// {
	// 	e := (src & 0xFFFFFFFF) | (dst << 32)
	// 	m := common.JUMP_EDGE_COUNT[h]
	// 	if m == nil {
	// 		m = map[uint64]uint{}
	// 		common.JUMP_EDGE_COUNT[h] = m
	// 	}
	// 	m[e] += 1
	// }
	{
		m := common.JUMP_CALLS[h]
		if m == nil {
			m = map[uint32]struct{}{}
			common.JUMP_CALLS[h] = m
		}
		m[common.CALLID] = struct{}{}
	}
	{
		m1 := common.JUMP_DST_CALLCOUNT[h]
		if m1 == nil {
			m1 = map[uint32]map[uint32]map[uint32]uint{}
			common.JUMP_DST_CALLCOUNT[h] = m1
		}
		m2 := m1[uint32(src)]
		if m2 == nil {
			m2 = map[uint32]map[uint32]uint{}
			m1[uint32(src)] = m2
		}
		m3 := m2[uint32(dst)]
		if m3 == nil {
			m3 = map[uint32]uint{}
			m2[uint32(dst)] = m3
		}
		m3[common.CALLID] += 1
	}
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
	if common.JUMP_TRACING { traceJump(callContext.contract.CodeHash, *pc, pos.Uint64()) }
	*pc = pos.Uint64()
	return nil, nil
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
		if common.JUMP_TRACING { traceJump(callContext.contract.CodeHash, *pc, pos.Uint64()) }
		*pc = pos.Uint64()
	} else {
		*pc++
	}
	return nil, nil
}

func opJumpdest(state *State) error {
	return nil, nil
}

func opPc(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetUint64(*pc))
	return nil, nil
}

func opMsize(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetUint64(uint64(callContext.memory.Len())))
	return nil, nil
}

func opGas(state *State) error {
	callContext.stack.Push(new(uint256.Int).SetUint64(callContext.contract.Gas))
	return nil, nil
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
	return nil, nil
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
	return nil, nil
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
	return nil, nil
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
	return nil, nil
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

		return nil, nil
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
	return nil, nil
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
		return nil, nil
	}
}

// make dup instruction function
func makeDup(size int64) executionFunc {
	return func(state *State) error {
		callContext.stack.Dup(int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) executionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size++
	return func(state *State) error {
		callContext.stack.Swap(int(size))
		return nil, nil
	}
}

