package elgamal

import (
	"fmt"
	"math/big"

	"../random"
)

// ProofOfKnowledge is a NonInteractive ZKP that the prover knows
// the secret key paired with a public key. We must have the public key
// to Verify
type ProofOfKnowledge struct {
	Cm, Ch, R *big.Int `json:",string"`
}

// on system to allow it to be used by both private and public keys
func (s *System) createPoKChallenge(x *big.Int) *big.Int {
	b := []byte("PoK:")
	b = append(b, x.Bytes()...)
	return random.Oracle(b, s.Q)
}

// ProofOfKnowledge generates a ZKP of knowledge of the secret key
func (sk *SecretKey) ProofOfKnowledge() (pok *ProofOfKnowledge) {
	pok = new(ProofOfKnowledge)
	// a random w
	w := random.Int(sk.Q)
	pok.Cm = new(big.Int).Exp(sk.G, w, sk.P)
	// to turn the commitment into a challenge we
	// SHA1 hash it and take the integer value of the bytes.
	// Any random oracle will work, as long as the same
	// one is used for verification as proof.
	pok.Ch = sk.createPoKChallenge(pok.Cm)
	// the response is now (w + sk.X * chall) % Q
	pok.R = new(big.Int).Mul(sk.X, pok.Ch)
	pok.R.Add(pok.R, w)
	pok.R.Mod(pok.R, sk.Q)
	return
}

// VerifyProof a proof of knowledge of the secret key associated with the given public key.
func (pk *PublicKey) VerifyProof(pok *ProofOfKnowledge) error {
	expectedChallenge := pk.createPoKChallenge(pok.Cm)
	if expectedChallenge.Cmp(pok.Ch) != 0 {
		return fmt.Errorf("Bad Challenge Value in ProofOfKnowledge")
	}
	// OK, now  g^response should equal commitment * y^challenge
	var lhs, rhs big.Int

	lhs.Exp(pk.G, pok.R, pk.P)
	rhs.Exp(pk.Y, pok.Ch, pk.P)
	rhs.Mul(&rhs, pok.Cm)
	rhs.Mod(&rhs, pk.P)

	if lhs.Cmp(&rhs) != 0 {
		return fmt.Errorf("ProofOfKnowledge invalid")
	}
	return nil
}
