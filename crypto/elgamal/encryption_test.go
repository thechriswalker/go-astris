package elgamal

import (
	"testing"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto/random"
)

func TestEncryption(t *testing.T) {
	eg := DH2048modp256()
	kp := GenerateKeyPair(eg)
	m := random.Int(eg.Q)
	ct := kp.Public().Encrypt(m, nil)
	pt := kp.Secret().Decrypt(ct)

	if pt.Cmp(m) != 0 {
		t.Fatal("encrypt/decrypt failed")
	}
}

func TestHomomorphism(t *testing.T) {
	eg := DH2048modp256()

	kp := GenerateKeyPair(eg)

	// this is a bit complex
	testAdd := func(expected uint64, values ...uint64) {
		agg := &CipherText{}
		n := new(big.Int)
		for _, v := range values {
			// exponentiate g^v%p
			n.SetUint64(v)
			n.Exp(eg.G, n, eg.P)
			enc := kp.Public().Encrypt(n, nil)
			agg.Mul(eg, enc)
		}
		// decrypt.
		dec := kp.Secret().Decrypt(agg)

		// max is expected value as that should match
		pt := DiscreteLogLookup(eg, expected, []*big.Int{dec})(dec)
		//t.Logf("SUM(%v) = %d", values, pt)
		if expected != pt {
			t.Logf("Addition failed sum(%v), expected:%d, got:%d", values, expected, pt)
			t.Fail()
		}
	}
	testAdd(0, 0, 0)
	testAdd(1, 0, 0, 0, 0, 1, 0, 0, 0)
	testAdd(10, 0, 1, 2, 3, 4, 0)
	testAdd(10, 0, 1, 0, 2, 0, 3, 0, 4, 0)
	testAdd(7, 0, 4, 0, 2, 0, 1)
}
