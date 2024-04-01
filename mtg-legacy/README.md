# MTG

This module can bootstrap a MTG application very effortlessly, it's as simple as a few lines of code.

```golang
func (rw *RefundWorker) ProcessOutput(ctx context.Context, out *mtg.Output) {
	receivers := []string{out.Sender}
	traceId := mixin.UniqueConversationID(out.UTXOID, "refund")
	err := rw.grp.BuildTransaction(ctx, out.AssetID, receivers, 1, out.Amount.String(), "refund", traceId)
	if err != nil {
		panic(err)
	}
}

group, _ := mtg.BuildGroup(ctx, db, conf)
rw := NewRefundrWorker(ctx, group, conf)
group.AddWorker(rw)
group.Run(ctx)
```

The group will call every workers added, and the worker just needs to implement the `ProcessOutput` interface. The code above is a very simple worker that refunds all the payments received.
