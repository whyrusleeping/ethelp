package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	oldkeccak "github.com/ethereum/go-ethereum/crypto/sha3"
	cli "github.com/urfave/cli"
)

var myethaddr string

func main() {
	myethaddr = os.Getenv("MY_ETH_ADDR")
	if myethaddr == "" {
		log.Fatalln("must set env var MY_ETH_ADDR to your ethereum address")
	}

	app := cli.NewApp()
	app.Commands = []cli.Command{
		createContract,
		getValue,
		setValue,
	}
	app.RunAndExitOnError()
}

type rpc struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type response struct {
	Error *struct {
		Code    int
		Message string
		Data    interface{}
	}
	Result interface{}
}

type transactionParams struct {
	From string `json:"from"`
	To   string `json:"to,omitempty"`
	Data string `json:"data,omitempty"`
}

var setValue = cli.Command{
	Name: "set",
	Action: func(c *cli.Context) error {
		if len(c.Args()) != 2 {
			return fmt.Errorf("must specify contract address and value to set")
		}

		caddr := c.Args()[0]
		val := c.Args()[1]

		method := "update(string)"
		methash := getMethodHashThing(method)
		fmt.Println(methash)

		fmt.Printf("setting value in %s to be %s\n", caddr, val)

		encval := encodeEthString(val)
		loc := fmt.Sprintf("%064x", 32)

		txp := &transactionParams{
			From: myethaddr,
			To:   caddr,
			Data: "0x" + methash + loc + encval,
		}

		tx, err := sendTransaction(txp)
		if err != nil {
			return err
		}
		fmt.Printf("created transaction: %s\n", tx)

		fmt.Println("waiting for transaction to be mined...")
		res, err := waitForTx(tx)
		if err != nil {
			return err
		}

		fmt.Printf("transaction mined in block %s\n", res.(map[string]interface{})["blockHash"])
		return nil
	},
}

var getValue = cli.Command{
	Name: "get",
	Action: func(c *cli.Context) error {
		if !c.Args().Present() {
			return fmt.Errorf("please specify contract address to query")
		}

		method := "getvalue()"

		addr := c.Args().First()
		fmt.Printf("calling '%s' on %s\n", method, addr)

		methash := getMethodHashThing(method)
		txp := &transactionParams{
			From: myethaddr,
			To:   addr,
			Data: fmt.Sprintf("0x%s", methash),
		}

		resp, err := makeRpcCall("eth_call", []interface{}{txp})
		if err != nil {
			return err
		}
		respval, err := decodeEthString(resp.(string)[66:])
		if err != nil {
			return err
		}

		fmt.Println(respval)
		return nil
	},
}

func getMethodHashThing(methodsig string) string {
	h := oldkeccak.NewKeccak256()
	h.Write([]byte(methodsig))
	out := h.Sum(nil)

	return fmt.Sprintf("%x", out[:4])
}

func decodeEthString(s string) (string, error) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}

	length := big.NewInt(0).SetBytes(raw[:32]).Uint64()
	return string(raw[32 : 32+length]), nil
}

func encodeEthString(s string) string {
	l := len(s)
	lstr := fmt.Sprintf("%064x", l)
	hexval := hex.EncodeToString([]byte(s))
	padding := 64 - (len(hexval) % 64)
	pad := strings.Repeat("0", padding)

	return lstr + hexval + pad
}

var createContract = cli.Command{
	Name: "create",
	Action: func(c *cli.Context) error {
		compiledContract := c.Args().First()
		if compiledContract == "" {
			return fmt.Errorf("must pass argument of compiled contract data")
		}

		data, err := ioutil.ReadFile(compiledContract)
		if err != nil {
			return err
		}
		txp := &transactionParams{
			From: myethaddr,
			Data: "0x" + string(data),
		}

		tx, err := sendTransaction(txp)
		if err != nil {
			return err
		}

		fmt.Println("Created contract on tx", tx)

		fmt.Println("waiting for contract to be mined...")
		before := time.Now()

		res, err := waitForTx(tx)
		if err != nil {
			return err
		}

		rmap := res.(map[string]interface{})
		fmt.Printf("took %s to mine block with contract\n", time.Since(before))
		fmt.Printf("new contract address is %s\n", rmap["contractAddress"])

		return nil
	},
}

func waitForTx(tx string) (interface{}, error) {
	for {
		res, err := makeRpcCall("eth_getTransactionReceipt", []interface{}{tx})
		if err != nil {
			return nil, err
		}
		if res != nil {
			return res, nil
		}
		time.Sleep(time.Second)
	}
}

func sendTransaction(txp *transactionParams) (string, error) {
	res, err := makeRpcCall("eth_sendTransaction", []interface{}{txp})
	if err != nil {
		return "", err
	}

	return res.(string), nil
}

func makeRpcCall(method string, params []interface{}) (interface{}, error) {
	r := &rpc{
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

	req, err := http.NewRequest("POST", "http://localhost:8545", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("error %d: %s", result.Error.Code, result.Error.Message)
	}
	return result.Result, nil
}
