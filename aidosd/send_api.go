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
	"sync"
	"time"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

var privileged bool
var mutex sync.RWMutex

func send(acc string, conf *Conf, trs []gadk.Transfer) (gadk.Trytes, error) {
	var mwm int64 = 18
	if conf.Testnet {
		mwm = 15
	}
	var result gadk.Trytes
	err := db.Update(func(tx *bolt.Tx) error {
		var ac *Account
		var err error
		if acc != "*" {
			ac, err = getAccount(tx, acc)
		} else {
			acs, errr := listAccount(tx)
			if errr != nil {
				return errr
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
		bhash, err := Send(conf.api, ac, mwm, trs)
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
	if len(data) < 2 || len(data) > 5 {
		return errors.New("invalid param length")
	}
	acc, ok := data[0].(string)
	if !ok {
		return errors.New("invalid account")
	}
	t, ok := data[1].(string)
	if !ok {
		return errors.New("param must be a  map string")
	}
	target := make(map[string]float64)
	if err := json.Unmarshal([]byte(t), &target); err != nil {
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
	if len(data) < 3 || len(data) > 6 {
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
	if len(data) > 5 || len(data) < 2 {
		return errors.New("invalid params")
	}
	adrstr, ok := data[0].(string)
	if !ok {
		return errors.New("invalid address")
	}
	value, ok := data[1].(float64)
	if !ok {
		return errors.New("invalid value")
	}
	tr.Address, err = gadk.ToAddress(adrstr)
	if err != nil {
		return err
	}

	tr.Value = int64(value * 100000000)
	res.Result, err = send("*", conf, []gadk.Transfer{tr})
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
	if len(data) != 2 {
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
