package mtg

import (
	"context"
	"fmt"
	"testing"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestAccountCRUD(t *testing.T) {
	assert := assert.New(t)

	client, err := mixin.NewFromKeystore(&mixin.Keystore{
		AppID:             "c60832ee-7c7b-438e-ba1e-eb2776edc936",
		SessionID:         "3f675da8-cea3-4cd3-8e3d-3915dd1fa18e",
		SessionPrivateKey: "26f21ad334edd1645b3e991a3c35dc929cdb5e8edc96274b607ad5e010fa4cfa",
		ServerPublicKey:   "911d6752b3f113f1103de152d262c7d46d7d47367a0f0b1bd348bfa5ee2a606e",
	})
	assert.Nil(err)

	tx, err := client.SafeReadMultisigRequests(context.Background(), "941a857ecaa76e22260c93dd3a24721a6ffbfdb9ffd06649a31a9facb6603eaf")
	assert.Nil(err)

	ver, _ := decodeTransactionWithExtra(tx.RawTransaction)
	fmt.Println(ver.AggregatedSignature, ver.SignaturesMap)
}
