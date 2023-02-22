package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

type MixinNetwork struct {
	httpClient *http.Client
	node       string
}

type Input struct {
	Hash    string              `json:"hash"`
	Index   int                 `json:"index"`
	Genesis string              `json:"genesis"`
	Deposit *common.DepositData `json:"deposit"`
	Mint    *common.MintData    `json:"mint"`
}

type Output struct {
	Type   uint8          `json:"type"`
	Amount number.Decimal `json:"amount"`
	Keys   []*crypto.Key  `json:"keys"`
	Script common.Script  `json:"script"`
	Mask   crypto.Key     `json:"mask"`
}

type Transaction struct {
	Version  uint8     `json:"version"`
	Asset    string    `json:"asset"`
	Inputs   []*Input  `json:"inputs"`
	Outputs  []*Output `json:"outputs"`
	Extra    string    `json:"extra"`
	Hash     string    `json:"hash"`
	Raw      string    `json:"hex"`
	Snapshot string    `json:"snapshot"`
}

type SnapshotWithTransaction struct {
	Hash        string      `json:"hash"`
	Timestamp   uint64      `json:"timestamp"`
	Topology    uint64      `json:"topology"`
	Transaction Transaction `json:"transaction"`
}

func NewMixinNetwork(node string) *MixinNetwork {
	return &MixinNetwork{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		node:       node,
	}
}

func (m *MixinNetwork) GetSnapshot(hash string) (*SnapshotWithTransaction, error) {
	body, err := m.callRPC("getsnapshot", []interface{}{hash})
	if err != nil {
		return nil, err
	}
	var s SnapshotWithTransaction
	err = json.Unmarshal(body, &s)
	if err != nil || s.Hash == "" {
		return nil, err
	}
	return &s, err
}

func (m *MixinNetwork) callRPC(method string, params []interface{}) ([]byte, error) {
	body, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", m.node, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data  interface{} `json:"data"`
		Error interface{} `json:"error"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("ERROR %s", result.Error)
	}

	return json.Marshal(result.Data)
}
