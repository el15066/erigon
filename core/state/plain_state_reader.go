package state

import (
	"fmt"
	"os"
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	// "encoding/base64"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/dbutils"
	"github.com/ledgerwatch/erigon/core/types/accounts"

	"github.com/ledgerwatch/erigon/bench"
)

// var ENC = base64.StdEncoding.EncodeToString
var ENC = hex.EncodeToString

var tracefile *bufio.Writer
var traceMu   common.Spinlock
var notrace = false

var _ StateReader = (*PlainStateReader)(nil)

// PlainStateReader reads data from so called "plain state".
// Data in the plain state is stored using un-hashed account/storage items
// as opposed to the "normal" state that uses hashes of merkle paths to store items.
type PlainStateReader struct {
	db kv.Getter
	blockID int
	txID    int
}

func NewPlainStateReader(db kv.Getter) *PlainStateReader {
	if common.STORAGE_TRACING {
		traceMu.Lock()
		if tracefile == nil && !notrace {
			_f, _err := os.OpenFile("logz/reads.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
			if _err == nil {
				tracefile = bufio.NewWriterSize(_f, 2*1024*1024)
			} else {
				notrace = true
				fmt.Println("\n\nWARNING: READS NOT RECORDED", _err)
			}
		}
		traceMu.Unlock()
	}
	return &PlainStateReader{
		db: db,
		blockID: -1,
		txID:    -1,
	}
}

func FlushStateReaderTracefile() {
	if common.STORAGE_TRACING && tracefile != nil {
		traceMu.Lock()
		tracefile.Flush()
		traceMu.Unlock()
	}
}

func (r *PlainStateReader) SetBlockID(n int) { r.blockID = n }
func (r *PlainStateReader) SetTxID(   n int) { r.txID    = n }

func (r *PlainStateReader) ReadAccountData(address common.Address) (*accounts.Account, error) {
	bench.Tick(60)

	enc, err := r.db.GetOne(kv.PlainState, address.Bytes())

	// HERE
	if common.STORAGE_TRACING && tracefile != nil {
		traceMu.Lock()
		tracefile.WriteString(fmt.Sprintf("A %8d %3d %s %s\n", r.blockID, r.txID, ENC(address.Bytes()), ENC(enc)))
		traceMu.Unlock()
	}

	if err != nil {
		bench.TiCk(61)
		return nil, err
	}
	if len(enc) == 0 {
		bench.TiCk(61)
		return nil, nil
	}
	var a accounts.Account
	if err = a.DecodeForStorage(enc); err != nil {
		bench.TiCk(61)
		return nil, err
	}

	bench.TiCk(61)
	return &a, nil
}

func (r *PlainStateReader) ReadAccountStorage(address common.Address, incarnation uint64, key *common.Hash) ([]byte, error) {
	bench.Tick(62)

	compositeKey := dbutils.PlainGenerateCompositeStorageKey(address.Bytes(), incarnation, key.Bytes())

	enc, err := r.db.GetOne(kv.PlainState, compositeKey)

	// HERE
	if common.STORAGE_TRACING && tracefile != nil {
		traceMu.Lock()
		tracefile.WriteString(fmt.Sprintf("S %8d %3d %s %s\n", r.blockID, r.txID, ENC(compositeKey), ENC(enc)))
		traceMu.Unlock()
	}

	if err != nil {
		bench.TiCk(63)
		return nil, err
	}
	if len(enc) == 0 {
		bench.TiCk(63)
		return nil, nil
	}

	bench.TiCk(63)
	return enc, nil
}
func (r *PlainStateReader) WriteTraceAccountStorage(address common.Address, incarnation uint64, key *common.Hash, enc []byte) {

	compositeKey := dbutils.PlainGenerateCompositeStorageKey(address.Bytes(), incarnation, key.Bytes())

	// HERE
	if common.STORAGE_TRACING && tracefile != nil {
		traceMu.Lock()
		tracefile.WriteString(fmt.Sprintf("S %8d %3d %s %s\n", r.blockID, r.txID, ENC(compositeKey), ENC(enc)))
		traceMu.Unlock()
	}
}

func (r *PlainStateReader) ReadAccountCode(address common.Address, incarnation uint64, codeHash common.Hash) ([]byte, error) {
	bench.Tick(64)

	if bytes.Equal(codeHash.Bytes(), emptyCodeHash) {
		bench.TiCk(65)
		return nil, nil
	}

	// HERE
	if common.STORAGE_TRACING && tracefile != nil {
		traceMu.Lock()
		tracefile.WriteString(fmt.Sprintf("C %8d %3d %s\n", r.blockID, r.txID, ENC(codeHash.Bytes())))
		traceMu.Unlock()
	}

	code, err := r.db.GetOne(kv.Code, codeHash.Bytes())
	if len(code) == 0 {
		bench.TiCk(65)
		return nil, nil
	}

	bench.TiCk(65)
	return code, err
}

func (r *PlainStateReader) ReadAccountCodeSize(address common.Address, incarnation uint64, codeHash common.Hash) (int, error) {
	code, err := r.ReadAccountCode(address, incarnation, codeHash)
	return len(code), err
}

func (r *PlainStateReader) ReadAccountIncarnation(address common.Address) (uint64, error) {
	bench.Tick(66)

	// HERE
	if common.STORAGE_TRACING && tracefile != nil {
		traceMu.Lock()
		tracefile.WriteString(fmt.Sprintf("I %8d %3d %s\n", r.blockID, r.txID, ENC(address.Bytes())))
		traceMu.Unlock()
	}

	b, err := r.db.GetOne(kv.IncarnationMap, address.Bytes())
	if err != nil {
		bench.TiCk(67)
		return 0, err
	}
	if len(b) == 0 {
		bench.TiCk(67)
		return 0, nil
	}

	bench.TiCk(67)
	return binary.BigEndian.Uint64(b), nil
}
