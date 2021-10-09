package bench

import (
	"fmt"
	"time"
)

var all_ticks      [1000]int64
var all_ticks_prev [1000]int64
var all_counts     [1000]int64

func Tick(index int) {
	all_ticks_prev[index] = all_ticks[index]
	all_ticks[index]     += time.Now().UnixNano()
	all_counts[index]    += 1
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
	return fmt.Sprintf("%14d %14d %11d", (ticksA - ticksB) / countsA, countsA, (ticksA - ticksB) / 1000)
}

func DiffStrAuto(indexA int, indexB int) string {
	if        all_counts[indexA] > all_counts[indexB] { return DiffStr(indexA,  true, indexB, false)
	} else if all_counts[indexA] < all_counts[indexB] { return DiffStr(indexA, false, indexB,  true)
	} else                                            { return DiffStr(indexA, false, indexB, false) }
}
