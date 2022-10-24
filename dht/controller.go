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

	http.HandleFunc("/", storageKeyHandler(node))
	http.HandleFunc("/neighbors", getNeighborsHandler(node))
	http.HandleFunc("/findKeyInOtherNode", findKeyInOtherNodeHandler(node))
	http.HandleFunc("/putKeyInOtherNode", putKeyInOtherNodeHandler(node))
	http.HandleFunc("/sim-crash", crashSimHandler(node))
	http.HandleFunc("/sim-recover", crashSimRecoveryHandler(node))
	http.HandleFunc("/nodePlaceSearch", nodePlaceSearchHandler(node))
	http.HandleFunc("/join", nodeJoinHandler(node))
	http.HandleFunc("/node-info", nodeInfoHandler(node))
	http.HandleFunc("/changeSuccessor", nodeChangeSuccessorHandler(node))
	http.HandleFunc("/changePredecessor", nodeChangePredecessorHandler(node))
	http.HandleFunc("/leave", nodeLeaveHandler(node))

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
			w.Write([]byte("Node recovered"))
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

			resp, error := http.Post("http://"+nprime+"/nodePlaceSearch", "application/json", bytes.NewBuffer(nodeIdRawBytes))
			if error != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error occured"))
			}
			respBody, _ := ioutil.ReadAll(resp.Body)
			var respBodyStruct NodePlaceSearchResponse
			json.Unmarshal(respBody, &respBodyStruct)

			respPredecessor, _ := http.Get("http://" + respBodyStruct.PredecessorIp + "/node-info")
			respPredecessorBody, _ := ioutil.ReadAll(respPredecessor.Body)
			var respPredecessorStruct NodeInfoResponse
			json.Unmarshal(respPredecessorBody, &respPredecessorStruct)
			preId, _ := strconv.Atoi(respPredecessorStruct.Node_hash)
			changeNodePredecessor(n, respBodyStruct.PredecessorIp, preId)

			respSuccessor, _ := http.Get("http://" + respBodyStruct.PredecessorIp + "/node-info")
			respSuccessorBody, _ := ioutil.ReadAll(respSuccessor.Body)
			var respSuccessorStruct NodeInfoResponse
			json.Unmarshal(respSuccessorBody, &respSuccessorStruct)
			sucId, _ := strconv.Atoi(respSuccessorStruct.Node_hash)
			changeNodeSuccessor(n, respBodyStruct.SuccessorIp, sucId)

			changeNeighborStruct := NodeChangeNeighbor{n.NodeAddress, n.NodeId}
			reqBody, _ := json.Marshal(changeNeighborStruct)

			http.Post("http://"+n.SuccessorIp+"/changePredecessor", "application/json", bytes.NewBuffer(reqBody))
			http.Post("http://"+n.PredecessorIp+"/changeSuccessor", "application/json", bytes.NewBuffer(reqBody))

			balanceNodeRecsSize(n)
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

			changeNodeSuccessor(n, requestBodyStruct.Hostname, requestBodyStruct.HostId)

			balanceNodeRecsSize(n)
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

			changeNodePredecessor(n, requestBodyStruct.Hostname, requestBodyStruct.HostId)

			balanceNodeRecsSize(n)
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

			http.Post("http://"+n.SuccessorIp+"/changePredecessor", "application/json", bytes.NewBuffer(reqBodySucc))
			http.Post("http://"+n.PredecessorIp+"/changeSuccessor", "application/json", bytes.NewBuffer(reqBodyPred))

			changeNodeSuccessor(n, n.NodeAddress, n.NodeId)
			changeNodePredecessor(n, n.NodeAddress, n.NodeId)

			balanceNodeRecsSize(n)
		}
	}
}
