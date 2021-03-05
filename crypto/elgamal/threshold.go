package elgamal

import (
	"bytes"
	"fmt"
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
	T int // T+1 participants required to decrypt
	L int // Total number of participants
}

// PublicParticipantPhase1 is the data for the initial participant setup that is made public
// at the start
type PublicParticipantPhase1 struct {
	Index     int               // index 0 <= L-1
	SigKey    *PublicKey        // the public key for verifying signatures
	EncKey    *PublicKey        // the public key for encryption
	EncProof  *ProofOfKnowledge // PoK of secret key for encryption key
	Exponents []*big.Int        // our public exponents from the private bits. a Commitment (length T+1)
	Signature *Signature        //signature over the data with the signing key (which is why we don't need a pok for that key)
}

// PublicParticipantPhase2 is the next published step, the private sharing of the
// points for the other trustees

type PublicParticipantPhase2 struct {
	Index  int // index 0 <= L-1
	Shares []*EncryptedShare
}

// EncryptedShare is the private data for the other trustees
type EncryptedShare struct {
	Sender    int         // sendder trustee (i)
	Recipient int         // recipient trustee (j)
	Point     *CipherText // the point data encrypted with the recipient's public encryption key
	Signature *Signature  // a signature over the data with the participants signing key
}

func (es *EncryptedShare) SignatureMessage() []byte {
	buf := &bytes.Buffer{}
	fmt.Fprintf(
		buf,
		"es:%d:%d:%s:%s",
		es.Sender,
		es.Recipient,
		es.Point.A.Text(16),
		es.Point.B.Text(16),
	)
	return buf.Bytes()
}

// PublicParticipantPhase3 is the acknowledgement of the validity of the shares from the other
// participants and the publishing of the public key shard and a pok of the shard of decryption key
type PublicParticipantPhase3 struct {
	Index      int               // the trustee index 0 <= L-1
	ShardKey   *PublicKey        // the shard of the public key
	ShardProof *ProofOfKnowledge // pok of the decryption part of the public key
	Signature  *Signature        // a signature over the data to authenticate it
}

// SecretParticipant is the secret information for a participant. It will hold
// the private keys for signing and encryption, the private coefficients for the
// shared decryption. we serialise this as just the Key bits.
type SecretParticipant struct {
	System       *ThresholdSystem
	Index        int
	Signing      *KeyPair
	Encryption   *KeyPair
	Coefficients []*big.Int
	// the rest of the fields may not be present due to the
	// fact that we only recieve the data after phase1 is complete
	Participants map[int]PublicParticipantPhase1
	Shares       map[int]*big.Int // decrypted shares (after verification)
	ShardKey     *KeyPair         // reconstructed Shard KeyPair hold the piece of the public key and the matching decryption shard
}

// PublicExponents raises G to the power of each coefficient
func (sp *SecretParticipant) PublicExponents() []*big.Int {
	exponents := make([]*big.Int, len(sp.Coefficients))
	for i, c := range sp.Coefficients {
		exponents[i] = new(big.Int).Exp(sp.System.G, c, sp.System.P)
	}
	return exponents
}

func (sp *SecretParticipant) secretShareFor(j int) *big.Int {
	bigJ := big.NewInt(int64(j))
	// we can recreate the polynomial by working backwards from t to 0
	s := big.NewInt(0)
	for n := sp.System.T; n >= 0; n-- {
		// each power is multiplied (to exponentiate) and the next (lower) coefficient is added
		s.Mul(s, bigJ)
		s.Add(s, sp.Coefficients[n])
		s.Mod(s, sp.System.P)
	}
	return s
}

// to sign the phase1 data we concatenate the
func (p1 *PublicParticipantPhase1) SignatureMessage() []byte {
	buf := &bytes.Buffer{}
	// we need to contain:
	// - the trustee index,
	// - encryption public key (just the number, as hex, lowercased)
	// - the exponents (there should be T, exponents)
	fmt.Fprintf(buf, "p1:%d:%s", p1.Index, p1.EncKey.Y.Text(16))
	for _, x := range p1.Exponents {
		fmt.Fprintf(buf, ":%s", x.Text(16))
	}
	return buf.Bytes()
}

// Phase1 generates the initial data.
func (sp *SecretParticipant) Phase1() *PublicParticipantPhase1 {
	p1 := &PublicParticipantPhase1{
		Index:     sp.Index,
		SigKey:    sp.Signing.Public(),
		EncKey:    sp.Encryption.Public(),
		EncProof:  sp.Encryption.Secret().ProofOfKnowledge(),
		Exponents: sp.PublicExponents(),
	}
	p1.Signature = sp.Signing.Secret().Sign(p1)
	return p1
}

// Phase2 creates the phase2 data, the secret shares, assumes we have
// phase1 data from all other trustees
func (sp *SecretParticipant) Phase2() *PublicParticipantPhase2 {
	p2 := &PublicParticipantPhase2{
		Index:  sp.Index,
		Shares: make([]*EncryptedShare, 0, sp.System.L-1), // one less than the number of participants
	}
	// remember to panic if we don't have the data from phase1 yet...
	if sp.Participants == nil || len(sp.Participants) != sp.System.L {
		panic("attempt to create phase2 data early")
	}
	// create the shares for each participant, we will need there encryption keys
	for _, p := range sp.Participants {
		if p.Index == sp.Index {
			// ignore
			continue
		}
		secretShare := sp.secretShareFor(p.Index)
		share := &EncryptedShare{
			Recipient: p.Index,
			Point:     p.EncKey.Encrypt(secretShare, nil),
		}
		// sign the data.
		share.Signature = sp.Signing.Secret().Sign(share)
		p2.Shares = append(p2.Shares, share)
	}
	return p2
}

// @TODO the validation of the data from the other phases from the other trustees

func (p3 *PublicParticipantPhase3) SignatureMessage() []byte {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "p3|%d|%s", p3.Index, p3.ShardKey.Y.Text(16))
	return buf.Bytes()
}

// Phase3 generates the data needed for phase3, assumes we have all the data from the
// other trustees from phase2 and have processed it
func (sp *SecretParticipant) Phase3() *PublicParticipantPhase3 {
	p3 := &PublicParticipantPhase3{
		Index:      sp.Index,
		ShardKey:   sp.ShardKey.Public(),
		ShardProof: sp.ShardKey.Secret().ProofOfKnowledge(),
	}
	p3.Signature = sp.Signing.Secret().Sign(p3)
	return p3
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

func deriveCoefficients(params *System, secret *big.Int, t int) []*big.Int {
	// t is the number of coefficients we want.
	// secret is our random secret that gives our deterministic "random"
	// coefficients and keys.
	// OK so we just want the coefficients.
	coefficients := make([]*big.Int, t+1) // t+1 coefficients i.e. c0 + c1*x + c2 * x^2 ... + ct * x^t

	buf := &bytes.Buffer{}

	for i := range coefficients {
		buf.Reset()
		fmt.Fprintf(
			buf,
			"coef|%s|%x|%s|%x",
			params.P.Text(16),
			t,
			secret.Text(16),
			i,
		)
		coefficients[i] = random.Oracle(buf.Bytes(), params.Q)
	}
	return coefficients
}
