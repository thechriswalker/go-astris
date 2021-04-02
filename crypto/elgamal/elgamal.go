package elgamal

import (
	"fmt"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto"
	"github.com/thechriswalker/go-astris/crypto/random"
)

// System represents the parameters for an ElGamal Cryptosystem
type System struct {
	P, Q, G *big.Int
}

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
	bigTwo  = big.NewInt(2)
)

// New creates a new ElGamal system with a prime of n-bits
// this is very slow for large primes (>1024bits)
// and sub-prime q = (p-1)/2
func New(bits int) (sys *System) {
	sys = &System{}
	sys.P, sys.Q = random.SafePrimes(bits)
	// find g
	var test big.Int
	for {
		sys.G = random.Int(sys.P)
		if test.Exp(sys.G, sys.Q, sys.P).Cmp(bigOne) == 0 {
			break
		}
	}
	return
}

// Validate checks the system params are OK. That is that
// P = Q * 2 +1 and that P and Q are (probably) prime
// and that G satisfies the exponentation test
func (s *System) Validate() error {
	// we don't know how many bits the prime should be here,
	// so just check for primeness
	// now q and p must be prime
	if !s.P.ProbablyPrime(20) {
		return fmt.Errorf("ElGamal System Invalid: p is not prime")
	}
	if !s.Q.ProbablyPrime(20) {
		return fmt.Errorf("ElGamal System Invalid: q is not prime")
	}
	// check that q divides p-1 (maybe 2?)
	pMinusOne := new(big.Int).Sub(s.P, bigOne)
	if new(big.Int).Rem(pMinusOne, s.Q).Cmp(bigZero) != 0 {
		return fmt.Errorf("ElGamal System Invalid: q does not divide p-1")
	}
	// now check g^q = 1 mod p
	if new(big.Int).Exp(s.G, s.Q, s.P).Cmp(bigOne) != 0 {
		return fmt.Errorf("ElGamal System invalid: g^q != 1 mod p")
	}
	return nil
}

// PublicKey is an ElGamal public key for encryption and signature verification
type PublicKey struct {
	*System
	Y *big.Int
}

func (pk *PublicKey) String() string {
	return fmt.Sprintf("pk:Y=%s", crypto.BigIntToJSON(pk.Y))
}

// SecretKey is an ElGamal secret key for decryption and signature creation
type SecretKey struct {
	*PublicKey
	X *big.Int
}

func (sk *SecretKey) String() string {
	return fmt.Sprintf("sk:X=%s", crypto.BigIntToJSON(sk.X))
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
// var agg = &CipherText{}
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
		ct = &CipherText{}
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

func (ct *CipherText) Equals(other *CipherText) bool {
	cmpA, cmpB := ct.A.Cmp(other.A), ct.B.Cmp(other.B)
	return cmpA == 0 && cmpB == 0
}

func (ct *CipherText) String() string {
	return fmt.Sprintf("CipherText[A=%s, B=%s]", ct.A, ct.B)
}

// Encrypt a plaintext with the public key and randomness r
func (pk *PublicKey) Encrypt(pt *big.Int, r *big.Int) (ct *CipherText) {
	ct = new(CipherText)
	if r == nil {
		r = random.Int(pk.Q)
	} else {
		// ensure in bounds
		r.Mod(r, pk.Q)
	}
	// set alpha to g^r mod p
	ct.A = new(big.Int).Exp(pk.G, r, pk.P)
	// set beta to (m * (h^r mod p)) mod p
	ct.B = new(big.Int).Exp(pk.Y, r, pk.P) // h^r mod p
	ct.B.Mul(ct.B, pt)                     // m * prev
	ct.B.Mod(ct.B, pk.P)                   // prev mod p
	return
}

// Validate that the Y value is within range for the system params
func (pk *PublicKey) Validate() error {
	// all we know is that the Y value should be an element of Z_p.
	// and we should know the system by this time in order to verify
	if pk.System == nil {
		return fmt.Errorf("PublicKey invalid: No ElGamal System Parameters")
	}
	// our signature and ZKP scheme requires y \in [1, p-1]
	if pk.Y.Cmp(bigOne) == -1 {
		// Y is less than one
		return fmt.Errorf("PublicKey invalid: y < 1")
	}
	if pk.Y.Cmp(pk.P) != -1 {
		// Y >= P i.e. Y > p-1
		return fmt.Errorf("PublicKey invalid: y > p-1")
	}
	return nil
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

// Validate that the X value is within range for the system params
// and that the PublicKey is correct (or generate it!)
func (sk *SecretKey) Validate() error {
	if sk.System == nil {
		return fmt.Errorf("SecretKey invalid: No ElGamal System Parameters")
	}
	// check X is element of Z_q
	// the secret key is from range [0, q-1]
	if sk.X.Cmp(bigZero) == -1 {
		return fmt.Errorf("SecretKey invalid: x < 0")
	}
	if sk.X.Cmp(sk.Q) != -1 {
		return fmt.Errorf("SecretKey invalid: x > q-1")
	}

	// and check public key
	if sk.PublicKey == nil {
		sk.PublicKey = &PublicKey{System: sk.System, Y: new(big.Int).Exp(sk.G, sk.X, sk.P)}
	} else {
		if err := sk.PublicKey.Validate(); err != nil {
			return fmt.Errorf("SecretKey invalid: %w", err)
		}
	}
	return nil
}
