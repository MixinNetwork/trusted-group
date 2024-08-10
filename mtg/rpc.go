package mtg

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MixinNetwork/mixin/common"
)

func GetKernelTransaction(rpc, hash string) (*common.VersionedTransaction, string, error) {
	raw, err := callMixinRPC(rpc, "gettransaction", []any{hash})
	if err != nil || raw == nil {
		return nil, "", err
	}
	var signed map[string]any
	err = json.Unmarshal(raw, &signed)
	if err != nil {
		panic(string(raw))
	}
	hex, err := hex.DecodeString(signed["hex"].(string))
	if err != nil {
		panic(string(raw))
	}
	ver, err := common.UnmarshalVersionedTransaction(hex)
	if err != nil {
		panic(string(raw))
	}
	if signed["snapshot"] == nil {
		return ver, "", nil
	}
	return ver, signed["snapshot"].(string), nil
}

func callMixinRPC(node, method string, params []any) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	body, err := json.Marshal(map[string]any{
		"method": method,
		"params": params,
	})
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", node, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data  any `json:"data"`
		Error any `json:"error"`
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	err = dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("callMixinRPC(%s, %s, %s) => %v", node, method, params, result.Error)
	}
	if result.Data == nil {
		return nil, nil
	}

	return json.Marshal(result.Data)
}
