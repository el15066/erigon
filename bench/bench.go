package bench

import (
	"time"
)

var all_ticks      [20]int64
var all_ticks_prev [20]int64
var all_counts     [20]int64

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
