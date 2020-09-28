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
	_ = flag.String("service", "message", "run a service")
	env := flag.String("env", "development", "set the environment")
	flag.Parse()

	config, err := configs.Init(*env)
	if err != nil {
		log.Panicln(err)
	}
	database := durable.OpenDatabaseClient(config)
	logger, err := durable.NewLoggerClient("", true)
	if err != nil {
		log.Panicln(err)
	}
	defer logger.Close()

	ctx := session.WithDatabase(context.Background(), database)
	ctx = session.WithLogger(ctx, durable.BuildLogger(logger, "multisig-message", nil))
	message := &services.MessageService{}
	go message.Run(ctx)
	go models.LoopingPendingPayments(ctx)
	models.LoopingPaidPayments(ctx)
}
