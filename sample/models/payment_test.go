package models

import (
	"testing"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestPaymentCRUD(t *testing.T) {
	ctx := setupTestContext()
	defer teardownTestContext(ctx)
	assert := assert.New(t)

	payments, err := FindPaymentsByState(ctx, PaymentStatePending, 100)
	assert.Nil(err)
	assert.Len(payments, 0)
	p := &bot.Payment{
		TraceId:   bot.UuidNewV4().String(),
		AssetId:   bot.UuidNewV4().String(),
		Amount:    "1",
		Threshold: 2,
		Receivers: pq.StringArray{bot.UuidNewV4().String(), bot.UuidNewV4().String()},
		Memo:      "",
		Status:    PaymentStatePending,
		CodeId:    bot.UuidNewV4().String(),
		CreatedAt: time.Now(),
	}
	payment, err := CreatedPayment(ctx, p, bot.UuidNewV4().String())
	assert.Nil(err)
	assert.NotNil(payment)
	payments, err = FindPaymentsByState(ctx, PaymentStatePending, 100)
	assert.Nil(err)
	assert.Len(payments, 1)
}
