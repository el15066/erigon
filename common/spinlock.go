// Copyright 2015 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// https://github.com/julienschmidt/spinlock/blob/master/rwmutex.go

// some modifications made and removed underflow checks

package common

import (
	"runtime"
	"sync"
	"sync/atomic"
)


type Spinlock struct {
	_    sync.Mutex // for compiler warning if copied, see https://github.com/tidwall/spinlock
	lock uint32
}
func (l *Spinlock) Lock() {
	for !atomic.CompareAndSwapUint32(&l.lock, 0, 1) {
		runtime.Gosched()
	}
}
func (l *Spinlock) RLock() { l.Lock() }

func (l *Spinlock) Unlock() {
	l.lock = 0
}
func (l *Spinlock) RUnlock() { l.Unlock() }


type RWSpinlock struct {
	_    sync.Mutex // for compiler warning if copied
	lock uint32
}
const (
	// Bit 1 is used as a flag for write mode
	// Bits 2-32 store the number of readers
	unlocked        = 0
	writer          = 0b0001
	readersIncrease = 0b0010
	writerDecrease  = ^uint32(writer          - 1)
	readersDecrease = ^uint32(readersIncrease - 1)
	// underflow    = ^uint32(writer)
)
func (l *RWSpinlock) RLock() {
	t := atomic.AddUint32(&l.lock, readersIncrease)
	//
	// if t & writer == 0 {
	// 	// common case, might actually be better to have this separate for the branch predictor (nope, all cases within 1-2%)
	// 	return
	// }
	//
	// for l.lock & writer != 0 { // might be a good idea not to let the compiler prefetch l.lock
	for t      & writer != 0 {    // here we force it to use t
		runtime.Gosched()
		t = l.lock
	}
}
func (l *RWSpinlock) RUnlock() {
	atomic.AddUint32(&l.lock, readersDecrease)
}
func (l *RWSpinlock) Lock() {
	for !atomic.CompareAndSwapUint32(&l.lock, unlocked, writer) {
		runtime.Gosched()
	}
}
func (l *RWSpinlock) Unlock() {
	atomic.AddUint32(&l.lock, writerDecrease)
}
