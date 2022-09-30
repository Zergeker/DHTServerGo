package dht

import (
	"crypto/sha1"
	"encoding/binary"
)

func HashString(input string, keyspace int) int {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	resKeyBytes := hasher.Sum(nil) //hasher.Sum(nil)
	resKeyInt := binary.BigEndian.Uint32(resKeyBytes[0:4]) % uint32(keyspace)
	return int(resKeyInt)
}
