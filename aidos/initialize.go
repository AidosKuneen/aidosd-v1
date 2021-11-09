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
    "bufio"
    "fmt"
    "os"
    "net/http"
    "time"
    "crypto/rand"
    "strings"
    "math/big"
    "strconv"
    "golang.org/x/term"
    "syscall"
  )

  var conf_g *(Conf)

  func DBExists()(bool){
    if _, err := os.Stat("aidosd.db"); err==nil {
      return true
    } else {
      return false
    }
  }

  func InitializeWallet()(error) {
    SetLog(true)
    if _, err := os.Stat("aidosd.conf"); errors.Is(err, os.ErrNotExist) {
      // path/to/aidosd.conf does not exist
      log.Fatal("ERROR: aidosd.conf does not exist. please create first.")
      return err
    }
    if _, err := os.Stat("aidosd.db"); err==nil {
      log.Fatal("ERROR: aidosd.db already exists. please delete the old database first.")
      // path/to/aidosd.db exists, check account exists
      return err
    }

    passwd := getPasswd()
    var err error
    conf_g, err = Prepare("aidosd.conf", passwd);
    if (err != nil){
      log.Panic(err)
    }

    // if we get here, we need a new account
    // no account exists, so lets set one up
    log.Println("######### WELCOME TO AIDOSD NEW ACCOUNT SETUP #######")
    log.Println("## You are seeing this because there is no account yet set up in your database")
    log.Println("## ")
    log.Println("## Let's get started. ")
    log.Println("## ")
    log.Println("## Enter (1) if you want to generate a NEW ACCOUNT/SEED (i.e. let aidosd generate a new seed for you), or")
    log.Println("## Enter (2) if you want to IMPORT an EXISTING SEED.")
    fmt.Print("## Type (1) or (2): ")
    char := "0"
    for {
      reader := bufio.NewReader(os.Stdin)
      char_s, _ := reader.ReadString('\n')
      char = char_s[0:1]
      if (char == "1" || char == "2") {
        break;
      }
      log.Println(" ")
      log.Println(" *** Invalid input")
      fmt.Print("## Type (1) or (2): ")
    }
    log.Println("Generating seed... ")
    if (char=="1"){ // NEW SEED
       seed, _ := GenerateRandomSEED(81)
       log.Println("Generating account from random new seed")
       seedTrytes, _ := gadk.ToTrytes(seed);
       err := SetupNewAddresses(conf_g, seedTrytes)
     	 if err != nil {
     			log.Fatal("Error initializing seed: %v\n", err)
       }
       log.Println("## Local database has been initialized..")
       log.Println("##")
       log.Println("## PLEASE WRITE DOWN YOUR SEED FOR BACKUP: ")
       log.Println("")
       log.Println("Seed:",seed)
       log.Println("")
    } else if char == "2" {
        log.Println("ENTER THE SEED OF THE WALLET YOU WANT TO RECOVER:")
        fmt.Print("-> ")
        reader := bufio.NewReader(os.Stdin)
        seed, _ := reader.ReadString('\n')
        // convert CRLF to LF
        seed = strings.Replace(seed, "\n", "", -1)
        seed = strings.Replace(seed, "\r", "", -1)
        log.Println("Restoring from a seed...")
      	if seedTrytes, err := gadk.ToTrytes(seed); err == nil {
      		err := ScanAndRestoreAddresses(conf_g, seedTrytes)
      		if err != nil {
      			log.Printf("Error restoring from the seed: %v\n", err)
      			return err
      		}
      		log.Println("local database has been restored")
      	} else {
      		log.Printf("Error parsing the seed: %v\n", err)
      		return err
      	}
    }
    SetLog(false)
    return nil
  }

  func GenerateRandomSEED(n int) (string, error) {
  	const letters = "9ABCDEFGHIJKLMNOPQRSTUVWXYZ"
  	ret := make([]byte, n)
  	for i := 0; i < n; i++ {
  		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
  		if err != nil {
  			return "", err
  		}
  		ret[i] = letters[num.Int64()]
  	}

  	return string(ret), nil
  }

  func ScanAndRestoreAddresses(conf *Conf, seed gadk.Trytes) error {
    acc := ""
    db.Update(func(tx *bolt.Tx) error {
      ac := &Account{
        Name: acc,
        Seed: seed,
      }

      log.Println("Checking for Address Balances.")
      fmt.Print("Enter how many addresses you want to scan (default 50000): ")
      reader := bufio.NewReader(os.Stdin)
      numaddrs_s, _ := reader.ReadString('\n')
      numaddrs_s = strings.Replace(numaddrs_s, "\n", "", -1)
      numaddrs_s = strings.Replace(numaddrs_s, "\r", "", -1)
      numaddrs, err_s := strconv.Atoi(numaddrs_s)
      if err_s != nil || numaddrs < 1 {
          numaddrs = 50000
      }
      chunksize := 8000
      addr_checked := 0
      var addr_chunk []gadk.Address
      var addr_allbal []gadk.Balance
      highest_index_with_balance := 0
      var total int64 = 0
      for (addr_checked < numaddrs) {
          //load chunks
          addr_chunk = []gadk.Address {}
          if (numaddrs - addr_checked < chunksize) {
            chunksize = numaddrs - addr_checked
          }
          log.Println("Checking Addresses ",addr_checked," to ",(addr_checked+chunksize),"...")
          for i := 0; i < chunksize; i++ {
            adr, _ := gadk.NewAddress(seed, addr_checked+i, 2)
            addr_chunk = append(addr_chunk, adr)
          }
          // get balance index
          var netClient = &http.Client{
            Timeout: time.Second * 120,
          }
          api := gadk.NewAPI(conf.Node, netClient)

          bals, err := allBalances(api, addr_chunk)
          if (err != nil){
            log.Panic("Error when calling api.Balances. node reachable?")
          }
        	for idxbal, b := range bals {
        		if (b.Value > 0) {
              log.Println("Found address with balance at index ",addr_checked+idxbal, "(",b.Value,")")
              highest_index_with_balance = addr_checked+idxbal
              total += b.Value
            }
        	}
          addr_allbal = append(addr_allbal, bals...) // remember all
          addr_checked += chunksize
      }
      log.Println("Total balance found: ",(total/100000000)," ADK (",total," uADK), highest address index with balance: ",highest_index_with_balance)
      log.Println("")
      log.Println("Now loading known accounts into database...")

      for idxbal, b := range addr_allbal {
        if idxbal > highest_index_with_balance {
          break
        }
        ac.Balances = append(ac.Balances, Balance { Balance: b })
      }
      return putAccount(tx, ac)
    })
    log.Println("Load complete. Now relaoding all transacations that already exist in the mesh, and store in DB")
    log.Println("Please be patient, this can take a while...")
    RefreshAccount(conf)
    log.Println("TX Load complete.")
    log.Println("Updating confirmation states (without notify shell call) Part 1")
    UpdateConfirmationState(conf)
    log.Println("Updating confirmation states (without notify shell call) Part 2")
    conf.Notify = ""
    Walletnotify(conf)
    log.Println("Confirmation state update complete.")
    return nil
  }

  func SetupNewAddresses(conf *Conf, seed gadk.Trytes) error {
    acc := ""
    return db.Update(func(tx *bolt.Tx) error {
      ac := &Account{
        Name: acc,
        Seed: seed,
      }

      var addr_chunk []gadk.Address
      adr, _ := gadk.NewAddress(seed, 0, 2) // create one address
      log.Println("got first address. Validating on mesh... pelase wait.")
      addr_chunk = append(addr_chunk, adr)
      var netClient = &http.Client{
        Timeout: time.Second * 120,
      }
      api := gadk.NewAPI(conf.Node, netClient)
      bals, err := allBalances(api, addr_chunk)
      if (err != nil){
        log.Panic("Error when calling api.Balances. node reachable?")
      }
      ac.Balances = append(ac.Balances, Balance{
  			Balance: bals[0],
  		})
      return putAccount(tx, ac)
    })
    log.Println("Load complete.")
    return nil
  }

  func getPasswd() []byte {
  	fmt.Print("Enter password: ")
  	pwd, err := term.ReadPassword(int(syscall.Stdin)) //int conversion is needed for win
  	log.Println("")
  	if err != nil {
  		panic(err)
  	}
  	return pwd
  }


  //Balances call GetBalances API and returns address-balance pair struct.
  func allBalances(api *gadk.API, adr []gadk.Address) (gadk.Balances, error) {
  	r, err := api.GetBalances(adr, 100)
  	if err != nil {
  		return nil, err
  	}
  	bs := make(gadk.Balances, 0, len(adr))
  	for i, bal := range r.Balances {
  		//if bal <= 0 {
//  			continue
  //		}
  		b := gadk.Balance{
  			Address: adr[i],
  			Value:   bal,
  		}
  		bs = append(bs, b)
  	}
  	return bs, nil
  }

  func UpdateConfirmationState(conf *Conf) (error) {
    var acc []Account
    var adrs []gadk.Address
    err := db.View(func(tx *bolt.Tx) error {
      //get all addresses
      var err2 error
      acc, err2 = listAccount(tx)
      return err2
    })

    if err != nil {
      return  err
    }
    if len(acc) == 0 {
      log.Println("no address in wallet.")
      return  nil
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
    chunksize := 8000
    if len(adrs) < chunksize {
      chunksize = len(adrs)
    }
    cntall := len(adrs)
    for len(adrs) > 0 { // need to break it into 100 chunks
        adrs_100 := adrs[0:chunksize]
        adrs = adrs[chunksize:]
        log.Println("Checking transactions for addresses ",(cntall-len(adrs))-chunksize," to ",(cntall-len(adrs)))
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
      return nil
    }
    log.Println("Storing transactions: ",len(extras), " please wait")
    //get newly added and newly confirmed trytes.
    compareHashes(conf.api, extras)

    return nil
  }
