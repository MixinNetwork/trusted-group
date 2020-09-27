package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MixinNetwork struct {
	httpClient *http.Client
	node       string
}

func NewMixinNetwork(node string) *MixinNetwork {
	return &MixinNetwork{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		node:       node,
	}
}

func (m *MixinNetwork) SendRawTransaction(raw string) (string, error) {
	body, err := m.callRPC("sendrawtransaction", []interface{}{raw})
	if err != nil {
		return "", err
	}
	var tx Transaction
	err = json.Unmarshal(body, &tx)
	return tx.Hash, err
}

func (m *MixinNetwork) GetTransaction(hash string) (*Transaction, error) {
	body, err := m.callRPC("gettransaction", []interface{}{hash})
	if err != nil {
		return nil, err
	}
	var tx Transaction
	err = json.Unmarshal(body, &tx)
	if err != nil || tx.Hash == "" {
		return nil, err
	}
	return &tx, err
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
