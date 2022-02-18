package bench

import (
	"fmt"
	// "time"
	"sync/atomic"

	// mclock "github.com/ledgerwatch/erigon/common/mclock"

	// _ "unsafe" // for go:linkname

	tsc "github.com/el15066/gotsc"
)

// The TSC frequency, usually the advertised base frequency
const TSC_FREQ_100MHZ = 38 // 3.8 GHz

// //go:noescape
// //go:linkname Nanotime runtime.nanotime
// func Nanotime() int64 // on linux: __vdso_clock_gettime

var all_ticks      [300]int64
var all_ticks_prev [300]int64
var all_counts     [300]int64

func Reset() {
	for i := range all_ticks {
		all_ticks[i]      = 0
		all_ticks_prev[i] = 0
		all_counts[i]     = 0
	}
}

// func Tick(index int) {
// 	// all_ticks_prev[index] = all_ticks[index] // TODO: atomic copy
// 	atomic.AddInt64(&all_counts[index], 1)
// 	// t := time.Now().UnixNano()
// 	// t := int64(mclock.Now())
// 	// t := Nanotime()
// 	// _  = all_ticks[ index] // cache it // unfortunately this kills inlining (and is optimized out)
// 	t := int64(tsc.BenchStart())
// 	atomic.AddInt64(&all_ticks[ index], t)
// }

// func Tick(index int) { TiCk(index) } // doesn't inline :(

func Tick(index int) {
	t := int64(tsc.BenchEnd())
	atomic.AddInt64(&all_ticks[ index], t)
	atomic.AddInt64(&all_counts[index], 1)
}

func TiCk(index int) {
	// t := time.Now().UnixNano()
	// t := int64(mclock.Now())
	// t := Nanotime()
	t := int64(tsc.BenchEnd())
	// all_ticks_prev[index] = all_ticks[index] // TODO: atomic copy
	atomic.AddInt64(&all_ticks[ index], t)
	atomic.AddInt64(&all_counts[index], 1)
}

func Get(index int, isPrev bool) (int64, int64) {
	var ticks  int64
	var counts int64
	if isPrev {
		ticks  = all_ticks_prev[index]
		counts = all_counts[index] - 1
	} else {
		ticks  = all_ticks[index]
		counts = all_counts[index]
	}
	return ticks, counts
}

func Diff(indexA int, isPrevA bool, indexB int, isPrevB bool) int64 {
	ticksA, countsA := Get(indexA, isPrevA)
	ticksB, countsB := Get(indexB, isPrevB)
	if countsA != countsB || countsA == 0 {
		return 0
	}
	return (ticksA - ticksB) / countsA
}

func DiffAuto(indexA int, indexB int) int64 {
	if        all_counts[indexA] > all_counts[indexB] { return Diff(indexA,  true, indexB, false)
	} else if all_counts[indexA] < all_counts[indexB] { return Diff(indexA, false, indexB,  true)
	} else                                            { return Diff(indexA, false, indexB, false) }
}

func DiffStrThese(ticksA, countsA, ticksB, countsB int64) string {
	if countsA != countsB || countsA == 0 {
		return "N/A"
	}
	//
	if TSC_FREQ_100MHZ == 0 {
		return fmt.Sprintf("%14d %14d %14d", (ticksA - ticksB) / countsA, countsA, (ticksA - ticksB) / 1000)
	} else {
		return fmt.Sprintf("%14d %14d %14d",
			((ticksA - ticksB) * 10) / (TSC_FREQ_100MHZ * countsA),
			countsA,
			 (ticksA - ticksB)       / (TSC_FREQ_100MHZ * 100),
		)
	}
}

func DiffStr(indexA int, isPrevA bool, indexB int, isPrevB bool) string {
	ticksA, countsA := Get(indexA, isPrevA)
	ticksB, countsB := Get(indexB, isPrevB)
	return DiffStrThese(ticksA, countsA, ticksB, countsB)
}

func DiffStrAuto(indexA int, indexB int) string {
	// if        all_counts[indexA] > all_counts[indexB] { return DiffStr(indexA,  true, indexB, false)
	// } else if all_counts[indexA] < all_counts[indexB] { return DiffStr(indexA, false, indexB,  true)
	// } else                                            { return DiffStr(indexA, false, indexB, false) }
	return DiffStr(indexA, false, indexB, false)
}

// func PrintAllNoCheck() {
// 	fmt.Println("__TO-FROM ___AVERAGE_ns_ _______COUNT__ _____TOTAL_us_")
// 	pairs := len(all_counts) - 1
// 	for i := 0; i < pairs; i += 1 {
// 		if all_counts[i] > 0 && all_counts[i+1] > 0 {
// 			fmt.Println(fmt.Sprintf("%4d-%4d %s", i+1, i, DiffStrAuto(i+1, i)))
// 		}
// 	}
// }

func PrintAll() {
	fmt.Println("__TO-FROM ___AVERAGE_ns_ _______COUNT__ _____TOTAL_us_")
	pairs := len(all_counts) - 1
	for i := 0; i < pairs; i += 1 {
		// we don't want to lock, because Tick() should be fast
		// so we approximate by reading until no change
		// it can still produce N/A but it's rare
		var  c0,  c1,  t0,  t1 int64
		var _c0, _c1, _t0, _t1 int64
		for {
			// atomic load here causes significant overhead, a measurable 0.5-1% in the whole execution benchmark (!!)
			// c0 = atomic.LoadInt64(&all_counts[i])
			// c1 = atomic.LoadInt64(&all_counts[i + 1])
			// t0 = atomic.LoadInt64(&all_ticks[ i])
			// t1 = atomic.LoadInt64(&all_ticks[ i + 1])
			c0 = all_counts[i]
			c1 = all_counts[i + 1]
			t0 = all_ticks[ i]
			t1 = all_ticks[ i + 1]
			if c0 == _c0 && c1 == _c1 && t0 == _t0 && t1 == _t1 { break }
			_c0, _c1, _t0, _t1 = c0, c1, t0, t1
		}
		if c0 > 0 && c1 > 0 {
			fmt.Println(fmt.Sprintf("%4d-%4d %s", i+1, i, DiffStrThese(t1, c1, t0, c0)))
		}
	}
}
