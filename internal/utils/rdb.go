package utils

import (
	"os"
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
	key := parseTable(c)
	str := key[4 : 4+key[3]]
	return string(str)
}
