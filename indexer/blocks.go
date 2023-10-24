package indexer

import (
	"context"
	"encoding/hex"
	"flare-ftso-indexer/config"
	"flare-ftso-indexer/indexer/abi"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

type BlockBatch struct {
	Blocks []*types.Block
	sync.Mutex
}

func NewBlockBatch(batchSize int) *BlockBatch {
	blockBatch := BlockBatch{}
	blockBatch.Blocks = make([]*types.Block, batchSize)

	return &blockBatch
}

func (ci *BlockIndexer) fetchLastBlockIndex() (int, error) {
	// todo: change to header by number when mocking is available
	var lastBlock *types.Block
	var err error
	for j := 0; j < config.ReqRepeats; j++ {
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(ci.params.TimeoutMillis)*time.Millisecond)
		lastBlock, err = ci.client.BlockByNumber(ctx, nil)
		cancelFunc()
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, err
	}

	return int(lastBlock.NumberU64()), nil
}

func (ci *BlockIndexer) requestBlocks(blockBatch *BlockBatch, start, stop, listIndex, lastIndex int, errChan chan error) {
	for i := start; i < stop; i++ {
		var block *types.Block
		var err error
		if i > lastIndex {
			block = &types.Block{}
		} else {
			for j := 0; j < config.ReqRepeats; j++ {
				ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(ci.params.TimeoutMillis)*time.Millisecond)
				block, err = ci.client.BlockByNumber(ctx, big.NewInt(int64(i)))
				cancelFunc()
				if err == nil {
					break
				}
			}
			if err != nil {
				errChan <- err
				return
			}
		}
		blockBatch.Lock()
		blockBatch.Blocks[listIndex+i-start] = block
		blockBatch.Unlock()
	}

	errChan <- nil
}

func (ci *BlockIndexer) processBlocks(blockBatch *BlockBatch, batchTransactions *TransactionsBatch, start, stop int, errChan chan error) {
	for i := start; i < stop; i++ {
		block := blockBatch.Blocks[i]
		for _, tx := range block.Transactions() {
			txData := hex.EncodeToString(tx.Data())
			if len(txData) < 8 {
				continue
			}
			// todo: check contract's address
			_, ok := abi.FtsoPrefixToFuncCall[txData[:8]]
			if !ok {
				continue
			}

			batchTransactions.Lock()
			batchTransactions.Transactions = append(batchTransactions.Transactions, tx)
			batchTransactions.toBlock = append(batchTransactions.toBlock, block)
			batchTransactions.toReceipt = append(batchTransactions.toReceipt, nil)
			batchTransactions.Unlock()
		}
	}
	errChan <- nil
}
