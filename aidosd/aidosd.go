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

package aidosd

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/natefinch/lumberjack"
	"golang.org/x/crypto/ssh/terminal"
)

var db *bolt.DB
var passPhrase = []byte("AidosKuneen")
var block *aesCrypto

type aesCrypto struct {
	block  cipher.Block
	pwd256 []byte
}

func newAESCrpto(pwd []byte) (*aesCrypto, error) {
	pwd2 := sha256.Sum256(pwd)
	pwd256 := pwd2[:]
	block, err := aes.NewCipher(pwd256)
	if err != nil {
		return nil, err
	}
	return &aesCrypto{
		block:  block,
		pwd256: pwd256,
	}, nil
}

func (a *aesCrypto) encrypt(pt []byte) []byte {
	ct := make([]byte, aes.BlockSize+len(pt))
	iv := ct[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	encryptStream := cipher.NewCTR(a.block, iv)
	encryptStream.XORKeyStream(ct[aes.BlockSize:], pt)
	return ct
}

func (a *aesCrypto) decrypt(ct []byte) []byte {
	pt := make([]byte, len(ct[aes.BlockSize:]))
	decryptStream := cipher.NewCTR(a.block, ct[:aes.BlockSize])
	decryptStream.XORKeyStream(pt, ct[aes.BlockSize:])
	return pt
}

//Conf is configuration for aidosd.
type Conf struct {
	RPCUser     string
	RPCPassword string
	RPCPort     string
	Notify      string
	Node        string
	Testnet     bool
	PassPhrase  bool
}

//ParseConf parses conf file.
func ParseConf() *Conf {
	conf := Conf{
		RPCPort:    "8332",
		PassPhrase: true,
	}

	f, err := os.Open("aidosd.conf")
	if err != nil {
		panic(err)
	}
	dat, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	str := string(dat)
	for i, state := range strings.Split(str, "\n") {
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
		default:
			log.Println(states[0] + " is ignored.")
		}
	}
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
func SetDB() {
	var err error
	db, err = bolt.Open("aidosd.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("exiting...")
		if err := db.Close(); err != nil {
			log.Println(err)
		}
		os.Exit(1)
	}()
}

func getPasswd() {
	pwd, err := terminal.ReadPassword(int(syscall.Stdin)) //int conversion is needed for win
	if err != nil {
		panic(err)
	}
	block, err = newAESCrpto(pwd)
	if err != nil {
		panic(err)
	}
}

//Password reads passowrd fron stdin and save password.
func Password() {
	err := db.Update(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("pass_phrase"))
		if b == nil {
			fmt.Print("It seems it's the first time to run aidosd. Please enter password: ")
			getPasswd()
			fmt.Println("")
			cipherText := block.encrypt(passPhrase)
			b, err = tx.CreateBucket([]byte("pass_phrase"))
			if err != nil {
				return err
			}
			if err := b.Put([]byte("pass_phrase"), cipherText); err != nil {
				return err
			}
			return nil
		}
		fmt.Print("Please enter password: ")
		getPasswd()
		fmt.Println("")
		ct := b.Get([]byte("pass_phrase"))
		pt := block.decrypt(ct)
		if !bytes.Equal(passPhrase, pt) {
			return errors.New("incorrect password")
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
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
	case "walletpassphrase":
		err = walletpassphrase(conf, &req, &res)
	case "sendmany":
		err = sendmany(conf, &req, &res)
	case "sendfrom":
		err = sendfrom(conf, &req, &res)
	case "gettransaction":
		err = gettransaction(conf, &req, &res)
	case "getbalance":
		err = getbalance(conf, &req, &res)
	case "sendtoaddress":
		err = sendtoaddress(conf, &req, &res)
	case "listtransactions":
		err = listtransactions(conf, &req, &res)
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

type txstate struct {
	Hash      gadk.Trytes
	Confirmed bool
}

func compareHashes(api *gadk.API, tx *bolt.Tx, hashes []gadk.Trytes) ([]gadk.Trytes, []gadk.Trytes, error) {
	var hs []*txstate
	var news []*txstate
	var confirmed []gadk.Trytes
	//search new tx
	b := tx.Bucket([]byte("hashes"))
	if b != nil {
		v := b.Get([]byte("hashes"))
		if v != nil {
			if err := json.Unmarshal(v, &hs); err != nil {
				return nil, nil, err
			}
		}
	}
	news = make([]*txstate, 0, len(hashes))
	for _, h1 := range hashes {
		exist := false
		for _, h2 := range hs {
			if h1 == h2.Hash {
				exist = true
				break
			}
		}
		if !exist {
			news = append(news, &txstate{Hash: h1})
		}
	}
	//search newly confirmed tx
	confirmed = make([]gadk.Trytes, 0, len(hs))
	hs = append(hs, news...)
	for _, h := range hs {
		if h.Confirmed {
			continue
		}
		inc, err := api.GetLatestInclusion([]gadk.Trytes{h.Hash})
		if err != nil {
			return nil, nil, err
		}
		if len(inc) > 0 && inc[0] {
			confirmed = append(confirmed, h.Hash)
			h.Confirmed = true
		}
	}
	//save txs
	b, err := tx.CreateBucketIfNotExists([]byte("hashes"))
	if err != nil {
		return nil, nil, err
	}
	bin, err := json.Marshal(hs)
	if err != nil {
		return nil, nil, err
	}
	if err = b.Put([]byte("hashes"), bin); err != nil {
		return nil, nil, err
	}

	ret := make([]gadk.Trytes, len(news))
	for i := range news {
		ret[i] = news[i].Hash
	}
	return ret, confirmed, nil
}

//Walletnotify exec walletnotify scripts when receivng tx and tx is confirmed.
func Walletnotify(conf *Conf) error {
	log.Println("starting walletnotify...")
	api := gadk.NewAPI(conf.Node, nil)
	bdls := make(map[gadk.Trytes]struct{})
	err := db.Update(func(tx *bolt.Tx) error {
		//get all addresses
		var adrs []gadk.Address
		acc, err := listAccount(tx)
		if err != nil {
			return err
		}
		if len(acc) == 0 {
			return nil
		}
		for _, ac := range acc {
			for _, b := range ac.Balances {
				adrs = append(adrs, b.Address)
			}
		}
		//get all trytes for all addresses
		ft := gadk.FindTransactionsRequest{
			Addresses: adrs,
		}
		r, err := api.FindTransactions(&ft)
		if err != nil {
			return err
		}
		if len(r.Hashes) == 0 {
			log.Println("no tx for addresses in wallet")
			return nil
		}
		//get newly added and newly confirmed trytes.
		news, confirmed, err := compareHashes(api, tx, r.Hashes)
		if err != nil {
			return err
		}
		if len(news)+len(confirmed) == 0 {
			log.Println("no tx to be handled")
			return nil
		}
		//add balances for all newly confirmed tx..
		resp, err := api.GetTrytes(confirmed)
		if err != nil {
			return err
		}
		for _, tr := range resp.Trytes {
			if tr.Value <= 0 {
				continue
			}
			acc, index, errr := findAddress(tx, tr.Address)
			if errr != nil {
				log.Println(errr)
				continue
			}
			if acc == nil {
				log.Println("acc shoud not be null")
				continue
			}
			acc.Balances[index].Value += tr.Value
			acc.Balances[index].Change = 0
			if errrr := putAccount(tx, acc); err != nil {
				return errrr
			}
		}
		//add bundle hash to bdls.
		nresp, err := api.GetTrytes(news)
		if err != nil {
			return err
		}
		for _, tr := range nresp.Trytes {
			if tr.Value != 0 {
				bdls[tr.Bundle] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
		return err
	}
	//exec cmds for all new txs. %s will be the bundle hash.
	if conf.Notify == "" {
		return nil
	}
	for bdl := range bdls {
		cmd := strings.Replace(conf.Notify, "%s", string(bdl), -1)
		args, err := shellwords.Parse(cmd)
		if err != nil {
			log.Println(err)
			return err
		}
		var out []byte
		if len(args) == 1 {
			out, err = exec.Command(args[0]).Output()
		} else {
			out, err = exec.Command(args[0], args[1:]...).Output()
		}
		if err != nil {
			log.Println(err)
			return err
		}
		delete(bdls, bdl)
		log.Println("executed ", cmd, ",output:", string(out))
	}
	log.Println("end of walletnotify")
	return nil
}
