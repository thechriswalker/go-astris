package elgamal

import (
	"math/big"

	"../random"
)

// System represents the parameters for an ElGamal Cryptosystem
type System struct {
	P, Q, G *big.Int
}

var (
	one = big.NewInt(1)
)

// New creates a new ElGamal system with a prime of n-bits
// this is very slow for large primes (>1024bits)
func New(nbits int) (sys *System) {
	sys = &System{}
	sys.P, sys.Q = random.SafePrimes(nbits)
	// find g
	var test big.Int
	for {
		sys.G = random.Int(sys.P)
		if test.Exp(sys.G, sys.Q, sys.P).Cmp(one) == 0 {
			break
		}
	}
	return
}

// PublicKey is an ElGamal public key for encryption and signature verification
type PublicKey struct {
	*System
	Y *big.Int
}

// SecretKey is an ElGamal secret key for decryption and signature creation
type SecretKey struct {
	*PublicKey
	X *big.Int
}

// CipherText is the output of encryption of a plaintext
type CipherText struct {
	A, B *big.Int
}

// Mul does a homomorphic multiplication of two cipher texts
// we assume they were created with the same system
// this property will be enforced externally by the ZKP that
// the text's encode a 0 or 1 in the given system
// this function mutates the reciever and is designed to be
// part of an aggregation, so the canonical usage is:
//
// var agg *CipherText
// agg.Mul(sys, other1) // first round simple sets to "other1"
// agg.Mul(sys, other2) // now set to other1 * other2
//
// NB that in order to do a homomorphic _addition_ (which is what we want)
// we must use the Exponential ElGamal (encoding g^m instead of m) and
// then using this same method for combining them. After decryption
// we must then find the discrete log see `expontential.go` for the code
// and the discrete log table used for lookups.
//
func (ct *CipherText) Mul(sys *System, other *CipherText) *CipherText {
	// if this is a "zero" cipher text then just update with the other
	if ct == nil {
		*ct = CipherText{}
	}
	if ct.A == nil {
		ct.A = new(big.Int).Set(other.A)
		ct.B = new(big.Int).Set(other.B)
	} else {
		// we need to add
		ct.A.Mul(ct.A, other.A)
		ct.A.Mod(ct.A, sys.P)
		ct.B.Mul(ct.B, other.B)
		ct.B.Mod(ct.B, sys.P)
	}
	return ct
}

// Encrypt a plaintext with the public key and randomness r
func (pk PublicKey) Encrypt(pt *big.Int, r *big.Int) (ct *CipherText) {
	ct = new(CipherText)
	if r == nil {
		r = random.Int(pk.Q)
	}
	// set alpha to g^r mod p
	ct.A = new(big.Int).Exp(pk.G, r, pk.P)
	// set beta to (m * (h^r mod p)) mod p
	ct.B = new(big.Int).Exp(pk.Y, r, pk.P) // h^r mod p
	ct.B.Mul(ct.B, pt)                     // m * prev
	ct.B.Mod(ct.B, pk.P)                   // prev mod p
	return
}

// Decrypt a ciphertext with this single key, no threshold work here
func (sk *SecretKey) Decrypt(ct *CipherText) (pt *big.Int) {
	// this is a single key decryption.
	pt = new(big.Int)
	// s = alpha^x
	pt.Exp(ct.A, sk.X, sk.P)
	// s^-1
	pt.ModInverse(pt, sk.P)
	// s^-1 * beta
	pt.Mul(pt, ct.B)
	pt.Mod(pt, sk.P)
	return
}
