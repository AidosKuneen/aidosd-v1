// Copyright (c) 2017 Aidos Developer

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package aidos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
	"github.com/natefinch/lumberjack"
)

var db *bolt.DB

type apis interface {
	FindTransactions(ft *gadk.FindTransactionsRequest) (*gadk.FindTransactionsResponse, error)
	GetTrytes(hashes []gadk.Trytes) (*gadk.GetTrytesResponse, error)
	Balances(adr []gadk.Address) (gadk.Balances, error)
	GetTransactionsToApprove(depth int64) (*gadk.GetTransactionsToApproveResponse, error)
	BroadcastTransactions(trytes []gadk.Transaction) error
	StoreTransactions(trytes []gadk.Transaction) error
	GetNodeInfo() (*gadk.GetNodeInfoResponse, error)
	GetInclusionStates([]gadk.Trytes, []gadk.Trytes) (*gadk.GetInclusionStatesResponse, error)
}

var mutex sync.RWMutex

//Conf is configuration for aidosd.
type Conf struct {
	RPCUser     string
	RPCPassword string
	RPCPort     string
	Notify      string
	Node        string
	Testnet     bool
	PassPhrase  bool
	Tag         string
	api         apis
}

//ParseConf parses conf file.
func ParseConf(cfile string) *Conf {
	conf := Conf{
		RPCPort:    "8332",
		PassPhrase: true,
	}

	f, err := os.Open(cfile)
	if err != nil {
		panic(err)
	}
	dat, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	str := string(dat)
	for i, state := range strings.Split(str, "\n") {
		state = strings.TrimSpace(state)
		if len(state) == 0 {
			continue
		}
		states := strings.SplitN(state, "=", 2)
		if len(states) != 2 {
			panic("illegal conf at line " + strconv.Itoa(i+1))
		}
		states[0] = strings.TrimSpace(states[0])
		states[1] = strings.TrimSpace(states[1])
		switch states[0] {
		case "rpcuser":
			conf.RPCUser = states[1]
		case "rpcpassword":
			conf.RPCPassword = states[1]
		case "rpcport":
			_, err = strconv.Atoi(states[1])
			if err != nil {
				panic("rcpport must be integer " + states[1])
			}
			conf.RPCPort = states[1]
		case "walletnotify":
			conf.Notify = states[1]
		case "aidos_node":
			conf.Node = states[1]
		case "testnet":
			switch states[1] {
			case "true":
				conf.Testnet = true
			case "false":
				//do nothing, it's default
			default:
				panic("testnet must be true or false")
			}
		case "passphrase":
			switch states[1] {
			case "true":
				//do nothing, it's default
			case "false":
				conf.PassPhrase = false
				privileged = true
			default:
				panic("passphrase must be true or false")
			}
		case "tag":
			for _, c := range states[1] {
				if !(c == '9' || (c >= 'A' && c <= 'Z')) {
					panic("invalid chracterin tag params. You can use characters 9 and A~Z ")
				}
			}
			if len(states[1]) > 20 {
				panic("tag is too long, must be under 20 characters.")
			}
			conf.Tag = states[1]

		default:
			log.Println(states[0] + " is ignored.")
		}
	}
	for i := len(conf.Tag); i < 20; i++ {
		conf.Tag += "9"
	}
	conf.Tag += "9AIDOSD"
	conf.api = gadk.NewAPI(conf.Node, nil)
	return &conf
}

//SetLog does log settings.
func SetLog(verbose bool) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	l := &lumberjack.Logger{
		Filename:   path.Join("aidosd.log"),
		MaxSize:    100, // megabytes
		MaxBackups: 10,
		MaxAge:     28, //days
	}
	if verbose {
		log.Println("output to stdout and file")
		m := io.MultiWriter(os.Stdout, l)
		log.SetOutput(m)
	} else {
		log.SetOutput(l)
	}
}

//SetDB setup db.
func setDB() {
	var err error
	db, err = bolt.Open("aidosd.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		Exit()
	}()
}

//Exit flushs DB and exit.
func Exit() {
	fmt.Println("exiting...")
	if err := db.Close(); err != nil {
		log.Println(err)
	}
	go func() {
		time.Sleep(3 * time.Second)
		os.Exit(1)
	}()
}

//Request is for parsing request from client.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

//Err represents error struct for response.
type Err struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

//Response is for respoding to clinete in jsonrpc.
type Response struct {
	Result interface{} `json:"result"`
	Error  *Err        `json:"error"`
	ID     interface{} `json:"id"`
}

func isValidAuth(r *http.Request, conf *Conf) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}
	return username == conf.RPCUser && password == conf.RPCPassword
}

//Handle handles api calls.
func Handle(conf *Conf, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			panic(err)
		}
	}()
	if !isValidAuth(r, conf) {
		log.Println("failed to auth")
		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		if _, err := w.Write([]byte("401 Unauthorized\n")); err != nil {
			log.Println(err)
		}
		return
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res := Response{
		ID: req.ID,
	}
	log.Println(req.Method, " is requested")
	var err error
	switch req.Method {
	case "getnewaddress":
		err = getnewaddress(conf, &req, &res)
	case "listaccounts":
		err = listaccounts(conf, &req, &res)
	case "listaddressgroupings":
		err = listaddressgroupings(conf, &req, &res)
	case "validateaddress":
		err = validateaddress(conf, &req, &res)
	case "settxfee":
		err = settxfee(conf, &req, &res)
	case "gettransaction":
		err = gettransaction(conf, &req, &res)
	case "getbalance":
		err = getbalance(conf, &req, &res)
	case "listtransactions":
		err = listtransactions(conf, &req, &res)
	case "walletpassphrase":
		err = walletpassphrase(conf, &req, &res)
	case "sendmany":
		err = sendmany(conf, &req, &res)
	case "sendfrom":
		err = sendfrom(conf, &req, &res)
	case "sendtoaddress":
		err = sendtoaddress(conf, &req, &res)
	default:
		err = errors.New(req.Method + " not supperted")
	}
	if err != nil {
		res.Error = &Err{
			Code:    -1,
			Message: err.Error(),
		}
	}
	result, err := json.Marshal(&res)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if _, err := w.Write(result); err != nil {
		panic(err)
	}
}

//Prepare prepares aidosd.
func Prepare(cfile string, passwd []byte) (*Conf, error) {
	setDB()

	if err := password(passwd); err != nil {
		fmt.Println(err)
		Exit()
		return nil, err
	}
	conf := ParseConf(cfile)

	return conf, nil
}
