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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

var privileged bool
var mutex = sync.RWMutex{}

//Balance represents balance, with change value.
type Balance struct {
	gadk.Balance
	Change int64
}

//Account represents account for bitcoind api.
type Account struct {
	Name     string
	Seed     gadk.Trytes `json:"-"`
	EncSeed  []byte
	Balances []Balance
}

func (a *Account) totalValueWithChange() int64 {
	var t int64
	for _, bals := range a.Balances {
		if bals.Balance.Value > 0 {
			t += bals.Balance.Value
		}
		t += bals.Change
	}
	return t
}

func findAddress(tx *bolt.Tx, adr gadk.Address) (*Account, int, error) {
	b := tx.Bucket([]byte("accounts"))
	if b == nil {
		return nil, -1, nil
	}
	c := b.Cursor()
	var result *Account
	index := -1
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var ac Account
		if err := json.Unmarshal(v, &ac); err != nil {
			return nil, -1, err
		}
		index = -1
		for i, a := range ac.Balances {
			if adr == a.Address {
				index = i
			}
		}
		if index == -1 {
			continue
		}
		seed := block.decrypt(ac.EncSeed)
		ac.Seed = gadk.Trytes(seed)
		result = &ac
		break
	}
	return result, index, nil
}
func listAccount(tx *bolt.Tx) ([]Account, error) {
	var asc []Account
	// Assume bucket exists and has keys
	b := tx.Bucket([]byte("accounts"))
	if b == nil {
		return nil, nil
	}
	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var ac Account
		if err := json.Unmarshal(v, &ac); err != nil {
			return nil, err
		}
		seed := block.decrypt(ac.EncSeed)
		ac.Seed = gadk.Trytes(seed)
		asc = append(asc, ac)
	}

	return asc, nil
}

func getAccount(tx *bolt.Tx, name string) (*Account, error) {
	var ac Account
	b := tx.Bucket([]byte("accounts"))
	if b == nil {
		return nil, nil
	}
	v := b.Get([]byte(name))
	if v == nil {
		return nil, nil
	}
	if err := json.Unmarshal(v, &ac); err != nil {
		return nil, err
	}
	seed := block.decrypt(ac.EncSeed)
	ac.Seed = gadk.Trytes(seed)
	return &ac, nil
}

func putAccount(tx *bolt.Tx, acc *Account) error {
	b, err := tx.CreateBucketIfNotExists([]byte("accounts"))
	if err != nil {
		return err
	}
	if acc.EncSeed == nil {
		acc.EncSeed = block.encrypt([]byte(acc.Seed))
	}
	bin, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	if err := b.Put([]byte(acc.Name), bin); err != nil {
		return err
	}
	return nil
}

func getOneParam(req *Request) (string, error) {
	ary, ok := req.Params.([]interface{})
	if !ok {
		return "", errors.New("invalid param")
	}
	if len(ary) != 1 {
		return "", errors.New("invalid param")
	}
	acc, ok := ary[0].(string)
	if !ok {
		return "", errors.New("invalid param")
	}
	return acc, nil
}

func getnewaddress(conf *Conf, req *Request, res *Response) error {
	acc, err := getOneParam(req)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		ac, err := getAccount(tx, acc)
		if err != nil {
			return err
		}
		if ac == nil {
			ac = &Account{
				Name: acc,
				Seed: gadk.NewSeed(),
			}
		}
		adr, err := gadk.NewAddress(ac.Seed, len(ac.Balances), 2)
		if err != nil {
			return err
		}
		ac.Balances = append(ac.Balances, Balance{
			Balance: gadk.Balance{
				Address: adr,
			},
		})
		res.Result = adr.WithChecksum()
		return putAccount(tx, ac)
	})
}
func getBalance(node string, tx *bolt.Tx) ([]Account, gadk.Balances, error) {
	api := gadk.NewAPI(node, nil)
	acs, err := listAccount(tx)
	if err != nil {
		log.Print(err)
		return nil, nil, err
	}
	var address []gadk.Address
	for _, ac := range acs {
		for _, b := range ac.Balances {
			address = append(address, b.Address)
		}
	}
	bals, err := api.Balances(address)
	return acs, bals, err
}
func listaddressgroupings(conf *Conf, req *Request, res *Response) error {
	var result [][][]interface{}
	err := db.View(func(tx *bolt.Tx) error {
		acs, bals, err := getBalance(conf.Node, tx)
		if err != nil {
			log.Print(err)
			return err
		}
		balmap := make(map[gadk.Address]int64)
		for _, b := range bals {
			balmap[b.Address] = b.Value
		}
		for _, ac := range acs {
			var r0 [][]interface{}
			for _, b := range ac.Balances {
				r1 := make([]interface{}, 3)
				r1[0] = b.Address.WithChecksum()
				r1[1] = 0
				if v, ok := balmap[b.Address]; ok {
					r1[1] = float64(v) / 100000000
				}
				r1[2] = ac.Name
				r0 = append(r0, r1)
			}
			result = append(result, r0)
		}
		return nil
	})
	res.Result = result
	return err
}
func getbalance(conf *Conf, req *Request, res *Response) error {
	err := db.View(func(tx *bolt.Tx) error {
		_, bals, err := getBalance(conf.Node, tx)
		if err != nil {
			return err
		}
		var total int64
		for _, b := range bals {
			total += b.Value
		}
		ftotal := float64(total) / 100000000
		res.Result = ftotal
		return nil
	})
	return err
}
func listaccounts(conf *Conf, req *Request, res *Response) error {
	api := gadk.NewAPI(conf.Node, nil)
	var addresses []gadk.Address
	result := make(map[string]float64)
	err := db.View(func(tx *bolt.Tx) error {
		acs, err := listAccount(tx)
		if err != nil {
			return err
		}
		for _, ac := range acs {
			for _, b := range ac.Balances {
				addresses = append(addresses, b.Address)
			}
			bals, err := api.Balances(addresses)
			if err != nil {
				return err
			}
			var sum int64
			for _, b := range bals {
				sum += b.Value
			}
			result[ac.Name] = float64(sum) / 100000000
		}
		return nil
	})
	res.Result = result
	return err
}

//only 'isvalid' params is valid, others may be incorrect.
func validateaddress(conf *Conf, req *Request, res *Response) error {
	type Info struct {
		IsValid      bool    `json:"isvalid"`
		Address      string  `json:"address"`
		ScriptPubKey string  `json:"scriptPubkey"`
		IsMine       bool    `json:"ismine"`
		IsWatchOnly  bool    `json:"iswatchonly"`
		IsScript     bool    `json:"isscript"`
		Pubkey       string  `json:"pubkey"`
		IsCompressed bool    `json:"iscompressed"`
		Account      *string `json:"account,omitempty"`
	}
	adrstr, err := getOneParam(req)
	if err != nil {
		return err
	}
	valid := false
	adr, err := gadk.ToAddress(adrstr)
	if err == nil {
		valid = true
	}
	var ac *Account
	err = db.View(func(tx *bolt.Tx) error {
		ac, _, err = findAddress(tx, adr)
		return err
	})

	info := Info{
		IsValid: valid,
		Address: adrstr,
		IsMine:  false,
	}
	if ac != nil {
		info.IsMine = true
		info.Account = &ac.Name
	}
	res.Result = &info
	return nil
}

func settxfee(conf *Conf, req *Request, res *Response) error {
	res.Result = true
	return nil
}

func send(acc string, conf *Conf, trs []gadk.Transfer) (gadk.Trytes, error) {
	var mwm int64 = 18
	if conf.Testnet {
		mwm = 15
	}
	api := gadk.NewAPI(conf.Node, nil)
	var result gadk.Trytes
	err := db.Update(func(tx *bolt.Tx) error {
		var ac *Account
		var err error
		if acc != "" {
			ac, err = getAccount(tx, acc)
		} else {
			acs, err := listAccount(tx)
			if err != nil {
				return err
			}
			if len(acs) == 0 {
				return errors.New("no accounts")
			}
			ac = &acs[0]
		}
		if err != nil {
			return err
		}
		if ac == nil {
			return errors.New("accout not found")
		}
		bhash, err := Send(api, ac, mwm, trs)
		if err == nil {
			if errr := putAccount(tx, ac); errr != nil {
				return errr
			}
			result = bhash
		}
		return err
	})
	return result, err
}

func sendmany(conf *Conf, req *Request, res *Response) error {
	mutex.RLock()
	if !privileged {
		mutex.RUnlock()
		return errors.New("not priviledged")
	}
	mutex.RUnlock()
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	if len(data) < 2 {
		return errors.New("invalid param length")
	}
	acc, ok := data[0].(string)
	if !ok {
		return errors.New("invalid account")
	}
	var target map[string]float64
	targetstr, ok := data[1].(string)
	if !ok {
		t, err := json.Marshal(data[1])
		if err != nil {
			return err
		}
		targetstr = string(t)
	}
	if err := json.Unmarshal([]byte(targetstr), &target); err != nil {
		return err
	}
	trs := make([]gadk.Transfer, len(target))
	i := 0
	var err error
	for k, v := range target {
		trs[i].Address, err = gadk.ToAddress(k)
		if err != nil {
			return err
		}
		trs[i].Value = int64(v * 100000000)
		trs[i].Tag = "AIDOSMARKET9A99999999999999"
		i++
	}
	res.Result, err = send(acc, conf, trs)
	return err
}

func sendfrom(conf *Conf, req *Request, res *Response) error {
	var err error
	mutex.RLock()
	if !privileged {
		mutex.RUnlock()
		return errors.New("not priviledged")
	}
	mutex.RUnlock()
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	if len(data) < 3 {
		return errors.New("invalid params")
	}
	acc, ok := data[0].(string)
	if !ok {
		return errors.New("invalid account")
	}
	var tr gadk.Transfer
	tr.Tag = "AIDOSMARKET9B99999999999999"
	adrstr, ok := data[1].(string)
	if !ok {
		return errors.New("invalid address")
	}
	tr.Address, err = gadk.ToAddress(adrstr)
	if err != nil {
		return err
	}
	value, ok := data[2].(float64)
	if !ok {
		return errors.New("invalid value")
	}
	tr.Value = int64(value * 100000000)
	res.Result, err = send(acc, conf, []gadk.Transfer{tr})
	return err
}
func sendtoaddress(conf *Conf, req *Request, res *Response) error {
	var err error
	mutex.RLock()
	if !privileged {
		mutex.RUnlock()
		return errors.New("not priviledged")
	}
	mutex.RUnlock()
	var tr gadk.Transfer
	tr.Tag = "AIDOSMARKET9C99999999999999"
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	if len(data) != 2 {
		return errors.New("invalid params")
	}
	adrstr, ok := data[0].(string)
	if !ok {
		return errors.New("invalid address")
	}
	tr.Address, err = gadk.ToAddress(adrstr)
	if err != nil {
		return err
	}
	value, ok := data[1].(float64)
	if !ok {
		return errors.New("invalid value")
	}
	tr.Value = int64(value * 100000000)
	res.Result, err = send("", conf, []gadk.Transfer{tr})
	return err
}

type details struct {
	Account  string      `json:"account"`
	Address  gadk.Trytes `json:"address"`
	Category string      `json:"category"`
	Amount   float64     `json:"amount"`
	Vout     int64       `json:"vout"`
	Fee      float64     `json:"fee"`
	Txid     gadk.Trytes `json:"txid,omitempty"`
}

type tx struct {
	Amount          float64     `json:"amount"`
	Fee             float64     `json:"fee"`
	Confirmations   int         `json:"confirmations"`
	Blockhash       string      `json:"blockhash"`
	Blockindex      int64       `json:"blockindex"`
	Blocktime       int64       `json:"blocktime"`
	Txid            gadk.Trytes `json:"txid"`
	Walletconflicts []string    `json:"walletconflicts"`
	Time            int64       `json:"time"`
	TimeReceived    int64       `json:"timereceived"`
	Details         []*details  `json:"details"`
	Hex             string      `json:"hex"`
}

func gettransaction(conf *Conf, req *Request, res *Response) error {
	bundlestr, err := getOneParam(req)
	if err != nil {
		return err
	}
	bundle := gadk.Trytes(bundlestr)
	ft := gadk.FindTransactionsRequest{
		Bundles: []gadk.Trytes{bundle},
	}
	api := gadk.NewAPI(conf.Node, nil)
	r, err := api.FindTransactions(&ft)
	if err != nil {
		return err
	}
	if len(r.Hashes) == 0 {
		return errors.New("bundle not found")
	}
	resp, err := api.GetTrytes(r.Hashes)
	if err != nil {
		return err
	}
	if len(resp.Trytes) != len(r.Hashes) {
		return fmt.Errorf("cannot get all txs %d/%d", len(r.Hashes), len(resp.Trytes))
	}
	inc, err := api.GetLatestInclusion(r.Hashes)
	if err != nil {
		return err
	}
	included := false
	for _, i := range inc {
		if i {
			included = true
			break
		}
	}
	detailss := make([]*details, 0, len(r.Hashes))
	var amount int64
	err = db.View(func(tx *bolt.Tx) error {
		for _, tr := range resp.Trytes {
			if included {
				inc, err := api.GetLatestInclusion([]gadk.Trytes{tr.Hash()})
				if err != nil {
					return err
				}
				if !inc[0] {
					continue
				}
			}
			if tr.Value == 0 {
				continue
			}
			ac, _, errr := findAddress(tx, tr.Address)
			if errr != nil {
				return errr
			}
			if ac == nil {
				continue
			}
			d := &details{
				Account:  ac.Name,
				Address:  tr.Address.WithChecksum(),
				Amount:   float64(tr.Value) / 100000000,
				Category: "send",
			}
			amount += tr.Value
			if tr.Value > 0 {
				d.Category = "receive"
			}
			detailss = append(detailss, d)
		}
		return nil
	})
	if err != nil {
		return err
	}
	nconf := 0
	if included {
		nconf = 100000
	}
	t := resp.Trytes[0].Timestamp.Unix()
	res.Result = &tx{
		Amount:          float64(amount) / 100000000,
		Confirmations:   nconf,
		Time:            t,
		TimeReceived:    t,
		Details:         detailss,
		Walletconflicts: []string{},
		Txid:            bundle,
	}
	return nil
}

//dont supprt over 1000 txs.
func listtransactions(conf *Conf, req *Request, res *Response) error {
	api := gadk.NewAPI(conf.Node, nil)
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	if len(data) != 2 {
		return errors.New("invalid params")
	}
	adrstr, ok := data[0].(string)
	if !ok {
		return errors.New("invalid address")
	}
	if adrstr != "*" {
		return errors.New("address must be *")
	}
	n, ok := data[1].(float64)
	if !ok {
		return errors.New("invalid number")
	}
	num := int(n)
	var ltx []*details
	err := db.View(func(tx *bolt.Tx) error {
		var hs []*txstate
		b := tx.Bucket([]byte("hashes"))
		if b == nil {
			return nil
		}
		v := b.Get([]byte("hashes"))
		if err := json.Unmarshal(v, &hs); err != nil {
			return err
		}
		txs := make([]gadk.Trytes, len(hs))
		for i := 0; i < len(hs); i++ {
			txs[i] = hs[len(hs)-1-i].Hash
		}
		resp, err := api.GetTrytes(txs)
		if err != nil {
			return err
		}
		ltx = make([]*details, 0, len(txs))
		for _, tr := range resp.Trytes {
			if tr.Value == 0 {
				continue
			}
			ac, _, errr := findAddress(tx, tr.Address)
			if errr != nil {
				return errr
			}
			if ac == nil {
				continue
			}
			dt := &details{
				Txid:     tr.Bundle,
				Address:  tr.Address.WithChecksum(),
				Amount:   float64(tr.Value) / 100000000,
				Category: "send",
			}
			if tr.Value > 0 {
				dt.Category = "receive"
			}
			ltx = append(ltx, dt)
			if len(ltx) >= num {
				return nil
			}
		}
		return nil
	})
	res.Result = ltx
	return err
}
func walletpassphrase(conf *Conf, req *Request, res *Response) error {
	mutex.RLock()
	if privileged {
		mutex.RUnlock()
		return nil
	}
	mutex.RUnlock()
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	if len(data) < 2 {
		return errors.New("invalid param length")
	}
	pwd, ok := data[0].(string)
	if !ok {
		return errors.New("invalid password")
	}
	sec, ok := data[1].(float64)
	if !ok {
		return errors.New("invalid time")
	}
	sum := sha256.Sum256([]byte(pwd))
	if !bytes.Equal(sum[:], block.pwd256) {
		return errors.New("invalid password")
	}
	go func() {
		mutex.Lock()
		privileged = true
		mutex.Unlock()
		time.Sleep(time.Second * time.Duration(sec))
		mutex.Lock()
		privileged = false
		mutex.Unlock()
	}()
	return nil
}
