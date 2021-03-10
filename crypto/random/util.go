package random

import (
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

// Int returns a random int <= max
func Int(max *big.Int) *big.Int {
	r, err := rand.Int(rand.Reader, max)
	if err != nil {
		// the rand.Reader is broken. Nothing we can do.
		panic(err)
	}
	return r
}

// Oracle is used for turning bytes into a random, but deterministic integer.
func Oracle(input []byte, max *big.Int) *big.Int {
	h := sha256.Sum256(input)
	var x big.Int
	x.SetBytes(h[:])
	x.Mod(&x, max)
	return &x
}

func SafePrimes(bits int) (*big.Int, *big.Int) {
	one, two := big.NewInt(1), big.NewInt(2)

	var q, p *big.Int
	var err error
	for {
		p, err = rand.Prime(rand.Reader, bits)
		// will only err on bad reader.
		if err != nil {
			panic(err)
		}
		// check is q = (p-1)/2 is prime
		q.Sub(p, one)
		q.Div(q, two)
		// we use 20 as that is what rand.Prime uses
		if q.ProbablyPrime(20) {
			return p, q
		}
	}
}
