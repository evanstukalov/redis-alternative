package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/store"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
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

type xReadOptions struct {
	Block          int
	Streams        []string
	Ids            []string
	exclusiveIndex *uint
}

func parseXREADCommand(args []string) (xReadOptions, error) {
	var options xReadOptions

	for i := 0; i < len(args); {
		switch strings.ToUpper(args[i]) {
		case "BLOCK":
			i++ // move to value of block

			if i < len(args) {
				val, err := strconv.Atoi(args[i])
				if err != nil {
					return options, fmt.Errorf("invalid block value %s", args[i])
				}

				options.Block = val
			}

		case "STREAMS":
			i++ // move to value of streams
			for i < len(args) && !isID(args[i]) {
				options.Streams = append(options.Streams, args[i])
				i++
			}

			for i < len(args) {
				if isID(args[i]) {
					options.Ids = append(options.Ids, args[i])
					i++
				} else {
					break
				}
			}
		default:
			i++ // ignore unknown options

		}
	}

	return options, nil
}

func handleBlockOption(
	ctx context.Context,
	options xReadOptions,
	args []string,
) (uint, error) {
	var index uint
	time.Sleep(time.Duration(options.Block) * time.Millisecond)

	if args[2] == "0" {
		blockCh, ok := utils.GetFromCtx[chan uint](ctx, "blockCh")

		if !ok {
			logrus.Error("No store in context")
			return 0, errors.New("no store in context")
		}

		index = <-blockCh
	}
	return index, nil
}

func isID(value string) bool {
	return len(value) > 0 && (value[0] >= '0' && value[0] <= '9' || value == "$")
}

type streamPair struct {
	streamKey string
	id        string
	messages  []store.StreamMessage
}

func makeStreamPairs(options xReadOptions) []streamPair {
	streamPairs := make([]streamPair, 0, len(options.Streams))

	for i := range options.Streams {
		streamPairs = append(streamPairs, streamPair{
			streamKey: options.Streams[i],
			id:        options.Ids[i],
			messages:  make([]store.StreamMessage, 0, 8),
		})
	}
	return streamPairs
}

func fillStreamPairsWithMessages(
	conn io.Writer,
	options xReadOptions,
	storeObj *store.Store,
	streamPairs *[]streamPair,
) {
	for index, streamPair := range *streamPairs {

		messages, err := storeObj.GetStreamsExclusive(
			streamPair.streamKey,
			streamPair.id,
			options.exclusiveIndex,
		)
		if err != nil {
			logrus.Error(err)
			return
		}

		(*streamPairs)[index].messages = messages

		if options.Block > 0 && len(messages) == 0 {
			conn.Write([]byte("$-1\r\n"))
			return
		}
	}
}
