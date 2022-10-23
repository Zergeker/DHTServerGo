package main

import (
	"os"
	"time"

	"example.com/DHTServer/dht"
	"github.com/spf13/viper"
)

func main() {
	time.AfterFunc(40*time.Minute, func() { os.Exit(0) })
	viper.SetConfigFile("config.env")
	viper.ReadInConfig()

	initialHost, _ := os.Hostname()

	keySpaceCellSize := viper.GetInt("KEYSPACE_SIZE")

	nodeId := dht.HashString(initialHost, keySpaceCellSize)

	n := dht.NewNode(nodeId, keySpaceCellSize, initialHost, initialHost, viper.GetInt("NODES_COUNT"))

	dht.StartController(n, viper.GetInt("PORT"))
}
