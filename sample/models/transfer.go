package models

import (
	"context"
	"database/sql"
	"fmt"
	"multisig/configs"
	"multisig/session"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/jmoiron/sqlx"
)

const (
	TransferStatePaid    = "paid"
	TransferStateMainnet = "mainnet"
	TransferStateRefund  = "refund"
)

type Transfer struct {
	TransferID string    `db:"transfer_id"`
	UserID     string    `db:"user_id"`
	AssetID    string    `db:"asset_id"`
	Amount     string    `db:"amount"`
	Memo       string    `db:"memo"`
	TraceID    string    `db:"trace_id"`
	State      string    `db:"state"`
	CreatedAt  time.Time `db:"created_at"`
}

var transfersColumnsFull = []string{"transfer_id", "user_id", "asset_id", "amount", "memo", "trace_id", "state", "created_at"}

func CreateTransfer(ctx context.Context, sid, uid, aid, amount, memo, traceID, state string, create time.Time) (*Transfer, error) {
	if aid != CNBAssetID || !number.FromString(amount).Equal(number.FromString("1")) {
		return nil, nil
	}
	t := &Transfer{
		TransferID: sid,
		UserID:     uid,
		AssetID:    aid,
		Amount:     amount,
		Memo:       memo,
		TraceID:    traceID,
		State:      state,
		CreatedAt:  create,
	}

	err := session.Database(ctx).RunInTransaction(ctx, nil, func(tx *sqlx.Tx) error {
		old, err := findTransferByID(ctx, tx, t.TransferID)
		if err != nil || old != nil {
			return err
		}
		query := fmt.Sprintf("INSERT INTO transfers (%s) VALUES (:%s)", strings.Join(transfersColumnsFull, ", "), strings.Join(transfersColumnsFull, ", :"))
		_, err = tx.NamedExec(query, t)
		return err
	})
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return t, nil
}

func findTransferByID(ctx context.Context, tx *sqlx.Tx, transferID string) (*Transfer, error) {
	if id, _ := bot.UuidFromString(transferID); id.String() != transferID {
		return nil, nil
	}
	p := &Transfer{}
	err := tx.Get(p, fmt.Sprintf("SELECT %s FROM transfers WHERE transfer_id=$1", strings.Join(transfersColumnsFull, ",")), transferID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func FindTransfersByState(ctx context.Context, state string, limit int64) ([]*Transfer, error) {
	var transfers []*Transfer
	query := fmt.Sprintf("SELECT %s FROM transfers WHERE state=$1 LIMIT $2", strings.Join(transfersColumnsFull, ","))
	err := session.Database(ctx).SelectContext(ctx, &transfers, query, state, limit)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return transfers, nil
}

func LoopingPaidTransfers(ctx context.Context) error {
	mixin := configs.AppConfig.Mixin
	receivers := []string{mixin.AppID}
	for _, user := range mixin.Users {
		receivers = append(receivers, user.UserID)
	}
	om := struct {
		Receivers []string `json:"receivers"`
		Threshold int64    `json:"threshold"`
	}{
		receivers, 2,
	}
	pr := &bot.PaymentRequest{
		AssetId:          CNBAssetID,
		Amount:           "1",
		OpponentMultisig: om,
	}
	for {
		transfers, err := FindTransfersByState(ctx, TransferStatePaid, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindTransfersByState %#v", err)
			continue
		}
		for _, transfer := range transfers {
			pr.TraceId = transfer.TransferID
			botPayment, err := bot.CreatePaymentRequest(ctx, pr, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("CreatePaymentRequest %#v", err)
				continue
			}
			payment, err := CreatedPayment(ctx, botPayment, transfer.UserID)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("CreatedPayment %#v", err)
				continue
			}

			opponent := struct {
				Receivers []string
				Threshold int64
			}{Receivers: om.Receivers, Threshold: om.Threshold}
			in := &bot.TransferInput{
				AssetId:          payment.AssetID,
				Amount:           number.FromString(payment.Amount),
				TraceId:          payment.PaymentID,
				Memo:             payment.Memo,
				OpponentMultisig: opponent,
			}
			_, err = bot.CreateMultisigTransaction(ctx, in, mixin.AppID, mixin.SessionID, mixin.PrivateKey, mixin.Pin, mixin.PinToken)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("CreatedPayment %#v", err)
				continue
			}
			query := "UPDATE transfers SET state='mainnet' WHERE transfer_id=$1"
			_, err = session.Database(ctx).ExecContext(ctx, query, transfer.TransferID)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("Updated transfers %#v", err)
				continue
			}
		}

		if len(transfers) < 10 {
			time.Sleep(10 * time.Second)
			continue
		}
	}
}
