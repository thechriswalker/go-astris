package elgamal

import (
	"bytes"
	"fmt"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto/random"
)

// Signature is a Schnorr signature over an arbitrary message
// based on https://tools.ietf.org/html/rfc8235
type Signature struct {
	C, R *big.Int
}

// defined on System so it is present on public and private keys
func (s *System) createSigningChallenge(V, A *big.Int, msg []byte) *big.Int {
	// we concat a fixed prefix, the randomness and the message
	var commit bytes.Buffer
	fmt.Fprintf(&commit, "sig|%x|%x|", V.Bytes(), A.Bytes())
	commit.Write(msg)
	// then hash and return the big.Int
	return random.Oracle(commit.Bytes(), s.Q)
}

// CreateSignature signs the given message with this key using Schnorr
func (sk *SecretKey) CreateSignature(msg []byte) (sig *Signature) {
	sig = new(Signature)
	v := random.Int(sk.Q)
	V := new(big.Int).Exp(sk.G, v, sk.P)
	//fmt.Println("V =", V)
	sig.C = sk.createSigningChallenge(V, sk.Y, msg)
	// the response is now (v - sk.X * C) % Q
	sig.R = new(big.Int).Mul(sk.X, sig.C)
	sig.R.Sub(v, sig.R)
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
func (pk *PublicKey) Verify(v Signable, s *Signature) error {
	return pk.VerifySignature(s, v.SignatureMessage())
}

// VerifySignature verifies a signature on a message
func (pk *PublicKey) VerifySignature(sig *Signature, message []byte) error {
	// we should validate the public key first...
	if err := pk.Validate(); err != nil {
		return fmt.Errorf("Signature invalid: public key not valid: %w", err)
	}
	// first we work out  g^r * A^c % p
	// g^r
	V := new(big.Int).Exp(pk.G, sig.R, pk.P)
	//fmt.Println("g^r =", V)
	// A^c
	Ac := new(big.Int).Exp(pk.Y, sig.C, pk.P)
	//fmt.Println("A^c =", Ac)
	// muliply, mod P
	V.Mul(V, Ac)
	V.Mod(V, pk.P)
	//fmt.Println("V = (g^r * A^c) % p =", V)
	expected := pk.createSigningChallenge(V, pk.Y, message)
	//fmt.Println("Verify> expected:", expected, "\n          actual:", sig.C)
	if expected.Cmp(sig.C) != 0 {
		return fmt.Errorf("Signature invalid: calculated challenge does not match expected")
	}
	return nil
}

var pokMessage = []byte("pok")

// we alias it so we can use the "right" one each time.
type ProofOfKnowledge = Signature

// ProofOfKnowledge generates a ZKP of knowledge of the secret key
// in essence it is just a signature with a fixed "message"
func (sk *SecretKey) ProofOfKnowledge() (pok *ProofOfKnowledge) {
	return sk.CreateSignature(pokMessage)
}

// VerifyProof a proof of knowledge of the secret key associated with the given public key.
func (pk *PublicKey) VerifyProof(pok *ProofOfKnowledge) error {
	return pk.VerifySignature(pok, pokMessage)
}
