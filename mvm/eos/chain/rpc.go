package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/MixinNetwork/tip/logger"
)

type GetTableRowsArgs struct {
	Json          bool   `json:"json"`
	Code          string `json:"code"`
	Scope         string `json:"scope"`
	Table         string `json:"table"`
	LowerBound    string `json:"lower_bound"`
	UpperBound    string `json:"upper_bound"`
	Limit         int    `json:"limit"`
	KeyType       string `json:"key_type"`
	IndexPosition int    `json:"index_position"`
	Reverse       bool   `json:"reverse"`
	ShowPayer     bool   `json:"show_payer"`
}

type GetRequiredKeysArgs struct {
	Transaction   *Transaction `json:"transaction"`
	AvailableKeys []string     `json:"available_keys"`
}

type GetAccountArgs struct {
	AccountName string `json:"account_name"`
}

//required_keys
type GetRequiredKeysResult struct {
	RequiredKeys []string `json:"required_keys"`
}

// struct get_actions_params {
// 	chain::account_name     account_name;
// 	std::optional<int32_t>  pos; /// a absolute sequence positon -1 is the end/last action
// 	std::optional<int32_t>  offset; ///< the number of actions relative to pos, negative numbers return [pos-offset,pos), positive numbers return [pos,pos+offset)
//  };

type GetActionsArgs struct {
	AccountName string `json:"account_name"`
	Pos         int    `json:"pos"`
	Offset      int    `json:"offset"`
}

type RpcError struct {
	err string
}

func (r *RpcError) Error() string {
	return r.err
}

func NewRpcError(err string) *RpcError {
	return &RpcError{err}
}

type Rpc struct {
	client *http.Client
	url    string
}

func NewRpc(url string) *Rpc {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    15 * time.Second,
		DisableCompression: true,
	}

	rpc := &Rpc{}
	rpc.url = url
	rpc.client = &http.Client{Transport: tr}
	return rpc
}

func (r *Rpc) GetInfo() (*ChainInfo, error) {
	var info ChainInfo
	result, err := r.Call("chain", "get_info", "")
	if err != nil {
		return nil, newError(err)
	}

	err = json.Unmarshal(result, &info)
	if err != nil {
		return nil, newError(err)
	}
	return &info, nil
}

func (r *Rpc) GetAccount(args *GetAccountArgs) (JsonObject, error) {
	_args, err := json.Marshal(args)
	if err != nil {
		return nil, newError(err)
	}

	result, err := r.Call("chain", "get_account", _args)
	if err != nil {
		return nil, newError(err)
	}
	return NewJsonObjectFromBytes(result)
}

func (r *Rpc) GetRequiredKeys(args *GetRequiredKeysArgs) (*GetRequiredKeysResult, error) {
	_args, err := json.Marshal(args)
	if err != nil {
		return nil, newError(err)
	}

	result, err := r.Call("chain", "get_required_keys", _args)
	if err != nil {
		return nil, newError(err)
	}

	result2 := &GetRequiredKeysResult{}
	err = json.Unmarshal(result, result2)
	if err != nil {
		return nil, newError(err)
	}
	return result2, nil
}

func (t *Rpc) GetTableRows(args *GetTableRowsArgs) (JsonObject, error) {
	r, err := t.Call("chain", "get_table_rows", args)
	if err != nil {
		return nil, err
	}
	return NewJsonObjectFromBytes(r)
}

func (t *Rpc) PushTransaction(packedTx *PackedTransaction) (JsonObject, error) {
	_packedTx, err := json.Marshal(packedTx)
	if err != nil {
		return nil, err
	}
	logger.Verbose("Packed transaction: %s", string(_packedTx))
	r, err := t.Call("chain", "push_transaction", _packedTx)
	if err != nil {
		return nil, err
	}
	return NewJsonObjectFromBytes(r)
}

func (t *Rpc) GetActions(args *GetActionsArgs) (JsonObject, error) {
	_args, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	r, err := t.Call("history", "get_actions", _args)
	if err != nil {
		return nil, err
	}
	return NewJsonObjectFromBytes(r)
}

func (r *Rpc) Call(api string, endpoint string, params interface{}) ([]byte, error) {
	var _params []byte
	reqUrl := fmt.Sprintf("%s/v1/%s/%s", r.url, api, endpoint)

	switch v := params.(type) {
	case string:
		_params = []byte(v)
	case []byte:
		_params = v
	default:
		var err error
		_params, err = json.Marshal(v)
		if err != nil {
			return nil, newError(err)
		}
	}

	if len(_params) == 0 {
		resp, err := r.client.Get(reqUrl)
		if err != nil {
			return nil, newError(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, newError(err)
		}
		return body, nil
	}

	buf := bytes.NewBuffer(_params)
	resp, err := r.client.Post(reqUrl, "application/json", buf)
	if err != nil {
		return nil, newError(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, newError(err)
	}
	return body, nil
}
