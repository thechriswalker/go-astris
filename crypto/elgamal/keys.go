package elgamal

import (
	"math/big"

	"../random"
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
	dk.Sig = deriveKey(system, dk.secret, new(big.Int).SetInt64(0))
	dk.Enc = deriveKey(system, dk.secret, new(big.Int).SetInt64(1))
	return
}

func deriveKey(sys *System, secret *big.Int, i *big.Int) *KeyPair {
	// We use our random Oracle and we feed it some of the system params
	// and our secret int and the I value.
	b := []byte("dk|")
	b = append(b, sys.P.Bytes()...)
	b = append(b, '|')
	b = append(b, secret.Bytes()...)
	b = append(b, '|')
	b = append(b, i.Bytes()...)

	return keypairForSecret(sys, random.Oracle(b, sys.Q))
}
