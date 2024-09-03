package transactions

import (
	"context"
	"log"
)

func GetTransactionsObj(ctx context.Context) *Transactions {
	transactionFromContext := ctx.Value("transactions")
	if transactionFromContext != nil {
		if transactions, ok := transactionFromContext.(*Transactions); !ok {
			log.Fatalf("Expected *master.Transaction, got %T", transactionFromContext)
		} else {
			return transactions
		}
	}
	return nil
}
