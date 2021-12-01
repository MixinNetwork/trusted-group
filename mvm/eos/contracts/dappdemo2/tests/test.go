package main

import (
	"github.com/uuosio/chain"
)

//contract test
type TransactionTest struct {
	Receiver      chain.Name
	FirstReceiver chain.Name
	Action        chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *TransactionTest {
	return &TransactionTest{receiver, firstReceiver, action}
}

//action sayhello
func (test *TransactionTest) SayHello() {
	payer := test.Receiver

	a := chain.NewAction(
		chain.PermissionLevel{test.Receiver, chain.ActiveName},
		test.Receiver,
		chain.NewName("txrequest"),
	)

	tx := chain.NewTransaction(100)
	tx.Actions = []*chain.Action{a}
	tx.Send(1, false, payer)
	chain.Println("transaction sent")
}

//action txrequest
func (test *TransactionTest) TxRequest() {
	//chain.Check(false, "bad request!")
}
