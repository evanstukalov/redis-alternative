package config

import (
	"context"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

type Services struct {
	StoreObj            *store.Store
	ClientsObj          *clients.Clients
	TransactionObj      commands.ITransactions
	BlockCh             chan struct{}
	ExpiredCollectorObj *store.ExpiredCollector
}

func InitializeServices() *Services {
	storeObj := store.NewStore()
	transactionObj := commands.NewTransaction()
	expiredCollectorObj := store.NewExpiredCollector(storeObj)
	clientsObj := clients.NewClients()
	blockCh := make(chan struct{})

	return &Services{
		StoreObj:            storeObj,
		ClientsObj:          clientsObj,
		TransactionObj:      transactionObj,
		BlockCh:             blockCh,
		ExpiredCollectorObj: expiredCollectorObj,
	}
}

func CreateContext(services *Services) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "store", services.StoreObj)
	ctx = context.WithValue(ctx, "clients", services.ClientsObj)
	ctx = context.WithValue(ctx, "transactions", services.TransactionObj)
	ctx = context.WithValue(ctx, "blockCh", services.BlockCh)

	return ctx
}
