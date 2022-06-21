package main

import (
	"context"

	"github.com/fox-one/mixin-sdk-go"
)

type User struct {
	PublicKey string
}

func (p *Proxy) createUser(ctx context.Context, addr string) (*User, error) {
	p.CreateUser(ctx, nil, addr)
	panic(0)
}

func (u *User) handle(s *mixin.Snapshot, act *Action) error {
	panic(0)
}

func (u *User) pass(s *mixin.Snapshot) error {
	// always do both bind and pass method
	panic(0)
}
