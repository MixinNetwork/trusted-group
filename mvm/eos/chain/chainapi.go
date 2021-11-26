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
	return api.rpc.GetAccount(&GetAccountArgs{AccountName: name})
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

func (api *ChainApi) getRequiredKeys(actions []Action) ([]string, error) {
	args := GetRequiredKeysArgs{
		Transaction:   NewTransaction(0),
		AvailableKeys: GetWallet().GetPublicKeys(),
	}
	for i := range actions {
		a := actions[i]
		a.Data = []byte{}
		args.Transaction.AddAction(&a)
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

	expiration := int(time.Now().Unix()) + 60
	tx := NewTransaction(expiration)
	tx.SetReferenceBlock(chainInfo.LastIrreversibleBlockID)
	for i := range actions {
		a := actions[i]
		tx.AddAction(a)
	}

	chainId := chainInfo.ChainID

	packedTx := NewPackedTransaction(tx)
	packedTx.SetChainId(chainId)

	pubKeys, err := api.getRequiredKeys(tx.Actions)
	if err != nil {
		return nil, err
	}
	if len(pubKeys) == 0 {
		return nil, newErrorf("sign key not found!")
	}

	for i := range pubKeys {
		pub := pubKeys[i]
		_, err = packedTx.Sign(pub)
		if err != nil {
			return nil, err
		}
	}

	r2, err := api.rpc.PushTransaction(packedTx)
	if err != nil {
		return nil, err
	}
	if _, err := r2.Get("error"); err == nil {
		return r2, newErrorf(r2.ToString())
	}
	return r2, nil
}

func (api *ChainApi) GetActions(account string, pos int, offset int) (JsonObject, error) {
	args := GetActionsArgs{
		AccountName: account,
		Pos:         pos,
		Offset:      offset,
	}
	return api.rpc.GetActions(&args)
}
