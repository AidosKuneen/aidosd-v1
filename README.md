[![GoDoc](https://godoc.org/github.com/AidosKuneen/aidosd?status.svg)](https://godoc.org/github.com/AidosKuneen/aidosd)
[![Build Status](https://travis-ci.org/AidosKuneen/aidosd.svg?branch=master)](https://travis-ci.org/AidosKuneen/aidosd)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/AidosKuneen/aidosd/LICENSE)
# aidosd

aidosd is a deamon which acts as bitcoind for adk. For now impletented APIs are :

* `getnewaddress`
* `listaccounts`
* `listaddressgroupings`
* `validateaddress`
* `settxfee`
* `walletpassphrase`
* `sendmany`
* `sendfrom`
* `gettransaction`
* `getbalance`
* `sendtoaddress`
* `listtransactions`

and `walletnotify` feature.

Refer Bitcoin API Reference (e.g. [here](https://bitcoin.org/en/developer-reference#rpcs)) for more details
about how to call these APIs.

See [incompatibility lists](https://github.com/AidosKuneen/aidosd/blob/master/incompatibilities.md)
for details about incompatibilities with Bitcoin APIs.

# NOTE: DON'T USE MULTIPLE ACCOUTS. Account feature will be removed in  a later version.

# Reqirements

* go 1.8+
* gcc (for linux)
* mingw (for windows)

Dependencies:
* "github.com/AidosKuneen/gadk"
* "github.com/boltdb/bolt"


# Build

```
	$ mkdir go
	$ cd go
	$ mkdir src
	$ mkdir bin
	$ mkdir pkg
	$ exoprt GOPATH=`pwd`
	$ cd src
	$ go get -u github.com/AidosKuneen/aidosd
	$ cd github.com/AidosKuneen/aidosd
	$ go build
```

# Configuration

Configurations are in `aidosd.conf`.

 * `rpcuser` : Username for JSON-RPC connections 
 * `rpcpassword`: Password for JSON-RPC connections 
 * `rpcport`: Listen for JSON-RPC connections on <port> (default: 8332) 
 * `walletnotify`: Execute command when a  transaction comes into a wallet (%s in cmd is replaced by bundle ID) 
 * `aidos_node`: Host address of an Aidos node server , which must be confifured  for wallet.
 * `testnet`: Set `true` if you are using `testnet` (default: false).
 * `passphrase`: Set `false` if your program sends tokens withtout `walletpassphrase` (default :true) .
 * `tag`: Set your identifier. You can use charcters 9 and A~Z and don't use other ones, and it must be under 20 characters.
 This is used as tag in transactions aidosd sends.

Note that `aidosd` always encrypts seeds with AES regardles `passphrase` settings. 

Examples of `aidosd.conf`:

```
rpcuser=put_your_username
rpcpassword=put_your_password
rpcport=8332
walletnotify=/home/adk/script/receive.sh %s
aidos_node = http://localhost:14266
testnet = false
passphrase = true
tag = "AWESOME9EXCHANGER"
```

```
rpcuser=put_your_username
rpcpassword=put_your_password
rpcport=8332
testnet = true
passphrase = false
aidos_node = http://localhost:15555
```


# Usage

When you run `aidosd` first time, you need to input a password to encrypt seeds in wallet.
If you run `aidosd` 2nd time or later, you need to input the password to decrypt seeds.

```
$ ./aidosd
enter password: <input your password> 
```

Then `aidosd` starts to run in background.

```
$ ./aidosd
Enter password: 
starting the aidosd server at port http://0.0.0.0:8332
aidosd is started
```

If you forget the password, YOU CANNOT ACCESS YOUR SEED ANYMORE (i.e. you cannot use your token).
Please remove the database in this case, i.e. remove `aidosd.db` .



To know if it is still running, run:

```
	$ ./aidosd status
```

This prints the status ("running" or "stopped").


When you want to stop:

```
	$ ./aidosd stop
```
