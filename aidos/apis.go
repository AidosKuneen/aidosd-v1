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
 	"github.com/AidosKuneen/gadk"
 	"github.com/boltdb-go/bolt"
 	"log"
 )
 
 func importwallet(conf *Conf, req *Request, res *Response) error {
 	mutex.Lock()
 	defer mutex.Unlock()
 	data, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("invalid params")
 	}
 	if len(data) != 1 {
 		return errors.New("invalid param length")
 	}
 	seed, ok := data[0].(string)
 	if !ok {
 		return errors.New("invalid seed")
 	}
 
 	log.Println("restoring from a seed...")
 	if seedTrytes, err := gadk.ToTrytes(seed); err == nil {
 		err := RestoreAddressesFromSeed(conf, seedTrytes)
 		if err != nil {
 			log.Printf("Error restoring from the seed: %v\n", err)
 
 			return err
 		}
 		RefreshAccount(conf)
 		log.Println("local database has been restored")
 	} else {
 		log.Printf("Error parsing the seed: %v\n", err)
 
 		return err
 	}
 
 	return nil
 }
 
 func getnewaddress(conf *Conf, req *Request, res *Response) error {
 	mutex.Lock()
 	defer mutex.Unlock()
 	data, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("invalid params")
 	}
 	acc := ""
 	switch len(data) {
 	case 1:
 		acc, ok = data[0].(string)
 		if !ok {
 			return errors.New("invalid txid")
 		}
 	case 0:
 	default:
 		return errors.New("invalid params")
 	}
 	return db.Update(func(tx *bolt.Tx) error {
 		ac, err := getAccount(tx, acc)
 		if err != nil {
 			return err
 		}
 		if ac == nil {
 			ac = &Account{
 				Name: acc,
 				Seed: gadk.NewSeed(),
 			}
 		}
 		// TODO Move "2" magic number to a constant
 		adr, err := gadk.NewAddress(ac.Seed, len(ac.Balances), 2)
 		if err != nil {
 			return err
 		}
 		ac.Balances = append(ac.Balances, Balance{
 			Balance: gadk.Balance{
 				Address: adr,
 			},
 		})
 		res.Result = adr.WithChecksum()
 		return putAccount(tx, ac)
 	})
 }
 func getBalance(api apis, tx *bolt.Tx) ([]Account, map[gadk.Address]int64, error) {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	acs, err := listAccount(tx)
 	if err != nil {
 		return nil, nil, err
 	}
 	var address []gadk.Address
 	for _, ac := range acs {
 		for _, b := range ac.Balances {
 			address = append(address, b.Address)
 		}
 	}
 	bals, err := api.Balances(address)
 	balmap := make(map[gadk.Address]int64)
 	for _, b := range bals {
 		balmap[b.Address] = b.Value
 	}
 	return acs, balmap, err
 }
 func listaddressgroupings(conf *Conf, req *Request, res *Response) error {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	var result [][][]interface{}
 	var r0 [][]interface{}
 	err := db.View(func(tx *bolt.Tx) error {
 		acs, balmap, err := getBalance(conf.api, tx)
 		if err != nil {
 			return err
 		}
 		for _, ac := range acs {
 			for _, b := range ac.Balances {
 				r1 := make([]interface{}, 3)
 				r1[0] = b.Address.WithChecksum()
 				r1[1] = 0
 				if v, ok := balmap[b.Address]; ok {
 					r1[1] = float64(v) / 100000000
 				}
 				r1[2] = ac.Name
 				r0 = append(r0, r1)
 			}
 		}
 		return nil
 	})
 	result = append(result, r0)
 	res.Result = result
 	return err
 }
 func getbalance(conf *Conf, req *Request, res *Response) error {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	data, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("param must be slice")
 	}
 	adrstr := "*"
 	switch len(data) {
 	case 3:
 		fallthrough
 	case 2:
 		n, okk := data[1].(float64)
 		if !okk {
 			return errors.New("invalid number")
 		}
 		if n == 0 {
 			return errors.New("not support unconfirmed transactions")
 		}
 		fallthrough
 	case 1:
 		adrstr, ok = data[0].(string)
 		if !ok {
 			return errors.New("invalid address")
 		}
 	case 0:
 	default:
 		return errors.New("invalid params")
 	}
 
 	err := db.View(func(tx *bolt.Tx) error {
 		acc, balmap, err := getBalance(conf.api, tx)
 		if err != nil {
 			return err
 		}
 		var total int64
 		if adrstr == "*" {
 			for _, v := range balmap {
 				total += v
 			}
 		} else {
 			var bal []Balance
 			for _, a := range acc {
 				if adrstr == a.Name {
 					bal = a.Balances
 				}
 			}
 			for _, b := range bal {
 				total += balmap[b.Address]
 			}
 		}
 		ftotal := float64(total) / 100000000
 		res.Result = ftotal
 		return nil
 	})
 	return err
 }
 func listaccounts(conf *Conf, req *Request, res *Response) error {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	ary, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("invalid param")
 	}
 	if len(ary) > 0 {
 		conf, ok := ary[0].(float64)
 		if !ok {
 			return errors.New("invalid param")
 		}
 		if conf == 0 {
 			return errors.New("not support unconfirmed transacton")
 		}
 	}
 	result := make(map[string]float64)
 	err := db.View(func(tx *bolt.Tx) error {
 		acs, err := listAccount(tx)
 		if err != nil {
 			return err
 		}
 		for _, ac := range acs {
 			var addresses []gadk.Address
 			for _, b := range ac.Balances {
 				addresses = append(addresses, b.Address)
 			}
 			bals, err := conf.api.Balances(addresses)
 			if err != nil {
 				return err
 			}
 			var sum int64
 			for _, b := range bals {
 				sum += b.Value
 			}
 			result[ac.Name] = float64(sum) / 100000000
 		}
 		return nil
 	})
 	res.Result = result
 	return err
 }
 
 type info struct {
 	IsValid      bool    `json:"isvalid"`
 	Address      string  `json:"address"`
 	ScriptPubKey string  `json:"scriptPubkey"`
 	IsMine       bool    `json:"ismine"`
 	IsWatchOnly  *bool   `json:"iswatchonly,omitempty"`
 	IsScript     *bool   `json:"isscript,omitempty"`
 	Pubkey       *string `json:"pubkey,omitempty"`
 	IsCompressed *bool   `json:"iscompressed,omitempty"`
 	Account      *string `json:"account,omitempty"`
 }
 
 //only 'isvalid' params is valid, others may be incorrect.
 func validateaddress(conf *Conf, req *Request, res *Response) error {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	data, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("invalid params")
 	}
 	if len(data) != 1 {
 		return errors.New("length of param must be 1")
 	}
 	adrstr, ok := data[0].(string)
 	if !ok {
 		return errors.New("invalid address")
 	}
 	valid := false
 	adr, err := gadk.ToAddress(adrstr)
 	if err == nil {
 		valid = true
 	}
 	var ac *Account
 	err = db.View(func(tx *bolt.Tx) error {
 		ac, _, err = findAddress(tx, adr)
 		return err
 	})
 
 	infoi := info{
 		IsValid: valid,
 		Address: adrstr,
 		IsMine:  false,
 	}
 	t := false
 	empty := ""
 	if ac != nil {
 		infoi.IsMine = true
 		infoi.Account = &ac.Name
 		infoi.IsWatchOnly = &t
 		infoi.IsScript = &t
 		infoi.Pubkey = &empty
 		infoi.IsCompressed = &t
 	}
 	res.Result = &infoi
 	return nil
 }
 
 func settxfee(conf *Conf, req *Request, res *Response) error {
 	res.Result = true
 	return nil
 }
 
 type details struct {
 	Account   string      `json:"account"`
 	Address   gadk.Trytes `json:"address"`
 	Category  string      `json:"category"`
 	Amount    float64     `json:"amount"`
 	Vout      int64       `json:"vout"`
 	Fee       float64     `json:"fee"`
 	Abandoned *bool       `json:"abandoned,omitempty"`
 }
 
 type tx struct {
 	Amount            float64     `json:"amount"`
 	Fee               float64     `json:"fee"`
 	Confirmations     int         `json:"confirmations"`
 	Blockhash         *string     `json:"blockhash,omitempty"`
 	Blockindex        *int64      `json:"blockindex,omitempty"`
 	Blocktime         *int64      `json:"blocktime,omitempty"`
 	Txid              gadk.Trytes `json:"txid"`
 	Walletconflicts   []string    `json:"walletconflicts"`
 	Time              int64       `json:"time"`
 	TimeReceived      int64       `json:"timereceived"`
 	BIP125Replaceable string      `json:"bip125-replaceable"`
 	Details           []*details  `json:"details"`
 	Hex               string      `json:"hex"`
 }
 
 func gettransaction(conf *Conf, req *Request, res *Response) error {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	data, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("invalid params")
 	}
 	bundlestr := ""
 	switch len(data) {
 	case 2:
 	case 1:
 		bundlestr, ok = data[0].(string)
 		if !ok {
 			return errors.New("invalid txid")
 		}
 	default:
 		return errors.New("invalid params")
 	}
 
 	var amount int64
 	nconf := 0
 	var dt *transaction
 	var detailss []*details
 	bundle := gadk.Trytes(bundlestr)
 	var allConfirmed bool = true
 
   // check inclusion state (live mesh lookup)
   var hashes_to_check []gadk.Trytes
 
 	err_check := db.View(func(tx *bolt.Tx) error {
 		trs, hs, err := findTX(tx, bundle)
 		if err != nil {
 			return err
 		}
 		hashes_to_check = make([]gadk.Trytes, 0, len(hs))
 		if len(trs) == 0 {
 			return errors.New("bundle not found")
 		}
     for i_check, tr_check_confirmed := range hs {
 			// if unconfirmed, trigger a live mesh lookup
 			if !hs[i_check].Confirmed {
 				allConfirmed = false; // have to trigger mesh lookup
 				hashes_to_check = append(hashes_to_check, tr_check_confirmed.Hash)
 			}
 		}
 		return nil
 	})
 	if err_check != nil {
 		return err_check
 	}
 
 	err := db.View(func(tx *bolt.Tx) error {
 		trs, hs, err := findTX(tx, bundle)
 		if err != nil {
 			return err
 		}
 		if len(trs) == 0 {
 			return errors.New("bundle not found")
 		}
 		detailss = make([]*details, 0, len(trs))
 		indice := make(map[int64]struct{})
 
 		// found some unconfirmed, so perform mesh lookup and DB Update
 		if !allConfirmed {
 			//get newly added and newly confirmed trytes direclty from mesh
 			ni, errNode := conf.api.GetNodeInfo()
 			if errNode != nil {
 				return nil
 			}
 			inc, err := conf.api.GetInclusionStates(hashes_to_check, []gadk.Trytes{ni.LatestMilestone})
 			if err != nil {
 				return err
 			}
 			if len(inc.States) > 0 {
 				for i, i_included := range inc.States {
 					if i_included {
 							hs[i].Confirmed = true
 					}
 				}
 			}
 		}
 
 		for i, tr := range trs {
 			dt2, errr := getTransaction(tx, conf, tr, hs[i].Confirmed)
 			if errr != nil {
 				return errr
 			}
 			if _, exist := indice[tr.CurrentIndex]; exist {
 				continue
 			}
 			indice[tr.CurrentIndex] = struct{}{}
 			dt = dt2
 			d := &details{
 				Account:   *dt.Account,
 				Address:   dt.Address,
 				Category:  dt.Category,
 				Amount:    dt.Amount,
 				Abandoned: dt.Abandoned,
 			}
 			if nconf == 0 {
 				nconf = dt.Confirmations
 			}
 			amount += tr.Value
 			detailss = append(detailss, d)
 		}
 		return nil
 	})
 	if err != nil {
 		return err
 	}
 	res.Result = &tx{
 		Amount:            float64(amount) / 100000000,
 		Confirmations:     nconf,
 		Blocktime:         dt.Blocktime,
 		Blockhash:         dt.Blockhash,
 		Blockindex:        dt.Blockindex,
 		Txid:              bundle,
 		Walletconflicts:   []string{},
 		Time:              dt.Time,
 		TimeReceived:      dt.TimeReceived,
 		BIP125Replaceable: "no",
 		Details:           detailss,
 	}
 	return nil
 }
 
 type transaction struct {
 	Account  *string     `json:"account"`
 	Address  gadk.Trytes `json:"address"`
 	Category string      `json:"category"`
 	Amount   float64     `json:"amount"`
 	// Label             string      `json:"label"`
 	Vout          int64   `json:"vout"`
 	Fee           float64 `json:"fee"`
 	Confirmations int     `json:"confirmations"`
 	Trusted       *bool   `json:"trusted,omitempty"`
 	// Generated         bool        `json:"generated"`
 	Blockhash       *string     `json:"blockhash,omitempty"`
 	Blockindex      *int64      `json:"blockindex,omitempty"`
 	Blocktime       *int64      `json:"blocktime,omitempty"`
 	Txid            gadk.Trytes `json:"txid"`
 	Walletconflicts []string    `json:"walletconflicts"`
 	Time            int64       `json:"time"`
 	TimeReceived    int64       `json:"timereceived"`
 	// Comment           string      `json:"string"`
 	// To                string `json:"to"`
 	// Otheraccount      string `json:"otheraccount"`
 	BIP125Replaceable string `json:"bip125-replaceable"`
 	Abandoned         *bool  `json:"abandoned,omitempty"`
 }
 
 //do not support over 1000 txs.
 func listtransactions(conf *Conf, req *Request, res *Response) error {
 	mutex.RLock()
 	defer mutex.RUnlock()
 	data, ok := req.Params.([]interface{})
 	if !ok {
 		return errors.New("invalid params")
 	}
 	acc := "*"
 	num := 10
 	skip := 0
 	switch len(data) {
 	case 4:
 		fallthrough
 	case 3:
 		n, okk := data[2].(float64)
 		if !okk {
 			return errors.New("invalid number")
 		}
 		skip = int(n)
 		fallthrough
 	case 2:
 		n, okk := data[1].(float64)
 		if !okk {
 			return errors.New("invalid number")
 		}
 		num = int(n)
 		fallthrough
 	case 1:
 		acc, ok = data[0].(string)
 		if !ok {
 			return errors.New("invalid account")
 		}
 	case 0:
 	default:
 		return errors.New("invalid params")
 	}
 	var ltx []*transaction
 	err := db.View(func(tx *bolt.Tx) error {
 		hs, err := getHashes(tx)
 		if err != nil {
 			return err
 		}
 		if len(hs) == 0 {
 			return nil
 		}
 		for skipped, i := 0, 0; i < len(hs) && len(ltx) < num; i++ {
 			//for replay bundles(i.e. multiple bundles with a same hash)
 			target := hs[len(hs)-1-i]
 			tr, err := getTX(tx, target.Hash)
 			if err != nil {
 				continue
 			}
 			inc := target.Confirmed
 			if !inc {
 				_, hs, err2 := findTX(tx, tr.Bundle)
 				if err2 != nil {
 					return err2
 				}
 				for _, h := range hs {
 					if h.Confirmed {
 						inc = true
 					}
 				}
 			}
 			dt, err := getTransaction(tx, conf, tr, inc)
 			if err != nil {
 				return err
 			}
 			if acc != "*" && *dt.Account != acc {
 				continue
 			}
 			if skipped++; skipped > skip {
 				ltx = append(ltx, dt)
 			}
 		}
 		return nil
 	})
 	res.Result = ltx
 	return err
 }
 
 func getTransaction(tx *bolt.Tx, conf *Conf, tr *gadk.Transaction, inc bool) (*transaction, error) {
 	ac, _, errr := findAddress(tx, tr.Address)
 	if errr != nil {
 		return nil, errr
 	}
 	f := false
 	emp := ""
 	var zero int64
 	dt := &transaction{
 		Address:           tr.Address.WithChecksum(),
 		Category:          "send",
 		Amount:            float64(tr.Value) / 100000000,
 		Txid:              tr.Bundle,
 		Walletconflicts:   []string{},
 		Time:              tr.Timestamp.Unix(),
 		TimeReceived:      tr.Timestamp.Unix(),
 		BIP125Replaceable: "no",
 		Abandoned:         &f,
 	}
 	if ac != nil {
 		dt.Account = &ac.Name
 	}
 	if inc {
 		dt.Blockhash = &emp
 		dt.Blocktime = &dt.Time
 		dt.Blockindex = &zero
 		dt.Confirmations = 100000
 	} else {
 		dt.Trusted = &f
 	}
 	if tr.Value > 0 {
 		dt.Category = "receive"
 		dt.Abandoned = nil
 	}
 	return dt, nil
 }
 