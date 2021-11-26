package chain

import "encoding/json"

type ChainInfo struct {
	ServerVersion             string `json:"server_version"`
	ChainID                   string `json:"chain_id"`
	HeadBlockNum              int64  `json:"head_block_num"`
	LastIrreversibleBlockNum  int64  `json:"last_irreversible_block_num"`
	LastIrreversibleBlockID   string `json:"last_irreversible_block_id"`
	HeadBlockID               string `json:"head_block_id"`
	HeadBlockTime             string `json:"head_block_time"`
	HeadBlockProducer         string `json:"head_block_producer"`
	VirtualBlockCPULimit      int64  `json:"virtual_block_cpu_limit"`
	VirtualBlockNetLimit      int64  `json:"virtual_block_net_limit"`
	BlockCPULimit             int64  `json:"block_cpu_limit"`
	BlockNetLimit             int64  `json:"block_net_limit"`
	ServerVersionString       string `json:"server_version_string"`
	ForkDBHeadBlockNum        int64  `json:"fork_db_head_block_num"`
	ForkDBHeadBlockID         string `json:"fork_db_head_block_id"`
	ServerFullVersionString   string `json:"server_full_version_string"`
	LastIrreversibleBlockTime string `json:"last_irreversible_block_time"`
}

func NewChainInfo(info []byte) (*ChainInfo, error) {
	chainInfo := &ChainInfo{}
	if err := json.Unmarshal(info, chainInfo); err != nil {
		return nil, newError(err)
	}
	return chainInfo, nil
}
