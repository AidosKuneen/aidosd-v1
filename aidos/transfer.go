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
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/AidosKuneen/gadk"
)

/*
we calulate spending values first without confirmation.
we will calulcate receiving values(including changes) after confirmation.
*/

func addOutputs(trs []gadk.Transfer) (gadk.Bundle, []gadk.Trytes, int64) {
	const sigSize = gadk.SignatureMessageFragmentTrinarySize / 3

	var bundle gadk.Bundle
	var frags []gadk.Trytes
	var total int64
	for _, tr := range trs {
		if len(tr.Message) > sigSize {
			panic("message size must be 0")
		}
		frags = append(frags, tr.Message)
		// Add first entries to the bundle
		// Slice the address in case the user provided a checksummed one
		bundle.Add(1, tr.Address, tr.Value, time.Now(), tr.Tag)
		// Sum up total value
		total += tr.Value
	}
	return bundle, frags, total
}

//PrepareTransfers gets an array of transfer objects as input,
//and then prepare the transfer by generating the correct bundle,
// as well as choosing and signing the inputs if necessary (if it's a value transfer).
func PrepareTransfers(api apis, ac *Account, trs []gadk.Transfer) (gadk.Bundle, error) {
	var err error

	bundle, frags, total := addOutputs(trs)
	// Get inputs if we are sending tokens
	if total <= 0 {
		// If no input required, don't sign and simply finalize the bundle
		bundle.Finalize(frags)
		return bundle, nil
	}

	if total > ac.totalValueWithChange() {
		return nil, errors.New("Not enough balance")
	}
	sufficient, err := addRemainder(api, &bundle, ac, total, false)
	if err != nil {
		return nil, err
	}
	if !sufficient {
		sufficient, err = addRemainder(api, &bundle, ac, total, true)
		if err != nil {
			return nil, err
		}
		if !sufficient {
			return nil, errors.New("insufficient balance")
		}
	}
	bundle.Finalize(frags)
	err = signInputs(ac, bundle)
	return bundle, err
}

func addRemainder(api apis, bundle *gadk.Bundle, ac *Account, total int64, useChange bool) (bool, error) {
	for i, bal := range ac.Balances {
		value := bal.Value
		if useChange {
			value = bal.Change
		}
		if value <= 0 {
			continue
		}
		// Add input as bundle entry
		bundle.Add(2, bal.Address, -value, time.Now(), gadk.EmptyHash)
		ac.Balances[i].Value -= value
		if useChange {
			ac.Balances[i].Change = 0
		}
		// If there is a remainder value
		// Add extra output to send remaining funds to
		if remain := value - total; remain > 0 {
			// If user has provided remainder address
			// Use it to send remaining funds to
			// Generate a new Address by calling getNewAddress
			adr, err := gadk.NewAddress(ac.Seed, len(ac.Balances), 2)
			if err != nil {
				return false, err
			}
			ac.Balances = append(ac.Balances, Balance{
				Balance: gadk.Balance{
					Address: adr,
				},
				Change: remain,
			})
			// Remainder bundle entry
			bundle.Add(1, adr, remain, time.Now(), gadk.EmptyHash)
			return true, nil
		}
		// If multiple inputs provided, subtract the totalTransferValue by
		// the inputs balance
		if total -= value; total == 0 {
			return true, nil
		}
	}
	return false, nil //balance is not sufficient
}

func signInputs(ac *Account, bundle gadk.Bundle) error {
	//  Get the normalized bundle hash
	nHash := bundle.Hash().Normalize()

	//  SIGNING OF INPUTS
	//
	//  Here we do the actual signing of the inputs
	//  Iterate over all bundle transactions, find the inputs
	//  Get the corresponding private key and calculate the signatureFragment
	for i, bd := range bundle {
		if bd.Value >= 0 {
			continue
		}
		// Get the corresponding keyIndex and security of the address
		index := -1
		for i, b := range ac.Balances {
			if b.Address == bd.Address {
				index = i
				break
			}
		}
		if index == -1 {
			return errors.New("cannot find address")
		}
		// Get corresponding private key of address
		key := gadk.NewKey(ac.Seed, index, 2)
		//  Calculate the new signatureFragment with the first bundle fragment
		bundle[i].SignatureMessageFragment = gadk.Sign(nHash[:27], key[:6561/3])

		// if user chooses higher than 27-tryte security
		// for each security level, add an additional signature
		//  Because the signature is > 2187 trytes, we need to
		//  find the subsequent transaction to add the remainder of the signature
		//  Same address as well as value = 0 (as we already spent the input)
		if bundle[i+1].Address == bd.Address && bundle[i+1].Value == 0 {
			//  Calculate the new signature
			nfrag := gadk.Sign(nHash[27:27*2], key[6561/3:2*6561/3])
			//  Convert signature to trytes and assign it again to this bundle entry
			bundle[i+1].SignatureMessageFragment = nfrag
		}
	}
	return nil
}

func doPow(tra *gadk.GetTransactionsToApproveResponse, depth int64, trytes []gadk.Transaction, mwm int64, pow gadk.PowFunc) error {
	var prev gadk.Trytes
	var err error
	for i := len(trytes) - 1; i >= 0; i-- {
		if i == len(trytes)-1 {
			trytes[i].TrunkTransaction = tra.TrunkTransaction
			trytes[i].BranchTransaction = tra.BranchTransaction
		} else {
			trytes[i].TrunkTransaction = prev
			trytes[i].BranchTransaction = tra.TrunkTransaction
		}
		trytes[i].Nonce, err = pow(trytes[i].Trytes(), int(mwm))
		if err != nil {
			return err
		}
		prev = trytes[i].Hash()
	}
	return nil
}

//PowTrytes does attachToMesh.
func PowTrytes(api apis, depth int64, trytes []gadk.Transaction, mwm int64, pow gadk.PowFunc) error {
	tra, err := api.GetTransactionsToApprove(depth)
	if err != nil {
		return err
	}
	if err := doPow(tra, depth, trytes, mwm, pow); err != nil {
		return err
	}
	if err := gadk.Bundle(trytes).IsValid(); err != nil {
		return err
	}
	for i, tr := range trytes {
		if !HasValidNonce(&tr, int(mwm)) {
			return fmt.Errorf("invlaid nonce for tx no %d", i)
		}
	}
	return nil
}

func broadcast(api apis, trytes []gadk.Transaction) error {
	// Broadcast and store tx
	if err := api.StoreTransactions(trytes); err != nil {
		return err
	}
	return api.BroadcastTransactions(trytes)
}

//HasValidNonce checks t's hash has valid MinWeightMagnitude.
func HasValidNonce(t *gadk.Transaction, mwm int) bool {
	h := t.Hash()
	for i := len(h) - 1; i > len(h)-1-mwm/3; i-- {
		if h[i] != '9' {
			return false
		}
	}
	return true
}

var pow gadk.PowFunc

func init() {
	_, pow = gadk.GetBestPoW()
}

var powMutex = sync.Mutex{}

//Send sends token.
//if you need to pow locally, you must specifiy pow func.
//otherwirse this calls AttachToMesh API.
func Send(conf *Conf, ac *Account, mwm int64, trs []gadk.Transfer) (gadk.Trytes, error) {
	bals := make([]Balance, len(ac.Balances))
	copy(bals, ac.Balances)
	bd, err := PrepareTransfers(conf.api, ac, trs)
	if err != nil {
		ac.Balances = bals
		return "", err
	}
	hash := bd.Hash()
	go func() {
		powMutex.Lock()
		defer powMutex.Unlock()
		log.Println("starting PoW...")
		ts := []gadk.Transaction(bd)
		for i := 0; ; i++ {
			err = PowTrytes(conf.api, gadk.Depth, ts, mwm, pow)
			if err == nil {
				break
			}
			log.Println(err, " waiting 3 minuites ", i)
			log.Println("failed to pow ", bd)
			time.Sleep(3 * time.Minute)
		}
		for i := 0; ; i++ {
			var bd2 gadk.Bundle
			if errr := broadcast(conf.api, ts); errr == nil {
				bd2 = gadk.Bundle(ts)
				log.Println("finish sending. bundle hash=", bd2.Hash())
				break
			}
			if _, ok := conf.api.(*gadk.API); ok && !conf.Testnet {
				for _, w := range []string{"http://wallet1.aidoskuneen.com:14266", "http://wallet2.aidoskuneen.com:14266"} {
					api2 := gadk.NewAPI(w, nil)
					if err := api2.BroadcastTransactions(ts); err != nil {
						log.Println(err)
					}
				}
			}
			log.Println("failed to send ", bd2)
			log.Println(err, " waiting 3 minuites ", i)
			time.Sleep(3 * time.Minute)
		}
		log.Println("finished PoW...")
	}()
	return hash, nil
}
