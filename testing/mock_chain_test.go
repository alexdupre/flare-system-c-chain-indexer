package testing

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ava-labs/coreth/ethclient"
	"github.com/stretchr/testify/assert"
)

func TestMockChain(t *testing.T) {
	ctx := context.Background()

	go func() {
		err := MockChain(5500, "chain_copy/blocks.json", "chain_copy/transactions.json")
		assert.NoError(t, err)
	}()

	time.Sleep(time.Second * 3)

	client, err := ethclient.Dial("http://localhost:5500")
	assert.NoError(t, err)

	blockNum := uint64(2001)
	block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	assert.NoError(t, err)
	assert.Equal(t, block.NumberU64(), blockNum)

	for _, tx := range block.Transactions() {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		assert.NoError(t, err)
		assert.Equal(t, receipt.TxHash, tx.Hash())
	}
}
