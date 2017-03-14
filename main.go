package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	cli "github.com/urfave/cli"
	util "github.com/whyrusleeping/ethelp/util"
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

var setValue = cli.Command{
	Name: "set",
	Action: func(c *cli.Context) error {
		if len(c.Args()) != 2 {
			return fmt.Errorf("must specify contract address and value to set")
		}

		caddr := c.Args()[0]
		val := c.Args()[1]

		method := "update(string)"
		methash := util.GetMethodHashThing(method)
		fmt.Println(methash)

		fmt.Printf("setting value in %s to be %s\n", caddr, val)

		encval := util.EncodeEthString(val)
		loc := fmt.Sprintf("%064x", 32)

		txp := &util.TransactionParams{
			From: myethaddr,
			To:   caddr,
			Data: "0x" + methash + loc + encval,
		}

		tx, err := util.SendTransaction(txp)
		if err != nil {
			return err
		}
		fmt.Printf("created transaction: %s\n", tx)

		fmt.Println("waiting for transaction to be mined...")
		res, err := util.WaitForTx(tx)
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
		fmt.Fprintf(os.Stderr, "calling '%s' on %s\n", method, addr)

		methash := util.GetMethodHashThing(method)
		txp := &util.TransactionParams{
			From: myethaddr,
			To:   addr,
			Data: fmt.Sprintf("0x%s", methash),
		}

		resp, err := util.MakeRpcCall("eth_call", []interface{}{txp})
		if err != nil {
			return err
		}
		respval, err := util.DecodeEthString(resp.(string)[66:])
		if err != nil {
			return err
		}

		fmt.Print(respval)
		return nil
	},
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
		txp := &util.TransactionParams{
			From: myethaddr,
			Data: "0x" + string(data),
		}

		tx, err := util.SendTransaction(txp)
		if err != nil {
			return err
		}

		fmt.Println("Created contract on tx", tx)

		fmt.Println("waiting for contract to be mined...")
		before := time.Now()

		res, err := util.WaitForTx(tx)
		if err != nil {
			return err
		}

		rmap := res.(map[string]interface{})
		fmt.Printf("took %s to mine block with contract\n", time.Since(before))
		fmt.Printf("new contract address is %s\n", rmap["contractAddress"])

		return nil
	},
}
