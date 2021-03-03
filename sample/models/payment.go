package models

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"multisig/configs"
	"multisig/session"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	PaymentStatePending = "pending"
	PaymentStatePaid    = "paid"
	PaymentStateRefund  = "refund"

	CNBAssetID      = "965e5c6e-434c-3fa9-b780-c50f43cd955c"
	CNBMixinAssetID = "b9f49cf777dc4d03bc54cd1367eebca319f8603ea1ce18910d09e2c540c630d8"
)

type Payment struct {
	PaymentID       string         `db:"payment_id"`
	AssetID         string         `db:"asset_id"`
	Amount          string         `db:"amount"`
	Threshold       int64          `db:"threshold"`
	Receivers       pq.StringArray `db:"receivers"`
	Memo            string         `db:"memo"`
	State           string         `db:"state"`
	CodeID          string         `db:"code_id"`
	TransactionHash sql.NullString `db:"transaction_hash"`
	RawTransaction  sql.NullString `db:"raw_transaction"`
	CreatedAt       time.Time      `db:"created_at"`
}

var paymentsColumnsFull = []string{"payment_id", "asset_id", "amount", "threshold", "receivers", "memo", "state", "code_id", "transaction_hash", "raw_transaction", "created_at"}

func CreatedPayment(ctx context.Context, payment *bot.Payment) (*Payment, error) {
	p := &Payment{
		PaymentID: payment.TraceId,
		AssetID:   payment.AssetId,
		Amount:    payment.Amount,
		Threshold: payment.Threshold,
		Receivers: payment.Receivers,
		Memo:      payment.Memo,
		State:     payment.Status,
		CodeID:    payment.CodeId,
		CreatedAt: payment.CreatedAt,
	}

	err := session.Database(ctx).RunInTransaction(ctx, nil, func(tx *sqlx.Tx) error {
		old, err := findPaymentByID(ctx, tx, p.PaymentID)
		if err != nil || old != nil {
			return err
		}

		query := fmt.Sprintf("INSERT INTO payments (%s) VALUES (:%s)", strings.Join(paymentsColumnsFull, ", "), strings.Join(paymentsColumnsFull, ", :"))
		_, err = tx.NamedExec(query, p)
		return err
	})
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return p, nil
}

func findPaymentByID(ctx context.Context, tx *sqlx.Tx, paymentID string) (*Payment, error) {
	if id, _ := bot.UuidFromString(paymentID); id.String() != paymentID {
		return nil, nil
	}
	p := &Payment{}
	err := tx.Get(p, fmt.Sprintf("SELECT %s FROM payments WHERE payment_id=$1", strings.Join(paymentsColumnsFull, ",")), paymentID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func FindPaymentByMemo(ctx context.Context, memo string) (*Payment, error) {
	if id, _ := bot.UuidFromString(memo); id.String() != memo {
		return nil, nil
	}
	p := &Payment{}
	err := session.Database(ctx).GetContext(ctx, p, fmt.Sprintf("SELECT %s FROM payments WHERE memo=$1 AND state='pending'", strings.Join(paymentsColumnsFull, ",")), memo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func FindPaymentsByState(ctx context.Context, state string, limit int64) ([]*Payment, error) {
	var payments []*Payment
	query := fmt.Sprintf("SELECT %s FROM payments WHERE state=$1 LIMIT $2", strings.Join(paymentsColumnsFull, ","))
	err := session.Database(ctx).SelectContext(ctx, &payments, query, state, limit)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return payments, nil
}

func LoopingPendingPayments(ctx context.Context) error {
	for {
		payments, err := FindPaymentsByState(ctx, PaymentStatePending, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindPaymentsByState %#v", err)
			continue
		}
		for _, payment := range payments {
			botPayment, err := bot.ReadPaymentByCode(ctx, payment.CodeID)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("ReadPaymentByCode %#v", err)
				continue
			}
			if botPayment.Status == PaymentStatePaid {
				query := "UPDATE payments SET state='paid' WHERE payment_id=$1"
				_, err = session.Database(ctx).ExecContext(ctx, query, payment.PaymentID)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("Updated payment %#v", err)
					continue
				}
			}
		}
		if len(payments) < 1 {
			time.Sleep(10 * time.Second)
		}
	}
}

func LoopingPaidPayments(ctx context.Context) error {
	network := NewMixinNetwork("http://mixin-node0.exinpool.com:8239")
	for {
		payments, err := FindPaymentsByState(ctx, PaymentStatePaid, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindPaymentsByState %#v", err)
			continue
		}
		for _, payment := range payments {
			err = payment.refund(ctx, network)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("refund %#v", err)
				continue
			}
		}
		if len(payments) < 1 {
			time.Sleep(10 * time.Second)
		}
	}
}

func (payment *Payment) refund(ctx context.Context, network *MixinNetwork) error {
	mixin := configs.AppConfig.Mixin
	input, err := ReadMultisig(ctx, payment.Amount, payment.Memo)
	if err != nil || input == nil {
		return err
	}
	if payment.RawTransaction.String != input.SignedTx {
		payment.RawTransaction = sql.NullString{String: input.SignedTx, Valid: true}
		query := "UPDATE payments SET raw_transaction=$1 WHERE payment_id=$2"
		_, err = session.Database(ctx).ExecContext(ctx, query, payment.RawTransaction, payment.PaymentID)
		if err != nil {
			return err
		}
	}
	if !payment.RawTransaction.Valid {
		var raw = ""
		if input.State == "signed" {
			raw = input.SignedTx
		}
		if raw == "" {
			key, err := bot.ReadGhostKeys(ctx, []string{payment.Memo}, 0, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
			if err != nil {
				return err
			}
			tx := &Transaction{
				Inputs:  []*Input{&Input{Hash: input.TransactionHash, Index: input.OutputIndex}},
				Outputs: []*Output{&Output{Mask: key.Mask, Keys: key.Keys, Amount: payment.Amount, Script: "fffe01"}},
				Asset:   CNBMixinAssetID,
			}
			data, err := json.Marshal(tx)
			if err != nil {
				return err
			}
			raw, err = buildTransaction(data)
			if err != nil {
				return err
			}
		}
		payment.RawTransaction = sql.NullString{String: raw, Valid: true}
		query := "UPDATE payments SET raw_transaction=$1 WHERE payment_id=$2"
		_, err = session.Database(ctx).ExecContext(ctx, query, payment.RawTransaction, payment.PaymentID)
		if err != nil {
			return err
		}
	}
	request, err := bot.CreateMultisig(ctx, "sign", payment.RawTransaction.String, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return err
	}
	if request.State == "initial" {
		pin, err := bot.EncryptPIN(ctx, mixin.Pin, mixin.PinToken, mixin.SessionID, mixin.PrivateKey, uint64(time.Now().UnixNano()))
		if err != nil {
			return err
		}
		request, err = bot.SignMultisig(ctx, request.RequestId, pin, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
		if err != nil {
			return err
		}
	}
	if payment.RawTransaction.String != request.RawTransaction {
		payment.TransactionHash = sql.NullString{String: request.TransactionHash, Valid: true}
		payment.RawTransaction = sql.NullString{String: request.RawTransaction, Valid: true}
		query := "UPDATE payments SET (transaction_hash,raw_transaction)=($1,$2) WHERE payment_id=$3"
		_, err = session.Database(ctx).ExecContext(ctx, query, payment.TransactionHash, payment.RawTransaction, payment.PaymentID)
		if err != nil {
			return err
		}
	}

	data, err := hex.DecodeString(payment.RawTransaction.String)
	if err != nil {
		return err
	}
	var stx common.SignedTransaction
	err = common.MsgpackUnmarshal(data, &stx)
	if err != nil {
		return err
	}
	if len(stx.Signatures) > 0 && len(stx.Signatures[0]) < int(payment.Threshold) {
		return nil
	}
	tx, err := network.GetTransaction(payment.TransactionHash.String)
	if tx == nil {
		_, err := network.SendRawTransaction(payment.RawTransaction.String)
		if err != nil {
			return err
		}
	}
	query := "UPDATE payments SET state='refund' WHERE payment_id=$1"
	_, err = session.Database(ctx).ExecContext(ctx, query, payment.PaymentID)
	return err
}

func HandleMessage(ctx context.Context, userID string) (string, error) {
	payment, err := FindPaymentByMemo(ctx, userID)
	if err != nil {
		return "", err
	}
	if payment != nil {
		return payment.CodeID, nil
	}
	mixin := configs.AppConfig.Mixin
	receivers := mixin.Receivers
	receivers = append(receivers, mixin.AppID)
	rand.Seed(time.Now().UnixNano())
	amount := number.FromString(fmt.Sprint(rand.Intn(10000))).Div(number.FromString("10000")).Persist()
	om := struct {
		Receivers []string `json:"receivers"`
		Threshold int64    `json:"threshold"`
	}{
		receivers, 2,
	}
	pr := &bot.PaymentRequest{
		AssetId:          CNBAssetID,
		Amount:           amount,
		TraceId:          bot.UuidNewV4().String(),
		OpponentMultisig: om,
		Memo:             userID,
	}
	botPayment, err := bot.CreatePaymentRequest(ctx, pr, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return "", err
	}
	payment, err = CreatedPayment(ctx, botPayment)
	if err != nil {
		return "", err
	}
	return payment.CodeID, nil
}
