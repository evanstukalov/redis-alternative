package transactions

import (
	"context"
	"log"
)

func GetTransactionBufferObj(ctx context.Context) *TransactionBuffer {
	transactionBufferFromContext := ctx.Value("transactionBuffer")
	if transactionBufferFromContext != nil {
		if transactionBuffer, ok := transactionBufferFromContext.(*TransactionBuffer); !ok {
			log.Fatalf("Expected *master.TransactionBuffer, got %T", transactionBufferFromContext)
		} else {
			return transactionBuffer
		}
	}
	return nil
}
