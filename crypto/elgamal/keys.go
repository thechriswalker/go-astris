package elgamal

import (
	"bytes"
	"fmt"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto/random"
)

// DerivedKeys are deterministically derived keys from
// an initial secret
type DerivedKeys struct {
	*System
	secret *big.Int // Secret value.
	Sig    *KeyPair
	Enc    *KeyPair
}

type KeyPair struct {
	sk *SecretKey
}

// Secret gets the private part of this keypair
func (kp *KeyPair) Secret() *SecretKey {
	return kp.sk
}

// Public gets the public half of this keypair
func (kp *KeyPair) Public() *PublicKey {
	return kp.sk.PublicKey
}

// GenerateKeyPair creates a new random key pair
func GenerateKeyPair(sys *System) *KeyPair {
	return keypairForSecret(sys, random.Int(sys.Q))
}

func keypairForSecret(sys *System, x *big.Int) (kp *KeyPair) {
	kp = new(KeyPair)
	y := new(big.Int).Exp(sys.G, x, sys.P)
	kp.sk = &SecretKey{
		PublicKey: &PublicKey{System: sys, Y: y},
		X:         x,
	}
	return
}

// DeriveKeys creates all the KeyPairs from the given secret, or create a new
// random secret
func DeriveKeys(system *System, secret *big.Int) (dk *DerivedKeys) {
	dk = new(DerivedKeys)
	dk.System = system
	if secret == nil {
		dk.secret = random.Int(system.P)
	} else {
		dk.secret = new(big.Int).Set(secret)
	}
	dk.Sig = deriveKey(system, dk.secret, "sig")
	dk.Enc = deriveKey(system, dk.secret, "enc")
	return
}

func deriveKey(sys *System, secret *big.Int, kind string) *KeyPair {
	// We use our random Oracle and we feed it some of the system params
	// and our secret int and the I value.
	var b bytes.Buffer
	fmt.Fprintf(&b, "dk|%x|%x|%s", sys.P.Bytes(), secret.Bytes(), kind)
	return keypairForSecret(sys, random.Oracle(b.Bytes(), sys.Q))
}
