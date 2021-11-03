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
  	"bytes"
  	"crypto/aes"
  	"crypto/cipher"
  	"crypto/rand"
  	"crypto/sha256"
  	"errors"
  	"io"
  
  	"github.com/boltdb-go/bolt"
  )
  
  var (
  	passPhrase = []byte("AidosKuneen") // Phrase that is encrypted/decrypted using a password
  	block      *aesCrypto
  	passDB     = []byte("pass_phrase") // Bucket name (in Bolt) for the encrypted passPhrase
  )
  
  type aesCrypto struct {
  	block  cipher.Block
  	pwd256 []byte
  }
  
  func newAESCrpto(pwd []byte) (*aesCrypto, error) {
  	pwd2 := sha256.Sum256(pwd)
  	pwd256 := pwd2[:]
  	block, err := aes.NewCipher(pwd256)
  	if err != nil {
  		return nil, err
  	}
  	return &aesCrypto{
  		block:  block,
  		pwd256: pwd256,
  	}, nil
  }
  
  func (a *aesCrypto) encrypt(pt []byte) []byte {
  	ct := make([]byte, aes.BlockSize+len(pt))
  	iv := ct[:aes.BlockSize]
  	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
  		panic(err)
  	}
  	encryptStream := cipher.NewCTR(a.block, iv)
  	encryptStream.XORKeyStream(ct[aes.BlockSize:], pt)
  	return ct
  }
  
  func (a *aesCrypto) decrypt(ct []byte) []byte {
  	pt := make([]byte, len(ct[aes.BlockSize:]))
  	decryptStream := cipher.NewCTR(a.block, ct[:aes.BlockSize])
  	decryptStream.XORKeyStream(pt, ct[aes.BlockSize:])
  	return pt
  }
  
  //Password gets a password, encrypt the defined passPhrase with it and saves the result in Bolt. When the Bolt bucket
  //bucket is available, it checks that the content can be decrypted using the provided password.
  func password(pwd []byte) error {
  	var err error
  	block, err = newAESCrpto(pwd)
  	if err != nil {
  		panic(err)
  	}
  	err = db.Update(func(tx *bolt.Tx) error {
  		var errr error
  		b := tx.Bucket(passDB)
  		if b == nil {
  			cipherText := block.encrypt(passPhrase)
  			b, errr = tx.CreateBucket(passDB)
  			if errr != nil {
  				return errr
  			}
  			return b.Put(passDB, cipherText)
  		}
  		ct := b.Get(passDB)
  		pt := block.decrypt(ct)
  		if !bytes.Equal(passPhrase, pt) {
  			return errors.New("incorrect password")
  		}
  		return nil
  	})
  	return err
  }
  