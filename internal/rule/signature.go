// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"crypto/ecdsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"math/big"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

// NewECDSAPublicKey creates a ECDSA public key from a PEM public key.
func NewECDSAPublicKey(PEMPublicKey string) (*ecdsa.PublicKey, error) {
	// decode the key, assuming it's in PEM format
	block, _ := pem.Decode([]byte(PEMPublicKey))
	if block == nil {
		return nil, sqerrors.New("failed to decode the PEM public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, sqerrors.Wrap(err, "failed to parse ECDSA public key")
	}
	publicKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, sqerrors.Errorf("unexpected public key type `%T`", pub)
	}
	return publicKey, nil
}

// Verify returns a non-nil error when message verification against the public
// key failed, nil otherwise.
func Verify(publicKey *ecdsa.PublicKey, hash []byte, signature []byte) error {
	// unmarshal the R and S components of the ASN.1-encoded signature into our
	// signature data structure
	var sig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(signature, &sig); err != nil {
		return err
	}
	valid := ecdsa.Verify(
		publicKey,
		hash,
		sig.R,
		sig.S,
	)
	if !valid {
		return sqerrors.New("invalid signature")
	}
	// signature is valid
	return nil
}

// VerifyRuleSignature returns a non-nil error when the rule signature is
// invalid, nil otherwise.
func VerifyRuleSignature(r *api.Rule, publicKey *ecdsa.PublicKey) error {
	signature := r.Signature.ECDSASignature
	// first decode the signature to extract the DER-encoded byte string
	der, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return sqerrors.Wrap(err, "base64 decoding")
	}
	hash := sha512.Sum512([]byte(signature.Message))
	return Verify(publicKey, hash[:], der)
}
