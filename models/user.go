package models

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"multisig/session"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/jmoiron/sqlx"
)

// useless just for test

type User struct {
	UserID              string    `db:"user_id"`
	IdentityNumber      string    `db:"identity_number"`
	FullName            string    `db:"full_name"`
	AvatarURL           string    `db:"avatar_url"`
	AccessToken         string    `db:"access_token"`
	AuthenticationToken string    `db:"authentication_token"`
	CreatedAt           time.Time `db:"created_at"`
}

var usersColumnFull = []string{"user_id", "identity_number", "full_name", "avatar_url", "access_token", "authentication_token", "created_at"}

func createUser(ctx context.Context, userID, accessToken, fullName, avatarURL, identityNumber string) (*User, error) {
	if id, _ := bot.UuidFromString(userID); id.String() != userID {
		return nil, nil
	}
	h := md5.New()
	io.WriteString(h, accessToken)
	user := &User{
		UserID:              userID,
		IdentityNumber:      identityNumber,
		FullName:            fullName,
		AvatarURL:           avatarURL,
		AccessToken:         accessToken,
		AuthenticationToken: fmt.Sprintf("%x", h.Sum(nil)),
		CreatedAt:           time.Now(),
	}

	err := session.Database(ctx).RunInTransaction(ctx, nil, func(tx *sqlx.Tx) error {
		old, err := findUserByID(ctx, tx, userID)
		if err != nil {
			return nil
		}

		var query string
		if old == nil {
			query = fmt.Sprintf("INSERT INTO users (%s) VALUES (:%s)", strings.Join(usersColumnFull, ", "), strings.Join(usersColumnFull, ", :"))
		} else {
			fields := []string{"authentication_token", "access_token", "full_name", "avatar_url"}
			query = fmt.Sprintf("UPDATE users SET (%s)=(:%s) WHERE user_id=:user_id", strings.Join(fields, ", "), strings.Join(fields, ", :"))
		}
		_, err = tx.NamedExec(query, user)
		return err
	})
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return user, nil
}

func AuthenticateUser(ctx context.Context, token string) (*User, error) {
	var user *User
	err := session.Database(ctx).RunInTransaction(ctx, nil, func(tx *sqlx.Tx) error {
		var err error
		user, err = findUserIdByToken(ctx, tx, token)
		return err
	})
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return user, nil
}

func findUserByID(ctx context.Context, tx *sqlx.Tx, id string) (*User, error) {
	if _, err := bot.UuidFromString(id); err != nil {
		return nil, nil
	}
	u := &User{}
	err := tx.Get(u, fmt.Sprintf("SELECT %s FROM users WHERE user_id=$1", strings.Join(usersColumnFull, ",")), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func findUserIdByToken(ctx context.Context, tx *sqlx.Tx, token string) (*User, error) {
	u := &User{}
	err := tx.Get(u, fmt.Sprintf("SELECT %s FROM users WHERE authentication_token=$1", strings.Join(usersColumnFull, ",")), token)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}
