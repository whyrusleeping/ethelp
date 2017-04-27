package util

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	oldkeccak "gx/ipfs/QmSYy1fYbh8wBH7TJyC9BJwfwt5UVWWLWaKpN4gSrNRRxi/go-ethereum/crypto/sha3"
)

type Rpc struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type Response struct {
	Error *struct {
		Code    int
		Message string
		Data    interface{}
	}
	Result interface{}
}

type TransactionParams struct {
	From string `json:"from"`
	To   string `json:"to,omitempty"`
	Data string `json:"data,omitempty"`
}

func GetMethodHashThing(methodsig string) string {
	h := oldkeccak.NewKeccak256()
	h.Write([]byte(methodsig))
	out := h.Sum(nil)

	return fmt.Sprintf("%x", out[:4])
}

func DecodeEthString(s string) (string, error) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}

	length := big.NewInt(0).SetBytes(raw[:32]).Uint64()
	return string(raw[32 : 32+length]), nil
}

func EncodeEthString(s string) string {
	l := len(s)
	lstr := fmt.Sprintf("%064x", l)
	hexval := hex.EncodeToString([]byte(s))
	padding := 64 - (len(hexval) % 64)
	pad := strings.Repeat("0", padding)

	return lstr + hexval + pad
}

func WaitForTx(host, tx string, timeout time.Duration) (interface{}, error) {
	start := time.Now()
	for time.Since(start) < timeout {
		res, err := MakeRpcCallWithHost(host, "eth_getTransactionReceipt", []interface{}{tx})
		if err != nil {
			return nil, err
		}
		if res != nil {
			return res, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for transaction to go through")
}

func SendTransaction(host string, txp *TransactionParams) (string, error) {
	res, err := MakeRpcCallWithHost(host, "eth_sendTransaction", []interface{}{txp})
	if err != nil {
		return "", err
	}

	return res.(string), nil
}

const defaultApi = "http://localhost:8545"

func MakeRpcCall(method string, params []interface{}) (interface{}, error) {
	return MakeRpcCallWithHost(defaultApi, method, params)
}

func MakeRpcCallWithHost(host, method string, params []interface{}) (interface{}, error) {
	r := &Rpc{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		Id:      1,
	}

	marshalled, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(marshalled)

	req, err := http.NewRequest("POST", host, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("error %d: %s", result.Error.Code, result.Error.Message)
	}
	return result.Result, nil
}
