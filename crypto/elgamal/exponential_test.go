package elgamal

import (
	"testing"

	big "github.com/ncw/gmp"
	"github.com/thechriswalker/go-astris/crypto/random"
)

func TestExponential(t *testing.T) {
	eg := Astris2048()
	//eg := EightBit()
	kp := GenerateKeyPair(eg)

	// lets keep these numbers fairly low
	m1 := random.Int(big.NewInt(17))
	m2 := random.Int(big.NewInt(17))

	msum := new(big.Int).Add(m1, m2)

	t.Logf("m1=%s, m2=%s, msum=%s", m1, m2, msum)

	expm1 := new(big.Int).Exp(eg.G, m1, eg.P)
	expm2 := new(big.Int).Exp(eg.G, m2, eg.P)

	ct1 := kp.Public().Encrypt(expm1, nil)
	ct2 := kp.Public().Encrypt(expm2, nil)

	ctsum := ct1.Mul(eg, ct2)

	pt := kp.Secret().Decrypt(ctsum)

	t.Logf("Decrypted (exponential): %s", pt)

	// this should be g^ptsum
	//expptsum := new(big.Int).Exp(eg.G, msum, eg.Q)

	//if expptsum.Cmp(pt) != 0 {
	//t.Fatal("encrypt/decrypt failed")
	//}
	cache := NewPlaintextOptionsCache(eg)
	t.Logf("Options: %#v", cache.GetOptions(34))

	dlog := DiscreteLogLookup(eg, msum.Uint64(), []*big.Int{pt})

	recovered := dlog(pt)
	t.Logf("Recovered: %d", recovered)
	if recovered != msum.Uint64() {
		t.Fatal("discrete log lookup fail")
	}
}
