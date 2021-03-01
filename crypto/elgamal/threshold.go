package elgamal

import (
	"math/big"

	"github.com/thechriswalker/go-astris/crypto/random"
)

// The threshold encryption scheme
//
// The aims:
//
// - Have a public key available
// - Each trustee has a private key share known only to them
// - We have a (K,N) decryption threshold
// - All data can be made public. (it will be on the blockchain)
// - the ElGamal Parameters will be known in advance, and probably
//   one of the RFC5114 standard DH params

// Process
//  - Each trustee i generates a random number from Z_q: secret key SKi
//  - Each trustee i generates a signing and encryption keypair derived from SKi
//    so they have KSi and KEi
//  - Each Trustee i gives the public keys to the election organiser along with a ZKP of knowledge
//    of the Secret key associations.
//  - The election initial block is now created
//  -----------
//  - next phase: each trustee derives the polynomial from the secret and
//    uploads the signed exponentials and the secret shares encrypted with the recipients public keys
//  -----------
//  - next phase: all trustees can now verify the secret shares and sign an acknowledgement
//  -----------
//  - next phase: once all acknowledgements are verified, we know that no complaints have been
//    raised and so the protocol has succeeded
//  ------------
//  - now all the trustees have their secrets, all is public and can be verified, public key is
//    computable and decryption can only happen with >K decryptors

// This is a form of the scheme in Belenios: https://hal.inria.fr/hal-02066930/document
//
// This scheme in some respects was added to Helios here.
// https://github.com/glondu/helios-server/commit/0e226a81ec9e2cd2fbd21c1e562434fca69d5987

// The building blocks are:
// - shared ElGamal Params with a T (amount of dishonest participants) and L (total participants) values
// - generate a random secret and derive (enc, sign) keys from it (randomOracle for keys derivation)
// - code for signing data and verifying the signature
// - a certificate scheme to provide signed publics keys for use as a private channel to participants.
// - The ElGamal PoK of secret key
// - ElGamal encryption with a public key.

// ThresholdSystem is the distributed decryption, public encryption scheme
// with L-T participants required to decrypt.
// I believe there is a requirement for L-T
type ThresholdSystem struct {
	*System
	T, L *big.Int `json:",string"`
}

// PublicParticipantInint is the data for the initial participant setup that is made public
type PublicParticipantInit struct {
	Index      int               // index 0 <= L-1
	VerifyKey  *PublicKey        // the public key for verifying signatures
	VerifyPoK  *ProofOfKnowledge // PoK of secret key for verify key
	EncryptKey *PublicKey        // the public key for encryption
	EncryptPoK *ProofOfKnowledge // PoK of secret key for encryption key
}

type PublicParticipantExponents struct {
	Index     int
	Exponents []*big.Int // length T
}

type PublicParticipantAcknowledgement struct {
	Index     int
	Signature *Signature // the commitment for this will be from the exponents of all participants
}

// SecretParticipant is the secret information for a participant. It will hold
// the key derivation secret
type SecretParticipant struct {
	Secret     *big.Int
	Signing    *KeyPair
	Encryption *KeyPair
	Init       *PublicParticipantInit
	Shares     []interface{} // I don't know what these shares look like yet
	Exponents  *PublicParticipantExponents
}

// mod-inverse of all the factors except the current index
func lagrange(indices []int, index int, modulus *big.Int) (r *big.Int) {
	r = new(big.Int).Set(one)
	var inv, idx big.Int
	for _, i := range indices {
		if i != index {
			// r = (r * i * inverse(i-index, modulus)) % modulus
			idx.SetInt64(int64(i))
			inv.SetInt64(int64(i - index))
			inv.ModInverse(&inv, modulus)
			r.Mul(r, &idx)
			r.Mul(r, &inv)
			r.Mod(r, modulus)
		}
	}
	return
}

func (p *SecretParticipant) PartialDecrypt(params *System, participants ...[]*PublicParticipantInit) *big.Int {
	// if len(factors) == 0 {
	// 	panic("no decryption factors passed to CipherText.Decrypt")
	// }
	// var dec, invf big.Int
	// dec.Set(ct.B)
	// for _, f := range factors {
	// 	invf.ModInverse(f, params.P)
	// 	dec.Mul(&dec, &invf)
	// 	dec.Mod(&dec, params.P)
	// }
	// return &dec
	return nil
}

func deriveCoefficients(params *System, secret *big.Int, t int) []*big.Int {
	// t is the number of coefficients we want.
	// secret is our random secret that gives our deterministic "random"
	// coefficients and keys.
	// OK so we just want the coefficients.
	coefficients := make([]*big.Int, t+1) // t+1 coefficients i.e. c0 + c1*x + c2 * x^2 ... + ct * x^t

	for i := range coefficients {
		b := []byte("coef|")
		b = append(b, params.P.Text(16)...)
		b = append(b, '|')
		b = append(b, big.NewInt(int64(t)).Text(16)...)
		b = append(b, '|')
		b = append(b, secret.Text(16)...)
		b = append(b, '|')
		b = append(b, big.NewInt(int64(i)).Text(16)...)
		coefficients[i] = random.Oracle(b, params.Q)
	}
	return coefficients
}
