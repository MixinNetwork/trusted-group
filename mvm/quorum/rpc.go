package quorum

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

const (
	quorumMinimumHeight = 8192
	etherPrecision      = 18
)

type RPC struct {
	client *http.Client
	host   string
}

func NewRPC(host string) (*RPC, error) {
	chain := &RPC{
		client: &http.Client{Timeout: 5 * time.Second},
		host:   host,
	}
	height, err := chain.GetBlockHeight()
	if err != nil {
		return nil, err
	}
	if height < quorumMinimumHeight {
		return nil, fmt.Errorf("block height too small %d", height)
	}
	return chain, nil
}

func (chain *RPC) GetBlockHeight() (uint64, error) {
	body, err := chain.call("eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}
	var resp struct {
		Result string         `json:"result"`
		Error  *EthereumError `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, resp.Error
	}
	return ethereumNumberToUint64(resp.Result)
}

func (chain *RPC) GetContractBirthBlock(address, hash string) (uint64, error) {
	body, err := chain.call("eth_getTransactionReceipt", []interface{}{hash})
	if err != nil {
		return 0, err
	}
	var resp struct {
		Result struct {
			BlockNumber     string `json:"blockNumber"`
			ContractAddress string `json:"contractAddress"`
		} `json:"result"`
		Error *EthereumError `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, resp.Error
	}
	if formatAddress(resp.Result.ContractAddress) != address {
		return 0, fmt.Errorf("malformed %s %s %s", address, hash, resp.Result.ContractAddress)
	}
	return ethereumNumberToUint64(resp.Result.BlockNumber)
}

func (chain *RPC) GetAddressNonce(address string) (uint64, error) {
	body, err := chain.call("eth_getTransactionCount", []interface{}{address, "latest"})
	if err != nil {
		return 0, err
	}
	var resp struct {
		Result string         `json:"result"`
		Error  *EthereumError `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, resp.Error
	}
	return ethereumNumberToUint64(resp.Result)
}

func (chain *RPC) GetAddressBalance(address string) (decimal.Decimal, error) {
	body, err := chain.call("eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		return decimal.Zero, err
	}
	var resp struct {
		Result string         `json:"result"`
		Error  *EthereumError `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return decimal.Zero, err
	}
	if resp.Error != nil {
		return decimal.Zero, err
	}
	return ethereumNumberToDecimal(resp.Result)
}

func (chain *RPC) GetLogs(address, topic string, from, to uint64) ([][]byte, error) {
	logger.Verbosef("RPC.GetLogs(%s, %s, %d, %d)", address, topic, from, to)
	body, err := chain.call("eth_getLogs", []interface{}{map[string]interface{}{
		"address":   address,
		"topics":    []string{topic},
		"fromBlock": fmt.Sprintf("0x%x", from),
		"toBlock":   fmt.Sprintf("0x%x", to),
	}})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Result []struct {
			Data string `json:"data"`
		} `json:"result"`
		Error *EthereumError `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var logs [][]byte
	for _, r := range resp.Result {
		b, err := hex.DecodeString(r.Data[2:])
		if err != nil {
			return nil, err
		}
		if len(b)%64 != 0 {
			return nil, fmt.Errorf("invalid log %s", r.Data)
		}
		if len(b) < 128 {
			return nil, fmt.Errorf("invalid log %s", r.Data)
		}
		bi := new(big.Int).SetBytes(b[:32])
		if !bi.IsInt64() {
			return nil, fmt.Errorf("invalid log %s", r.Data)
		}
		if bi.Int64() != 0x20 {
			return nil, fmt.Errorf("invalid log %s", r.Data)
		}
		bi = new(big.Int).SetBytes(b[32:64])
		if !bi.IsInt64() {
			return nil, fmt.Errorf("invalid log %s", r.Data)
		}
		if bi.Int64() > 512 {
			return nil, fmt.Errorf("invalid log %s", r.Data)
		}
		logs = append(logs, b[64:64+bi.Int64()])
	}
	return logs, nil
}

func (chain *RPC) SendRawTransaction(raw string) (string, error) {
	body, err := chain.call("eth_sendRawTransaction", []interface{}{raw})
	if err != nil {
		return "", err
	}
	var resp struct {
		Result string         `json:"result"`
		Error  *EthereumError `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result, nil
}

func (chain *RPC) call(method string, params []interface{}) ([]byte, error) {
	data := map[string]interface{}{
		"method":  method,
		"params":  params,
		"id":      time.Now().UnixNano(),
		"jsonrpc": "2.0",
	}

	body := encoding.JSONMarshalPanic(data)
	req, err := http.NewRequest("POST", chain.host, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	resp, err := chain.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func formatAddress(to string) string {
	ab, err := hex.DecodeString(to[2:])
	if err != nil {
		panic(err)
	}
	var address common.Address
	copy(address[:], ab)
	return address.Hex()
}

func ethereumNumberToUint64(hex string) (uint64, error) {
	if !strings.HasPrefix(hex, "0x") {
		return 0, fmt.Errorf("invalid hex %s", hex)
	}
	value, success := new(big.Int).SetString(hex, 0)
	if !success {
		return 0, fmt.Errorf("invalid hex %s", hex)
	}
	if !value.IsUint64() {
		return 0, fmt.Errorf("invalid uint64 %s", hex)
	}
	return value.Uint64(), nil
}

func ethereumNumberToDecimal(hex string) (decimal.Decimal, error) {
	if !strings.HasPrefix(hex, "0x") {
		return decimal.Zero, fmt.Errorf("invalid hex %s", hex)
	}
	value, success := new(big.Int).SetString(hex, 0)
	if !success {
		return decimal.Zero, fmt.Errorf("invalid hex %s", hex)
	}
	d, err := decimal.NewFromString(value.String())
	return d.Div(decimal.New(1, etherPrecision)), err
}

type EthereumError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err *EthereumError) Error() string {
	return fmt.Sprintf("RPC ERROR Ethereum %d %s", err.Code, err.Message)
}
