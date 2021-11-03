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
  	"log"
  
  	"github.com/AidosKuneen/gadk"
  )
  
  func getTrytes(api *gadk.API, resp1 *gadk.FindTransactionsResponse, txs map[gadk.Trytes]gadk.Transaction, counter map[gadk.Trytes]int) error {
  	var resp2 *gadk.GetTrytesResponse
  	var err error
  	log.Println("getting txs for bundle...")
  	for i := 0; i < 5; i++ {
  		resp2, err = api.GetTrytes(resp1.Hashes)
  		if err == nil {
  			break
  		}
  	}
  	if err != nil {
  		return err
  	}
  	for _, tx := range resp2.Trytes {
  		if tx.HasValidNonceMWM(18) {
  			txs[tx.Hash()] = tx
  			counter[tx.Hash()]++
  		}
  	}
  	return nil
  }
  
  func sendToWalletServers(apis []*gadk.API, txs map[gadk.Trytes]gadk.Transaction, counter map[gadk.Trytes]int) error {
  	var txss []gadk.Transaction
  	var err error
  
  	for h, tx := range txs {
  		if counter[h] < len(apis) {
  			log.Println("added", h, counter[h])
  			txss = append(txss, tx)
  		}
  	}
  
  	log.Println("number of txs", len(txss))
  	for _, api := range apis {
  		log.Println("broadcasting...")
  		for i := 0; i < 5; i++ {
  			if err = api.BroadcastTransactions(txss); err == nil {
  				break
  			}
  		}
  	}
  	return err
  }
  
  //Recast assembles unsent txs from public wallets servers and recast them to all wallets servers.
  func Recast(wallet string) error {
  	// TODO Remove hardcoded URLs
  	var wallets = []string{"http://wallet1.aidoskuneen.com:14266", "http://wallet2.aidoskuneen.com:14266"}
  
  	apis := make([]*gadk.API, 0, len(wallets)+1)
  	for _, w := range wallets {
  		apis = append(apis, gadk.NewAPI(w, nil))
  	}
  	apis = append(apis, gadk.NewAPI(wallet, nil))
  	var resp1 *gadk.GetTipsResponse
  	var resp2 *gadk.GetTrytesResponse
  	var resp3 *gadk.FindTransactionsResponse
  	txs := make(map[gadk.Trytes]gadk.Transaction)
  	counter := make(map[gadk.Trytes]int)
  	var bundles []gadk.Trytes
  	c := make(map[gadk.Trytes]struct{})
  
  	var err error
  	for j, api := range apis {
  		if j < len(wallets) {
  			log.Println("runnin on", wallets[j])
  		} else {
  			log.Println("runnin on", wallet)
  		}
  		log.Println("getting tips...")
  		for i := 0; i < 5; i++ {
  			resp1, err = api.GetTips()
  			if err == nil {
  				break
  			}
  		}
  		if err != nil {
  			return err
  		}
  		log.Println("getting txs for tips...")
  		for i := 0; i < 5; i++ {
  			resp2, err = api.GetTrytes(resp1.Hashes)
  			if err == nil {
  				break
  			}
  		}
  		if err != nil {
  			return err
  		}
  		for _, tx := range resp2.Trytes {
  			if _, ok := c[tx.Hash()]; !ok {
  				bundles = append(bundles, tx.Bundle)
  				c[tx.Hash()] = struct{}{}
  			}
  		}
  	}
  	for _, api := range apis {
  		log.Println("getting bundle hashes for tips...")
  		for i := 0; i < 5; i++ {
  			resp3, err = api.FindTransactions(&gadk.FindTransactionsRequest{
  				Bundles: bundles,
  			})
  			if err == nil {
  				break
  			}
  		}
  		if err != nil {
  			return err
  		}
  		if err := getTrytes(api, resp3, txs, counter); err != nil {
  			return err
  		}
  	}
  	return sendToWalletServers(apis, txs, counter)
  }
  