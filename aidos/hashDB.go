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

	"github.com/AidosKuneen/gadk"
	"github.com/boltdb/bolt"
)

var hashDB = []byte("hashes")

type txstate struct {
	Hash      gadk.Trytes
	Confirmed bool
}

func getHashes(tx *bolt.Tx) ([]*txstate, error) {
	var hs []*txstate
	b := tx.Bucket(hashDB)
	if b == nil {
		return nil, nil
	}
	v := b.Get(hashDB)
	if v != nil {
		if err := json.Unmarshal(v, &hs); err != nil {
			return nil, err
		}
	}
	return hs, nil
}
func putHashes(tx *bolt.Tx, hs []*txstate) error {
	b, err := tx.CreateBucketIfNotExists(hashDB)
	if err != nil {
		return err
	}
	bin, err := json.Marshal(hs)
	if err != nil {
		return err
	}
	return b.Put(hashDB, bin)
}

var txDB = []byte("transactions")

var errTxNotFound = errors.New("tx is not found")

var emptysig gadk.Trytes

func init() {
	for i := 0; i < gadk.SignatureSize/3; i++ {
		emptysig += "9"
	}
}

func getTX(tx *bolt.Tx, hash gadk.Trytes) (*gadk.Transaction, error) {
	b := tx.Bucket(txDB)
	if b == nil {
		return nil, errTxNotFound
	}
	v := b.Get([]byte(hash))
	if v == nil {
		return nil, errTxNotFound
	}
	trytes := emptysig + gadk.Trytes(v)
	return gadk.NewTransaction(trytes)
}

func getTXs(tx *bolt.Tx, hash []gadk.Trytes) ([]*gadk.Transaction, error) {
	b := tx.Bucket(txDB)
	if b == nil {
		return nil, errTxNotFound
	}
	ret := make([]*gadk.Transaction, 0, len(hash))
	for i := range hash {
		v := b.Get([]byte(hash[i]))
		if v == nil {
			return nil, errTxNotFound
		}
		trytes := emptysig + gadk.Trytes(v)
		tr, err := gadk.NewTransaction(trytes)
		if err != nil {
			return nil, err
		}
		ret = append(ret, tr)
	}
	return ret, nil
}
func putTX(tx *bolt.Tx, tr *gadk.Transaction) error {
	b, err := tx.CreateBucketIfNotExists(txDB)
	if err != nil {
		return err
	}
	trytes := tr.Trytes()[gadk.SignatureSize/3:]
	return b.Put([]byte(tr.Hash()), []byte(trytes))
}

func findTX(tx *bolt.Tx, bundle gadk.Trytes) ([]*gadk.Transaction, []*txstate, error) {
	b := tx.Bucket(txDB)
	if b == nil {
		return nil, nil, errTxNotFound
	}
	c := b.Cursor()
	var trs []*gadk.Transaction
	for k, v := c.First(); k != nil; k, v = c.Next() {
		trytes := emptysig + gadk.Trytes(v)
		tr, err := gadk.NewTransaction(trytes)
		if err != nil {
			return nil, nil, err
		}
		if tr.Bundle == bundle {
			trs = append(trs, tr)
		}
	}
	var hashes []*txstate
	hs, err := getHashes(tx)
	if err != nil {
		return nil, nil, err
	}
	for _, tr := range trs {
		var found bool
		trh := tr.Hash()
		for _, h := range hs {
			if h.Hash == trh {
				found = true
				hashes = append(hashes, h)
			}
		}
		if !found {
			return nil, nil, errors.New("hash not found for " + string(tr.Hash()))
		}
	}
	return trs, hashes, nil
}

//UpdateTXs update TX db from hashes DB.
func UpdateTXs(conf *Conf) error {
	log.Println("Updating transactions in DB...")
	err := db.Update(func(tx *bolt.Tx) error {
		hs, err := getHashes(tx)
		if err != nil {
			return err
		}
		var req []gadk.Trytes
		for _, h := range hs {
			_, err2 := getTX(tx, h.Hash)
			if err2 != errTxNotFound {
				if err2 != nil {
					return err2
				}
				continue
			}
			req = append(req, h.Hash)
		}
		if len(req) == 0 {
			return nil
		}
		resp, err := conf.api.GetTrytes(req)
		if err != nil {
			return err
		}
		for _, h := range resp.Trytes {
			if err := putTX(tx, &h); err != nil {
				return err
			}
		}
		return nil
	})
	log.Println("Update done.")
	return err
}
