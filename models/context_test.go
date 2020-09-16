package models

import (
	"context"
	"io/ioutil"
	"log"
	"multisig/configs"
	"multisig/durable"
	"multisig/session"
)

const (
	testEnvironment = "test"
	testDatabase    = "multisig_test"
)

const (
	dropPaymentsDDL = `DROP TABLE IF EXISTS payments;`
	dropUsersDDL    = `DROP TABLE IF EXISTS users;`
)

func teardownTestContext(ctx context.Context) {
	tables := []string{
		dropPaymentsDDL,
		dropUsersDDL,
	}
	for _, t := range tables {
		session.Database(ctx).MustExec(t)
	}
}

func setupTestContext() context.Context {
	config, err := configs.Init(testEnvironment)
	if err != nil {
		log.Panicln(err)
	}
	if config.Environment != testEnvironment || config.Database.Name != testDatabase {
		log.Panicln(config.Environment, config.Database.Name)
	}

	db := durable.OpenDatabaseClient(config)
	data, err := ioutil.ReadFile("./schema.sql")
	if err != nil {
		log.Panicln(err)
	}
	db.MustExec(string(data))
	return session.WithDatabase(context.Background(), db)
}
