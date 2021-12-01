package chain

import (
	"time"
)

type ChainApi struct {
	rpc *Rpc
}

func NewChainApi(rpcUrl string) *ChainApi {
	rpc := NewRpc(rpcUrl)
	chainApi := &ChainApi{rpc: rpc}
	return chainApi
}

func (api *ChainApi) GetAccount(name string) (JsonObject, error) {
	ret, err := api.rpc.GetAccount(&GetAccountArgs{AccountName: name})
	if err != nil {
		return nil, newError(err)
	}
	if _, err := ret.Get("error"); err == nil {
		return ret, newErrorf(ret.ToString())
	}
	return ret, nil
}

func (api *ChainApi) GetTableRows(
	json bool,
	code string,
	scope string,
	table string,
	lowerbound string,
	upperbound string,
	limit int,
	keyType string,
	indexPosition int,
	reverse bool,
	showPayer bool,
) (JsonObject, error) {
	args := GetTableRowsArgs{
		Json:          json,
		Code:          code,
		Scope:         scope,
		Table:         table,
		LowerBound:    lowerbound,
		UpperBound:    upperbound,
		Limit:         limit,
		KeyType:       keyType,
		IndexPosition: indexPosition,
		Reverse:       reverse,
		ShowPayer:     showPayer,
	}
	return api.rpc.GetTableRows(&args)
}

func (api *ChainApi) getRequiredKeys(actions []*Action) ([]string, error) {
	args := GetRequiredKeysArgs{
		Transaction:   NewTransaction(0),
		AvailableKeys: GetWallet().GetPublicKeys(),
	}
	for i := range actions {
		a := actions[i]
		a.Data = []byte{}
		args.Transaction.AddAction(a)
	}
	r, err := api.rpc.GetRequiredKeys(&args)
	if err != nil {
		return nil, newError(err)
	}
	return r.RequiredKeys, nil
}

func (api *ChainApi) PushAction(action *Action) (JsonObject, error) {
	return api.PushActions([]*Action{action})
}

func (api *ChainApi) PushActions(actions []*Action) (JsonObject, error) {
	chainInfo, err := api.rpc.GetInfo()
	if err != nil {
		return nil, err
	}

	expiration := uint32(time.Now().Unix()) + 60
	tx := NewTransaction(expiration)
	tx.SetReferenceBlock(chainInfo.LastIrreversibleBlockID)
	for i := range actions {
		a := actions[i]
		tx.AddAction(a)
	}

	pubKeys, err := api.getRequiredKeys(tx.Actions)
	if err != nil {
		return nil, err
	}
	if len(pubKeys) == 0 {
		return nil, newErrorf("signing key not found!")
	}

	chainId, err := NewBytes32FromHex(chainInfo.ChainID)
	if err != nil {
		panic(err)
	}

	signatures := []string{}
	for i := range pubKeys {
		pub := pubKeys[i]
		sign, err := tx.SignWithPublicKey(pub, chainId)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, sign.String())
	}

	r2, err := api.rpc.PushTransaction(tx, signatures, false)
	if err != nil {
		return nil, err
	}
	return r2, nil
}

func (t *ChainApi) PushTransaction(tx *Transaction, signatures []string, comporess bool) (JsonObject, error) {
	r, err := t.rpc.PushTransaction(tx, signatures, comporess)
	if err != nil {
		return nil, newError(err)
	}
	if _, err := r.Get("error"); err == nil {
		return r, newErrorf(r.ToString())
	}
	return r, nil
}

func (api *ChainApi) GetActions(account string, pos int, offset int) (JsonObject, error) {
	args := GetActionsArgs{
		AccountName: account,
		Pos:         pos,
		Offset:      offset,
	}
	return api.rpc.GetActions(&args)
}
