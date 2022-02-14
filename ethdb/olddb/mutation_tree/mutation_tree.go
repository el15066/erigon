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
	FREELIST_SIZE = 32
	DEGREE        = 32
	MAX_ITEMS     = DEGREE * 2 - 1
)

// var (
// 	nilItems    = make([]*MutationItem, DEGREE)
// 	nilChildren = make([]*node,         DEGREE)
// )

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
	// n_items    int
	// n_children int
	items    []*MutationItem
	children []*node
}

func newNode() (n *node) {
	return &node{
		items:    make([]*MutationItem, 0, MAX_ITEMS),
		children: make([]*node,         0, MAX_ITEMS + 1),
	}
}

func (n *node) insertItemAt(i int, item *MutationItem) {
	l := len(n.items)
	n.items = n.items[:l+1]
	if i <= l {
		copy(n.items[i+1:], n.items[i:])
	}
	n.items[i] = item
}

func (n *node) truncateItems() {
	i       := MAX_ITEMS / 2
	// toClear := n.items[i:]
	n.items  = n.items[:i]
	// copy(toClear, nilItems)
}

func (n *node) findItem(item *MutationItem) (int, bool) {
	i := sort.Search(len(n.items), func(j int) bool {
		return item.Less(n.items[j])
	})
	if i > 0 && !n.items[i-1].Less(item) {
		return i - 1, true
	}
	return i, false
}

func (n *node) insertChildAt(i int, child *node) {
	l := len(n.children)
	n.children = n.children[:l+1]
	if i <= l {
		copy(n.children[i+1:], n.children[i:])
	}
	n.children[i] = child
}

func (n *node) truncateChildren() {
	i          := MAX_ITEMS / 2 + 1
	// toClear    := n.children[i:]
	n.children  = n.children[:i]
	// copy(toClear, nilChildren)
}

func (n *node) split() (*MutationItem, *node) {
	i    := MAX_ITEMS / 2
	item := n.items[i]
	next := newNode()
	next.items = append(next.items, n.items[i+1:]...)
	n.truncateItems()
	if len(n.children) > 0 {
		next.children = append(next.children, n.children[i+1:]...)
		n.truncateChildren()
	}
	return item, next
}

func (n *node) maybeSplitChild(i int) bool {
	if len(n.children[i].items) < MAX_ITEMS {
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
	if len(n.children) == 0 {
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
	} else if len(n.children) > 0 {
		return n.children[i].get(key)
	}
	return nil
}

type BTree struct {
	root *node
}

func (t *BTree) ReplaceOrInsert(item *MutationItem) *MutationItem {
	if t.root == nil {
		t.root       = newNode()
		t.root.items = append(t.root.items, item)
		return nil
	} else {
		if len(t.root.items) >= MAX_ITEMS {
			item2, second  := t.root.split()
			oldroot        := t.root
			t.root          = newNode()
			t.root.items    = append(t.root.items, item2)
			t.root.children = append(t.root.children, oldroot, second)
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
