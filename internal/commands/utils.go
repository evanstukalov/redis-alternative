package commands

import (
	"bytes"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func arrayResp(elements int) string {
	return fmt.Sprintf("*%d\r\n", elements)
}

func stringResp(value string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
}

func writeStreamMessage(bb *bytes.Buffer, streamKey string, streamMessages []store.StreamMessage) {
	bb.WriteString(arrayResp(2))
	bb.WriteString(stringResp(streamKey))

	bb.WriteString(arrayResp(len(streamMessages)))

	for _, msg := range streamMessages {
		bb.WriteString(arrayResp(2))
		bb.WriteString(stringResp(msg.ID))

		fields := msg.Fields
		bb.WriteString(arrayResp(len(fields) * 2))

		for k, v := range fields {
			bb.WriteString(stringResp(k))
			bb.WriteString(stringResp(v))
		}
	}
}
