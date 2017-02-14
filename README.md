# ethelp

Currently, a very single purpose ethereum helper tool.

## Installation
```
go get github.com/whyrusleeping/ethelp
```

## Commands

### ethelp create <compiled.bin>
Publishes a smart contract to the network.
- Takes as an argument a file containing the compiled evm code.

### ethelp set <contract> <value>
Sets a value in a contract.
- First argument is the address of the contract you created.
- Second argument is the value to set.

Note: currently only calls `update(string)` on the contract, if your contract
has a different method signature, this won't work.

### ethelp get <contract>
Gets a value stored in a contract.

Note: currently only calls `getvalue()`, if your contract has a different
method signature, this won't work.


