package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
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

	go dht.StartController(n, viper.GetInt("PORT"))

	for true == true {
		time.Sleep(10 * time.Second)
		resp, err := http.Get("http://" + n.SuccessorIp + "/node-info")
		if err != nil || resp.StatusCode == 503 {
			respBodyStruct := dht.NodeChangeNeighbor{n.NodeAddress, n.NodeId}
			respBodyJson, _ := json.Marshal(respBodyStruct)
			predResp, err := http.Post("http://"+n.PredecessorIp+"/checkPredecessorCrash", "application/json", bytes.NewBuffer(respBodyJson))
			if err != nil {
				dht.ChangeNodeSuccessor(n, n.NodeAddress, n.NodeId)
				dht.ChangeNodePredecessor(n, n.NodeAddress, n.NodeId)
				dht.BalanceNodeRecsSize(n)
			} else {
				var predRespBodyStruct dht.NodeChangeNeighbor
				predRespBodyJson, _ := ioutil.ReadAll(predResp.Body)
				json.Unmarshal(predRespBodyJson, &predRespBodyStruct)

				dht.ChangeNodeSuccessor(n, predRespBodyStruct.Hostname, predRespBodyStruct.HostId)
				dht.BalanceNodeRecsSize(n)
			}
		}
	}
}
