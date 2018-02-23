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
	"math"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/AidosKuneen/gadk"
)

type dummy1 struct {
	acc2adr map[string][]gadk.Address
	adr2acc map[gadk.Address]string
	vals    map[gadk.Address]int64
	mtrytes map[gadk.Trytes]gadk.Transaction
	t       *testing.T
	isConf  bool
	ch      chan struct{}

	txs         map[gadk.Address][]*gadk.Transaction
	bundle      gadk.Bundle
	broadcasted []gadk.Transaction
	stored      []gadk.Transaction
}

func (d *dummy1) bundleAmount() float64 {
	var amount int64
	for _, tx := range d.bundle {
		amount += tx.Value
	}
	return float64(amount) / 100000000
}
func (d *dummy1) list(ac string, count, skip int) []*gadk.Transaction {
	var res []*gadk.Transaction
	sort.Slice(d.acc2adr[ac], func(i, j int) bool {
		return strings.Compare(string(d.acc2adr[ac][i]), string(d.acc2adr[ac][j])) > 0
	})
	for _, adr := range d.acc2adr[ac] {
		res = append(res, d.txs[adr]...)
	}
	for i := len(res)/2 - 1; i >= 0; i-- {
		res[i], res[len(res)-1-i] = res[len(res)-1-i], res[i]
	}
	return res[skip : skip+count]
}
func (d *dummy1) listall() []*gadk.Transaction {
	var adrs []gadk.Address
	for adr := range d.txs {
		adrs = append(adrs, adr)
	}
	sort.Slice(adrs, func(i, j int) bool {
		return strings.Compare(string(adrs[i]), string(adrs[j])) > 0
	})
	var res []*gadk.Transaction
	for _, adr := range adrs {
		res = append(res, d.txs[adr]...)
	}
	for i := len(res)/2 - 1; i >= 0; i-- {
		res[i], res[len(res)-1-i] = res[len(res)-1-i], res[i]
	}
	return res
}

func newdummy(accadr map[string][]gadk.Address, t *testing.T) *dummy1 {
	rand.Seed(time.Now().Unix())
	adr2acc := make(map[gadk.Address]string)
	for k, v := range accadr {
		for _, vv := range v {
			adr2acc[vv] = k
		}
	}
	d1 := &dummy1{
		acc2adr: accadr,
		adr2acc: adr2acc,
		vals:    make(map[gadk.Address]int64),
		mtrytes: make(map[gadk.Trytes]gadk.Transaction),
		t:       t,
		isConf:  false,
		ch:      make(chan struct{}),

		txs: make(map[gadk.Address][]*gadk.Transaction),
	}
	d1.setupTXs()
	return d1
}

func (d *dummy1) setupTXs() {
	c := []string{"A", "B", "C"}
	for adr := range d.adr2acc {
		var sum int64
		for i := 0; i < 5; i++ {
			val := int64(rand.Int31() - math.MaxInt32/2)
			sum += val
			tx := &gadk.Transaction{
				Address:   adr,
				Value:     val,
				Timestamp: time.Now().Add(time.Duration(rand.Int31()-math.MaxInt32/2) * time.Second),
				Bundle:    gadk.Trytes("B"+c[i%3]) + gadk.EmptyHash[2:],
			}
			if i == 4 {
				for sum < 0.2*100000000 {
					val = int64(rand.Int31())
					tx.Value += val
					sum += val
				}
			}
			d.txs[adr] = append(d.txs[adr], tx)
		}
		d.vals[adr] = sum
	}
	for i := 0; i < 5; i++ {
		tx := gadk.Transaction{
			Address:      gadk.EmptyAddress,
			Value:        int64(rand.Int31() - math.MaxInt32/2),
			Timestamp:    time.Now().Add(time.Duration(-rand.Int31()%100000) * time.Second),
			CurrentIndex: int64(i),
		}
		for tx.Address = range d.adr2acc {
		}
		if i == 3 {
			tx.Value = 0
		}
		d.vals[tx.Address] += tx.Value
		d.bundle = append(d.bundle, tx)
	}

	for i := range d.bundle {
		d.bundle[i].Bundle = d.bundle.Hash()
	}

}

func (d *dummy1) Balances(adr []gadk.Address) (gadk.Balances, error) {
	b := make(gadk.Balances, len(adr))
	for i, a := range adr {
		v, ok := d.vals[a]
		if !ok {
			d.t.Error("invalid adr in balances")
		}
		b[i] = gadk.Balance{
			Address: a,
			Value:   v,
		}
	}
	return b, nil
}

func (d *dummy1) FindTransactions(ft *gadk.FindTransactionsRequest) (*gadk.FindTransactionsResponse, error) {
	var res gadk.FindTransactionsResponse
	if ft.Addresses != nil {
		for _, tx := range d.bundle {
			res.Hashes = append(res.Hashes, tx.Hash())
		}
		sort.Slice(ft.Addresses, func(i, j int) bool {
			return strings.Compare(string(ft.Addresses[i]), string(ft.Addresses[j])) > 0
		})
		for _, a := range ft.Addresses {
			for _, tx := range d.txs[a] {
				res.Hashes = append(res.Hashes, tx.Hash())
			}
		}

	}
	return &res, nil
}
func (d *dummy1) GetTrytes(hashes []gadk.Trytes) (*gadk.GetTrytesResponse, error) {
	var res gadk.GetTrytesResponse
	for _, h := range hashes {
		exist := false
	loop:
		for _, txs := range d.txs {
			for _, tx := range txs {
				if tx.Hash() == h {
					res.Trytes = append(res.Trytes, *tx)
					exist = true
					break loop
				}
			}
		}
		if exist {
			continue
		}
	loop2:
		for _, tx := range d.bundle {
			if tx.Hash() == h {
				res.Trytes = append(res.Trytes, tx)
				exist = true
				break loop2
			}
		}
		if !exist {
			d.t.Error("invalid hashe in gettrytes", h)
		}
	}
	return &res, nil
}

var trunk = gadk.EmptyHash[:len(gadk.EmptyAddress)-1] + "T"
var branch = gadk.EmptyHash[:len(gadk.EmptyAddress)-1] + "B"

func (d *dummy1) GetTransactionsToApprove(depth int64) (*gadk.GetTransactionsToApproveResponse, error) {

	return &gadk.GetTransactionsToApproveResponse{
		TrunkTransaction:  trunk,
		BranchTransaction: branch,
	}, nil
}
func (d *dummy1) BroadcastTransactions(trytes []gadk.Transaction) error {
	d.broadcasted = trytes
	d.ch <- struct{}{}
	return nil
}
func (d *dummy1) StoreTransactions(trytes []gadk.Transaction) error {
	d.stored = trytes
	return nil
}
func (d *dummy1) GetInclusionStates(tx []gadk.Trytes, tips []gadk.Trytes) (*gadk.GetInclusionStatesResponse, error) {
	ret := make([]bool, len(tx))
	if d.isConf {
		for i := range ret {
			ret[i] = true
		}
	}

	return &gadk.GetInclusionStatesResponse{
		States: ret,
	}, nil
}
func (d *dummy1) GetNodeInfo() (*gadk.GetNodeInfoResponse, error) {
	return &gadk.GetNodeInfoResponse{}, nil
}
