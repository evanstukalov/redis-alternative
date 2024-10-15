package utils

import (
	"context"
	"log"
)

func GetFromCtx[T any](ctx context.Context, key string) (T, bool) {
	if ctx.Value(key) != nil {
		if val, ok := ctx.Value(key).(T); !ok {
			log.Fatalf("Expected %T, got %T", *new(T), val)
		} else {
			return val, true
		}
	}

	var zeroValue T
	return zeroValue, false
}
