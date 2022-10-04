package dht

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/spf13/viper"
)

type GetInternalKeyRequest struct {
	HashedKey   int
	OriginalKey string
}

type PutInternalKeyRequest struct {
	HashedKey   int
	OriginalKey string
	Value       string
}

func StartController(node *Node) {
	viper.SetConfigFile("config.env")
	viper.ReadInConfig()

	http.HandleFunc("/", storageKeyHandler(node, viper.GetInt("NODES_COUNT")))
	http.HandleFunc("/neighbors", getNeighborsHandler(node))
	http.HandleFunc("/findKeyInOtherNode", findKeyInOtherNodeHandler(node))
	http.HandleFunc("/putKeyInOtherNode", putKeyInOtherNodeHandler(node))

	err := http.ListenAndServe(":"+strconv.Itoa(viper.GetInt("PORT")), nil)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("server closed")
	} else if err != nil {
		fmt.Println("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func getRootHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("got / request\n")
		io.WriteString(w, path.Base(r.URL.Path))
	}
}

func storageKeyHandler(n *Node, nodesCount int) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			OriginalKey := path.Base(r.URL.Path)
			inputKeyHashed := HashString(OriginalKey, n.KeySpaceSize)
			if (inputKeyHashed >= n.NodeId && inputKeyHashed < n.SuccessorNodeId) || nodesCount == 1 {
				foundFlag := false
				if len(n.Records[inputKeyHashed-n.NodeId]) > 1 {
					for i := 0; i < len(n.Records[inputKeyHashed-n.NodeId]); i++ {
						if n.Records[inputKeyHashed-n.NodeId][i].OrigKey == OriginalKey {
							fmt.Fprint(w, n.Records[inputKeyHashed-n.NodeId][i].Value)
							foundFlag = true
						}
					}
					if foundFlag == false {
						http.Error(w, "No such key", http.StatusNotFound)
					}
				} else if len(n.Records[inputKeyHashed-n.NodeId]) == 1 {
					if n.Records[inputKeyHashed-n.NodeId][0].OrigKey == OriginalKey {
						fmt.Fprintf(w, n.Records[inputKeyHashed-n.NodeId][0].Value)
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
			if (inputKeyHashed >= n.NodeId && inputKeyHashed < n.SuccessorNodeId) || nodesCount == 1 {
				if len(n.Records[inputKeyHashed-n.NodeId]) == 0 {
					n.Records[inputKeyHashed-n.NodeId] = append(n.Records[inputKeyHashed-n.NodeId], NewRecord(OriginalKey, string(body), n.KeySpaceSize))
					fmt.Fprint(w, "Success!\n")
				} else {
					foundFlag := false
					for i := 0; i < len(n.Records[inputKeyHashed-n.NodeId]); i++ {
						if n.Records[inputKeyHashed-n.NodeId][i].OrigKey == OriginalKey {
							foundFlag = true
						}
					}

					if foundFlag {
						fmt.Fprint(w, "Key is already put\n")
					} else {
						n.Records[inputKeyHashed-n.NodeId] = append(n.Records[inputKeyHashed-n.NodeId], NewRecord(OriginalKey, string(body), n.KeySpaceSize))
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

func findKeyInOtherNodeHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respBody, _ := ioutil.ReadAll(r.Body)
		var responseBodyStruct GetInternalKeyRequest
		json.Unmarshal(respBody, &responseBodyStruct)

		if responseBodyStruct.HashedKey >= n.NodeId && responseBodyStruct.HashedKey < n.SuccessorNodeId {
			if len(n.Records[responseBodyStruct.HashedKey-n.NodeId]) > 1 {
				foundFlag := false
				for i := 0; i < len(n.Records[responseBodyStruct.HashedKey-n.NodeId]); i++ {
					if n.Records[responseBodyStruct.HashedKey-n.NodeId][i].OrigKey == responseBodyStruct.OriginalKey {
						fmt.Fprint(w, n.Records[responseBodyStruct.HashedKey-n.NodeId][i].Value)
						foundFlag = true
					}
				}
				if foundFlag == false {
					http.Error(w, "No such key", http.StatusNotFound)
				}
			} else if len(n.Records[responseBodyStruct.HashedKey-n.NodeId]) == 1 {
				if n.Records[responseBodyStruct.HashedKey-n.NodeId][0].OrigKey == responseBodyStruct.OriginalKey {
					fmt.Fprintf(w, n.Records[responseBodyStruct.HashedKey-n.NodeId][0].Value)
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

func putKeyInOtherNodeHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respBody, _ := ioutil.ReadAll(r.Body)
		var responseBodyStruct PutInternalKeyRequest
		json.Unmarshal(respBody, &responseBodyStruct)

		if responseBodyStruct.HashedKey >= n.NodeId && responseBodyStruct.HashedKey < n.SuccessorNodeId {
			if len(n.Records[responseBodyStruct.HashedKey-n.NodeId]) == 0 {
				n.Records[responseBodyStruct.HashedKey-n.NodeId] = append(n.Records[responseBodyStruct.HashedKey-n.NodeId], NewRecord(responseBodyStruct.OriginalKey, responseBodyStruct.Value, n.KeySpaceSize))
				fmt.Fprint(w, "Success!\n")
			} else {
				foundFlag := false
				for i := 0; i < len(n.Records[responseBodyStruct.HashedKey-n.NodeId]); i++ {
					if n.Records[responseBodyStruct.HashedKey-n.NodeId][i].OrigKey == responseBodyStruct.OriginalKey {
						foundFlag = true
					}
				}

				if foundFlag {
					fmt.Fprint(w, "Key is already put\n")
				} else {
					n.Records[responseBodyStruct.HashedKey-n.NodeId] = append(n.Records[responseBodyStruct.HashedKey-n.NodeId], NewRecord(responseBodyStruct.OriginalKey, responseBodyStruct.Value, n.KeySpaceSize))
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

func getNeighborsHandler(n *Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		responseBody, _ := json.Marshal([]string{n.PredecessorIp, n.SuccessorIp})
		fmt.Fprintf(w, string(responseBody))
	}
}
