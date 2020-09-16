package models

import (
	"context"
	"database/sql"
	"fmt"
	"multisig/session"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	PaymentStatusPending = "pending"
	PaymentStatusPaid    = "paid"
)

type Payment struct {
	PaymentID string         `db:"payment_id"`
	AssetID   string         `db:"asset_id"`
	Amount    string         `db:"amount"`
	Threshold int64          `db:"threshold"`
	Receivers pq.StringArray `db:"receivers"`
	Memo      string         `db:"memo"`
	Status    string         `db:"status"`
	CodeID    string         `db:"code_id"`
	CreatedAt time.Time      `db:"created_at"`
}

var paymentsColumnsFull = []string{"payment_id", "asset_id", "amount", "threshold", "receivers", "memo", "status", "code_id", "created_at"}

func CreatedPayment(ctx context.Context, payment *bot.Payment) (*Payment, error) {
	p := &Payment{
		PaymentID: payment.TraceId,
		AssetID:   payment.AssetId,
		Amount:    payment.Amount,
		Threshold: payment.Threshold,
		Receivers: payment.Receivers,
		Memo:      payment.Memo,
		Status:    payment.Status,
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

func FindPendingPayments(ctx context.Context, limit int64) ([]*Payment, error) {
	var payments []*Payment
	query := fmt.Sprintf("SELECT %s FROM payments WHERE status='pending' LIMIT $1", strings.Join(paymentsColumnsFull, ","))
	err := session.Database(ctx).SelectContext(ctx, &payments, query, limit)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return payments, nil
}

func LoopingPendingPayment(ctx context.Context) error {
	for {
		payments, err := FindPendingPayments(ctx, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindPendingPayments %#v", err)
			continue
		}
		for _, payment := range payments {
			botPayment, err := bot.ReadPaymentByCode(ctx, payment.CodeID)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("ReadPaymentByCode %#v", err)
				continue
			}
			if botPayment.Status == PaymentStatusPaid {
				query := "UPDATE payments SET status='paid' WHERE payment_id=$1"
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
	return nil
}
