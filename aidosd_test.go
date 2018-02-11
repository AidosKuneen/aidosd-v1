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
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/AidosKuneen/aidosd/aidosd"
)

func spawn(t *testing.T) string {
	cdir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	fdb := filepath.Join(cdir, "aidosd.db")
	_ = os.Remove(fdb)
	cmd := exec.Command("go", "build", "-o", "adkd")
	if err = cmd.Run(); err != nil {
		t.Error(err)
	}
	cmd = exec.Command("go", "build", "-o", "adkd.exe")
	if err = cmd.Run(); err != nil {
		t.Error(err)
	}
	command := filepath.Join(cdir, "adkd")
	if err = runParent([]byte("test"), command); err != nil {
		t.Error(err)
	}
	stat, err := callStatus()
	if err != nil {
		t.Error(err)
	}
	if stat != working {
		t.Error("stat must bt working, but", stat)
	}
	return command
}

func TestAidosd(t *testing.T) {
	command := spawn(t)

	if err := callStop(); err != nil {
		t.Error(err)
	}
	time.Sleep(5 * time.Second)
	if _, err := callStatus(); err == nil {
		t.Error("should be error")
	}
	if err := runParent([]byte("test2"), command); err == nil {
		t.Error("should be error")
	}
	time.Sleep(5 * time.Second)
	if _, err := callStatus(); err == nil {
		t.Error("should be error")
	}
}

type postparam struct {
	body string
	resp interface{}
}

var (
	user = "test"
	pwd  = "test"
)

func (p *postparam) post() error {
	client := &http.Client{}

	auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + pwd))
	req, err := http.NewRequest("POST", "http://localhost:8332/", bytes.NewBuffer([]byte(p.body)))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if s, ok := p.resp.(*string); ok {
		*s = string(dat)
		return nil
	}
	return json.Unmarshal(dat, p.resp)
}

func TestAPIFee(t *testing.T) {

	spawn(t)
	str := ""
	setfee := &postparam{
		body: `{"jsonrpc": "1.0", "id":"curltest", "method": "settxfee", "params": [0.00001] }`,
		resp: &str,
	}
	user = "test2"
	pwd = "test2"
	if err := setfee.post(); err != nil {
		t.Error(err)
	}
	if str != "401 Unauthorized\n" {
		t.Error("should be error")
		t.Log(str)
	}

	user = "test"
	pwd = "test"
	resp := &struct {
		Result bool        `json:"result"`
		Error  *aidosd.Err `json:"error"`
		ID     string      `json:"id"`
	}{}
	setfee2 := &postparam{
		body: `{"jsonrpc": "1.0", "id":"curltest", "method": "settxfee", "params": [0.00001] }`,
		resp: resp,
	}

	if err := setfee2.post(); err != nil {
		t.Error(err)
	}
	//	`{"result":true,"error":null,"id":"curltest"}`,
	if resp.Error != nil {
		t.Error("should not be error")
	}
	if !resp.Result {
		t.Error("result should be true")
	}
	if resp.ID != "curltest" {
		t.Error("id must be curltest")
	}

	if err := callStop(); err != nil {
		t.Error(err)
	}

}

func TestMain(m *testing.M) {
	if _, err := os.Stat("aidosd.conf"); err == nil {
		os.Rename("aidosd.conf", "_aidosd.conf_")
	}
	err := ioutil.WriteFile("aidosd.conf", []byte(`
rpcuser=test
rpcpassword=test
rpcport=8332
walletnotify=echo %s
passphrase = true
#aidos_node = http://localhost:14266
testnet = true
aidos_node = http://78.46.250.88:15555
#testnet = false`), 0664)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	if _, err := os.Stat("_aidosd.conf_"); err == nil {
		os.Rename("_aidosd.conf_", "aidosd.conf")
	}
	os.Exit(code)
}
