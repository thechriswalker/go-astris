package elgamal

import (
	"testing"

	big "github.com/ncw/gmp"
)

func TestProofOfKnowledge(t *testing.T) {
	eg := DH2048modp256()
	kp := GenerateKeyPair(eg)

	pok := kp.Secret().ProofOfKnowledge()

	if err := kp.Public().VerifyProof(pok); err != nil {
		t.Logf("ProofOfKnowledge verify fail: %v", err)
		t.Fail()
	}

	// screw it up
	pok.R.Add(pok.R, big.NewInt(1))

	if err := kp.Public().VerifyProof(pok); err == nil {
		t.Logf("ProofOfKnowledge verify passeed incorrectly Response tampered")
		t.Fail()
	}
}
