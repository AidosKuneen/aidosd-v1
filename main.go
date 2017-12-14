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
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/AidosKuneen/aidosd/aidosd"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	stopping = byte(iota)
	working

	controlURL = "127.0.0.1:33631"
)

//Version is aidosd's version. It shoud be overwritten when building on travis.
var Version = "unstable"

func main() {
	aidosd.SetLog(true)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "aidosd version %v\n", Version)
		fmt.Fprintf(os.Stderr, "%s <options>\n", os.Args[0])
		flag.PrintDefaults()
	}
	var cmd string
	flag.StringVar(&cmd, "cmd", "start", "command")
	flag.Parse()

	switch cmd {
	case "child":
		if err := runChild(); err != nil {
			panic(err)
		}
	case "start":
		passwd := getPasswd()
		if err := runParent(passwd, os.Args...); err != nil {
			panic(err)
		}
		fmt.Println("aidosd is started")
	case "status":
		stat, err := callStatus()
		if err != nil {
			fmt.Println("aidosd is not running")
		}
		switch stat {
		case working:
			fmt.Println("aidosd is working")
		case stopping:
			fmt.Println("aidosd is stopping")
		default:
			fmt.Println("unknown status")
		}
	case "stop":
		if err := callStop(); err != nil {
			panic(err)
		}
		fmt.Println("aidosd has stopped")
	default:
		fmt.Println("unknown cmd")
	}
}

func callStatus() (byte, error) {
	var stat byte
	err := call("Control.Status", &struct{}{}, &stat)
	return stat, err
}

func callStop() error {
	return call("Control.Stop", &struct{}{}, &struct{}{})
}

//Control is a struct for controling child.
type Control struct {
	status byte
}

//Start starts aidosd with password.
func (c *Control) Start(r *http.Request, args *[]byte, reply *struct{}) error {
	conf, err := aidosd.Prepare("aidosd.conf", *args)
	if err != nil {
		return err
	}
	go func() {
		for {
			if _, err := aidosd.Walletnotify(conf); err != nil {
				log.Print(err)
			}
			time.Sleep(time.Minute)
		}
	}()
	fmt.Println("starting the aidosd server at port http://0.0.0.0:" + conf.RPCPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		aidosd.Handle(conf, w, r)
	})
	go func() {
		if err := http.ListenAndServe("0.0.0.0:"+conf.RPCPort, mux); err != nil {
			log.Println(err)
		}
	}()
	c.status = working
	return nil
}

//Stop stops aidosd.
func (c *Control) Stop(r *http.Request, args *struct{}, reply *struct{}) error {
	aidosd.Exit()
	c.status = stopping
	return nil
}

//Status returns if aidosd is working or stopping.
func (c *Control) Status(r *http.Request, args *struct{}, reply *byte) error {
	*reply = c.status
	return nil
}

func call(method string, args interface{}, ret interface{}) error {
	url := "http://" + controlURL + "/control"
	message, err := json.EncodeClientRequest(method, args)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error in sending request to %s. %s", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	return json.DecodeClientResponse(resp.Body, ret)
}

func runParent(passwd []byte, oargs ...string) error {
	args := []string{"--cmd", "child"}
	args = append(args, oargs[1:]...)
	cmd := exec.Command(oargs[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)

	return call("Control.Start", &passwd, &struct{}{})
}

func getPasswd() []byte {
	fmt.Print("Enter password: ")
	pwd, err := terminal.ReadPassword(int(syscall.Stdin)) //int conversion is needed for win
	fmt.Println("")
	if err != nil {
		panic(err)
	}
	return pwd
}

func runChild() error {
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	if err := s.RegisterService(new(Control), ""); err != nil {
		panic(err)
	}
	http.Handle("/rpc", s)

	mux := http.NewServeMux()
	mux.Handle("/control", s)
	log.Println("started  control server on aidosd...")
	return http.ListenAndServe(controlURL, mux)
}
