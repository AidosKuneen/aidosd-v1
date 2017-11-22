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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/AidosKuneen/aidosd/aidosd"
)

func main() {
	var isVerbose bool
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <options>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&isVerbose, "v", false, "print logs")
	flag.Parse()
	conf := aidosd.ParseConf()
	aidosd.SetLog(isVerbose)
	aidosd.SetDB()
	aidosd.Password()
	go func() {
		for {
			if err := aidosd.Walletnotify(conf); err != nil {
				log.Print(err)
			}
			time.Sleep(time.Minute)
		}
	}()
	log.Println("starting the aidosd server at port http://0.0.0.0:" + conf.RPCPort)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		aidosd.Handle(conf, w, r)

	})
	log.Fatal(http.ListenAndServe("0.0.0.0:"+conf.RPCPort, nil))
}
