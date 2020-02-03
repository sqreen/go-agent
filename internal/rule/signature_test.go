// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/rule"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestNewECDSAPublicKey(t *testing.T) {
	t.Run("invalid  format pubkey", func(t *testing.T) {
		_, err := rule.NewECDSAPublicKey(testlib.RandPrintableUSASCIIString(0, 100))
		require.Error(t, err)
	})

	t.Run("invalid pem pubkey", func(t *testing.T) {
		const publicKey string = `-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQA39oWMHR8sxb9LRaM5evZ7mw03iwJ
WNHuDeGqgPo1HmvuMfLnAyVLwaMXpGPuvbqhC1U65PG90bTJLpvNokQf0VMA5Tpi
m+NXwl7bjqa03vO/HExLbq3zBRysrZnC4OhJOF1jazkAg0psQOea2r5HcMcPHgMK
fnWXiKWnZX+uOWPuerE=
-----END PUBLIC KEY-----`
		_, err := rule.NewECDSAPublicKey(publicKey)
		require.Error(t, err)
	})

	t.Run("valid pem pubkey but not ecdsa", func(t *testing.T) {
		const publicKey string = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO
3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX
7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS
j+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd
OrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ
5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl
FQIDAQAB
-----END PUBLIC KEY-----`
		_, err := rule.NewECDSAPublicKey(publicKey)
		require.Error(t, err)
	})

	t.Run("valid ecdsa pem pubkey", func(t *testing.T) {
		pub, err := rule.NewECDSAPublicKey(config.PublicKey)
		require.NoError(t, err)
		require.NotNil(t, pub)
	})

}

func TestVerify(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	msg := []byte("hello, world")
	hash := sha512.Sum512(msg)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	require.NoError(t, err)

	t.Run("invalid signature", func(t *testing.T) {
		signature, err := asn1.Marshal(struct{ R, S *big.Int }{R: big.NewInt(0).Add(r, big.NewInt(33)), S: s})
		require.NoError(t, err)
		err = rule.Verify(&privateKey.PublicKey, hash[:], signature)
		require.Error(t, err)
	})

	t.Run("invalid asn1", func(t *testing.T) {
		err = rule.Verify(&privateKey.PublicKey, hash[:], []byte("oops"))
		require.Error(t, err)
	})

	t.Run("valid signature", func(t *testing.T) {
		signature, err := asn1.Marshal(struct{ R, S *big.Int }{R: r, S: s})
		require.NoError(t, err)
		err = rule.Verify(&privateKey.PublicKey, hash[:], signature)
		require.NoError(t, err)
	})
}
