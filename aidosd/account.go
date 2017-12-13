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

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

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

func toKey(name string) []byte {
	key := []byte(name)
	if name == "" {
		key = []byte{0}
	}
	return key
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
	v := b.Get(toKey(name))
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
	if err := b.Put(toKey(acc.Name), bin); err != nil {
		return err
	}
	return nil
}
