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
  	"os/exec"
  	"strings"

  	"github.com/AidosKuneen/gadk"
  	"github.com/boltdb-go/bolt"
  	shellwords "github.com/mattn/go-shellwords"
  )

  func compareHashes(api apis, hashes []gadk.Trytes) ([]gadk.Trytes, []gadk.Trytes, error) {
  	var confirmed []gadk.Trytes
  	var hs, news []*txstate
  	err := db.Update(func(tx *bolt.Tx) error {
  		//search new tx
  		var err error
  		hs, err = getHashes(tx)
  		if err != nil {
  			return err
  		}
  		news = make([]*txstate, 0, len(hashes))
  		nhashes := make([]gadk.Trytes, 0, len(hashes))
  		for _, h1 := range hashes {
  			exist := false
  			for _, h2 := range hs {
  				if h1 == h2.Hash {
  					exist = true
  					break
  				}
  			}
  			if !exist {
  				news = append(news, &txstate{Hash: h1})
  				nhashes = append(nhashes, h1)
  			}
  		}
  		trs, err := api.GetTrytes(nhashes)
  		if err != nil {
  			return err
  		}
  		for _, tr := range trs.Trytes {
  			if err := putTX(tx, &tr); err != nil {
  				return err
  			}
  		}
  		return nil
  	})
  	if err != nil {
  		return nil, nil, err
  	}

  	//search newly confirmed tx
  	confirmed = make([]gadk.Trytes, 0, len(hs))
  	hs = append(hs, news...)
  	ni, err2 := api.GetNodeInfo()
  	if err2 != nil {
  		return nil, nil, err2
  	}
  	for _, h := range hs {
  		if h.Confirmed {
  			continue
  		}
  		inc, err := api.GetInclusionStates([]gadk.Trytes{h.Hash}, []gadk.Trytes{ni.LatestMilestone})
  		if err != nil {
  			log.Println(err)
  			continue
  		}
  		if len(inc.States) > 0 && inc.States[0] {
  			confirmed = append(confirmed, h.Hash)
  			h.Confirmed = true
  		}
  	}

  	err = db.Update(func(tx *bolt.Tx) error {
  		return putHashes(tx, hs)
  	})
  	if err != nil {
  		return nil, nil, err
  	}

  	ret := make([]gadk.Trytes, len(news))
  	for i := range news {
  		ret[i] = news[i].Hash
  	}
  	return ret, confirmed, nil
  }

  var ignoreAddr []gadk.Address
	var addressCcheckPerformed bool = false
  //Walletnotify exec walletnotify scripts when receivng tx and tx is confirmed.
  func Walletnotify(conf *Conf) ([]string, error) {
  	log.Println("starting walletnotify...")
  	bdls := make(map[gadk.Trytes]struct{})
  	var acc []Account
  	var adrs []gadk.Address
  	err := db.View(func(tx *bolt.Tx) error {
  		//get all addresses
  		var err2 error
  		acc, err2 = listAccount(tx)
  		return err2
  	})
  	if err != nil {
  		return nil, err
  	}
  	if len(acc) == 0 {
  		log.Println("no address in wallet.")
  		return nil, nil
  	}
  	for _, ac := range acc {
  		for _, b := range ac.Balances {
				if !contains(ignoreAddr, b.Address){
  				adrs = append(adrs, b.Address)
				}
  		}
  	}
  	//get all trytes for all addresses

		var extras []gadk.Trytes
    chunksize := 100
		if len(adrs) < chunksize {
			chunksize = len(adrs)
		}
		for len(adrs) > 0 { // need to break it into 100 chunks
			  adrs_100 := adrs[0:chunksize]
				adrs = adrs[chunksize:]

				ft := gadk.FindTransactionsRequest{
					Addresses: adrs_100,
				}

				r, err := conf.api.FindTransactions(&ft)
				if err != nil {
					// invalid address. lets find it
					for adi, _ := range adrs_100 {
							 ftx := gadk.FindTransactionsRequest{
								 Addresses: adrs_100[adi:adi+1],
							 }
							 rx, errx := conf.api.FindTransactions(&ftx)
							 if (errx!=nil){
								 ignoreAddr = append(ignoreAddr,adrs_100[adi] )
							 } else {
								 extras = append(extras,rx.Hashes ...)
							 }
				  }
				} else {
					extras = append(extras,r.Hashes ...)
				}
				if len(adrs) < chunksize {
					chunksize = len(adrs)
				}
		}

		if len(extras) == 0 {
  		log.Println("no tx for addresses in wallet")
  		return nil, nil
  	}
		log.Println("Checking transactions: ",len(extras))
  	//get newly added and newly confirmed trytes.
  	news, confirmed, err := compareHashes(conf.api, extras)
  	if err != nil {
  		return nil, err
  	}
  	err = db.Update(func(tx *bolt.Tx) error {
  		if len(news) == 0 && len(confirmed) == 0 {
  			log.Println("no tx to be handled")
  			return nil
  		}
  		//add balances for all newly confirmed tx..
  		trs, err := getTXs(tx, confirmed)
  		if err != nil {
  			return err
  		}
  		for _, tr := range trs {
  			if tr.Value == 0 {
  				continue
  			}
  			bdls[tr.Bundle] = struct{}{}
  			acc, index, errr := findAddress(tx, tr.Address)
  			if errr != nil {
  				log.Println(errr)
  				continue
  			}
  			if acc == nil {
  				log.Println("acc shoud not be null")
  				continue
  			}
  			acc.Balances[index].Value += tr.Value
  			acc.Balances[index].Change = 0
  			if errrr := putAccount(tx, acc); err != nil {
  				return errrr
  			}
  		}
  		//add bundle hash to bdls.
  		nresp, err := conf.api.GetTrytes(news)
  		if err != nil {
  			return err
  		}
  		for _, tr := range nresp.Trytes {
  			if tr.Value != 0 {
  				bdls[tr.Bundle] = struct{}{}
  			}
  		}
  		return nil
  	})
  	if err != nil {
  		log.Println(err)
  		return nil, err
  	}
  	//exec cmds for all new txs. %s will be the bundle hash.
  	if conf.Notify == "" {
  		log.Println("end of walletnotify")
  		return nil, nil
  	}
  	result := make([]string, 0, len(bdls))
  	for bdl := range bdls {
  		cmd := strings.Replace(conf.Notify, "%s", string(bdl), -1)
  		args, err := shellwords.Parse(cmd)
  		if err != nil {
  			log.Println(err)
  			return nil, err
  		}
  		var out []byte
  		if len(args) == 1 {
  			out, err = exec.Command(args[0]).Output()
  		} else {
  			out, err = exec.Command(args[0], args[1:]...).Output()
  		}
  		if err != nil {
  			log.Println(err)
  			return nil, err
  		}
  		delete(bdls, bdl)
  		log.Println("executed ", cmd, ",output:", string(out))
  		result = append(result, string(out))
  	}
  	log.Println("end of walletnotify")
  	return result, nil
  }

	func contains(s []gadk.Address, str gadk.Address) bool {
			for _, v := range s {
				if string(v) == string(str) {
					return true
				}
			}

			return false
		}
