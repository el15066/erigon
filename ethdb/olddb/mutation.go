package olddb

import (
	"context"
	"encoding/binary"
	// "fmt"
	// "strings"
	// "sync"
	// "time"
	"unsafe"

	// "github.com/google/btree"
	mtree "github.com/ledgerwatch/erigon/ethdb/olddb/mutation_tree"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/ethdb"
	// "github.com/ledgerwatch/log/v3"

	"github.com/ledgerwatch/erigon/bench"
)

var BucketsMap  = map[string]int{}
var BucketNames = []string{} // should be the same as kv.ChaindataTables

func tableNameToID(name string) int {
	i, ok := BucketsMap[name]
	if !ok {
		panic("Bucket name not in BucketsMap")
	}
	return i
}
func tableIDToName(i int) string {
	return BucketNames[i]
}

type Mutation struct {
	trees      []*mtree.BTree
	db         kv.RwTx
	quit       <-chan struct{}
	clean      func()
	// mu         sync.RWMutex
	mu         common.RWSpinlock
	size       int
}

// NewBatch - starts in-mem batch
//
// Common pattern:
//
// batch := db.NewBatch()
// defer batch.Rollback()
// ... some calculations on `batch`
// batch.Commit()
func NewBatch(tx kv.RwTx, quit <-chan struct{}) *Mutation {
	if len(BucketsMap) == 0 {
		for i, v := range kv.ChaindataTables {
			BucketsMap[v] = i
			BucketNames   = append(BucketNames, v)
		}
	}
	clean := func() {}
	if quit == nil {
		ch := make(chan struct{})
		clean = func() { close(ch) }
		quit = ch
	}
	m := &Mutation{
		db:    tx,
		quit:  quit,
		clean: clean,
	}
	for range kv.ChaindataTables {
		m.trees = append(m.trees, mtree.New())
	}
	return m
}

func (m *Mutation) RwKV() kv.RwDB {
	if casted, ok := m.db.(ethdb.HasRwKV); ok {
		return casted.RwKV()
	}
	return nil
}

func (m *Mutation) getMem(table string, key []byte) ([]byte, bool) {
	// bench.Tick(500) // expensive !
	m.mu.RLock()
	// bench.TiCk(501)
	t := mtree.MutationItem{ key, nil }
	i := m.trees[tableNameToID(table)].Get(t)
	m.mu.RUnlock()
	return i.Value, i.Key != nil
}

func (m *Mutation) IncrementSequence(bucket string, amount uint64) (res uint64, err error) {
	v, ok := m.getMem(kv.Sequence, []byte(bucket))
	if !ok && m.db != nil {
		v, err = m.db.GetOne(kv.Sequence, []byte(bucket))
		if err != nil {
			return 0, err
		}
	}

	var currentV uint64 = 0
	if len(v) > 0 {
		currentV = binary.BigEndian.Uint64(v)
	}

	newVBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(newVBytes, currentV+amount)
	if err = m.Put(kv.Sequence, []byte(bucket), newVBytes); err != nil {
		return 0, err
	}

	return currentV, nil
}
func (m *Mutation) ReadSequence(bucket string) (res uint64, err error) {
	v, ok := m.getMem(kv.Sequence, []byte(bucket))
	if !ok && m.db != nil {
		v, err = m.db.GetOne(kv.Sequence, []byte(bucket))
		if err != nil {
			return 0, err
		}
	}
	var currentV uint64 = 0
	if len(v) > 0 {
		currentV = binary.BigEndian.Uint64(v)
	}

	return currentV, nil
}

// Can only be called from the worker thread
func (m *Mutation) GetOne(table string, key []byte) ([]byte, error) {
	if value, ok := m.getMem(table, key); ok {
		if value == nil {
			return nil, nil
		}
		return value, nil
	}
	if m.db != nil {
		// TODO: simplify when tx can no longer be parent of Mutation
		bench.Tick(70)
		value, err := m.db.GetOne(table, key)
		if err != nil {
			bench.TiCk(71)
			return nil, err
		}

		bench.TiCk(71)
		return value, nil
	}
	return nil, nil
}

// Can only be called from the worker thread
func (m *Mutation) Get(table string, key []byte) ([]byte, error) {
	value, err := m.GetOne(table, key)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, ethdb.ErrKeyNotFound
	}

	return value, nil
}

func (m *Mutation) Last(table string) ([]byte, []byte, error) {
	c, err := m.db.Cursor(table)
	if err != nil {
		return nil, nil, err
	}
	defer c.Close()
	return c.Last()
}

func (m *Mutation) hasMem(table string, key []byte) bool {
	// bench.Tick(502)
	m.mu.RLock()
	// bench.TiCk(503)
	defer m.mu.RUnlock()
	t := mtree.MutationItem{ key, nil }
	return m.trees[tableNameToID(table)].Has(t)
}

func (m *Mutation) Has(table string, key []byte) (bool, error) {
	if m.hasMem(table, key) {
		return true, nil
	}
	if m.db != nil {
		// bench.Tick(72)
		res, err := m.db.Has(table, key)
		// bench.TiCk(73)
		return res, err
	}
	return false, nil
}

func (m *Mutation) Put(table string, key []byte, value []byte) error {
	// bench.Tick(220) // expensive !
	m.mu.Lock()
	// bench.TiCk(221)

	newMi := mtree.MutationItem{ key, value }
	i := m.trees[tableNameToID(table)].ReplaceOrInsert(newMi)
	m.size += int(unsafe.Sizeof(newMi)) + len(key) + len(value)
	if i.Key != nil {
		m.size -= (int(unsafe.Sizeof(i)) + len(i.Key) + len(i.Value))
	}

	m.mu.Unlock()
	return nil
}

func (m *Mutation) Append(table string, key []byte, value []byte) error {
	return m.Put(table, key, value)
}

func (m *Mutation) AppendDup(table string, key []byte, value []byte) error {
	return m.Put(table, key, value)
}

func (m *Mutation) BatchSize() int {
	// no need to lock, if we reeeealy need precision, use atomic load
	// m.mu.RLock()
	// defer m.mu.RUnlock()
	return m.size
}

func (m *Mutation) ForEach(bucket string, fromPrefix []byte, walker func(k, v []byte) error) error {
	m.panicOnEmptyDB()
	return m.db.ForEach(bucket, fromPrefix, walker)
}

func (m *Mutation) ForPrefix(bucket string, prefix []byte, walker func(k, v []byte) error) error {
	m.panicOnEmptyDB()
	return m.db.ForPrefix(bucket, prefix, walker)
}

func (m *Mutation) ForAmount(bucket string, prefix []byte, amount uint32, walker func(k, v []byte) error) error {
	m.panicOnEmptyDB()
	return m.db.ForAmount(bucket, prefix, amount, walker)
}

func (m *Mutation) Delete(table string, k, v []byte) error {
	if v != nil {
		return m.db.Delete(table, k, v) // TODO: mutation to support DupSort deletes
	}
	//m.mu.Lock()
	//defer m.mu.Unlock()
	//m.puts.Delete(table, k)
	return m.Put(table, k, nil)
}

// func (m *Mutation) doCommit(tx kv.RwTx) error {
// 	var prevTable int
// 	var c kv.RwCursor
// 	var innerErr error
// 	var isEndOfBucket bool
// 	logEvery := time.NewTicker(30 * time.Second)
// 	defer logEvery.Stop()
// 	count := 0
// 	total := float64(m.puts.Len())

// 	m.puts.Ascend(func(i btree.Item) bool {
// 		mi := i.(*MutationItem)
// 		if mi.table != prevTable {
// 			if c != nil {
// 				c.Close()
// 			}
// 			var err error
// 			c, err = tx.RwCursor(tableIDToName(mi.table))
// 			if err != nil {
// 				innerErr = err
// 				return false
// 			}
// 			prevTable = mi.table
// 			firstKey, _, err := c.Seek(mi.key)
// 			if err != nil {
// 				innerErr = err
// 				return false
// 			}
// 			isEndOfBucket = firstKey == nil
// 		}
// 		if isEndOfBucket {
// 			if len(mi.value) > 0 {
// 				if err := c.Append(mi.key, mi.value); err != nil {
// 					innerErr = err
// 					return false
// 				}
// 			}
// 		} else if len(mi.value) == 0 {
// 			if err := c.Delete(mi.key, nil); err != nil {
// 				innerErr = err
// 				return false
// 			}
// 		} else {
// 			if err := c.Put(mi.key, mi.value); err != nil {
// 				innerErr = err
// 				return false
// 			}
// 		}

// 		count++

// 		select {
// 		default:
// 		case <-logEvery.C:
// 			progress := fmt.Sprintf("%.1fM/%.1fM", float64(count)/1_000_000, total/1_000_000)
// 			log.Info("Write to db", "progress", progress, "current table", tableIDToName(mi.table))
// 			tx.CollectMetrics()
// 		case <-m.quit:
// 			innerErr = common.ErrStopped
// 			return false
// 		}
// 		return true
// 	})
// 	tx.CollectMetrics()
// 	return innerErr
// }

func (m *Mutation) Commit() error {

	return nil // -readonly

	// if m.db == nil {
	// 	return nil
	// }
	// m.mu.Lock()
	// defer m.mu.Unlock()
	// if err := m.doCommit(m.db); err != nil {
	// 	return err
	// }

	// m.puts.Clear(false /* addNodesToFreelist */)
	// m.size = 0
	// m.clean()
	// return nil
}

func (m *Mutation) Rollback() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.trees {
		t.Clear()
	}
	m.size = 0
	m.clean()
}

func (m *Mutation) Close() {
	m.Rollback()
}

func (m *Mutation) Begin(ctx context.Context, flags ethdb.TxFlags) (ethdb.DbWithPendingMutations, error) {
	panic("mutation can't start transaction, because doesn't own it")
}

func (m *Mutation) panicOnEmptyDB() {
	if m.db == nil {
		panic("Not implemented")
	}
}

func (m *Mutation) SetRwKV(kv kv.RwDB) {
	m.db.(ethdb.HasRwKV).SetRwKV(kv)
}

// Read-only sub-interfaces

type RoMutation struct {
	m    *Mutation
	rodb kv.Tx
}

// func (m *Mutation) addRoDB(rodb *kv.Tx) int {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	//
// 	i := len(m.rodbs)
// 	m.rodbs = append(m.rodbs, rodb)
// 	return i
// }
func (m *Mutation) UsingRoDB(rodb kv.Tx) *RoMutation {
	return &RoMutation{ m, rodb }
}

func (rom *RoMutation) Close() {
	rom.rodb.Rollback()
}
func (rom *RoMutation) GetOne(table string, key []byte) ([]byte, error) {
	if value, ok := rom.m.getMem(table, key); ok {
		return value, nil
	}
	bench.Tick(75)
	res, err := rom.rodb.GetOne(table, key)
	bench.TiCk(76)
	return res, err
}
func (rom *RoMutation) Has(table string, key []byte) (bool, error) {
	if rom.m.hasMem(table, key) {
		return true, nil
	}
	return rom.rodb.Has(table, key)
}
func (rom *RoMutation) ForEach(bucket string, fromPrefix []byte, walker func(k, v []byte) error) error {
	return rom.rodb.ForEach(bucket, fromPrefix, walker)
}
func (rom *RoMutation) ForPrefix(bucket string, prefix []byte, walker func(k, v []byte) error) error {
	return rom.rodb.ForPrefix(bucket, prefix, walker)
}
func (rom *RoMutation) ForAmount(bucket string, prefix []byte, amount uint32, walker func(k, v []byte) error) error {
	return rom.rodb.ForAmount(bucket, prefix, amount, walker)
}
