// Copyright 2014 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copied from https://github.com/google/btree
// and modified for this specific use case.

package mutation_tree

import (
	"sort"
	"bytes"
)

type MutationItem struct {
	Key   []byte
	Value []byte
}

func (mi *MutationItem) Less(i *MutationItem) bool {
	return bytes.Compare(mi.Key, i.Key) < 0
}

const (
	DEGREE    = 128
	MAX_ITEMS = DEGREE * 2 - 1
)

// var nilItems    = [DEGREE]*MutationItem{}
// var nilChildren = [DEGREE]*node{}

type FreeList struct {
	freelist []*node
}

func New() *BTree {
	return &BTree{}
}

// `node` must at all times maintain the invariant that either
//   * len(children) == 0, len(items) unconstrained
//   * len(children) == len(items) + 1
type node struct {
	items_len    int
	children_len int
	items        [MAX_ITEMS  ]*MutationItem
	children     [MAX_ITEMS+1]*node
}

func newNode() (n *node) {
	return new(node)
}

func (n *node) insertItemAt(i int, item *MutationItem) {
	if i <= n.items_len {
		copy(n.items[i+1:], n.items[i:])
	}
	n.items_len += 1
	n.items[i]   = item
}

func (n *node) truncateItems() {
	const i     = MAX_ITEMS / 2
	n.items_len = i
	for j := i; j < len(n.items); j += 1 { n.items[j] = nil }
}

func (n *node) findItem(item *MutationItem) (int, bool) {
	i := sort.Search(n.items_len, func(j int) bool {
		return item.Less(n.items[j])
	})
	if i > 0 && !n.items[i-1].Less(item) {
		return i - 1, true
	}
	return i, false
}

func (n *node) insertChildAt(i int, child *node) {
	if i <= n.children_len {
		copy(n.children[i+1:], n.children[i:])
	}
	n.children_len += 1
	n.children[i]   = child
}

func (n *node) truncateChildren() {
	const i        = MAX_ITEMS / 2 + 1
	n.children_len = i
	for j := i; j < len(n.children); j += 1 { n.children[j] = nil }
}

func (n *node) split() (*MutationItem, *node) {
	const i = MAX_ITEMS / 2
	item := n.items[i]
	next := newNode()
	//
	next.items_len = MAX_ITEMS-i-1
	copy(next.items[:MAX_ITEMS-i-1], n.items[i+1:])
	n.truncateItems()
	//
	if n.children_len > 0 {
		next.children_len = MAX_ITEMS-i
		copy(next.children[:MAX_ITEMS-i], n.children[i+1:])
		n.truncateChildren()
	}
	return item, next
}

func (n *node) maybeSplitChild(i int) bool {
	if n.children[i].items_len < MAX_ITEMS {
		return false
	}
	first := n.children[i]
	item, second := first.split()
	n.insertItemAt(i, item)
	n.insertChildAt(i+1, second)
	return true
}

func (n *node) insert(item *MutationItem) *MutationItem {
	i, found := n.findItem(item)
	if found {
		out := n.items[i]
		n.items[i] = item
		return out
	}
	if n.children_len == 0 {
		n.insertItemAt(i, item)
		return nil
	}
	if n.maybeSplitChild(i) {
		inTree := n.items[i]
		switch {
		case item.Less(inTree):
			// no change, we want first split node
		case inTree.Less(item):
			i++ // we want second split node
		default:
			out := n.items[i]
			n.items[i] = item
			return out
		}
	}
	return n.children[i].insert(item)
}

func (n *node) get(key *MutationItem) *MutationItem {
	i, found := n.findItem(key)
	if found {
		return n.items[i]
	} else if n.children_len > 0 {
		return n.children[i].get(key)
	}
	return nil
}

type BTree struct {
	root *node
}

func (t *BTree) ReplaceOrInsert(item *MutationItem) *MutationItem {
	if t.root == nil {
		t.root           = newNode()
		t.root.items_len = 1
		t.root.items[0]  = item
		return nil
	} else {
		if t.root.items_len >= MAX_ITEMS {
			item2, second       := t.root.split()
			oldroot             := t.root
			t.root               = newNode()
			t.root.items_len     = 1
			t.root.items[0]      = item2
			t.root.children_len  = 2
			t.root.children[0]   = oldroot
			t.root.children[1]   = second
		}
	}
	return t.root.insert(item)
}

func (t *BTree) Get(key *MutationItem) *MutationItem {
	if t.root == nil {
		return nil
	}
	return t.root.get(key)
}

func (t *BTree) Has(key *MutationItem) bool {
	return t.Get(key) != nil
}

func (t *BTree) Clear() {
	t.root = nil
}
