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


Note that 

* These don't have full features, these are just for a few exchange programs.
* Error codes and error string from these commands are not same as ones in bitcoin.
* Deposit addresses must be changed per every deposits e.g. by calling `getnewaddress`. 

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
	$ (copy giota and aidosd to current directory)
	$ cd aidosd
	$ go get
	$ go build -o adkd
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

When you run `aidosd` first time, you need to input passwords to encrypt seeds in wallet.

```
$ ./adkd
It seems it's the first time to run aidosd. Please enter password: <input your password> 
```

If you run `aidosd` 2nd time or later, you need to input password to decrypt seeds.

```
$ ./adkd
Please enter password: <input your password> 
```

Then, normally you need to `aidosd` to be deamon, so push `Ctrl-Z` to be backgound, and:

```
	$ bg
	$ ps -ef | grep adkd
	$ disown <process id of adkd>
```

If you forget the password, YOU CANNOT ACCESS YOUR SEED ANYMORE (i.e. you cannot use your token).
Please remove the database in this case, i.e. remove `aidosd.db` .

When you want to stop:

```
	$ kill -SIGINT <process id of adkd>
	<wait for a few dozen of seconds>
	$ ps -ef|grep <process id> # confirm the adkd was killed
```
