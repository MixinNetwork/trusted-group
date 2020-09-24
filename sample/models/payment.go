package models

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"multisig/configs"
	"multisig/session"
	"net/http"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	PaymentStatePending = "pending"
	PaymentStatePaid    = "paid"
	PaymentStateRefund  = "refund"

	CNBAssetID = "965e5c6e-434c-3fa9-b780-c50f43cd955c"
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
	TransactionHash string         `db:"transaction_hash"`
	RawTransaction  string         `db:"raw_transaction"`
	UserID          string         `db:"user_id"`
	CreatedAt       time.Time      `db:"created_at"`
}

var paymentsColumnsFull = []string{"payment_id", "asset_id", "amount", "threshold", "receivers", "memo", "state", "code_id", "transaction_hash", "raw_transaction", "user_id", "created_at"}

func CreatedPayment(ctx context.Context, payment *bot.Payment, userID string) (*Payment, error) {
	p := &Payment{
		PaymentID: payment.TraceId,
		AssetID:   payment.AssetId,
		Amount:    payment.Amount,
		Threshold: payment.Threshold,
		Receivers: payment.Receivers,
		Memo:      payment.Memo,
		State:     payment.Status,
		CodeID:    payment.CodeId,
		UserID:    userID,
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

func FindPaymentsByState(ctx context.Context, state string, limit int64) ([]*Payment, error) {
	var payments []*Payment
	query := fmt.Sprintf("SELECT %s FROM payments WHERE state=$1 LIMIT $2", strings.Join(paymentsColumnsFull, ","))
	err := session.Database(ctx).SelectContext(ctx, &payments, query, state, limit)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
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
	network := NewMixinNetwork("http://35.234.74.25:8239")
	mixin := configs.AppConfig.Mixin
	for {
		payments, err := FindPaymentsByState(ctx, PaymentStatePaid, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindPaymentsByState %#v", err)
			continue
		}
		for _, payment := range payments {
			if payment.RawTransaction == "" {
				input, err := ReadMultisig(ctx, payment.Amount)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("ReadMultisig %#v", err)
					continue
				}
				key, err := bot.ReadGhostKeys(ctx, []string{payment.UserID}, 0, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("ReadGhostKeys %#v", err)
					continue
				}
				tx := &Transaction{
					Inputs:  []*Input{&Input{Hash: input.TransactionHash, Index: input.OutputIndex}},
					Outputs: []*Output{&Output{Mask: key.Mask, Keys: key.Keys, Amount: payment.Amount, Script: "fffe01"}},
					Asset:   "b9f49cf777dc4d03bc54cd1367eebca319f8603ea1ce18910d09e2c540c630d8",
				}
				data, err := json.Marshal(tx)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("json Marshal %#v", err)
					continue
				}
				raw, err := buildTransaction(data)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("buildTransaction %#v", err)
					continue
				}
				payment.RawTransaction = raw
				query := "UPDATE payments SET raw_transaction=$1 WHERE payment_id=$2"
				_, err = session.Database(ctx).ExecContext(ctx, query, payment.RawTransaction, payment.PaymentID)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("Updated payment %#v", err)
					continue
				}
			}
			request, err := bot.CreateMultisig(ctx, "sign", payment.RawTransaction, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("CreateMultisig %s %#v", mixin.AppID, err)
				continue
			}
			if request.State == "initial" {
				pin, err := bot.EncryptPIN(ctx, mixin.Pin, mixin.PinToken, mixin.SessionID, mixin.PrivateKey, uint64(time.Now().UnixNano()))
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("EncryptPIN %s %#v", mixin.AppID, err)
					continue
				}
				request, err = bot.SignMultisig(ctx, request.RequestId, pin, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("SignMultisig %s %#v", mixin.AppID, err)
					continue
				}
			}
			user := mixin.Users[0]
			request, err = bot.CreateMultisig(ctx, "sign", request.RawTransaction, user.UserID, user.SessionID, user.PrivateKey)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("CreateMultisig %s %#v", mixin.AppID, err)
				continue
			}
			if request.State == "initial" {
				pin, err := bot.EncryptPIN(ctx, user.Pin, user.PinToken, user.SessionID, user.PrivateKey, uint64(time.Now().UnixNano()))
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("EncryptPIN %s %#v", user.UserID, err)
					continue
				}
				request, err = bot.SignMultisig(ctx, request.RequestId, pin, user.UserID, user.SessionID, user.PrivateKey)
				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("SignMultisig %s %#v", user.UserID, err)
					continue
				}
			}
			hash, err := network.SendRawTransaction(request.RawTransaction)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("SendRawTransaction  %#v", err)
				continue
			}
			query := "UPDATE payments SET (state, transaction_hash)=('refund',$1) WHERE payment_id=$2"
			_, err = session.Database(ctx).ExecContext(ctx, query, hash, payment.PaymentID)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("Update Payment %#v", err)
				continue
			}
		}
		if len(payments) < 1 {
			time.Sleep(10 * time.Second)
		}
	}
	return nil
}

type MixinNetwork struct {
	httpClient *http.Client
	node       string
}

func NewMixinNetwork(node string) *MixinNetwork {
	return &MixinNetwork{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		node:       node,
	}
}

func (m *MixinNetwork) SendRawTransaction(raw string) (string, error) {
	body, err := m.callRPC("sendrawtransaction", []interface{}{raw})
	if err != nil {
		return "", err
	}
	var tx Transaction
	err = json.Unmarshal(body, &tx)
	return tx.Hash, err
}

func (m *MixinNetwork) callRPC(method string, params []interface{}) ([]byte, error) {
	body, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", m.node, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data  interface{} `json:"data"`
		Error interface{} `json:"error"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("ERROR %s", result.Error)
	}

	return json.Marshal(result.Data)
}
