package elgamal

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/thechriswalker/go-astris/crypto/random"
)

// Signature is a Schnorr signature over an arbitrary message
type Signature struct {
	Ch, R *big.Int
}

// defined on System so it is present on public and private keys
func (s *System) createSigningChallenge(a *big.Int, msg []byte) *big.Int {
	// we concat a fixed prefix, the randomness and the message
	var commit bytes.Buffer
	fmt.Fprintf(&commit, "sig|%s|%s|", s.P.Text(16), a.Text(16))
	commit.Write(msg)
	// then hash and return the big.Int
	return random.Oracle(commit.Bytes(), s.Q)
}

// CreateSignature signs the given message with this key using Schnorr
func (sk *SecretKey) CreateSignature(msg []byte) (sig *Signature) {
	sig = new(Signature)
	w := random.Int(sk.Q)
	A := new(big.Int).Exp(sk.G, w, sk.P)
	//fmt.Println("A =", A)
	sig.Ch = sk.createSigningChallenge(A, msg)
	// the response is now (w - sk.X * chall) % Q
	sig.R = new(big.Int).Mul(sk.X, sig.Ch)
	sig.R.Sub(w, sig.R)
	sig.R.Mod(sig.R, sk.Q)
	return
}

// Signable interface represents an object that can be signed
type Signable interface {
	SignatureMessage() []byte
}

// Sign a signable object
func (sk *SecretKey) Sign(v Signable) *Signature {
	return sk.CreateSignature(v.SignatureMessage())
}

// Verify a signable object
func (sk *SecretKey) Verify(v Signable, s *Signature) error {
	return sk.VerifySignature(s, v.SignatureMessage())
}

// VerifySignature verifies a signature on a message
func (pk *PublicKey) VerifySignature(sig *Signature, message []byte) error {
	// rv = G^R * Y^C % P
	rv := new(big.Int).Exp(pk.G, sig.R, pk.P)
	//fmt.Println("g^r =", rv)
	yc := new(big.Int).Exp(pk.Y, sig.Ch, pk.P)
	//fmt.Println("y^c =", yc)
	rv.Mul(rv, yc)
	rv.Mod(rv, pk.P)
	//fmt.Println("A = (g^r * y^c) % p =", rv)
	expected := pk.createSigningChallenge(rv, message)
	//fmt.Println("Verify> expected:", expected, "\n          actual:", sig.Ch)
	if expected.Cmp(sig.Ch) != 0 {
		return fmt.Errorf("Signature invalid")
	}
	return nil
}
