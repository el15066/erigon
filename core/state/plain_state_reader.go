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
)

// var ENC = base64.StdEncoding.EncodeToString
var ENC = hex.EncodeToString

var tracefile *bufio.Writer
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
	if tracefile == nil && common.STORAGE_TRACING == true && notrace == false {
		_f, _err := os.OpenFile("logz/reads.txt", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
		if _err == nil {
			tracefile = bufio.NewWriterSize(_f, 128*1024)
		} else {
			notrace = true
			fmt.Println("\n\nWARNING: READS NOT RECORDED", _err)
		}
	}
	return &PlainStateReader{
		db: db,
		blockID: -1,
		txID:    -1,
	}
}

func FlushStateReaderTracefile() {
	if tracefile != nil {
		tracefile.Flush()
	}
}

func (r *PlainStateReader) SetBlockID(n int) { r.blockID = n }
func (r *PlainStateReader) SetTxID(   n int) { r.txID    = n }

func (r *PlainStateReader) ReadAccountData(address common.Address) (*accounts.Account, error) {

	enc, err := r.db.GetOne(kv.PlainState, address.Bytes())

	// HERE
	if tracefile != nil {
		tracefile.WriteString(fmt.Sprintf("A %8d %3d %s %s\n", r.blockID, r.txID, ENC(address.Bytes()), ENC(enc)))
	}

	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	var a accounts.Account
	if err = a.DecodeForStorage(enc); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PlainStateReader) ReadAccountStorage(address common.Address, incarnation uint64, key *common.Hash) ([]byte, error) {
	compositeKey := dbutils.PlainGenerateCompositeStorageKey(address.Bytes(), incarnation, key.Bytes())

	enc, err := r.db.GetOne(kv.PlainState, compositeKey)

	// HERE
	if tracefile != nil {
		tracefile.WriteString(fmt.Sprintf("S %8d %3d %s %s\n", r.blockID, r.txID, ENC(compositeKey), ENC(enc)))
	}

	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	return enc, nil
}

func (r *PlainStateReader) ReadAccountCode(address common.Address, incarnation uint64, codeHash common.Hash) ([]byte, error) {
	if bytes.Equal(codeHash.Bytes(), emptyCodeHash) {
		return nil, nil
	}

	// HERE
	if tracefile != nil {
		tracefile.WriteString(fmt.Sprintf("C %8d %3d %s\n", r.blockID, r.txID, ENC(codeHash.Bytes())))
	}

	code, err := r.db.GetOne(kv.Code, codeHash.Bytes())
	if len(code) == 0 {
		return nil, nil
	}
	return code, err
}

func (r *PlainStateReader) ReadAccountCodeSize(address common.Address, incarnation uint64, codeHash common.Hash) (int, error) {
	code, err := r.ReadAccountCode(address, incarnation, codeHash)
	return len(code), err
}

func (r *PlainStateReader) ReadAccountIncarnation(address common.Address) (uint64, error) {

	// HERE
	if tracefile != nil {
		tracefile.WriteString(fmt.Sprintf("I %8d %3d %s\n", r.blockID, r.txID, ENC(address.Bytes())))
	}

	b, err := r.db.GetOne(kv.IncarnationMap, address.Bytes())
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, nil
	}
	return binary.BigEndian.Uint64(b), nil
}
