
package prediction

import (
	"fmt"
	"encoding/hex"

	common "github.com/ledgerwatch/erigon/common"
)

type Mem struct {
	data     [65536]byte
	msize    uint64
	modified bool
}

func (mem *Mem) Init() {
	mem.msize    = 0
	mem.modified = false
}

func (mem *Mem) Msize() uint64 {
	return mem.msize
}

func (mem *Mem) debug() {
	if common.DEBUG_TX && mem.modified {
		mem.modified = false
		for j := uint64(0); j < mem.msize; j += 0x20 {
			w := mem.data[j:j+0x20]
			if string(w) != ZEROS32 {
				fmt.Println(fmt.Sprintf("  mem %4x  %s", j, hex.EncodeToString(w)))
			}
		}
	}
}

func (mem *Mem) updateMsize(i1 uint64) {
	m1 := mem.msize
	m2 := (i1 + 31) & ^uint64(31)
	if m2 > m1 {
		t := mem.data[m1:m2]
		for i := range t { t[i] = 0 }
		mem.msize = m2
	}
}

func (mem *Mem) get(i0, s uint64) []byte {
	i1 := i0 + s
	if i1 > uint64(len(mem.data)) || i0 > i1 { return nil }
	mem.updateMsize(i1)
	return mem.data[i0:i1]
}

func (mem *Mem) set(i0, s uint64, data []byte) {
	i1 := i0 + s
	if i1 > uint64(len(mem.data)) || i0 > i1 { return }
	mem.updateMsize(i1)
	mem.modified = true
	copy(mem.data[i0:i1], data)
}

func (mem *Mem) setUnknown(i0, s uint64) {
	i1 := i0 + s
	if i1 > uint64(len(mem.data)) || i0 > i1 { return }
	mem.updateMsize(i1)
	mem.modified = true
	copy(mem.data[i0:i1], random_byte_string)
	// copy(mem.data[i0:i1], random_byte_string[i0&0x3:]) // tiny 0.001% worse
}

func (mem *Mem) setByte(i uint64, b byte) {
	if i >= uint64(len(mem.data))            { return }
	mem.updateMsize(i + 1)
	mem.modified = true
	mem.data[i]  = b
}

func (mem *Mem) getByte(i uint64) byte {
	if i >= uint64(len(mem.data))            { return 0 }
	mem.updateMsize(i + 1)
	return mem.data[i]
}

func (mem *Mem) get32(i0 uint64)        []byte  { return mem.get(i0, 32)          }
func (mem *Mem) set32(i0 uint64, data [32]byte) {        mem.set(i0, 32, data[:]) }
