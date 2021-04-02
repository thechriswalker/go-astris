package elgamal

import (
	"math/rand"
	"testing"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto/random"
)

func TestBAKZKP(t *testing.T) {
	eg := EightBit()
	kp := GenerateKeyPair(eg)

	r := random.Int(eg.Q)

	plainZero := big.NewInt(0)
	plainZero.Exp(eg.G, plainZero, eg.P)

	plainOne := big.NewInt(1)
	plainOne.Exp(eg.G, plainOne, eg.P)

	ct := kp.Public().Encrypt(plainZero, r)

	zkpDec := ProveDecryption(kp.Secret(), ct)
	errDec := VerifyDecryptionProof(zkpDec, kp.Public(), ct, plainZero)
	if errDec != nil {
		t.Logf("Decryption Verification Fail: %s", errDec)
		t.Fail()
	}

	options := []*big.Int{
		plainZero,
		plainOne,
	}
	meta := []byte("test")
	zkpEnc := ProveEncryption(kp.Public(), ct, options, 0, r, meta)
	errEnc := VerifyEncryptionProof(zkpEnc, kp.Public(), ct, options, meta)
	if errEnc != nil {
		t.Logf("Encryption Verification Fail: %s", errEnc)
		t.Fail()
	}

}

func TestZKP(t *testing.T) {
	//eg := EightBit()
	eg := Astris2048()
	kp := GenerateKeyPair(eg)
	//kp := keypairForSecret(eg, big.NewInt(33))

	r := random.Int(eg.Q)

	plainA := random.Int(eg.P)
	plainA.Exp(eg.G, plainA, eg.P)

	plainB := random.Int(eg.P)
	plainB.Exp(eg.G, plainB, eg.P)

	// which to encrpt?
	choice := rand.Intn(2)

	var plain *big.Int
	if choice == 0 {
		plain = plainA
	} else {
		plain = plainB
	}

	ct := kp.Public().Encrypt(plain, r)

	zkpDec := ProveDecryption(kp.Secret(), ct)
	errDec := VerifyDecryptionProof(zkpDec, kp.Public(), ct, plain)
	if errDec != nil {
		t.Logf("Decryption Verification Fail: %s", errDec)
		t.Fail()
	}

	options := []*big.Int{
		plainA,
		plainB,
	}
	meta := []byte("test")
	zkpEnc := ProveEncryption(kp.Public(), ct, options, choice, r, meta)
	errEnc := VerifyEncryptionProof(zkpEnc, kp.Public(), ct, options, meta)
	if errEnc != nil {
		t.Logf("Encryption Verification Fail: %s", errEnc)
		t.Fail()
	}

}
