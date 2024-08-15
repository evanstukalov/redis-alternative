package utils

import (
	"context"
	"log"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
)

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
