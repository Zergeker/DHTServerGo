package dht

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
)

var state = 1 //0 = crashed, 1 = active

type GetInternalKeyRequest struct {
	HashedKey   int
	OriginalKey string
}

type PutInternalKeyRequest struct {
	HashedKey   int
	OriginalKey string
	Value       string
}

type NodePlaceSearchResponse struct {
	SuccessorIp   string
	PredecessorIp string
}

type NodeInfoResponse struct {
	Node_hash string   `json:"node_hash"`
	Successor string   `json:"successor"`
	Others    []string `json:"others"`
}

type NodeChangeNeighbor struct {
	Hostname string
	HostId   int
}

func StartController(node *Node, port int) {

	//methods for client usage
	http.HandleFunc("/", storageKeyHandler(node))
	http.HandleFunc("/neighbors", getNeighborsHandler(node))
	http.HandleFunc("/sim-crash", crashSimHandler(node))
	http.HandleFunc("/sim-recover", crashSimRecoveryHandler(node))
	http.HandleFunc("/join", nodeJoinHandler(node))
	http.HandleFunc("/leave", nodeLeaveHandler(node))
	http.HandleFunc("/node-info", nodeInfoHandler(node))

	//methods for internal calls
	http.HandleFunc("/findKeyInOtherNode", findKeyInOtherNodeHandler(node))
	http.HandleFunc("/putKeyInOtherNode", putKeyInOtherNodeHandler(node))
	http.HandleFunc("/nodePlaceSearch", nodePlaceSearchHandler(node))
	http.HandleFunc("/changeSuccessor", nodeChangeSuccessorHandler(node))
	http.HandleFunc("/changePredecessor", nodeChangePredecessorHandler(node))
	http.HandleFunc("/checkPredecessorCrash", checkPredecessorCrashHandler(node))

	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("server closed")
	} else if err != nil {
		fmt.Println("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func storageKeyHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			switch r.Method {
			case "GET":
				OriginalKey := path.Base(r.URL.Path)
				inputKeyHashed := HashString(OriginalKey, n.KeySpaceSize)
				if n.NodeId == n.SuccessorNodeId {
					foundFlag := false
					if len(n.Records[inputKeyHashed]) > 1 {
						for i := 0; i < len(n.Records[inputKeyHashed]); i++ {
							if n.Records[inputKeyHashed][i].OrigKey == OriginalKey {
								fmt.Fprint(w, n.Records[inputKeyHashed][i].Value)
								foundFlag = true
							}
						}
						if foundFlag == false {
							http.Error(w, "No such key", http.StatusNotFound)
						}
					} else if len(n.Records[inputKeyHashed]) == 1 {
						if n.Records[inputKeyHashed][0].OrigKey == OriginalKey {
							fmt.Fprintf(w, n.Records[inputKeyHashed][0].Value)
						} else {
							http.Error(w, "No such key", http.StatusNotFound)
						}
					} else {
						http.Error(w, "No such key", http.StatusNotFound)
					}
					//TBD: optimize the condition below (or make it more readable)
				} else if (n.NodeId < n.SuccessorNodeId && inputKeyHashed >= n.NodeId && inputKeyHashed < n.SuccessorNodeId) || (n.NodeId > n.SuccessorNodeId && ((inputKeyHashed >= n.NodeId) || (inputKeyHashed < n.SuccessorNodeId && inputKeyHashed >= 0))) {
					keyIndex := 0
					if n.NodeId < n.SuccessorNodeId {
						keyIndex = inputKeyHashed - n.NodeId
					} else {
						if inputKeyHashed >= n.NodeId {
							keyIndex = inputKeyHashed - n.NodeId
						} else {
							keyIndex = n.KeySpaceSize - n.NodeId + inputKeyHashed
						}
					}
					foundFlag := false
					if len(n.Records[keyIndex]) > 1 {
						for i := 0; i < len(n.Records[keyIndex]); i++ {
							if n.Records[keyIndex][i].OrigKey == OriginalKey {
								fmt.Fprint(w, n.Records[keyIndex][i].Value)
								foundFlag = true
							}
						}
						if foundFlag == false {
							http.Error(w, "No such key", http.StatusNotFound)
						}
					} else if len(n.Records[keyIndex]) == 1 {
						if n.Records[keyIndex][0].OrigKey == OriginalKey {
							fmt.Fprintf(w, n.Records[keyIndex][0].Value)
						} else {
							http.Error(w, "No such key", http.StatusNotFound)
						}
					} else {
						http.Error(w, "No such key", http.StatusNotFound)
					}
				} else {
					requestBodyStruct := GetInternalKeyRequest{inputKeyHashed, OriginalKey}
					requestBodyJson, _ := json.Marshal(requestBodyStruct)
					resp, error := http.Post("http://"+n.SuccessorIp+"/findKeyInOtherNode", "application/json", bytes.NewBuffer(requestBodyJson))
					if error != nil {
						http.Error(w, "Key not found", http.StatusNotFound)
					} else {
						rBody, _ := ioutil.ReadAll(resp.Body)
						fmt.Fprintf(w, string(rBody))
					}
				}
			case "PUT":
				OriginalKey := path.Base(r.URL.Path)
				inputKeyHashed := HashString(OriginalKey, n.KeySpaceSize)
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				if n.NodeId == n.SuccessorNodeId {
					if len(n.Records[inputKeyHashed]) == 0 {
						n.Records[inputKeyHashed] = append(n.Records[inputKeyHashed], NewRecord(OriginalKey, string(body), n.KeySpaceSize))
						fmt.Fprint(w, "Success!\n")
					} else {
						foundFlag := false
						for i := 0; i < len(n.Records[inputKeyHashed]); i++ {
							if n.Records[inputKeyHashed][i].OrigKey == OriginalKey {
								foundFlag = true
							}
						}

						if foundFlag {
							fmt.Fprint(w, "Key is already put\n")
						} else {
							n.Records[inputKeyHashed] = append(n.Records[inputKeyHashed], NewRecord(OriginalKey, string(body), n.KeySpaceSize))
							fmt.Fprint(w, "Success!\n")
						}
					}
					//TBD: optimize the condition below (or make it more readable)
				} else if (n.NodeId < n.SuccessorNodeId && inputKeyHashed >= n.NodeId && inputKeyHashed < n.SuccessorNodeId) || (n.NodeId > n.SuccessorNodeId && ((inputKeyHashed >= n.NodeId) || (inputKeyHashed < n.SuccessorNodeId && inputKeyHashed >= 0))) {
					keyIndex := 0
					if n.NodeId < n.SuccessorNodeId {
						keyIndex = inputKeyHashed - n.NodeId
					} else {
						if inputKeyHashed >= n.NodeId {
							keyIndex = inputKeyHashed - n.NodeId
						} else {
							keyIndex = n.KeySpaceSize - n.NodeId + inputKeyHashed
						}
					}
					if len(n.Records[keyIndex]) == 0 {
						n.Records[keyIndex] = append(n.Records[keyIndex], NewRecord(OriginalKey, string(body), n.KeySpaceSize))
						fmt.Fprint(w, "Success!\n")
					} else {
						foundFlag := false
						for i := 0; i < len(n.Records[keyIndex]); i++ {
							if n.Records[keyIndex][i].OrigKey == OriginalKey {
								foundFlag = true
							}
						}

						if foundFlag {
							fmt.Fprint(w, "Key is already put\n")
						} else {
							n.Records[keyIndex] = append(n.Records[keyIndex], NewRecord(OriginalKey, string(body), n.KeySpaceSize))
							fmt.Fprint(w, "Success!\n")
						}
					}
				} else {
					requestBodyStruct := PutInternalKeyRequest{inputKeyHashed, OriginalKey, string(body)}
					requestBodyJson, _ := json.Marshal(requestBodyStruct)
					resp, error := http.Post("http://"+n.SuccessorIp+"/putKeyInOtherNode", "application/json", bytes.NewBuffer(requestBodyJson))
					if error != nil {
						http.Error(w, "Key was not put", http.StatusNotFound)
					} else {
						rBody, _ := ioutil.ReadAll(resp.Body)
						fmt.Fprintf(w, string(rBody))
					}
				}
				r.Body.Close()
			default:
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			}
		}
	}
}

func findKeyInOtherNodeHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			respBody, _ := ioutil.ReadAll(r.Body)
			var responseBodyStruct GetInternalKeyRequest
			json.Unmarshal(respBody, &responseBodyStruct)

			//TBD: optimize the condition below (or make it more readable)
			if (n.NodeId < n.SuccessorNodeId && responseBodyStruct.HashedKey >= n.NodeId && responseBodyStruct.HashedKey < n.SuccessorNodeId) || (n.NodeId > n.SuccessorNodeId && ((responseBodyStruct.HashedKey >= n.NodeId) || (responseBodyStruct.HashedKey < n.SuccessorNodeId && responseBodyStruct.HashedKey >= 0))) {
				keyIndex := 0
				if n.NodeId < n.SuccessorNodeId {
					keyIndex = responseBodyStruct.HashedKey - n.NodeId
				} else {
					if responseBodyStruct.HashedKey >= n.NodeId {
						keyIndex = responseBodyStruct.HashedKey - n.NodeId
					} else {
						keyIndex = n.KeySpaceSize - n.NodeId + responseBodyStruct.HashedKey
					}
				}
				if len(n.Records[keyIndex]) > 1 {
					foundFlag := false
					for i := 0; i < len(n.Records[keyIndex]); i++ {
						if n.Records[keyIndex][i].OrigKey == responseBodyStruct.OriginalKey {
							fmt.Fprint(w, n.Records[keyIndex][i].Value)
							foundFlag = true
						}
					}
					if foundFlag == false {
						http.Error(w, "No such key", http.StatusNotFound)
					}
				} else if len(n.Records[keyIndex]) == 1 {
					if n.Records[keyIndex][0].OrigKey == responseBodyStruct.OriginalKey {
						fmt.Fprintf(w, n.Records[keyIndex][0].Value)
					} else {
						http.Error(w, "No such key", http.StatusNotFound)
					}
				} else {
					http.Error(w, "No such key", http.StatusNotFound)
				}
			} else {
				requestBodyStruct := GetInternalKeyRequest{responseBodyStruct.HashedKey, responseBodyStruct.OriginalKey}
				requestBodyJson, _ := json.Marshal(requestBodyStruct)
				resp, error := http.Post("http://"+n.SuccessorIp+"/findKeyInOtherNode", "application/json", bytes.NewBuffer(requestBodyJson))
				if error != nil {
					http.Error(w, "Key not found", http.StatusNotFound)
				} else {
					rBody, _ := ioutil.ReadAll(resp.Body)
					fmt.Fprintf(w, string(rBody))
				}
			}
		}
	}
}

func putKeyInOtherNodeHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			respBody, _ := ioutil.ReadAll(r.Body)
			var responseBodyStruct PutInternalKeyRequest
			json.Unmarshal(respBody, &responseBodyStruct)

			//TBD: optimize the condition below (or make it more readable)
			if (n.NodeId < n.SuccessorNodeId && responseBodyStruct.HashedKey >= n.NodeId && responseBodyStruct.HashedKey < n.SuccessorNodeId) || (n.NodeId > n.SuccessorNodeId && ((responseBodyStruct.HashedKey >= n.NodeId) || (responseBodyStruct.HashedKey < n.SuccessorNodeId && responseBodyStruct.HashedKey >= 0))) {
				keyIndex := 0
				if n.NodeId < n.SuccessorNodeId {
					keyIndex = responseBodyStruct.HashedKey - n.NodeId
				} else {
					if responseBodyStruct.HashedKey >= n.NodeId {
						keyIndex = responseBodyStruct.HashedKey - n.NodeId
					} else {
						keyIndex = n.KeySpaceSize - n.NodeId + responseBodyStruct.HashedKey
					}
				}
				if len(n.Records[keyIndex]) == 0 {
					n.Records[keyIndex] = append(n.Records[keyIndex], NewRecord(responseBodyStruct.OriginalKey, responseBodyStruct.Value, n.KeySpaceSize))
					fmt.Fprint(w, "Success!\n")
				} else {
					foundFlag := false
					for i := 0; i < len(n.Records[keyIndex]); i++ {
						if n.Records[keyIndex][i].OrigKey == responseBodyStruct.OriginalKey {
							foundFlag = true
						}
					}

					if foundFlag {
						fmt.Fprint(w, "Key is already put\n")
					} else {
						n.Records[keyIndex] = append(n.Records[keyIndex], NewRecord(responseBodyStruct.OriginalKey, responseBodyStruct.Value, n.KeySpaceSize))
						fmt.Fprint(w, "Success!\n")
					}
				}
			} else {
				requestBodyStruct := PutInternalKeyRequest{responseBodyStruct.HashedKey, responseBodyStruct.OriginalKey, responseBodyStruct.Value}
				requestBodyJson, _ := json.Marshal(requestBodyStruct)
				resp, error := http.Post("http://"+n.SuccessorIp+"/putKeyInOtherNode", "application/json", bytes.NewBuffer(requestBodyJson))
				if error != nil {
					http.Error(w, "Key was not put", http.StatusNotFound)
				} else {
					rBody, _ := ioutil.ReadAll(resp.Body)
					fmt.Fprintf(w, string(rBody))
				}
			}
		}
	}
}

func getNeighborsHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			responseBody, _ := json.Marshal([]string{n.PredecessorIp, n.SuccessorIp})
			fmt.Fprintf(w, string(responseBody))
		}
	}
}

func crashSimHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			state = 0
			w.Write([]byte("Node crash simulated"))
		}
	}
}

func crashSimRecoveryHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 1 {
			w.Write([]byte("Node is not crashed"))
		} else {
			state = 1
			respPred, err := http.Get("http://" + n.PredecessorIp + "/node-info")
			//if predecessor does not respond, check successor
			if err != nil || respPred.StatusCode == 503 {
				respSucc, err := http.Get("http://" + n.SuccessorIp + "/node-info")
				//if successor does not respond, go to a singular node mode
				if err != nil || respSucc.StatusCode == 503 {
					ChangeNodeSuccessor(n, n.NodeAddress, n.NodeId)
					ChangeNodePredecessor(n, n.NodeAddress, n.NodeId)
					BalanceNodeRecsSize(n)
					w.Write([]byte("Node recovered, could not return to network"))
				} else {
					var respSuccStruct NodeInfoResponse
					respSuccJson, _ := ioutil.ReadAll(respSucc.Body)
					json.Unmarshal(respSuccJson, &respSuccStruct)

					if respSuccStruct.Others[0] == n.NodeAddress {
						w.Write([]byte("Node recovered, stayed in network during crash"))
					} else {
						reqJoin, err := http.Post("http://"+n.NodeAddress+"/join?nprime="+n.SuccessorIp, "text/plain", bytes.NewReader([]byte("")))
						if err != nil || reqJoin.StatusCode == 503 {
							ChangeNodeSuccessor(n, n.NodeAddress, n.NodeId)
							ChangeNodePredecessor(n, n.NodeAddress, n.NodeId)
							BalanceNodeRecsSize(n)
							w.Write([]byte("Node recovered, could not return to network"))
						} else {
							w.Write([]byte("Node recovered and returned to network"))
						}
					}
				}
			} else {
				var respPredStruct NodeInfoResponse
				respPredJson, _ := ioutil.ReadAll(respPred.Body)
				json.Unmarshal(respPredJson, &respPredStruct)

				if respPredStruct.Successor == n.NodeAddress {
					w.Write([]byte("Node recovered, stayed in network during crash"))
				} else {
					reqJoin, err := http.Post("http://"+n.NodeAddress+"/join?nprime="+n.PredecessorIp, "text/plain", bytes.NewReader([]byte("")))
					if err != nil || reqJoin.StatusCode == 503 {
						ChangeNodeSuccessor(n, n.NodeAddress, n.NodeId)
						ChangeNodePredecessor(n, n.NodeAddress, n.NodeId)
						BalanceNodeRecsSize(n)
						w.Write([]byte("Node recovered, could not return to network"))
					} else {
						w.Write([]byte("Node recovered and returned to network"))
					}
				}
			}
		}
	}
}

func nodePlaceSearchHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			reqBody, _ := ioutil.ReadAll(r.Body)
			searcherId, _ := strconv.Atoi(string(reqBody))
			if n.NodeId == n.SuccessorNodeId {
				if searcherId != n.NodeId {
					respBodyStruct := NodePlaceSearchResponse{n.SuccessorIp, n.PredecessorIp}
					respBodyJson, _ := json.Marshal(respBodyStruct)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write(respBodyJson)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("NodeId duplicates"))
				}
				//TBD: optimize the condition below (or make it more readable)
			} else if (n.NodeId < n.SuccessorNodeId && searcherId > n.NodeId && searcherId < n.SuccessorNodeId) || (n.NodeId > n.SuccessorNodeId && (searcherId > n.NodeId || searcherId < n.SuccessorNodeId)) {
				respBodyStruct := NodePlaceSearchResponse{n.SuccessorIp, n.NodeAddress}
				respBodyJson, _ := json.Marshal(respBodyStruct)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(respBodyJson)
			} else if searcherId == n.NodeId || searcherId == n.SuccessorNodeId {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("NodeId duplicates"))
			} else {
				resp, error := http.Post("http://"+n.SuccessorIp+"/nodePlaceSearch", "application/json", bytes.NewBuffer(reqBody))
				if error != nil {
					http.Error(w, "Node was not inserted", http.StatusInternalServerError)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				rBody, _ := ioutil.ReadAll(resp.Body)
				w.Write(rBody)
			}
		}
	}
}

func nodeJoinHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			nprime := r.URL.Query().Get("nprime")
			nodeIdRawBytes := []byte(strconv.Itoa(n.NodeId))

			//searching for new node's successor and predecessor
			resp, error := http.Post("http://"+nprime+"/nodePlaceSearch", "application/json", bytes.NewBuffer(nodeIdRawBytes))
			if error != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error occured"))
			}
			respBody, _ := ioutil.ReadAll(resp.Body)
			var respBodyStruct NodePlaceSearchResponse
			json.Unmarshal(respBody, &respBodyStruct)

			//setting new node's predecessor
			respPredecessor, _ := http.Get("http://" + respBodyStruct.PredecessorIp + "/node-info")
			respPredecessorBody, _ := ioutil.ReadAll(respPredecessor.Body)
			var respPredecessorStruct NodeInfoResponse
			json.Unmarshal(respPredecessorBody, &respPredecessorStruct)
			preId, _ := strconv.Atoi(respPredecessorStruct.Node_hash)
			ChangeNodePredecessor(n, respBodyStruct.PredecessorIp, preId)

			//setting new node's successor
			respSuccessor, _ := http.Get("http://" + respBodyStruct.SuccessorIp + "/node-info")
			respSuccessorBody, _ := ioutil.ReadAll(respSuccessor.Body)
			var respSuccessorStruct NodeInfoResponse
			json.Unmarshal(respSuccessorBody, &respSuccessorStruct)
			sucId, _ := strconv.Atoi(respSuccessorStruct.Node_hash)
			ChangeNodeSuccessor(n, respBodyStruct.SuccessorIp, sucId)

			//notifying node's successor and predecessor about their new neighbour
			changeNeighborStruct := NodeChangeNeighbor{n.NodeAddress, n.NodeId}
			reqBody, _ := json.Marshal(changeNeighborStruct)
			http.Post("http://"+n.SuccessorIp+"/changePredecessor", "application/json", bytes.NewBuffer(reqBody))
			http.Post("http://"+n.PredecessorIp+"/changeSuccessor", "application/json", bytes.NewBuffer(reqBody))

			BalanceNodeRecsSize(n)
		}
	}
}

func nodeInfoHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			respBodyStruct := NodeInfoResponse{strconv.Itoa(n.NodeId), n.SuccessorIp, []string{n.PredecessorIp}}
			respBodyJson, _ := json.Marshal(respBodyStruct)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(respBodyJson)
		}
	}
}

func nodeChangeSuccessorHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			var requestBodyStruct NodeChangeNeighbor
			json.Unmarshal(body, &requestBodyStruct)

			ChangeNodeSuccessor(n, requestBodyStruct.Hostname, requestBodyStruct.HostId)

			BalanceNodeRecsSize(n)
			w.Write([]byte("Success"))
		}
	}
}

func nodeChangePredecessorHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			var requestBodyStruct NodeChangeNeighbor
			json.Unmarshal(body, &requestBodyStruct)

			ChangeNodePredecessor(n, requestBodyStruct.Hostname, requestBodyStruct.HostId)

			BalanceNodeRecsSize(n)
			w.Write([]byte("Success"))
		}
	}
}

func nodeLeaveHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			changeSuccessorStruct := NodeChangeNeighbor{n.PredecessorIp, n.NodeId}
			reqBodySucc, _ := json.Marshal(changeSuccessorStruct)

			changePredecessorStruct := NodeChangeNeighbor{n.SuccessorIp, n.NodeId}
			reqBodyPred, _ := json.Marshal(changePredecessorStruct)

			//notification for node's neighbours about it's leaving
			http.Post("http://"+n.SuccessorIp+"/changePredecessor", "application/json", bytes.NewBuffer(reqBodySucc))
			http.Post("http://"+n.PredecessorIp+"/changeSuccessor", "application/json", bytes.NewBuffer(reqBodyPred))

			//change node's parameters for working as a singular node
			ChangeNodeSuccessor(n, n.NodeAddress, n.NodeId)
			ChangeNodePredecessor(n, n.NodeAddress, n.NodeId)
			BalanceNodeRecsSize(n)

			w.Write([]byte("Success"))
		}
	}
}

func checkPredecessorCrashHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == 0 {
			w.WriteHeader(503)
		} else {
			reqBodyJson, _ := ioutil.ReadAll(r.Body)
			//check if node's predecessor is alive
			resp, err := http.Get("http://" + n.PredecessorIp + "/node-info")

			//if it isn't, change the node's predecessor to the initial node which detected crashed node
			if err != nil || resp.StatusCode == 503 {
				var reqBodyStruct NodeChangeNeighbor
				json.Unmarshal(reqBodyJson, &reqBodyStruct)
				ChangeNodePredecessor(n, reqBodyStruct.Hostname, reqBodyStruct.HostId)

				respBodyStruct := NodeChangeNeighbor{n.NodeAddress, n.NodeId}
				respBodyJson, _ := json.Marshal(respBodyStruct)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(respBodyJson)
			} else {
				// if predecessor is alive, send let it check its predecessor
				resp, _ := http.Post("http://"+n.PredecessorIp+"/checkPredecessorCrash", "application/json", bytes.NewBuffer(reqBodyJson))
				respBodyJson, _ := ioutil.ReadAll(resp.Body)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				w.Write(respBodyJson)
			}
		}
	}
}
