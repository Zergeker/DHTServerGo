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

	port := viper.GetInt("PORT")

	sucFullIp := sucIp + ":" + strconv.Itoa(port)
	preFullIp := preIp + ":" + strconv.Itoa(port)

	n := Node{id, recs, 0, 0, sucFullIp, preFullIp, port, viper.GetInt("KEYSPACE_SIZE")}
	return &n
}

func changeNodeSuccessor(n *Node, sucIp string, sucId int) {
	n.SuccessorIp = sucIp
	n.SuccessorNodeId = sucId
}

func changeNodePredecessor(n *Node, preIp string, preId int) {
	n.PredecessorIp = preIp
	n.PredecessorNodeId = preId
}

func balanceNodeRecsSize(n *Node) {
	newRecsSize := n.SuccessorNodeId - n.NodeId
	if n.NodeId > n.SuccessorNodeId {
		newRecsSize = n.KeySpaceSize - n.NodeId + n.SuccessorNodeId
	}

	n.Records = make([][]*Record, newRecsSize)
}
