package elgamal

import (
	"fmt"
	"testing"
)

func TestSchnorrSignature(t *testing.T) {
	eg := DH2048modp256()

	kp := GenerateKeyPair(eg)
	m := []byte("hello")
	sig := kp.Secret().CreateSignature(m)
	err := kp.Public().VerifySignature(sig, m)
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err)
		t.Fatal("signature verification failed")
	}
}
