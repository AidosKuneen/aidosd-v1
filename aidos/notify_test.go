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
	"log"
	"sort"
	"strings"
	"testing"

	"github.com/AidosKuneen/gadk"
)

func TestNotify1(t *testing.T) {
	conf := prepareTest(t)
	acc := make(map[string][]gadk.Address)
	for _, ac := range []string{"ac1"} {
		acc[ac] = newAddress(t, conf, ac)
	}
	d1 := newdummy(acc, t)
	conf.api = d1
	if err := check(conf, d1); err != nil {
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
	if err = check(conf, d1); err != nil {
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
	conf := prepareTest(t)
	acc := make(map[string][]gadk.Address)
	for _, ac := range []string{"ac1"} {
		acc[ac] = newAddress(t, conf, ac)
	}
	d1 := newdummy(acc, t)
	conf.api = d1
	if err := check(conf, d1); err != nil {
		t.Error(err)
	}
	d1.isConf = true
	conf.api = d1
	if err := check(conf, d1); err != nil {
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

func check(conf *Conf, d1 *dummy1) error {
	outs, err := Walletnotify(conf)
	if err != nil {
		return err
	}
	if len(outs) != 4 {
		log.Println(len(outs))
		return errors.New("invalid out1")
	}
	sort.Slice(outs, func(i, j int) bool {
		return strings.Compare(outs[i], outs[j]) < 0
	})
	res := []string{
		string(d1.bundle.Hash()),
		string("BA" + gadk.EmptyHash[2:]),
		string("BB" + gadk.EmptyHash[2:]),
		string("BC" + gadk.EmptyHash[2:]),
	}
	sort.Slice(res, func(i, j int) bool {
		return strings.Compare(res[i], res[j]) < 0
	})
	for i, o := range outs {
		o = strings.TrimSpace(o)
		if o != res[i] {
			return errors.New("invalid out2")
		}
	}
	return nil
}
