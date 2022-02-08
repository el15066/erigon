package bench

import (
	"fmt"
	"time"
	"sync/atomic"
)

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

func Tick(index int) {
	// all_ticks_prev[index] = all_ticks[index] // TODO: atomic copy
	atomic.AddInt64(&all_counts[index], 1)
	t := time.Now().UnixNano()
	atomic.AddInt64(&all_ticks[ index], t)
}

func TiCk(index int) {
	t := time.Now().UnixNano()
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

func DiffStr(indexA int, isPrevA bool, indexB int, isPrevB bool) string {
	ticksA, countsA := Get(indexA, isPrevA)
	ticksB, countsB := Get(indexB, isPrevB)
	if countsA != countsB || countsA == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%14d %14d %14d", (ticksA - ticksB) / countsA, countsA, (ticksA - ticksB) / 1000)
}

func DiffStrAuto(indexA int, indexB int) string {
	// if        all_counts[indexA] > all_counts[indexB] { return DiffStr(indexA,  true, indexB, false)
	// } else if all_counts[indexA] < all_counts[indexB] { return DiffStr(indexA, false, indexB,  true)
	// } else                                            { return DiffStr(indexA, false, indexB, false) }
	return DiffStr(indexA, false, indexB, false)
}

func PrintAll() {
	fmt.Println("__TO-FROM ___AVERAGE_ns_ _______COUNT__ _____TOTAL_us_")
	pairs := len(all_counts) - 1
	for i := 0; i < pairs; i += 1 {
		if all_counts[i] > 0 && all_counts[i+1] > 0 {
			fmt.Println(fmt.Sprintf("%4d-%4d %s", i+1, i, DiffStrAuto(i+1, i)))
		}
	}
}
