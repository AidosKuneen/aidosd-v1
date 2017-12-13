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
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

func preparetSend(t *testing.T) (*Conf, *dummy1) {
	cdir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	fdb := filepath.Join(cdir, "aidosd.db")
	if err := os.Remove(fdb); err != nil {
		t.Log(err)
	}
	conf, err := Prepare("../aidosd.conf", []byte("test"))
	if err != nil {
		t.Error(err)
	}
	acc := make(map[string][]gadk.Address)
	vals := make(map[gadk.Address]int64)
	for _, ac := range []string{"ac1", "ac2", ""} {
		adr := newAddress(t, conf, ac)
		for _, a := range adr {
			acc[ac] = append(acc[ac], a)
			vals[a] = int64(rand.Int31() + 0.2*100000000)
		}
	}
	return conf, &dummy1{
		acc2adr: acc,
		vals:    vals,
		mtrytes: make(map[gadk.Trytes]gadk.Transaction),
		t:       t,
		isConf:  true,
		ch:      make(chan struct{}),
	}
}
func TestSend(t *testing.T) {
	conf, d1 := preparetSend(t)
	conf.api = d1
	d1.setupTXs()
	err := sendmany(conf, nil, nil)
	if err.Error() != "not priviledged" {
		t.Error("should be error")
	}
	if testwalletpassphrase1(conf, d1); err == nil {
		t.Error("should be error")
	}
	d1.isConf = false
	testwalletpassphrase2(conf, d1)
	testsendmany(conf, d1, true)

	d1.isConf = true
	_, err = Walletnotify(conf)
	if err != nil {
		t.Error(err)
	}
	testwalletpassphrase2(conf, d1)
	testsendmany(conf, d1, false)
}

func TestSend2(t *testing.T) {
	conf, d1 := preparetSend(t)
	conf.api = d1
	d1.setupTXs()
	if _, err := Walletnotify(conf); err != nil {
		t.Error(err)
	}
	testwalletpassphrase2(conf, d1)
	testsendfrom(conf, d1)
}

func TestSend3(t *testing.T) {
	conf, d1 := preparetSend(t)
	conf.api = d1
	d1.setupTXs()
	if _, err := Walletnotify(conf); err != nil {
		t.Error(err)
	}
	testwalletpassphrase2(conf, d1)
	testsendtoaddress(conf, d1)
}

func testwalletpassphrase1(conf *Conf, d1 *dummy1) error {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "walletpassphrase",
		Params:  []interface{}{"invalid", 60},
	}
	var resp Response
	return walletpassphrase(conf, req, &resp)
}

func testwalletpassphrase2(conf *Conf, d1 *dummy1) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "walletpassphrase",
		Params:  []interface{}{"test", float64(60)},
	}
	var resp Response
	if err := walletpassphrase(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	if resp.Result != nil {
		d1.t.Error("should be nil")
	}
}

func testsendmany(conf *Conf, d1 *dummy1, isErr bool) {
	adr1 := gadk.Address("A" + gadk.EmptyAddress[1:])
	adr2 := gadk.Address("B" + gadk.EmptyAddress[1:])
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "sendmany",
		Params: []interface{}{"ac1",
			map[string]interface{}{
				string(adr1.WithChecksum()): 0.1,
				string(adr2.WithChecksum()): 0.2,
			}},
	}
	var resp Response
	d1.broadcasted = nil
	d1.stored = nil
	var acc0 []Account
	err := db.Update(func(tx *bolt.Tx) error {
		var err error
		acc0, err = listAccount(tx)
		return err
	})
	if err != nil {
		d1.t.Error(err)
	}
	err = sendmany(conf, req, &resp)
	if isErr {
		if err == nil {
			d1.t.Error("should be error")
		}
		return
	}
	if err != nil {
		d1.t.Error(err)
	}
	var acc1 []Account
	err = db.Update(func(tx *bolt.Tx) error {
		var err error
		acc1, err = listAccount(tx)
		return err
	})
	diff := getDiff(acc0, acc1)
	checkResponse(diff, "ac1", d1, &resp, map[gadk.Address]int64{
		adr1: int64(0.1 * 100000000),
		adr2: int64(0.2 * 100000000),
	})
}

func testsendtoaddress(conf *Conf, d1 *dummy1) {
	adr1 := gadk.Address("A" + gadk.EmptyAddress[1:])
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "sendtoaddress",
		Params:  []interface{}{string(adr1.WithChecksum()), 0.1},
	}
	var resp Response
	d1.broadcasted = nil
	d1.stored = nil
	var acc0 []Account
	err := db.Update(func(tx *bolt.Tx) error {
		var err error
		acc0, err = listAccount(tx)
		return err
	})
	if err != nil {
		d1.t.Error(err)
	}
	err = sendtoaddress(conf, req, &resp)
	if err != nil {
		d1.t.Error(err)
	}
	var acc1 []Account
	err = db.Update(func(tx *bolt.Tx) error {
		var err error
		acc1, err = listAccount(tx)
		return err
	})
	diff := getDiff(acc0, acc1)
	checkResponse(diff, "*", d1, &resp, map[gadk.Address]int64{
		adr1: int64(0.1 * 100000000),
	})
}

func getDiff(acc0, acc1 []Account) map[gadk.Address]int64 {
	diff := make(map[gadk.Address]int64)

	bal0 := make(map[gadk.Address]int64)
	for _, ac := range acc0 {
		for _, b := range ac.Balances {
			bal0[b.Address] = b.Value
		}
	}
	bal1 := make(map[gadk.Address]int64)
	for _, ac := range acc1 {
		for _, b := range ac.Balances {
			bal1[b.Address] = b.Value + b.Change
		}
	}
	for adr, val := range bal0 {
		if v := bal1[adr] - val; v != 0 {
			diff[adr] = v
		}
	}
	for adr, val := range bal1 {
		if v := val - bal0[adr]; v != 0 {
			diff[adr] = v
		}
	}
	return diff
}

func checkResponse(diff map[gadk.Address]int64, acc string,
	d1 *dummy1, resp *Response, sendto map[gadk.Address]int64) {
	select {
	case <-d1.ch:
	case <-time.After(time.Minute):
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.(gadk.Trytes)
	if !ok {
		d1.t.Error("result must be slice")
	}
	if d1.broadcasted == nil || d1.stored == nil {
		d1.t.Error("must be broadcasted and stored", d1.broadcasted, d1.stored)
	}
	b := gadk.Bundle(d1.broadcasted)
	if err := b.IsValid(); err != nil {
		d1.t.Error(err)
	}
	if b.Hash() != result {
		d1.t.Error("invalid returned hash")
	}
	var acc1 []Account
	err := db.Update(func(tx *bolt.Tx) error {
		var err error
		acc1, err = listAccount(tx)
		return err
	})
	if err != nil {
		d1.t.Error(err)
	}

	nsend := 0
	nvalue := 0
	for i, tx := range d1.broadcasted {
		if d1.stored[i].Hash() != tx.Hash() {
			d1.t.Error("invalid trytes for store transaction")
		}
		if !tx.HasValidNonceMWM(15) {
			d1.t.Error("invalid nonce")
		}
		if tx.Value > 0 {
			v, ok := sendto[tx.Address]
			if ok {
				nsend++
				if tx.Value != v {
					d1.t.Error("invalid value")
				}
			} else {
				nvalue++
				v, ok := diff[tx.Address]
				if !ok {
					d1.t.Error("invalid change address", tx.Address)
				}
				if tx.Value != v {
					d1.t.Error("invalid value")
				}
			}
		}
		if tx.Value < 0 {
			nvalue++
			v, ok := diff[tx.Address]
			if !ok {
				d1.t.Error("invalid address", tx.Address)
			}
			if tx.Value != v {
				d1.t.Error("invalid value")
			}
			v, ok = d1.vals[tx.Address]
			if !ok {
				d1.t.Error("invalid send address")
			}
			if v != -tx.Value {
				d1.t.Error("invalid send address", v, tx.Value, tx.Address)
			}
			ac, ok := d1.adr2acc[tx.Address]
			if !ok {
				d1.t.Error("invalid account")
			}
			if acc != "*" && acc != ac {
				d1.t.Error("invalid account")
			}
		}
		if i != len(d1.broadcasted)-1 {
			if tx.TrunkTransaction != d1.broadcasted[i+1].Hash() {
			}
			if tx.BranchTransaction != trunk {
				d1.t.Error("invalid branch hash")
			}
		} else {
			if tx.TrunkTransaction != trunk {
				d1.t.Error("invalid trunk hash")
			}
			if tx.BranchTransaction != branch {
				d1.t.Error("invalid trunk hash")
			}
		}
	}
	if nvalue != len(diff) {
		d1.t.Error("invalid accout database", nvalue, len(diff))
	}
	if nsend != len(sendto) {
		d1.t.Error("invalid number of send address")
	}
}

func testsendfrom(conf *Conf, d1 *dummy1) {
	adr1 := gadk.Address("A" + gadk.EmptyAddress[1:])
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "sendfrom",
		Params:  []interface{}{"ac2", string(adr1.WithChecksum()), 0.1},
	}
	var resp Response
	d1.broadcasted = nil
	d1.stored = nil
	var acc0 []Account
	err := db.Update(func(tx *bolt.Tx) error {
		var err error
		acc0, err = listAccount(tx)
		return err
	})
	if err != nil {
		d1.t.Error(err)
	}
	err = sendfrom(conf, req, &resp)
	if err != nil {
		d1.t.Error(err)
	}
	var acc1 []Account
	err = db.Update(func(tx *bolt.Tx) error {
		var err error
		acc1, err = listAccount(tx)
		return err
	})
	diff := getDiff(acc0, acc1)
	checkResponse(diff, "ac2", d1, &resp, map[gadk.Address]int64{
		adr1: int64(0.1 * 100000000),
	})
}
