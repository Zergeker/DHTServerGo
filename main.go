package main

import (
	"os"
	"strconv"

	"example.com/DHTServer/dht"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigFile("config.env")
	viper.ReadInConfig()

	predecessorHost := os.Args[1]
	successorHost := os.Args[2]
	nodeId, _ := strconv.Atoi(os.Args[3])

	keySpaceCellSize := viper.GetInt("KEYSPACE_SIZE") / viper.GetInt("NODES_COUNT")

	n := dht.NewNode(nodeId, keySpaceCellSize, successorHost, predecessorHost, viper.GetInt("NODES_COUNT"))

	dht.StartController(n)
}
