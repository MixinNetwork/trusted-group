package main

import (
	"context"
	"flag"
	"log"
	"multisig/configs"
	"multisig/durable"
	"multisig/models"
	"multisig/services"
	"multisig/session"
)

func main() {
	env := flag.String("env", "development", "set the environment")
	flag.Parse()

	config, err := configs.Init(*env)
	if err != nil {
		log.Panicln(err)
	}
	logger, err := durable.NewLoggerClient("", true)
	if err != nil {
		log.Panicln(err)
	}
	defer logger.Close()

	ctx := session.WithLogger(context.Background(), durable.BuildLogger(logger, "multisig-message", nil))
	if config.Mixin.Master {
		database := durable.OpenDatabaseClient(config)
		ctx = session.WithDatabase(ctx, database)

		message := &services.MessageService{}
		go message.Run(ctx)

		go models.LoopingPendingPayments(ctx)
		go models.LoopingPaidPayments(ctx)
	}
	models.LoopingSignMultisig(ctx)
}
