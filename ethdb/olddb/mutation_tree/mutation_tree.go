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

// Package btree implements in-memory B-Trees of arbitrary degree.
//
// btree implements an in-memory B-Tree for use as an ordered data structure.
// It is not meant for persistent storage solutions.
//
// It has a flatter structure than an equivalent red-black or other binary tree,
// which in some cases yields better memory usage and/or performance.
// See some discussion on the matter here:
//   http://google-opensource.blogspot.com/2013/01/c-containers-that-save-memory-and-time.html
// Note, though, that this project is in no way related to the C++ B-Tree
// implementation written about there.
//
// Within this tree, each node contains a slice of items and a (possibly nil)
// slice of children.  For basic numeric values or raw structs, this can cause
// efficiency differences when compared to equivalent C++ template code that
// stores values in arrays within the node:
//   * Due to the overhead of storing values as interfaces (each
//     value needs to be stored as the value itself, then 2 words for the
//     interface pointing to that value and its type), resulting in higher
//     memory use.
//   * Since interfaces can point to values anywhere in memory, values are
//     most likely not stored in contiguous blocks, resulting in a higher
//     number of cache misses.
// These issues don't tend to matter, though, when working with strings or other
// heap-allocated structures, since C++-equivalent structures also must store
// pointers and also distribute their values across the heap.
//
// This implementation is designed to be a drop-in replacement to gollrb.LLRB
// trees, (http://github.com/petar/gollrb), an excellent and probably the most
// widely used ordered tree implementation in the Go ecosystem currently.
// Its functions, therefore, exactly mirror those of
// llrb.LLRB where possible.  Unlike gollrb, though, we currently don't
// support storing multiple equivalent values.
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
	// i := than.(*MutationItem)
	// c := mi.table - i.table
	// if c != 0 {
	// 	return c < 0
	// }
	return bytes.Compare(mi.Key, i.Key) < 0
	// return string(mi.key) < string(i.key) // TODO
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

// FreeList represents a free list of btree nodes. By default each
// BTree has its own FreeList, but multiple BTrees can share the same
// FreeList.
// Two Btrees using the same freelist are safe for concurrent write access.
type FreeList struct {
	freelist []*node
}

func newNode() (n *node) {
	return &node{
		items:    make(items,    0, MAX_ITEMS),
		children: make(children, 0, MAX_ITEMS + 1),
	}
}

// New creates a new B-Tree with the given degree.
//
// New(2), for example, will create a 2-3-4 tree (each node contains 1-3 items
// and 2-4 children).
func New() *BTree {
	return &BTree{}
}

// items stores items in a node.
type items []*MutationItem

// insertAt inserts a value into the given index, pushing all subsequent values
// forward.
func (s *items) insertAt(index int, item *MutationItem) {
	*s = append(*s, nil)
	if index < len(*s) {
		copy((*s)[index+1:], (*s)[index:])
	}
	(*s)[index] = item
}

func (s *items) truncate() {
	index   := MAX_ITEMS / 2
	toClear := (*s)[index:]
	*s       = (*s)[:index]
	copy(toClear, nilItems)
	// for len(toClear) > 0 {
	// 	toClear = toClear[copy(toClear, nilItems):]
	// }
}

// find returns the index where the given item should be inserted into this
// list.  'found' is true if the item already exists in the list at the given
// index.
func (s items) find(item *MutationItem) (index int, found bool) {
	i := sort.Search(len(s), func(i int) bool {
		return item.Less(s[i])
	})
	if i > 0 && !s[i-1].Less(item) {
		return i - 1, true
	}
	return i, false
}

// children stores child nodes in a node.
type children []*node

// insertAt inserts a value into the given index, pushing all subsequent values
// forward.
func (s *children) insertAt(index int, n *node) {
	*s = append(*s, nil)
	if index < len(*s) {
		copy((*s)[index+1:], (*s)[index:])
	}
	(*s)[index] = n
}

func (s *children) truncate() {
	index   := MAX_ITEMS / 2 + 1
	toClear := (*s)[index:]
	*s       = (*s)[:index]
	copy(toClear, nilChildren)
	// var toClear children
	// *s, toClear = (*s)[:index], (*s)[index:]
	// for len(toClear) > 0 {
	// 	toClear = toClear[copy(toClear, nilChildren):]
	// }
}

// node is an internal node in a tree.
//
// It must at all times maintain the invariant that either
//   * len(children) == 0, len(items) unconstrained
//   * len(children) == len(items) + 1
type node struct {
	items    items
	children children
}

// split splits the given node at the given index.  The current node shrinks,
// and this function returns the item that existed at that index and a new node
// containing all items/children after it.
func (n *node) split() (*MutationItem, *node) {
	i    := MAX_ITEMS / 2
	item := n.items[i]
	next := newNode()
	next.items = append(next.items, n.items[i+1:]...)
	n.items.truncate()
	if len(n.children) > 0 {
		next.children = append(next.children, n.children[i+1:]...)
		n.children.truncate()
	}
	return item, next
}

// maybeSplitChild checks if a child should be split, and if so splits it.
// Returns whether or not a split occurred.
func (n *node) maybeSplitChild(i int) bool {
	if len(n.children[i].items) < MAX_ITEMS {
		return false
	}
	first := n.children[i]
	item, second := first.split()
	n.items.insertAt(i, item)
	n.children.insertAt(i+1, second)
	return true
}

// insert inserts an item into the subtree rooted at this node, making sure
// no nodes in the subtree exceed MAX_ITEMS items.  Should an equivalent item be
// be found/replaced by insert, it will be returned.
func (n *node) insert(item *MutationItem) *MutationItem {
	i, found := n.items.find(item)
	if found {
		out := n.items[i]
		n.items[i] = item
		return out
	}
	if len(n.children) == 0 {
		n.items.insertAt(i, item)
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

// get finds the given key in the subtree and returns it.
func (n *node) get(key *MutationItem) *MutationItem {
	i, found := n.items.find(key)
	if found {
		return n.items[i]
	} else if len(n.children) > 0 {
		return n.children[i].get(key)
	}
	return nil
}

// BTree is an implementation of a B-Tree.
//
// BTree stores *MutationItem instances in an ordered structure, allowing easy insertion,
// removal, and iteration.
//
// Write operations are not safe for concurrent mutation by multiple
// goroutines, but Read operations are.
type BTree struct {
	root *node
}

// ReplaceOrInsert adds the given item to the tree.  If an item in the tree
// already equals the given one, it is removed from the tree and returned.
// Otherwise, nil is returned.
//
// nil cannot be added to the tree (will panic).
func (t *BTree) ReplaceOrInsert(item *MutationItem) *MutationItem {
	if item == nil {
		panic("nil item being added to BTree")
	}
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

// Get looks for the key item in the tree, returning it.  It returns nil if
// unable to find that item.
func (t *BTree) Get(key *MutationItem) *MutationItem {
	if t.root == nil {
		return nil
	}
	return t.root.get(key)
}

// Has returns true if the given key is in the tree.
func (t *BTree) Has(key *MutationItem) bool {
	return t.Get(key) != nil
}

func (t *BTree) Clear() {
	t.root = nil
}
