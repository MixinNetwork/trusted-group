package main

import (
	"context"
	"encoding/binary"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/MixinNetwork/trusted-group/mvm/quorum"
	"github.com/MixinNetwork/trusted-group/mvm/rpc"
	"github.com/MixinNetwork/trusted-group/mvm/store"
	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/urfave/cli/v2"
)

func bootCmd(c *cli.Context) error {
	logger.SetLevel(logger.VERBOSE)
	ctx := context.Background()

	cp := c.String("config")
	if strings.HasPrefix(cp, "~/") {
		usr, _ := user.Current()
		cp = filepath.Join(usr.HomeDir, (cp)[2:])
	}
	conf, err := config.ReadConfiguration(cp)
	if err != nil {
		return err
	}

	if c.Int64("offset") > 0 && strings.TrimSpace(c.String("address")) != "" {
		en := quorum.Build(conf.Quorum)
		return en.FlushDataByOffset(c.String("address"), uint64(c.Int64("offset")))
	}

	bp := c.String("dir")
	if strings.HasPrefix(bp, "~/") {
		usr, _ := user.Current()
		bp = filepath.Join(usr.HomeDir, (bp)[2:])
	}
	db, err := store.OpenBadger(ctx, bp)
	if err != nil {
		return err
	}
	defer db.Close()

	handleUnifiedOutputCheckpoints(db)
	handleInvalidCollectibleTransactions(db)
	go func() {
		if !c.Bool("profile") {
			return
		}
		err := http.ListenAndServe(":9239", http.DefaultServeMux)
		if err != nil {
			panic(err)
		}
	}()

	group, err := mtg.BuildGroup(ctx, db, conf.MTG)
	if err != nil {
		return err
	}

	s := &mixin.Keystore{
		ClientID:   conf.Messenger.UserId,
		SessionID:  conf.Messenger.SessionId,
		PrivateKey: conf.Messenger.Key,
	}
	mixin, err := mixin.NewFromKeystore(s)
	if err != nil {
		return err
	}

	messenger, err := messenger.NewMixinMessenger(ctx, conf.Messenger)
	if err != nil {
		return err
	}
	im, err := machine.Boot(conf.Machine, group, db, messenger, mixin)
	if err != nil {
		return err
	}

	en, err := quorum.Boot(conf.Quorum)
	if err != nil {
		return err
	}
	im.AddEngine(machine.ProcessPlatformQuorum, en)

	go func() {
		if c.Int("port") < 1000 {
			return
		}
		server := rpc.NewServer(en, db, conf, c.Int("port"))
		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	go im.Loop(ctx)
	go RunMonitor(ctx, messenger, db)

	group.SetOutputGrouper(machine.OutputGrouper)
	group.AddWorker(im)
	group.Run(ctx)

	return nil
}

func handleUnifiedOutputCheckpoints(db *store.BadgerStore) {
	val, err := db.ReadProperty([]byte("outputs-draining-checkpoint"))
	if err != nil {
		panic(err)
	}
	if len(val) == 0 {
		return
	}
	ckpt := time.Unix(0, int64(binary.BigEndian.Uint64(val)))
	hack, _ := time.Parse(time.RFC3339Nano, "2023-02-15T00:17:05.90874844Z")
	if ckpt.Before(hack) {
		panic(ckpt)
	}

	err = handleUnifiedOutputCreatedCheckpoint(db, ckpt)
	if err != nil {
		panic(err)
	}
	err = handleUnifiedOutputUpdatedCheckpoint(db, ckpt)
	if err != nil {
		panic(err)
	}
}

func handleUnifiedOutputCreatedCheckpoint(db *store.BadgerStore, ckpt time.Time) error {
	key := "outputs-draining-checkpoint-by-created"
	val, err := db.ReadProperty([]byte(key))
	if err != nil {
		return err
	}

	if len(val) > 0 && binary.BigEndian.Uint64(val) > uint64(ckpt.UnixNano()) {
		return nil
	}
	return db.WriteProperty([]byte(key), tsToBytes(ckpt))
}

func handleUnifiedOutputUpdatedCheckpoint(db *store.BadgerStore, ckpt time.Time) error {
	key := "outputs-draining-checkpoint-by-updated"
	val, err := db.ReadProperty([]byte(key))
	if err != nil {
		return err
	}

	if len(val) > 0 && binary.BigEndian.Uint64(val) > uint64(ckpt.UnixNano()) {
		return nil
	}
	return db.WriteProperty([]byte(key), tsToBytes(ckpt))
}

func handleInvalidCollectibleTransactions(db *store.BadgerStore) {
	ctxs, err := db.ListCollectibleTransactions(mtg.TransactionStateInitial, 100)
	if err != nil {
		panic(err)
	}
	for _, tx := range ctxs {
		log.Println("initial", tx)
	}
	ctxs, err = db.ListCollectibleTransactions(mtg.TransactionStateSigning, 100)
	if err != nil {
		panic(err)
	}
	for _, tx := range ctxs {
		log.Println("signinig", tx)
	}
	ctxs, err = db.ListCollectibleTransactions(mtg.TransactionStateSigned, 100)
	if err != nil {
		panic(err)
	}
	for _, tx := range ctxs {
		log.Println("signed", tx)
	}
	ctxs, err = db.ListCollectibleTransactions(mtg.TransactionStateSnapshot, 100)
	if err != nil {
		panic(err)
	}
	for _, tx := range ctxs {
		log.Println("snapshot", tx)
	}

	err = removeInvalidCollectibleTransaction(db.Badger(), "8dc64911-4fc7-3d6d-9b9a-dc54b0330003")
	if err != nil {
		panic(err)
	}
	err = removeInvalidCollectibleTransaction(db.Badger(), "7ba7ac8a-9018-3792-ab00-7bde27b14c9b")
	if err != nil {
		panic(err)
	}
}

func removeInvalidCollectibleTransaction(db *badger.DB, traceId string) error {
	prefixCollectibleTransactionPayload := "COLLECTIBLES:TRANSACTION:PAYLOAD:"
	prefixCollectibleTransactionHash := "COLLECTIBLES:TRANSACTION:HASH:"

	return db.Update(func(txn *badger.Txn) error {
		pk := []byte(prefixCollectibleTransactionPayload + traceId)
		item, err := txn.Get(pk)
		if err == badger.ErrKeyNotFound {
			return nil
		} else if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		var tx mtg.CollectibleTransaction
		err = mtg.MsgpackUnmarshal(val, &tx)
		if err != nil {
			return err
		}

		if len(tx.Raw) > 0 {
			if !tx.Hash.HasValue() {
				panic(tx.TraceId)
			}
			hk := append([]byte(prefixCollectibleTransactionHash), tx.Hash[:]...)
			err = txn.Delete(hk)
			if err != nil {
				return err
			}
		}

		sk := buildCollectibleTransactionTimedKey(&tx)
		err = txn.Delete(sk)
		if err != nil {
			return err
		}
		return txn.Delete(pk)
	})
}

func buildCollectibleTransactionTimedKey(tx *mtg.CollectibleTransaction) []byte {
	buf := tsToBytes(tx.UpdatedAt)
	prefix := collectibleTransactionStatePrefix(tx.State)
	key := append([]byte(prefix), buf...)
	return append(key, []byte(tx.TraceId)...)
}

func collectibleTransactionStatePrefix(state int) string {
	prefix := "COLLECTIBLES:TRANSACTION:STATE:"
	switch state {
	case mtg.TransactionStateInitial:
		return prefix + "initiall"
	case mtg.TransactionStateSigning:
		return prefix + "signingg"
	case mtg.TransactionStateSigned:
		return prefix + "signeddd"
	case mtg.TransactionStateSnapshot:
		return prefix + "snapshot"
	}
	panic(state)
}

func tsToBytes(ts time.Time) []byte {
	buf := make([]byte, 8)
	d := ts.UnixNano()
	binary.BigEndian.PutUint64(buf, uint64(d))
	return buf
}
