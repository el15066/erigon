package stagedsync

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"bufio"
	"io"
	"fmt"
	"runtime"
	"sort"
	"time"

	"github.com/holiman/uint256"
	"github.com/c2h5oh/datasize"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/changeset"
	"github.com/ledgerwatch/erigon/common/dbutils"
	"github.com/ledgerwatch/erigon/common/etl"
	"github.com/ledgerwatch/erigon/consensus"
	"github.com/ledgerwatch/erigon/core"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/core/types/accounts"
	"github.com/ledgerwatch/erigon/core/vm"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/ethdb/olddb"
	"github.com/ledgerwatch/erigon/ethdb/prune"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/turbo/shards"
	"github.com/ledgerwatch/erigon/cmd/utils"
	"github.com/ledgerwatch/log/v3"

	"github.com/ledgerwatch/erigon/bench"

	"github.com/ledgerwatch/erigon/prediction"
)

const (
	logInterval = 30 * time.Second
)

type HasChangeSetWriter interface {
	ChangeSetWriter() *state.ChangeSetWriter
}

type ChangeSetHook func(blockNum uint64, wr *state.ChangeSetWriter)

type ExecuteBlockCfg struct {
	db            kv.RwDB
	batchSize     datasize.ByteSize
	prune         prune.Mode
	changeSetHook ChangeSetHook
	chainConfig   *params.ChainConfig
	engine        consensus.Engine
	vmConfig      *vm.Config
	tmpdir        string
	stateStream   bool
	accumulator   *shards.Accumulator
}

func StageExecuteBlocksCfg(
	kv kv.RwDB,
	prune prune.Mode,
	batchSize datasize.ByteSize,
	changeSetHook ChangeSetHook,
	chainConfig *params.ChainConfig,
	engine consensus.Engine,
	vmConfig *vm.Config,
	accumulator *shards.Accumulator,
	stateStream bool,
	tmpdir string,
) ExecuteBlockCfg {
	return ExecuteBlockCfg{
		db:            kv,
		prune:         prune,
		batchSize:     batchSize,
		changeSetHook: changeSetHook,
		chainConfig:   chainConfig,
		engine:        engine,
		vmConfig:      vmConfig,
		tmpdir:        tmpdir,
		accumulator:   accumulator,
		stateStream:   stateStream,
	}
}

func readBlock(blockNum uint64, tx kv.Tx) (*types.Block, error) {
	blockHash, err := rawdb.ReadCanonicalHash(tx, blockNum)
	if err != nil {
		return nil, err
	}
	b, _, err := rawdb.ReadBlockWithSenders(tx, blockHash, blockNum)
	return b, err
}

func executeBlock(
	block *types.Block,
	tx kv.RwTx,
	batch ethdb.Database,
	cfg ExecuteBlockCfg,
	vmConfig vm.Config, // emit copy, because will modify it
	writeChangesets bool,
	writeReceipts bool,
	writeCallTraces bool,
	contractHasTEVM func(contractHash common.Hash) (bool, error),
	initialCycle bool,
) error {
	blockNum := block.NumberU64()
	stateReader, stateWriter, err := newStateReaderWriter(batch, tx, block, writeChangesets, cfg.accumulator, initialCycle, cfg.stateStream)
	if err != nil {
		return err
	}

	// where the magic happens
	getHeader := func(hash common.Hash, number uint64) *types.Header { return rawdb.ReadHeader(tx, hash, number) }

	callTracer := NewCallTracer(contractHasTEVM)
	vmConfig.Debug = true
	vmConfig.Tracer = callTracer
	receipts, err := core.ExecuteBlockEphemerally(cfg.chainConfig, &vmConfig, getHeader, cfg.engine, block, stateReader, stateWriter, epochReader{tx: tx}, chainReader{config: cfg.chainConfig, tx: tx}, contractHasTEVM)
	if err != nil {
		return err
	}

	if writeReceipts {
		if err = rawdb.AppendReceipts(tx, blockNum, receipts); err != nil {
			return err
		}
	}

	if cfg.changeSetHook != nil {
		if hasChangeSet, ok := stateWriter.(HasChangeSetWriter); ok {
			cfg.changeSetHook(blockNum, hasChangeSet.ChangeSetWriter())
		}
	}

	if writeCallTraces {
		callTracer.tos[block.Coinbase()] = false
		for _, uncle := range block.Uncles() {
			callTracer.tos[uncle.Coinbase] = false
		}
		list := make(common.Addresses, len(callTracer.froms)+len(callTracer.tos))
		i := 0
		for addr := range callTracer.froms {
			copy(list[i][:], addr[:])
			i++
		}
		for addr := range callTracer.tos {
			copy(list[i][:], addr[:])
			i++
		}
		sort.Sort(list)
		// List may contain duplicates
		var blockNumEnc [8]byte
		binary.BigEndian.PutUint64(blockNumEnc[:], blockNum)
		var prev common.Address
		var created bool
		for j, addr := range list {
			if j > 0 && prev == addr {
				continue
			}
			var v [common.AddressLength + 1]byte
			copy(v[:], addr[:])
			if _, ok := callTracer.froms[addr]; ok {
				v[common.AddressLength] |= 1
			}
			if _, ok := callTracer.tos[addr]; ok {
				v[common.AddressLength] |= 2
			}
			// TEVM marking still untranslated contracts
			if vmConfig.EnableTEMV {
				if created = callTracer.tos[addr]; created {
					v[common.AddressLength] |= 4
				}
			}
			if j == 0 {
				if err = tx.Append(kv.CallTraceSet, blockNumEnc[:], v[:]); err != nil {
					return err
				}
			} else {
				if err = tx.AppendDup(kv.CallTraceSet, blockNumEnc[:], v[:]); err != nil {
					return err
				}
			}
			copy(prev[:], addr[:])
		}
	}

	return nil
}

func newStateReaderWriter(
	batch ethdb.Database,
	tx kv.RwTx,
	block *types.Block,
	writeChangesets bool,
	accumulator *shards.Accumulator,
	initialCycle bool,
	stateStream bool,
) (state.StateReader, state.WriterWithChangeSets, error) {

	var stateReader state.StateReader
	var stateWriter state.WriterWithChangeSets

	stateReader = state.NewPlainStateReader(batch)

	if !initialCycle && stateStream {
		txs, err := rawdb.RawTransactionsRange(tx, block.NumberU64(), block.NumberU64())
		if err != nil {
			return nil, nil, err
		}
		var blockBaseFee uint64
		if block.BaseFee() != nil {
			blockBaseFee = block.BaseFee().Uint64()
		}
		accumulator.StartChange(block.NumberU64(), block.Hash(), txs, blockBaseFee, false)
	} else {
		accumulator = nil
	}
	if writeChangesets {
		stateWriter = state.NewPlainStateWriter(batch, tx, block.NumberU64()).SetAccumulator(accumulator)
	} else {
		stateWriter = state.NewPlainStateWriterNoHistory(batch).SetAccumulator(accumulator)
	}

	return stateReader, stateWriter, nil
}

type tx_dump struct {
	Block      uint64
	Index      int
	Coinbase   string
	Timestamp  uint64
	Difficulty uint64
	Gaslimit   uint64
	// Chainid    uint64
	Address    string
	Origin     string
	Callvalue  *uint256.Int
	Calldata   string
	// GasPrice   string
}

var _buf = [64]byte{}

func fetchBlocks(cfg ExecuteBlockCfg, batch *olddb.Mutation, blockChan chan *types.Block, errChan chan error, quitChan chan int, from uint64, to uint64) {
	var ENC = hex.EncodeToString
	var tracefile *bufio.Writer
	if common.PREFETCH_TRACING {
		_f, _err := os.OpenFile("logz/prefetches.txt", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
		if _err == nil {
			defer _f.Close()
			tracefile = bufio.NewWriterSize(_f, 128*1024)
			defer tracefile.Flush()
		} else {
			log.Warn("Prefetches not recorded", "error", _err)
		}
	}
	var dumpfile *bufio.Writer
	if common.TX_DUMPING {
		_f, _err := os.OpenFile("logz/tx_dump.txt", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
		if _err == nil {
			defer _f.Close()
			dumpfile = bufio.NewWriterSize(_f, 128*1024)
			defer dumpfile.Flush()
		} else {
			log.Warn("Transactions not dumped", "error", _err)
		}
	}
	var storage_prefetch_b = uint64(7500000) // where the file starts
	var storage_prefetch_i = -1
	var storage_prefetch_file *bufio.Reader
	if common.USE_STORAGE_PREFETCH_FILE {
		_f, _err := os.OpenFile("logz/reads_s.bin", os.O_RDONLY, 0664)
		if _err == nil {
			defer _f.Close()
			storage_prefetch_file = bufio.NewReaderSize(_f, 128*1024)
			storage_prefetch_i    = 0
		} else {
			log.Warn("Storage prefetch file", "error", _err)
		}
	}
	// var err   error
	// var block *types.Block
	// var db    kv.Tx
	rodb, err := cfg.db.BeginRo(context.Background())
	if err == nil {
		defer rodb.Rollback()
		db := batch.UsingRoDB(rodb)
		//
		if common.USE_PREDICTORS {
			prediction.InitCtx(db)
		}
		Loop: for blockNum := from; blockNum <= to; blockNum++ {
			block, err := readBlock(blockNum, rodb)
			if err != nil {
				log.Error("Bad block", "(block==nil)", block == nil, "error", err)
				break Loop
			}
			select {
				case blockChan <- block:
				case <-quitChan:
					break Loop
			}
			if common.USE_PREDICTORS {
				prediction.SetBlockVars(
					block.Coinbase(),
					block.Difficulty(),
					blockNum,
					block.Time(),
					block.GasLimit(),
				)
			}
			if common.PREFETCH_ACCOUNTS {
				for i, tx := range block.Transactions() {
					bench.Tick(100)
					// prefetch 'from' account
					//
					// readBlock() above calls ReadBlockWithSenders() which saves the senders to the transactions,
					// meaning we can use .GetSender() directly and we don't need to derive it by .Sender(Signer)
					from_addr, ok := tx.GetSender()
					if !ok {
						log.Error("Sender not in tx", "block_number", blockNum, "tx_index", i)
						break Loop
					}
					if common.PREFETCH_TRACING && tracefile != nil {
						tracefile.WriteString(fmt.Sprintf("A %8d %3d %s\n", blockNum, i, ENC(from_addr.Bytes())))
					}
					from_data, _ := db.GetOne(kv.PlainState, from_addr.Bytes())
					//
					bench.Tick(101)
					// prefetch 'to' account (nil if contract creation)
					to_addr := tx.GetTo()
					if to_addr != nil && *to_addr != from_addr {
						bench.Tick(105)
						if common.PREFETCH_TRACING && tracefile != nil {
							tracefile.WriteString(fmt.Sprintf("A %8d %3d %s\n", blockNum, i, ENC(to_addr.Bytes())))
						}
						to_data, _ := db.GetOne(kv.PlainState, to_addr.Bytes())
						//
						bench.Tick(106)
						// prefetch its code if it's a contract
						if common.PREFETCH_CODE {
							var to_acc accounts.Account
							to_acc.DecodeForStorage(to_data)
							// 
							if !to_acc.IsEmptyCodeHash() {
								//
								if common.PREFETCH_TRACING && tracefile != nil {
									tracefile.WriteString(fmt.Sprintf("C %8d %3d %s\n", blockNum, i, ENC(to_acc.CodeHash.Bytes())))
								}
								bench.Tick(110)
								rodb.GetOne(kv.Code, to_acc.CodeHash.Bytes()) // if we use its value, remember to change rodb to db
								bench.Tick(111)
								//
								if common.TX_DUMPING && dumpfile != nil {
									t, _ := json.Marshal(&tx_dump{
										Block:      blockNum,
										Index:      i,
										// Blockhash: not here (array of 256 most recent)
										Coinbase:   ENC(block.Coinbase().Bytes()),
										Timestamp:  block.Time(),
										Difficulty: block.Difficulty().Uint64(),
										Gaslimit:   block.GasLimit(),
										// Chainid:    cfg.chainConfig.ChainID.Uint64(),
										// Selfbalance: is dynamic
										Address:    ENC(to_addr.Bytes()),
										// Balance: is dynamic
										Origin:     ENC(from_addr.Bytes()),
										// Caller: same as origin for now
										Callvalue:  tx.GetValue(),
										Calldata:   ENC(tx.GetData()),
										// Gasprice: n/a after LONDON // TODO: verify below is correct
										// Gasprice: ENC(tx.GetPrice().Bytes32()),
										// Extcode: not here
										// Returndata: n/a
									})
									dumpfile.Write(append(t, byte('\n')))
								}
								//
								if common.USE_PREDICTORS {
									var from_acc accounts.Account
									from_acc.DecodeForStorage(from_data)
									//
									prediction.PredictTX(
										i,
										*to_addr,
										//
										from_addr,
										tx.GetPrice(),
										//
										tx.GetValue(),
										tx.GetData(),
									)
									bench.Tick(112)
								}
							}
							bench.Tick(107)
						}
					}
					bench.Tick(102)
					//
					if common.USE_STORAGE_PREFETCH_FILE {
						// GET storage prefetch locations
						// read 2 bytes
						// if 0 -> update tx index
						// else that is # of addrs, read 20B contract address + 8B incarnation + #*32B storage addresses
						// fetch all those addresses
						// loop
						// why loop? -> 1 tx can call many contracts
						for i == storage_prefetch_i {
							_, _err := io.ReadFull(storage_prefetch_file, _buf[0:4])
							if _err != nil {
								log.Warn("Read from storage prefetch file", "error", _err)
								storage_prefetch_i    = -1
								storage_prefetch_file = nil
								break
							}
							count := int(binary.BigEndian.Uint16(_buf[0:2]))
							//
							// fmt.Println(i, "count", count)
							if count != 0 {
								io.ReadFull(storage_prefetch_file, _buf[4:30])
								for j := 0; j < count; j++ {
									io.ReadFull(storage_prefetch_file, _buf[30:62])
									db.GetOne(kv.PlainState, _buf[2:62])
									// fmt.Println(i, ENC(_buf[2:62]))
								}
							} else {
								diff := int(binary.BigEndian.Uint16(_buf[2:4]))
								// fmt.Println(i, "diff", diff)
								if diff != 0 { storage_prefetch_i += diff
								} else       { storage_prefetch_i  = -1   }
								break
							}
						}
						bench.Tick(103)
					}
				}
			}
			if common.USE_PREDICTORS {
				prediction.BlockEnded()
			}
			if common.USE_STORAGE_PREFETCH_FILE && storage_prefetch_file != nil {
				if storage_prefetch_i == -1 {
					if blockNum == storage_prefetch_b {
						io.ReadFull(storage_prefetch_file,      _buf[0:2])
						bdiff := uint64(binary.BigEndian.Uint16(_buf[0:2]))
						// fmt.Println("bdiff", storage_prefetch_b, "+", bdiff)
						storage_prefetch_b += bdiff
					}
					if blockNum + 1 == storage_prefetch_b {
						storage_prefetch_i = 0
						// fmt.Println("~~~~~ SWITCH ~~~~~", storage_prefetch_b)
					}
				} else {
					log.Warn("Malformed storage prefetch file", "storage_prefetch_b", storage_prefetch_b, "storage_prefetch_i", storage_prefetch_i)
					storage_prefetch_i    = -1
					storage_prefetch_file = nil
				}
			}
		}
	}
	log.Info("Prefetch thread exiting", "error", err)
	close(blockChan)
	<-quitChan
	errChan <- err
	log.Info("Prefetch thread exited")
}

func read_or_fetch_block(blockNum uint64, tx kv.Tx, blockChan chan *types.Block) (*types.Block) {
	if common.PREFETCH_BLOCKS {
		block := <-blockChan
		return block
	} else {
		block, err := readBlock(blockNum, tx)
		if err != nil {
			return nil
		}
		return block
	}
}

func SpawnExecuteBlocksStage(s *StageState, u Unwinder, tx kv.RwTx, toBlock uint64, ctx context.Context, cfg ExecuteBlockCfg, initialCycle bool) (err error) {
	bench.Reset()
	quit := ctx.Done()
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	prevStageProgress, errStart := stages.GetStageProgress(tx, stages.Senders)
	if errStart != nil {
		return errStart
	}
	nextStageProgress, err := stages.GetStageProgress(tx, stages.HashState)
	if err != nil {
		return err
	}
	nextStagesExpectData := nextStageProgress > 0 // Incremental move of next stages depend on fully written ChangeSets, Receipts, CallTraceSet

	logPrefix := s.LogPrefix()
	var to = prevStageProgress
	if toBlock > 0 {
		to = min(prevStageProgress, toBlock)
	}
	to = min(to, common.MAX_BLOCK)
	if to <= s.BlockNumber {
		return nil
	}
	if to > s.BlockNumber+16 {
		log.Info(fmt.Sprintf("[%s] Blocks execution", logPrefix), "from", s.BlockNumber, "to", to)
	}

	// <-quit

	log.Info("Globals",
		//
		"STORAGE_TRACING",           common.STORAGE_TRACING,
		"PREFETCH_TRACING",          common.PREFETCH_TRACING,
		"TX_DUMPING",                common.TX_DUMPING,
		"CODE_DUMPING",              common.CODE_DUMPING,
		"JUMP_TRACING",              common.JUMP_TRACING,
		//
		"PREFETCH_BLOCKS",           common.PREFETCH_BLOCKS,
		"BLOCK_READAHEAD",           common.BLOCK_READAHEAD,
		"PREFETCH_ACCOUNTS",         common.PREFETCH_ACCOUNTS,
		"PREFETCH_CODE",             common.PREFETCH_CODE,
		"USE_PREDICTORS",            common.USE_PREDICTORS,
		"USE_STORAGE_PREFETCH_FILE", common.USE_STORAGE_PREFETCH_FILE,
		//
		"TRACE_PREDICTED",           common.TRACE_PREDICTED,
		//
		"PREDICTOR_CACHE_SIZE",      common.PREDICTOR_CACHE_SIZE,
		"PREDICTOR_INITIAL_GAZ",     common.PREDICTOR_INITIAL_GAZ,
		"PREDICTOR_RESERVE_GAZ_DIV", common.PREDICTOR_RESERVE_GAZ_DIV,
		"PREDICTOR_CALL_GAZ_BONUS",  common.PREDICTOR_CALL_GAZ_BONUS,
		//
		"DEBUG_TX",                  common.DEBUG_TX,
		"DEBUG_TX_BLOCK",            common.DEBUG_TX_BLOCK,
		"DEBUG_TX_INDEX",            common.DEBUG_TX_INDEX,
		//
		"PREDICTOR_DB_PATH",         common.PREDICTOR_DB_PATH,
	)

	batch := olddb.NewBatch(tx, quit)
	defer batch.Rollback()

	logEvery := time.NewTicker(logInterval)
	defer logEvery.Stop()
	stageProgress := s.BlockNumber
	logBlock := stageProgress
	logTx, lastLogTx := uint64(0), uint64(0)
	logTime := time.Now()
	var gas uint64

	if common.USE_PREDICTORS {
		prediction.Init()
		defer prediction.Close()
	}

	blockChan := make(chan *types.Block, common.BLOCK_READAHEAD - 1)
	errChan   := make(chan error)
	quitChan  := make(chan int)

	if common.PREFETCH_BLOCKS {
		go fetchBlocks(cfg, batch, blockChan, errChan, quitChan, stageProgress + 1, to)
	}

	if common.STORAGE_TRACING {
		defer state.FlushStateReaderTracefile()
	}

	var stoppedErr error

	for blockNum := stageProgress + 1; blockNum <= to; blockNum++ {

		bench.Tick(0)

		block := read_or_fetch_block(blockNum, tx, blockChan)

		bench.Tick(1)

		if stoppedErr = common.Stopped(quit); stoppedErr != nil {
			break
		}
		// bench.Tick(2)

		var err error

		lastLogTx += uint64(block.Transactions().Len())

		var contractHasTEVM func(contractHash common.Hash) (bool, error)

		if cfg.vmConfig.EnableTEMV {
			contractHasTEVM = ethdb.GetHasTEVM(tx)
		}

		bench.Tick(3)
		// Incremental move of next stages depend on fully written ChangeSets, Receipts, CallTraceSet
		writeChangeSets := nextStagesExpectData || blockNum > cfg.prune.History.PruneTo(to)
		writeReceipts := nextStagesExpectData || blockNum > cfg.prune.Receipts.PruneTo(to)
		writeCallTraces := nextStagesExpectData || blockNum > cfg.prune.CallTraces.PruneTo(to)
		if err = executeBlock(block, tx, batch, cfg, *cfg.vmConfig, writeChangeSets, writeReceipts, writeCallTraces, contractHasTEVM, initialCycle); err != nil {
			log.Error(fmt.Sprintf("[%s] Execution failed", logPrefix), "block", blockNum, "hash", block.Hash().String(), "error", err)
			u.UnwindTo(blockNum-1, block.Hash())
			break
		}
		bench.Tick(4)
		stageProgress = blockNum

		// updateProgress := batch.BatchSize() >= int(cfg.batchSize)
		updateProgress := false // -readonly

		if updateProgress {
			if err = batch.Commit(); err != nil {
				return err
			}
			if !useExternalTx {
				if err = s.Update(tx, stageProgress); err != nil {
					return err
				}
				if err = tx.Commit(); err != nil {
					return err
				}
				tx, err = cfg.db.BeginRw(context.Background())
				if err != nil {
					return err
				}
				// TODO: This creates stacked up deferrals
				defer tx.Rollback()
			}
			batch = olddb.NewBatch(tx, quit)
			// TODO: This creates stacked up deferrals
			defer batch.Rollback()
		}

		gas = gas + block.GasUsed()

		// bench.Tick(5)

		select {
		default:
		case <-logEvery.C:
			logBlock, logTx, logTime = logProgress(logPrefix, logBlock, logTime, blockNum, logTx, lastLogTx, gas, batch)
			gas = 0
			tx.CollectMetrics()
			syncMetrics[stages.Execution].Set(blockNum)
			bench.PrintAll()
		}

		// bench.Tick(6)

		bench.Tick(9)
		bench.Tick(10)
		bench.Tick(12)
		bench.TiCk(11)
	}

	if common.PREFETCH_BLOCKS {
		close(quitChan)
		err2 := <-errChan
		if err == nil { err = err2 }
	}

	// HERE
	bench.PrintAll()

	if common.CODE_DUMPING {
		var ENC = hex.EncodeToString
		{
			_f, _err := os.OpenFile("logz/code_counts.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
			if _err == nil {
				f := bufio.NewWriterSize(_f, 128*1024)
				for k, v := range common.CONTRACT_CODE_COUNT {
					if v > 100 {
						f.WriteString(fmt.Sprintf("h_%s %5d\n", ENC(k.Bytes()), v))
					}
				}
				f.Flush()
				_f.Close()
			} else {
				log.Error("Code counts not saved", "error", _err)
			}
		}
		{
			_f, _err := os.OpenFile("logz/code_alias.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
			if _err == nil {
				f := bufio.NewWriterSize(_f, 128*1024)
				for k, v := range common.CONTRACT_CODE_ALIAS {
					// if common.CONTRACT_CODE_COUNT[v] > 100 {
					f.WriteString(fmt.Sprintf("%s h_%s\n", ENC(k.Bytes()), ENC(v.Bytes())))
					// }
				}
				f.Flush()
				_f.Close()
			} else {
				log.Error("Code alias not saved", "error", _err)
			}
		}
		{
			for k, v := range common.CONTRACT_CODE {
				// c, ok := common.CONTRACT_CODE_COUNT[k]
				// fmt.Println(k, c, ok, )
				if common.CONTRACT_CODE_COUNT[k] > 100 {
					_f, _err := os.OpenFile(fmt.Sprintf("logz/code/h_%s.runbin.hex", ENC(k.Bytes())), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
					if _err == nil {
						f := bufio.NewWriterSize(_f, 128*1024)
						f.WriteString(ENC(v))
						f.Flush()
						_f.Close()
					} else {
						log.Error("Code not saved", "error", _err)
						break
					}
				}
			}
		}
	}

	if common.JUMP_TRACING {
		var ENC = hex.EncodeToString
		{
			_f, _err := os.OpenFile("logz/jump_counts.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
			if _err == nil {
				f := bufio.NewWriterSize(_f, 128*1024)
				// for h, m := range common.JUMP_EDGE_COUNT {
				// 	total := common.JUMP_COUNT[h]
				// 	if total > 1000 {
				// 		f.WriteString(fmt.Sprintf("h_%s %9d\n", ENC(h.Bytes()), total))
				// 		for e, c := range m {
				// 			src := e &  0xFFFFFFFF
				// 			dst := e >> 32
				// 			f.WriteString(fmt.Sprintf("%08x %08x %8d\n", src, dst, c))
				// 		}
				// 	}
				// }
				for h, m1 := range common.JUMP_DST_CALLCOUNT {
					total :=     common.JUMP_COUNT[h]
					calls := len(common.JUMP_CALLS[h])
					if calls >= 100 {
						f.WriteString(fmt.Sprintf("h_%s %7d %9d\n", ENC(h.Bytes()), calls, total))
						for src, m2 := range m1 {
							f.WriteString(fmt.Sprintf(" %06x\n", src))
							for dst, m3 := range m2 {
								f.WriteString(fmt.Sprintf("  %06x\n", dst))
								for cid, c := range m3 {
									f.WriteString(fmt.Sprintf("   %8d %8d\n", cid, c))
								}
							}
						}
					}
				}
				f.Flush()
				_f.Close()
			} else {
				log.Error("Jump counts not saved", "error", _err)
			}
		}
	}

	utils.NotifySIGINT()
	<-quit
	return common.ErrStopped

	return fmt.Errorf("early stop")
	if err = s.Update(batch, stageProgress); err != nil {
		return err
	}
	if err = batch.Commit(); err != nil {
		return fmt.Errorf("batch commit: %v", err)
	}

	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}

	log.Info(fmt.Sprintf("[%s] Completed on", logPrefix), "block", stageProgress)
	return stoppedErr
}

func pruneChangeSets(tx kv.RwTx, logPrefix string, table string, pruneTo uint64, logEvery *time.Ticker, ctx context.Context) error {
	c, err := tx.RwCursorDupSort(table)
	if err != nil {
		return fmt.Errorf("failed to create cursor for pruning %w", err)
	}
	defer c.Close()

	for k, _, err := c.First(); k != nil; k, _, err = c.NextNoDup() {
		if err != nil {
			return fmt.Errorf("failed to move %s cleanup cursor: %w", table, err)
		}
		blockNum := binary.BigEndian.Uint64(k)
		if blockNum >= pruneTo {
			break
		}
		select {
		case <-logEvery.C:
			log.Info(fmt.Sprintf("[%s]", logPrefix), "table", table, "block", blockNum)
		case <-ctx.Done():
			return common.ErrStopped
		default:
		}
		if err = c.DeleteCurrentDuplicates(); err != nil {
			return fmt.Errorf("failed to remove for block %d: %w", blockNum, err)
		}
	}
	return nil
}

func logProgress(logPrefix string, prevBlock uint64, prevTime time.Time, currentBlock uint64, prevTx, currentTx uint64, gas uint64, batch ethdb.DbWithPendingMutations) (uint64, uint64, time.Time) {
	currentTime := time.Now()
	interval := currentTime.Sub(prevTime)
	speed := float64(currentBlock-prevBlock) / (float64(interval) / float64(time.Second))
	speedTx := float64(currentTx-prevTx) / (float64(interval) / float64(time.Second))
	speedMgas := float64(gas) / 1_000_000 / (float64(interval) / float64(time.Second))
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	var logpairs = []interface{}{
		"number", currentBlock,
		"blk/s", speed,
		"tx/s", speedTx,
		"Mgas/s", speedMgas,
	}
	if batch != nil {
		logpairs = append(logpairs, "batch", common.StorageSize(batch.BatchSize()))
	}
	logpairs = append(logpairs, "alloc", common.StorageSize(m.Alloc), "sys", common.StorageSize(m.Sys))
	log.Info(fmt.Sprintf("[%s] Executed blocks", logPrefix), logpairs...)

	return currentBlock, currentTx, currentTime
}

func UnwindExecutionStage(u *UnwindState, s *StageState, tx kv.RwTx, ctx context.Context, cfg ExecuteBlockCfg, initialCycle bool) (err error) {
	quit := ctx.Done()
	if u.UnwindPoint >= s.BlockNumber {
		return nil
	}
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(context.Background())
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}
	logPrefix := u.LogPrefix()
	log.Info(fmt.Sprintf("[%s] Unwind Execution", logPrefix), "from", s.BlockNumber, "to", u.UnwindPoint)

	if err = unwindExecutionStage(u, s, tx, quit, cfg, initialCycle); err != nil {
		return err
	}
	if err = u.Done(tx); err != nil {
		return err
	}

	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func unwindExecutionStage(u *UnwindState, s *StageState, tx kv.RwTx, quit <-chan struct{}, cfg ExecuteBlockCfg, initialCycle bool) error {
	logPrefix := s.LogPrefix()
	stateBucket := kv.PlainState
	storageKeyLength := common.AddressLength + common.IncarnationLength + common.HashLength

	var accumulator *shards.Accumulator
	if !initialCycle && cfg.stateStream {
		accumulator = cfg.accumulator

		hash, err := rawdb.ReadCanonicalHash(tx, u.UnwindPoint)
		if err != nil {
			return fmt.Errorf("read canonical hash of unwind point: %w", err)
		}
		txs, err := rawdb.RawTransactionsRange(tx, u.UnwindPoint, s.BlockNumber)
		if err != nil {
			return err
		}
		targetHeader := rawdb.ReadHeader(tx, hash, u.UnwindPoint)
		var protocolBaseFee uint64
		if targetHeader.BaseFee != nil {
			protocolBaseFee = targetHeader.BaseFee.Uint64()
		}
		accumulator.StartChange(u.UnwindPoint, hash, txs, protocolBaseFee, true /* unwind */)
	}

	changes := etl.NewCollector(cfg.tmpdir, etl.NewOldestEntryBuffer(etl.BufferOptimalSize))
	defer changes.Close(logPrefix)
	errRewind := changeset.RewindData(tx, s.BlockNumber, u.UnwindPoint, changes, quit)
	if errRewind != nil {
		return fmt.Errorf("getting rewind data: %w", errRewind)
	}

	if err := changes.Load(logPrefix, tx, stateBucket, func(k, v []byte, table etl.CurrentTableReader, next etl.LoadNextFunc) error {
		if len(k) == 20 {
			if len(v) > 0 {
				var acc accounts.Account
				if err := acc.DecodeForStorage(v); err != nil {
					return err
				}

				// Fetch the code hash
				recoverCodeHashPlain(&acc, tx, k)
				var address common.Address
				copy(address[:], k)

				// cleanup contract code bucket
				original, err := state.NewPlainStateReader(tx).ReadAccountData(address)
				if err != nil {
					return fmt.Errorf("read account for %x: %w", address, err)
				}
				if original != nil {
					// clean up all the code incarnations original incarnation and the new one
					for incarnation := original.Incarnation; incarnation > acc.Incarnation && incarnation > 0; incarnation-- {
						err = tx.Delete(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(address[:], incarnation), nil)
						if err != nil {
							return fmt.Errorf("writeAccountPlain for %x: %w", address, err)
						}
					}
				}

				newV := make([]byte, acc.EncodingLengthForStorage())
				acc.EncodeForStorage(newV)
				if accumulator != nil {
					accumulator.ChangeAccount(address, acc.Incarnation, newV)
				}
				if err := next(k, k, newV); err != nil {
					return err
				}
			} else {
				if accumulator != nil {
					var address common.Address
					copy(address[:], k)
					accumulator.DeleteAccount(address)
				}
				if err := next(k, k, nil); err != nil {
					return err
				}
			}
			return nil
		}
		if accumulator != nil {
			var address common.Address
			var incarnation uint64
			var location common.Hash
			copy(address[:], k[:common.AddressLength])
			incarnation = binary.BigEndian.Uint64(k[common.AddressLength:])
			copy(location[:], k[common.AddressLength+common.IncarnationLength:])
			accumulator.ChangeStorage(address, incarnation, location, common.CopyBytes(v))
		}
		if len(v) > 0 {
			if err := next(k, k[:storageKeyLength], v); err != nil {
				return err
			}
		} else {
			if err := next(k, k[:storageKeyLength], nil); err != nil {
				return err
			}
		}
		return nil

	}, etl.TransformArgs{Quit: quit}); err != nil {
		return err
	}

	if err := changeset.Truncate(tx, u.UnwindPoint+1); err != nil {
		return err
	}

	if err := rawdb.DeleteNewerReceipts(tx, u.UnwindPoint+1); err != nil {
		return fmt.Errorf("walking receipts: %w", err)
	}
	if err := rawdb.DeleteNewerEpochs(tx, u.UnwindPoint+1); err != nil {
		return fmt.Errorf("walking epoch: %w", err)
	}

	// Truncate CallTraceSet
	keyStart := dbutils.EncodeBlockNumber(u.UnwindPoint + 1)
	c, err := tx.RwCursorDupSort(kv.CallTraceSet)
	if err != nil {
		return err
	}
	defer c.Close()
	for k, _, err := c.Seek(keyStart); k != nil; k, _, err = c.NextNoDup() {
		if err != nil {
			return err
		}
		err = c.DeleteCurrentDuplicates()
		if err != nil {
			return err
		}
	}

	return nil
}

func recoverCodeHashPlain(acc *accounts.Account, db kv.Tx, key []byte) {
	var address common.Address
	copy(address[:], key)
	if acc.Incarnation > 0 && acc.IsEmptyCodeHash() {
		if codeHash, err2 := db.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(address[:], acc.Incarnation)); err2 == nil {
			copy(acc.CodeHash[:], codeHash)
		}
	}
}

func min(a, b uint64) uint64 {
	if a <= b {
		return a
	}
	return b
}

func PruneExecutionStage(s *PruneState, tx kv.RwTx, cfg ExecuteBlockCfg, ctx context.Context, initialCycle bool) (err error) {
	logPrefix := s.LogPrefix()
	useExternalTx := tx != nil
	if !useExternalTx {
		tx, err = cfg.db.BeginRw(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
	}

	logEvery := time.NewTicker(logInterval)
	defer logEvery.Stop()

	if cfg.prune.History.Enabled() {
		if err = pruneChangeSets(tx, logPrefix, kv.AccountChangeSet, cfg.prune.History.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
			return err
		}
		if err = pruneChangeSets(tx, logPrefix, kv.StorageChangeSet, cfg.prune.History.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
			return err
		}
	}

	if cfg.prune.Receipts.Enabled() {
		if err = pruneReceipts(tx, logPrefix, cfg.prune.Receipts.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
			return err
		}
	}
	if cfg.prune.CallTraces.Enabled() {
		if err = pruneCallTracesSet(tx, logPrefix, cfg.prune.CallTraces.PruneTo(s.ForwardProgress), logEvery, ctx); err != nil {
			return err
		}
	}

	if err = s.Done(tx); err != nil {
		return err
	}
	if !useExternalTx {
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func pruneReceipts(tx kv.RwTx, logPrefix string, pruneTo uint64, logEvery *time.Ticker, ctx context.Context) error {
	c, err := tx.RwCursor(kv.Receipts)
	if err != nil {
		return fmt.Errorf("failed to create cursor for pruning %w", err)
	}
	defer c.Close()

	for k, _, err := c.First(); k != nil; k, _, err = c.Next() {
		if err != nil {
			return err
		}

		blockNum := binary.BigEndian.Uint64(k)
		if blockNum >= pruneTo {
			break
		}
		select {
		case <-logEvery.C:
			log.Info(fmt.Sprintf("[%s]", logPrefix), "table", kv.Receipts, "block", blockNum)
		case <-ctx.Done():
			return common.ErrStopped
		default:
		}
		if err = c.DeleteCurrent(); err != nil {
			return fmt.Errorf("failed to remove for block %d: %w", blockNum, err)
		}
	}

	c, err = tx.RwCursor(kv.Log)
	if err != nil {
		return fmt.Errorf("failed to create cursor for pruning %w", err)
	}
	defer c.Close()

	for k, _, err := c.First(); k != nil; k, _, err = c.Next() {
		if err != nil {
			return err
		}
		blockNum := binary.BigEndian.Uint64(k)
		if blockNum >= pruneTo {
			break
		}
		select {
		case <-logEvery.C:
			log.Info(fmt.Sprintf("[%s]", logPrefix), "table", kv.Log, "block", blockNum)
		case <-ctx.Done():
			return common.ErrStopped
		default:
		}
		if err = c.DeleteCurrent(); err != nil {
			return fmt.Errorf("failed to remove for block %d: %w", blockNum, err)
		}
	}
	return nil
}

func pruneCallTracesSet(tx kv.RwTx, logPrefix string, pruneTo uint64, logEvery *time.Ticker, ctx context.Context) error {
	c, err := tx.RwCursorDupSort(kv.CallTraceSet)
	if err != nil {
		return fmt.Errorf("failed to create cursor for pruning %w", err)
	}
	defer c.Close()

	for k, _, err := c.First(); k != nil; k, _, err = c.NextNoDup() {
		if err != nil {
			return err
		}
		blockNum := binary.BigEndian.Uint64(k)
		if blockNum >= pruneTo {
			break
		}
		select {
		case <-logEvery.C:
			log.Info(fmt.Sprintf("[%s]", logPrefix), "table", kv.CallTraceSet, "block", blockNum)
		case <-ctx.Done():
			return common.ErrStopped
		default:
		}
		if err = c.DeleteCurrentDuplicates(); err != nil {
			return fmt.Errorf("failed to remove for block %d: %w", blockNum, err)
		}
	}
	return nil
}
