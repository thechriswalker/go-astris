package elgamal

import (
	big "github.com/ncw/gmp"
)

type dlogLookup struct {
	exp *big.Int
	log uint64
}

const dlogSparseLimit = 100000

// Note that in a national sized election the num of votes will be large
// meaning this table of lookups will be large.
// a 1024bit number has 128bytes and we need 2 for each entry in the table.
// so computing 50million votes needs ~ 6GB of memory. and scales linearly.
// this is a lot but not ridiculous.
// on the other hand there likely < 15 candidates so we could find the exp's
// first and lookup the results in one go, then we don't need to store them all
// but instead we have to compare every iteration.
// 50million * 15 calculations vs 6GB memory and 50million calculations.
// so we will initialise the table with the numbers we want, rather than holding them all in
// memory.
// do not modify targets after calling this.
// over course with small max it is easier just to generate all the values.
func DiscreteLogLookup(sys *System, max uint64, targets []*big.Int) func(n *big.Int) uint64 {
	if max < dlogSparseLimit {
		return lazyLookup(sys, max)
	}
	// we cannot key our map on the big.Int as it is a pointer.
	// that is we iterate until we have found them all.
	remaining := len(targets)
	found := make([]*dlogLookup, len(targets))
	last := big.NewInt(1)
	counter := uint64(0)
	for counter <= max && remaining > 0 {
		for i := range targets {
			if found[i] == nil && targets[i].Cmp(last) == 0 {
				found[i] = &dlogLookup{exp: new(big.Int).Set(last), log: counter}
				remaining--
			}
		}
		counter++
		last.Mul(last, sys.G)
		last.Mod(last, sys.P)
	}
	// now we have found them all!
	return func(n *big.Int) uint64 {
		for _, t := range found {
			if t.exp.Cmp(n) == 0 {
				return t.log
			}
		}
		panic("requested value not targetted")
	}
}

// the real problem with this is that the *big.Int is not comparable
// so we cannot make a "map" with the keys. on the other hand all values should be
// less than 2048bits so we could use a fixed length slice of the data...
func lazyLookup(sys *System, max uint64) func(n *big.Int) uint64 {
	cache := map[[256]byte]uint64{}
	last := big.NewInt(1)
	counter := uint64(0)
	key := func(x *big.Int) (b [256]byte) {
		bs := x.Bytes()
		copy(b[len(b)-len(bs):], bs)
		return
	}
	incr := func(x *big.Int) uint64 {
		for counter <= max*2 {
			counter++
			last.Mul(last, sys.G)
			last.Mod(last, sys.P)
			//	fmt.Printf("DLog lookup: counter:%d value:%s\n", counter, last.String())
			k := key(last)
			cache[k] = counter
			if x.Cmp(last) == 0 {
				return counter
			}
		}
		panic("lazy dlog max exceeded")
	}
	// initial value
	cache[key(last)] = counter

	return func(n *big.Int) uint64 {
		k := key(n)
		if x, ok := cache[k]; ok {
			return x
		}
		return incr(n)
	}
}
