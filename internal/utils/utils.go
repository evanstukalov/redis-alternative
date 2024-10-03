package utils

import (
	"context"
	"log"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func GetBlockChObj(ctx context.Context) chan struct{} {
	blockChFromContext := ctx.Value("blockCh")
	if blockChFromContext != nil {
		if blockCh, ok := blockChFromContext.(chan struct{}); !ok {
			log.Fatalf("Expected chan struct{}, got %T", blockChFromContext)
		} else {
			return blockCh
		}
	}
	return nil
}

func GetClientsObj(ctx context.Context) *clients.Clients {
	clientsFromContext := ctx.Value("clients")
	if clientsFromContext != nil {
		if clients, ok := clientsFromContext.(*clients.Clients); !ok {
			log.Fatalf("Expected *master.Clients, got %T", clientsFromContext)
		} else {
			return clients
		}
	}
	return nil
}

func GetStoreObj(ctx context.Context) *store.Store {
	storeFromContext := ctx.Value("store")

	if storeFromContext != nil {
		if store, ok := storeFromContext.(*store.Store); !ok {
			log.Fatalf("Expected *store.Store, got %T", storeFromContext)
		} else {
			return store
		}
	}
	return nil
}
