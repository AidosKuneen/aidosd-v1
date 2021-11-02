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
	"log"
	"math"

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb-go/bolt"
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

var globalAccountNo int = -1

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
	if (globalAccountNo == -1){ //no account unspecified
			return asc, nil
		}
	return asc[globalAccountNo:globalAccountNo+1], nil // return specific account slice
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

func RestoreAddressesFromSeed(conf *Conf, seed gadk.Trytes) error {
	acc := ""

	return db.Update(func(tx *bolt.Tx) error {
		ac, err := getAccount(tx, acc)
		if err != nil {
			return err
		}
		if ac != nil {
			return errors.New("an account already exists")
		}
		ac = &Account{
			Name: acc,
			Seed: seed,
		}

		var count int
		for i := 0; i < math.MaxInt32; i++ {
			// TODO Move "2" magic number to a constant
			adr, err := gadk.NewAddress(ac.Seed, i, 2)
			if err != nil {
				break
			}

			resp, err := conf.api.FindTransactions(&gadk.FindTransactionsRequest{
				Addresses: []gadk.Address{adr},
			})
			if err != nil {
				break
			}
			if len(resp.Hashes) == 0 {
				// OK, this is the last one, break the search and remember i
				count = i

				break
			}
		}

		addresses, err := gadk.NewAddresses(ac.Seed, 0, count, 2)
		if err != nil {
			return err
		}
		for _, adr := range addresses {
			ac.Balances = append(ac.Balances, Balance{
				Balance: gadk.Balance{
					Address: adr,
				},
			})
		}
		return putAccount(tx, ac)
	})
}

func ListAndSelectAccount(conf  *Conf){
	  globalAccountNo = conf.accountNo
	  log.Println("Checking for multiple accounts: ")
		db.Update(func(tx *bolt.Tx) error {
			acc, err2 := listAccount(tx)
			if err2 != nil {
				return err2
			}
			var cnt int = 0
			for idx, ac := range acc {
				// get known balance
				bal := ac.totalValueWithChange()
				log.Printf("Account found: Account number %v : %s, Balance: %v \n", idx, ac.Name, bal)
				cnt++
			}
			if cnt > 1 && conf.accountNo == -1 {
				log.Fatal("\n**************************\nERROR: More than one account found! Please specify the account to use in aidosd.conf: e.g. account_no=0  \n\n**********************\n  ")
			} else if conf.accountNo >= 0 {
				log.Println("Account selected by aidosd.conf: ", conf.accountNo)
			}
			return nil
		})

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
				for i, ab := range ac.Balances {
					if b.Address == ab.Address {
						ac.Balances[i].Balance = b
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

//ResetDB reset hashes and balances (basically remove all the hashes and set balances to 0)
func ResetDB(conf *Conf) {
	err := db.Update(func(tx *bolt.Tx) error {
		var hs []*txstate
		if err := putHashes(tx, hs); err != nil {
			return err
		}
		acc, err2 := listAccount(tx)
		if err2 != nil {
			return err2
		}
		for _, ac := range acc {
			for i := range ac.Balances {
				ac.Balances[i].Balance.Value = 0
			}
			if err := putAccount(tx, &ac); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
