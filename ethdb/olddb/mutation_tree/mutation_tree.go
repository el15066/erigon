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

var (
	nilItems    = make(items,    DEGREE)
	nilChildren = make(children, DEGREE)
)

type FreeList struct {
	freelist []*node
}

func newNode() (n *node) {
	return new(node)
}

func New() *BTree {
	return &BTree{}
}

type items []*MutationItem

func (s *items) insertItemAt(index int, item *MutationItem) {
	*s = append(*s, nil)
	if index < len(*s) {
		copy((*s)[index+1:], (*s)[index:])
	}
	(*s)[index] = item
}

func (s *items) truncateItems() {
	index   := MAX_ITEMS / 2
	toClear := (*s)[index:]
	*s       = (*s)[:index]
	copy(toClear, nilItems)
}

func (s *items) findItem(item *MutationItem) (int, bool) {
	i := sort.Search(len((*s)), func(i int) bool {
		return item.Less((*s)[i])
	})
	if i > 0 && !(*s)[i-1].Less(item) {
		return i - 1, true
	}
	return i, false
}

type children []*node

func (s *children) insertChildAt(index int, n *node) {
	*s = append(*s, nil)
	if index < len(*s) {
		copy((*s)[index+1:], (*s)[index:])
	}
	(*s)[index] = n
}

func (s *children) truncateChildren() {
	index   := MAX_ITEMS / 2 + 1
	toClear := (*s)[index:]
	*s       = (*s)[:index]
	copy(toClear, nilChildren)
}

// `node` must at all times maintain the invariant that either
//   * len(children) == 0, len(items) unconstrained
//   * len(children) == len(items) + 1
type node struct {
	items    items
	children children
}

func (n *node) split() (*MutationItem, *node) {
	i    := MAX_ITEMS / 2
	item := n.items[i]
	next := newNode()
	next.items = append(next.items, n.items[i+1:]...)
	n.items.truncateItems()
	if len(n.children) > 0 {
		next.children = append(next.children, n.children[i+1:]...)
		n.children.truncateChildren()
	}
	return item, next
}

func (n *node) maybeSplitChild(i int) bool {
	if len(n.children[i].items) < MAX_ITEMS {
		return false
	}
	first := n.children[i]
	item, second := first.split()
	n.items.insertItemAt(i, item)
	n.children.insertChildAt(i+1, second)
	return true
}

func (n *node) insert(item *MutationItem) *MutationItem {
	i, found := n.items.findItem(item)
	if found {
		out := n.items[i]
		n.items[i] = item
		return out
	}
	if len(n.children) == 0 {
		n.items.insertItemAt(i, item)
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
	i, found := n.items.findItem(key)
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
