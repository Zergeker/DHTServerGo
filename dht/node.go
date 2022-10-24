package dht

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Node struct {
	NodeId            int
	NodeAddress       string
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
	address, _ := os.Hostname()

	address = address + ":" + strconv.Itoa(port)

	sucFullIp := sucIp + ":" + strconv.Itoa(port)
	preFullIp := preIp + ":" + strconv.Itoa(port)

	n := Node{id, address, recs, id, id, sucFullIp, preFullIp, port, viper.GetInt("KEYSPACE_SIZE")}
	return &n
}

func ChangeNodeSuccessor(n *Node, sucIp string, sucId int) {
	n.SuccessorIp = sucIp
	n.SuccessorNodeId = sucId
}

func ChangeNodePredecessor(n *Node, preIp string, preId int) {
	n.PredecessorIp = preIp
	n.PredecessorNodeId = preId
}

func BalanceNodeRecsSize(n *Node) {
	newRecsSize := n.SuccessorNodeId - n.NodeId
	if n.SuccessorNodeId == n.NodeId {
		newRecsSize = n.KeySpaceSize
	} else if n.NodeId > n.SuccessorNodeId {
		newRecsSize = n.KeySpaceSize - n.NodeId + n.SuccessorNodeId
	}

	n.Records = make([][]*Record, newRecsSize)
}
