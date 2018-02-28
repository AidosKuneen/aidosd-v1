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
	"log"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

var accountDB = []byte("accounts")

var lastAccount *Account

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

func (a *Account) search(adr gadk.Address) int {
	index := -1
	for i, bal := range a.Balances {
		if adr == bal.Address {
			index = i
		}
	}
	return index
}

func findAddress(tx *bolt.Tx, adr gadk.Address) (*Account, int, error) {
	if lastAccount != nil {
		i := lastAccount.search(adr)
		if i >= 0 {
			return lastAccount, i, nil
		}
	}
	b := tx.Bucket(accountDB)
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
		lastAccount = result
		break
	}
	return result, index, nil
}
func listAccount(tx *bolt.Tx) ([]Account, error) {
	var asc []Account
	// Assume bucket exists and has keys
	b := tx.Bucket(accountDB)
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
	if lastAccount != nil && lastAccount.Name == name {
		return lastAccount, nil
	}
	var ac Account
	b := tx.Bucket(accountDB)
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
	if lastAccount != nil && lastAccount.Name == acc.Name {
		lastAccount = acc
	}
	b, err := tx.CreateBucketIfNotExists(accountDB)
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
	return b.Put(toKey(acc.Name), bin)
}

//RefreshAccount refresh all hashes and accounts from address in address.
func RefreshAccount(conf *Conf) {
	log.Println("starting refresh...")
	// ni, err2 := conf.api.GetNodeInfo()
	// if err2 != nil {
	// 	log.Fatal(err2)
	// }
	err := db.Update(func(tx *bolt.Tx) error {
		acc, err2 := listAccount(tx)
		if err2 != nil {
			return err2
		}
		hs, err2 := getHashes(tx)
		if err2 != nil {
			return err2
		}
		for _, ac := range acc {
			log.Println("processing account", ac.Name)
			var adrs []gadk.Address
			for _, b := range ac.Balances {
				adrs = append(adrs, b.Address)
			}

			ft := gadk.FindTransactionsRequest{
				Addresses: adrs,
			}
			r, err2 := conf.api.FindTransactions(&ft)
			if err2 != nil {
				return err2
			}
			log.Println("updating hashes")
			for _, h1 := range r.Hashes {
				exist := false
				for _, h2 := range hs {
					if h1 == h2.Hash {
						exist = true
						break
					}
				}
				if !exist {
					hs = append(hs, &txstate{
						Hash: h1,
					})
				}
			}
			// log.Println("updating confirmation")
			// for i := range hs {
			// 	if i%10 == 0 {
			// 		log.Println("processing no.", i, "/", len(hs))
			// 	}
			// 	inc, err := conf.api.GetInclusionStates([]gadk.Trytes{hs[i].Hash}, []gadk.Trytes{ni.LatestMilestone})
			// 	if err != nil {
			// 		return err
			// 	}
			// 	hs[i].Confirmed = inc.States[0]
			// }
			log.Println("updating balance")
			bals, err2 := conf.api.Balances(adrs)
			if err2 != nil {
				return err2
			}
			for _, b := range bals {
				for _, ab := range ac.Balances {
					if b.Address == ab.Address {
						ab.Balance = b
					}
				}
			}
			if err := putAccount(tx, &ac); err != nil {
				return err
			}
		}
		return putHashes(tx, hs)
	})
	if err != nil {
		log.Fatal(err)
	}

}
