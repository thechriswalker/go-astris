package random

import (
	"crypto/rand"
	"crypto/sha256"

	gbig "math/big"

	big "github.com/ncw/gmp"
)

// Int returns a random int <= max
func Int(max *big.Int) *big.Int {
	r, err := rand.Int(rand.Reader, new(gbig.Int).SetBytes(max.Bytes()))
	if err != nil {
		// the rand.Reader is broken. Nothing we can do.
		panic(err)
	}
	return new(big.Int).SetBytes(r.Bytes())
}

// Oracle is used for turning bytes into a random, but deterministic integer.
func Oracle(input []byte, max *big.Int) *big.Int {
	h := sha256.Sum256(input)
	var x big.Int
	x.SetBytes(h[:])
	x.Mod(&x, max)
	return &x
}

// SafePrimes returns two primes P and Q where P is pbits bits
// and P = 2Q + 1
// Note that not all ElGamal uses primes of this form, but if we
// generate them, it is safest to use this method.
func SafePrimes(bits int) (*big.Int, *big.Int) {
	one := gbig.NewInt(1)
	alpha := gbig.NewInt(2)

	q, p := new(gbig.Int), new(gbig.Int)
	var err error
	for {
		p, err = rand.Prime(rand.Reader, bits)
		// will only err on bad reader.
		if err != nil {
			panic(err)
		}
		// check is q = (p-1)/alpha is prime
		q.Sub(p, one)
		q.Div(q, alpha)
		// we use 20 as that is what rand.Prime uses
		if q.ProbablyPrime(20) {
			P := new(big.Int).SetBytes(p.Bytes())
			Q := new(big.Int).SetBytes(q.Bytes())
			return P, Q
		}
	}
}
