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
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

func getnewaddress(conf *Conf, req *Request, res *Response) error {
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	acc := ""
	switch len(data) {
	case 1:
		acc, ok = data[0].(string)
		if !ok {
			return errors.New("invalid txid")
		}
	case 0:
	default:
		return errors.New("invalid params")
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
func getBalance(api apis, tx *bolt.Tx) ([]Account, map[gadk.Address]int64, error) {
	acs, err := listAccount(tx)
	if err != nil {
		return nil, nil, err
	}
	var address []gadk.Address
	for _, ac := range acs {
		for _, b := range ac.Balances {
			address = append(address, b.Address)
		}
	}
	bals, err := api.Balances(address)
	balmap := make(map[gadk.Address]int64)
	for _, b := range bals {
		balmap[b.Address] = b.Value
	}
	return acs, balmap, err
}
func listaddressgroupings(conf *Conf, req *Request, res *Response) error {
	var result [][][]interface{}
	var r0 [][]interface{}
	err := db.View(func(tx *bolt.Tx) error {
		acs, balmap, err := getBalance(conf.api, tx)
		if err != nil {
			return err
		}
		for _, ac := range acs {
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
		}
		return nil
	})
	result = append(result, r0)
	res.Result = result
	return err
}
func getbalance(conf *Conf, req *Request, res *Response) error {
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("param must be slice")
	}
	adrstr := "*"
	switch len(data) {
	case 3:
		fallthrough
	case 2:
		n, okk := data[1].(float64)
		if !okk {
			return errors.New("invalid number")
		}
		if n == 0 {
			return errors.New("not support unconfirmed transactions")
		}
		fallthrough
	case 1:
		adrstr, ok = data[0].(string)
		if !ok {
			return errors.New("invalid address")
		}
	case 0:
	default:
		return errors.New("invalid params")
	}

	err := db.View(func(tx *bolt.Tx) error {
		acc, balmap, err := getBalance(conf.api, tx)
		if err != nil {
			return err
		}
		var total int64
		if adrstr == "*" {
			for _, v := range balmap {
				total += v
			}
		} else {
			var bal []Balance
			for _, a := range acc {
				if adrstr == a.Name {
					bal = a.Balances
				}
			}
			for _, b := range bal {
				total += balmap[b.Address]
			}
		}
		ftotal := float64(total) / 100000000
		res.Result = ftotal
		return nil
	})
	return err
}
func listaccounts(conf *Conf, req *Request, res *Response) error {
	ary, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid param")
	}
	if len(ary) > 0 {
		conf, ok := ary[0].(float64)
		if !ok {
			return errors.New("invalid param")
		}
		if conf == 0 {
			return errors.New("not support unconfirmed transacton")
		}
	}
	result := make(map[string]float64)
	err := db.View(func(tx *bolt.Tx) error {
		acs, err := listAccount(tx)
		if err != nil {
			return err
		}
		for _, ac := range acs {
			var addresses []gadk.Address
			for _, b := range ac.Balances {
				addresses = append(addresses, b.Address)
			}
			bals, err := conf.api.Balances(addresses)
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

type info struct {
	IsValid      bool    `json:"isvalid"`
	Address      string  `json:"address"`
	ScriptPubKey string  `json:"scriptPubkey"`
	IsMine       bool    `json:"ismine"`
	IsWatchOnly  *bool   `json:"iswatchonly,omitempty"`
	IsScript     *bool   `json:"isscript,omitempty"`
	Pubkey       *string `json:"pubkey,omitempty"`
	IsCompressed *bool   `json:"iscompressed,omitempty"`
	Account      *string `json:"account,omitempty"`
}

//only 'isvalid' params is valid, others may be incorrect.
func validateaddress(conf *Conf, req *Request, res *Response) error {
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	if len(data) != 1 {
		return errors.New("length of param must be 1")
	}
	adrstr, ok := data[0].(string)
	if !ok {
		return errors.New("invalid address")
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

	infoi := info{
		IsValid: valid,
		Address: adrstr,
		IsMine:  false,
	}
	t := false
	empty := ""
	if ac != nil {
		infoi.IsMine = true
		infoi.Account = &ac.Name
		infoi.IsWatchOnly = &t
		infoi.IsScript = &t
		infoi.Pubkey = &empty
		infoi.IsCompressed = &t
	}
	res.Result = &infoi
	return nil
}

func settxfee(conf *Conf, req *Request, res *Response) error {
	res.Result = true
	return nil
}

type details struct {
	Account   string      `json:"account"`
	Address   gadk.Trytes `json:"address"`
	Category  string      `json:"category"`
	Amount    float64     `json:"amount"`
	Vout      int64       `json:"vout"`
	Fee       float64     `json:"fee"`
	Abandoned *bool       `json:"abandoned,omitempty"`
}

type tx struct {
	Amount            float64     `json:"amount"`
	Fee               float64     `json:"fee"`
	Confirmations     int         `json:"confirmations"`
	Blockhash         *string     `json:"blockhash,omitempty"`
	Blockindex        *int64      `json:"blockindex,omitempty"`
	Blocktime         *int64      `json:"blocktime,omitempty"`
	Txid              gadk.Trytes `json:"txid"`
	Walletconflicts   []string    `json:"walletconflicts"`
	Time              int64       `json:"time"`
	TimeReceived      int64       `json:"timereceived"`
	BIP125Replaceable string      `json:"bip125-replaceable"`
	Details           []*details  `json:"details"`
	Hex               string      `json:"hex"`
}

func gettransaction(conf *Conf, req *Request, res *Response) error {
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	bundlestr := ""
	switch len(data) {
	case 2:
	case 1:
		bundlestr, ok = data[0].(string)
		if !ok {
			return errors.New("invalid txid")
		}
	default:
		return errors.New("invalid params")
	}
	bundle := gadk.Trytes(bundlestr)
	ft := gadk.FindTransactionsRequest{
		Bundles: []gadk.Trytes{bundle},
	}
	r, err := conf.api.FindTransactions(&ft)
	if err != nil {
		return err
	}
	if len(r.Hashes) == 0 {
		return errors.New("bundle not found")
	}
	resp, err := conf.api.GetTrytes(r.Hashes)
	if err != nil {
		return err
	}
	if len(resp.Trytes) != len(r.Hashes) {
		return fmt.Errorf("cannot get all txs %d/%d", len(r.Hashes), len(resp.Trytes))
	}

	detailss := make([]*details, 0, len(r.Hashes))
	var amount int64
	nconf := 0
	var dt *transaction
	indice := make(map[int64]struct{})
	err = db.View(func(tx *bolt.Tx) error {
		for _, tr := range resp.Trytes {
			dt2, errr := getTransaction(tx, conf, &tr)
			if errr != nil {
				return err
			}
			if _, exist := indice[tr.CurrentIndex]; exist {
				continue
			}
			indice[tr.CurrentIndex] = struct{}{}
			dt = dt2
			d := &details{
				Account:   *dt.Account,
				Address:   dt.Address,
				Category:  dt.Category,
				Amount:    dt.Amount,
				Abandoned: dt.Abandoned,
			}
			nconf = dt.Confirmations
			amount += tr.Value
			detailss = append(detailss, d)
		}
		return nil
	})
	if err != nil {
		return err
	}
	res.Result = &tx{
		Amount:            float64(amount) / 100000000,
		Confirmations:     nconf,
		Blocktime:         dt.Blocktime,
		Blockhash:         dt.Blockhash,
		Blockindex:        dt.Blockindex,
		Txid:              bundle,
		Walletconflicts:   []string{},
		Time:              dt.Time,
		TimeReceived:      dt.TimeReceived,
		BIP125Replaceable: "no",
		Details:           detailss,
	}
	return nil
}

type transaction struct {
	Account  *string     `json:"account"`
	Address  gadk.Trytes `json:"address"`
	Category string      `json:"category"`
	Amount   float64     `json:"amount"`
	// Label             string      `json:"label"`
	Vout          int64   `json:"vout"`
	Fee           float64 `json:"fee"`
	Confirmations int     `json:"confirmations"`
	Trusted       *bool   `json:"trusted,omitempty"`
	// Generated         bool        `json:"generated"`
	Blockhash       *string     `json:"blockhash,omitempty"`
	Blockindex      *int64      `json:"blockindex,omitempty"`
	Blocktime       *int64      `json:"blocktime,omitempty"`
	Txid            gadk.Trytes `json:"txid"`
	Walletconflicts []string    `json:"walletconflicts"`
	Time            int64       `json:"time"`
	TimeReceived    int64       `json:"timereceived"`
	// Comment           string      `json:"string"`
	// To                string `json:"to"`
	// Otheraccount      string `json:"otheraccount"`
	BIP125Replaceable string `json:"bip125-replaceable"`
	Abandoned         *bool  `json:"abandoned,omitempty"`
}

//dont supprt over 1000 txs.
func listtransactions(conf *Conf, req *Request, res *Response) error {
	data, ok := req.Params.([]interface{})
	if !ok {
		return errors.New("invalid params")
	}
	acc := "*"
	num := 10
	skip := 0
	switch len(data) {
	case 4:
		fallthrough
	case 3:
		n, okk := data[2].(float64)
		if !okk {
			return errors.New("invalid number")
		}
		skip = int(n)
		fallthrough
	case 2:
		n, okk := data[1].(float64)
		if !okk {
			return errors.New("invalid number")
		}
		num = int(n)
		fallthrough
	case 1:
		acc, ok = data[0].(string)
		if !ok {
			return errors.New("invalid account")
		}
	case 0:
	default:
		return errors.New("invalid params")
	}
	var ltx []*transaction
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
		m := make(map[gadk.Trytes]int)
		txs := make([]gadk.Trytes, len(hs))
		for i := 0; i < len(hs); i++ {
			m[hs[i].Hash] = i
			txs[i] = hs[i].Hash
		}
		resp, err := conf.api.GetTrytes(txs)
		if err != nil {
			return err
		}
		sort.Slice(resp.Trytes, func(i, j int) bool {
			return !(m[resp.Trytes[i].Hash()] < m[resp.Trytes[j].Hash()])
			// return !resp.Trytes[i].Timestamp.Before(resp.Trytes[j].Timestamp)
		})
		ltx = make([]*transaction, 0, num)
		index := 0
		for i := 0; i < len(resp.Trytes) && len(ltx) < num; i++ {
			tr := resp.Trytes[i]
			dt, err := getTransaction(tx, conf, &tr)
			if err != nil {
				return err
			}
			if dt.Amount == 0 {
				continue
			}

			if acc != "*" && *dt.Account != acc {
				continue
			}
			if index++; index-1 < skip {
				continue
			}
			ltx = append(ltx, dt)
		}
		return nil
	})
	res.Result = ltx
	return err
}

func getTransaction(tx *bolt.Tx, conf *Conf, tr *gadk.Transaction) (*transaction, error) {
	ac, _, errr := findAddress(tx, tr.Address)
	if errr != nil {
		return nil, errr
	}
	inc, err := conf.api.GetLatestInclusion([]gadk.Trytes{tr.Hash()})
	if err != nil {
		return nil, err
	}
	f := false
	emp := ""
	var zero int64
	dt := &transaction{
		Address:           tr.Address.WithChecksum(),
		Category:          "send",
		Amount:            float64(tr.Value) / 100000000,
		Txid:              tr.Bundle,
		Walletconflicts:   []string{},
		Time:              tr.Timestamp.Unix(),
		TimeReceived:      tr.Timestamp.Unix(),
		BIP125Replaceable: "no",
		Abandoned:         &f,
	}
	if ac != nil {
		dt.Account = &ac.Name
	}
	if inc[0] {
		dt.Blockhash = &emp
		dt.Blocktime = &dt.Time
		dt.Blockindex = &zero
		dt.Confirmations = 100000
	} else {
		dt.Trusted = &f
	}
	if tr.Value > 0 {
		dt.Category = "receive"
		dt.Abandoned = nil
	}
	return dt, nil
}
