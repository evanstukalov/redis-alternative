package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	opCodeModuleAux    byte = 247 /* Module auxiliary data. */
	opCodeIdle         byte = 248 /* LRU idle time. */
	opCodeFreq         byte = 249 /* LFU frequency. */
	opCodeAux          byte = 250 /* RDB aux field. */
	opCodeResizeDB     byte = 251 /* Hash table resize hint. */
	opCodeExpireTimeMs byte = 252 /* Expire time in milliseconds. */
	opCodeExpireTime   byte = 253 /* Old expire time in seconds. */
	opCodeSelectDB     byte = 254 /* DB number of the following keys. */
	opCodeEOF          byte = 255
)

func sliceIndex(data []byte, sep byte) int {
	for i, b := range data {
		if b == sep {
			return i
		}
	}
	return -1
}

func parseTable(bytes []byte) []byte {
	start := sliceIndex(bytes, opCodeResizeDB)
	end := sliceIndex(bytes, opCodeEOF)
	return bytes[start+1 : end]
}

func ReadFile(path string) string {
	c, _ := os.ReadFile(path)
	logrus.Debug(c)
	logrus.Debug(string(c))
	key := parseTable(c)
	str := key[4 : 4+key[3]]
	return string(str)
}

func LoadRDB(ctx context.Context, dir string, dbFileName string) {
	path := fmt.Sprintf("%s/%s", dir, dbFileName)
	content, _ := os.ReadFile(path)
	if len(content) == 0 {
		logrus.Info("RDB file is empty")
		return
	}

	line := parseTable(content)
	key := string(line[4 : 4+line[3]])
	value := string(line[5+line[3]:])

	logrus.WithFields(logrus.Fields{
		"key":   key,
		"value": value,
	}).Debug("LoadRDB")

	storeObj := GetStoreObj(ctx)
	storeObj.Set(key, value, nil)
}
