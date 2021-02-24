package elgamal

import (
	"fmt"
	"testing"
)

func TestSignature(t *testing.T) {
	// j := json.NewEncoder(os.Stdout)
	// j.SetIndent("", " ")
	eg := dh2048modp256
	//fmt.Println("ElGamal")
	//j.Encode(eg)

	kp := GenerateKeyPair(eg)
	//fmt.Println("KeyPair: Secret")
	//j.Encode(kp.Secret())
	//fmt.Println("KeyPair: Public")
	//j.Encode(kp.Public())

	//fmt.Println("message")
	m := []byte("hello")
	//j.Encode(m)

	sig := kp.Secret().CreateSignature(m)
	//fmt.Println("Signature")
	//j.Encode(sig)

	err := kp.Public().VerifySignature(sig, m)

	if err != nil {
		fmt.Println("Error")
		fmt.Println(err)
		t.Fatal("signature verification failed")
	}
}
