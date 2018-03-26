package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/EducationEKT/EKT/io/ekt8/core/common"
	"github.com/EducationEKT/EKT/io/ekt8/db"
	"github.com/EducationEKT/EKT/io/ekt8/i_consensus"
	"github.com/EducationEKT/EKT/io/ekt8/tx_pool"
)

var BackboneChainId []byte

func init() {
	BackboneChainId, _ = hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
}

const (
	CurrentBlockKey       = "CurrentBlock"
	BackboneConsensus     = i_consensus.DPOS
	BackboneBlockInterval = 3
	InitStatus            = 0
	OpenStatus            = 100
	CaculateHashStatus    = 150
)

type BlockChain struct {
	ChainId       []byte
	Consensus     i_consensus.ConsensusType
	CurrentBlock  *Block
	Locker        sync.RWMutex
	Status        int // 100 正在计算MTProot, 150停止计算root,开始计算block Hash
	Fee           int64
	Difficulty    []byte
	TxPool        *tx_pool.TxPool
	CurrentHeight int64
}

func (blockchain *BlockChain) SyncBlockChain() error {
	blockchain.Locker.Lock()
	defer blockchain.Locker.Unlock()
	blockchain.Status = OpenStatus
	return nil
}

func (blockchain *BlockChain) GetStatus() int {
	blockchain.Locker.RLock()
	defer blockchain.Locker.RUnlock()
	return blockchain.Status
}

func (blockchain *BlockChain) ValidateBlock(block *Block) bool {
	return false
}

func (blockchain *BlockChain) NewBlock(block *Block) error {
	blockchain.Locker.Lock()
	defer blockchain.Locker.Unlock()
	if err := block.Validate(); err != nil {
		return err
	}
	db.GetDBInst().Set(block.Hash(), block.Bytes())
	//TODO sync tx and stat
	// TODO refact block的产生和交易模块
	block.UpdateMPTPlusRoot()
	return db.GetDBInst().Set(blockchain.CurrentBlockKey(), block.Hash())
}

func (blockchain *BlockChain) LastBlock() (*Block, error) {
	var err error = nil
	var block *Block
	if currentBlock == nil {
		key := blockchain.CurrentBlockKey()
		blockValue, err := db.GetDBInst().Get(key)
		if err == nil {
			err = json.Unmarshal(blockValue, &block)
			currentBlock = block
			return block, err
		}
	}
	return currentBlock, err
}

func (blockchain *BlockChain) CurrentBlockKey() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteString(CurrentBlockKey)
	buffer.Write(blockchain.ChainId)
	return buffer.Bytes()
}

func (blockchain *BlockChain) WaitAndPack() {
	time.Sleep(BackboneBlockInterval * time.Second)
	blockchain.Locker.Lock()
	defer blockchain.Locker.Unlock()
	blockchain.Pack()
}

func (blockchain *BlockChain) NewTransaction(tx *common.Transaction) {
	blockchain.Locker.Lock()
	defer blockchain.Locker.Unlock()
	if blockchain.Status == OpenStatus {
		blockchain.CurrentBlock.NewTransaction(tx)
	} else {
		blockchain.TxPool.Park(tx, tx_pool.Ready)
	}
}

// consensus 模块调用这个函数，获得一个block对象之后发送给其他节点，其他节点同意之后调用上面的NewBlock方法
func (blockchain *BlockChain) Pack() *Block {
	block := blockchain.CurrentBlock
	block.Locker.Lock()
	defer block.Locker.Unlock()
	start := time.Now().Nanosecond()
	for ; !bytes.HasPrefix(block.CaculateHash(), []byte("FFFFFF")); block.NewNonce() {
	}
	end := time.Now().Nanosecond()
	fmt.Printf(`\ndifficulty="FFFFFF", cost=%d\n`, (end-start)/1e6)
	return block
}
