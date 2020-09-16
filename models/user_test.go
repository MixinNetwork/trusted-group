package models

import (
	"context"
	"multisig/session"
	"testing"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestUserCRUD(t *testing.T) {
	ctx := setupTestContext()
	defer teardownTestContext(ctx)
	assert := assert.New(t)

	uid := bot.UuidNewV4().String()
	user, err := createUser(ctx, uid, "access_token", "Li Yuqing", "", "123456")
	assert.Nil(err)
	assert.NotNil(user)
	assert.Equal(uid, user.UserID)
	assert.Equal("Li Yuqing", user.FullName)
	user, err = createUser(ctx, uid, "access_new_token", "Li", "", "123456")
	assert.Nil(err)
	assert.NotNil(user)
	assert.Equal(uid, user.UserID)
	assert.Equal("Li", user.FullName)
	user, err = findTestUser(ctx, user.UserID)
	assert.Nil(err)
	assert.NotNil(user)
	assert.Equal(uid, user.UserID)
	assert.Equal("Li", user.FullName)
	assert.Equal("access_new_token", user.AccessToken)
	user, err = AuthenticateUser(ctx, user.AuthenticationToken)
	assert.Equal(uid, user.UserID)
	assert.Equal("Li", user.FullName)
}

func findTestUser(ctx context.Context, id string) (*User, error) {
	var user *User
	err := session.Database(ctx).RunInTransaction(ctx, nil, func(tx *sqlx.Tx) error {
		var err error
		user, err = findUserByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return user, nil
}
