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

package aidosd

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/AidosKuneen/gadk"
)

func TestNotify1(t *testing.T) {
	cdir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	fdb := filepath.Join(cdir, "aidosd.db")
	if err := os.Remove(fdb); err != nil {
		t.Log(err)
	}
	conf, err := Prepare("../aidosd.conf", []byte("test"))
	if err != nil {
		t.Error(err)
	}
	acc := make(map[string][]gadk.Address)
	vals := make(map[gadk.Address]int64)
	for _, ac := range []string{"ac1"} {
		adr := newAddress(t, conf, ac)
		for _, a := range adr {
			acc[ac] = append(acc[ac], a)
			vals[a] = int64(rand.Int31())
		}
	}
	d1 := &dummy1{
		acc2adr: acc,
		vals:    vals,
		mtrytes: make(map[gadk.Trytes]gadk.Transaction),
		t:       t,
		isConf:  false,
	}
	conf.api = d1
	d1.setupTXs()
	if err := check(conf); err != nil {
		t.Error(err)
	}

	outs, err := Walletnotify(conf)
	if err != nil {
		t.Error(err)
	}
	if len(outs) != 0 {
		t.Error("invalid out")
	}

	d1.isConf = true
	if err := check(conf); err != nil {
		t.Error(err)
	}

	outs, err = Walletnotify(conf)
	if err != nil {
		t.Error(err)
	}
	if len(outs) != 0 {
		t.Error("invalid out")
	}
}

func TestNotify2(t *testing.T) {
	cdir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	fdb := filepath.Join(cdir, "aidosd.db")
	if err := os.Remove(fdb); err != nil {
		t.Log(err)
	}
	conf, err := Prepare("../aidosd.conf", []byte("test"))
	if err != nil {
		t.Error(err)
	}
	acc := make(map[string][]gadk.Address)
	vals := make(map[gadk.Address]int64)
	for _, ac := range []string{"ac1"} {
		adr := newAddress(t, conf, ac)
		for _, a := range adr {
			acc[ac] = append(acc[ac], a)
			vals[a] = int64(rand.Int31())
		}
	}
	d1 := &dummy1{
		acc2adr: acc,
		vals:    vals,
		mtrytes: make(map[gadk.Trytes]gadk.Transaction),
		t:       t,
		isConf:  true,
	}
	conf.api = d1
	d1.setupTXs()
	if err := check(conf); err != nil {
		t.Error(err)
	}

	outs, err := Walletnotify(conf)
	if err != nil {
		t.Error(err)
	}
	if len(outs) != 0 {
		t.Error("invalid out")
	}
}

func check(conf *Conf) error {
	outs, err := Walletnotify(conf)
	if err != nil {
		return err
	}
	if len(outs) != 3 {
		return errors.New("invalid out")
	}
	sort.Slice(outs, func(i, j int) bool {
		return outs[i][1] < outs[j][1]
	})
	res := []string{
		string("BA" + gadk.EmptyHash[2:]),
		string("BB" + gadk.EmptyHash[2:]),
		string("BC" + gadk.EmptyHash[2:]),
	}
	for i, o := range outs {
		o = strings.Trim(o, "\n")
		if o != res[i] {
			return errors.New("invalid out")
		}
	}
	return nil
}
