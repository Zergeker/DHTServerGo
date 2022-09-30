package dht

import (
	"strconv"

	"github.com/spf13/viper"
)

type Node struct {
	NodeId            int
	Records           [][]*Record
	SuccessorNodeId   int
	PredecessorNodeId int
	SuccessorIp       string
	PredecessorIp     string
	Port              int
	KeySpaceSize      int
}

func NewNode(id int, keySpaceCellSize int, sucIp string, preIp string, nodesCount int) *Node {
	viper.SetConfigFile("config.env")
	viper.ReadInConfig()

	recs := make([][]*Record, keySpaceCellSize)

	sucFullIp := sucIp + ":" + strconv.Itoa(viper.GetInt("PORT"))
	preFullIp := preIp + ":" + strconv.Itoa(viper.GetInt("PORT"))
	sucId := 0
	preId := id * keySpaceCellSize

	if id+keySpaceCellSize < nodesCount*keySpaceCellSize {
		sucId = id*keySpaceCellSize + keySpaceCellSize
	}

	if preId-keySpaceCellSize >= 0 {
		preId = preId - keySpaceCellSize
	} else {
		preId = (nodesCount - 1) * keySpaceCellSize
	}

	n := Node{id * keySpaceCellSize, recs, sucId, preId, sucFullIp, preFullIp, viper.GetInt("PORT"), viper.GetInt("KEYSPACE_SIZE")}
	return &n
}
