package elgamal

import (
	"bytes"
	"errors"
	"fmt"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto/random"
)

// ZKP in general form.
// This struct doesn't contain enough information to validate the ZKP
// that can only be done in context. Sometime the A and B values will
// directly lead to the challenge, but mostly we just check the given
// challenge. This means that depending on context more than `verifyZKP`
// It also means that _some_ ZKPs don't need C and other do.
//
// The general form is:
//   CreateZKP(g, h, x, p, q, challengeFn)
//     w = random()
//     A = g^w % p
//     B = h^w % p
//     C = challengeFn(A, B) % q
//     R = (w + x*C) % q
//     return { A, B, R }
//
//   VerifyZKP(A, B, C, R, g, h, G, H, p, q)
//     check g^R % p === (A * G^C) % p
//     check h^R % p === (B * H^C) % p
//
// But g, p, q are always the "System"
//
//  We prove we know x such that G = g^x and H = h^x
//
//
// We have three forms of ZKP in our system.
// First we have the proof of knowledge of a secret key which is defined separately as it is simpler.
//
// Then we have 2 other proofs:
//   Proof of Correct Encryption to one of a set of plaintexts (the OR proof)
//   Proof of Correct Decryption of a ciphertext
type ZKP struct {
	A, B, C, R *big.Int
}

type cFn = func(A, B, q *big.Int) *big.Int

func createZKP(s *System, h, x *big.Int, fn cFn) *ZKP {
	w := random.Int(s.Q)
	A := new(big.Int).Exp(s.G, w, s.P)
	B := new(big.Int).Exp(h, w, s.P)
	C := fn(A, B, s.Q)
	R := new(big.Int).Mul(x, C)
	R.Add(R, w)
	R.Mod(R, s.Q)

	// 	fmt.Printf(`ZKP:Create
	// system(G,P,Q) = (%v, %v, %v)

	// g = %v (system.g)
	// G = %v (g^x = PK)

	// h = %v (alpha)
	// H = %v (alpha ^ x = beta/m)

	// x = %v (secret key)
	// w = %v (random int)
	// A = %v (g ^ w = system.g ^ w)
	// B = %v (h ^ w = alpha ^ w)
	// C = %v (random (derived from A,B,...))
	// R = %v (w + xC mod Q)
	// `, s.G, s.P, s.Q, s.G, new(big.Int).Exp(s.G, x, s.P), h, new(big.Int).Exp(h, x, s.P), x, w, A, B, C, R)

	return &ZKP{A: A, B: B, C: C, R: R}
}

func verifyZKP(zkp *ZKP, s *System, h, G, H *big.Int) error {
	lhs, rhs := new(big.Int), new(big.Int)

	// 	fmt.Printf(`ZKP:Verify
	// A = %v
	// B = %v
	// C = %v
	// R = %v
	// `, zkp.A, zkp.B, zkp.C, zkp.R)

	// check g^R % p === (A * G^C) % p
	lhs.Exp(s.G, zkp.R, s.P)
	rhs.Exp(G, zkp.C, s.P)
	rhs.Mul(rhs, zkp.A)
	rhs.Mod(rhs, s.P)

	// 	fmt.Printf(`ZKP:Check1
	// g = %v
	// G = %v
	//     g^R = %v
	// A * G^C = %v
	// `, s.G, G, lhs, rhs)

	if lhs.Cmp(rhs) != 0 {
		return errors.New("ZKP invalid: g^R % p != (A * G^C) % p")
	}
	// check h^R % p === (B * H^C) % p
	lhs.Exp(h, zkp.R, s.P)
	rhs.Exp(H, zkp.C, s.P)
	rhs.Mul(rhs, zkp.B)
	rhs.Mod(rhs, s.P)

	// 	fmt.Printf(`ZKP:Check2
	// h = %v
	// H = %v
	//     h^R = %v
	// B * H^C = %v
	// `, h, H, lhs, rhs)

	if lhs.Cmp(rhs) != 0 {
		return errors.New("ZKP invalid: h^R % p != (B * H^C) % p")
	}
	// ok!
	return nil
}

// ProveDecryption provides a ZKP of correct decryption
// given the keypair, and ciphertext:
// prove equality of discrete log with Chaum Pederson, and that discrete log
// is X (the secret key).
//
// For the ZKP:
//	h is the ciphertext alpha
//  x is the private key
//
// h = g^x
func ProveDecryption(sk *SecretKey, ct *CipherText) (zkp *ZKP) {
	return createZKP(sk.System, ct.A, sk.X, zkpOfDecryptionCommitment)
}

// we don't need to add more info to the commitment here as
// it includes the secret key of the decryptor. So it cannot
// be forged by a different party
func zkpOfDecryptionCommitment(ca, cb, Q *big.Int) *big.Int {
	var commit bytes.Buffer
	fmt.Fprintf(&commit, "zkp:dec:%x|%x", ca.Bytes(), cb.Bytes())
	return random.Oracle(commit.Bytes(), Q)
}

// VerifyDecryptionProof validates the ZKP of correction decryption
//
// For the ZKP
//  h = public key
//  G = ciphertext alpha
//  H = ciphertext beta / plaintext
func VerifyDecryptionProof(zkp *ZKP, pk *PublicKey, ct *CipherText, pt *big.Int) error {
	// H = beta / pt
	H := big.NewInt(0)
	H.ModInverse(pt, pk.P)
	H.Mul(H, ct.B)
	H.Mod(H, pk.P)
	return VerifyPartialDecryptionProof(zkp, pk, ct, H)
}

// This is the partial decryption proof
// doesn't use the beta/plaintext for validation, but the form is identical to the general decryption proof
func VerifyPartialDecryptionProof(zkp *ZKP, pk *PublicKey, ct *CipherText, partial *big.Int) error {
	// first we verify the commitment is correct:
	C := zkpOfDecryptionCommitment(zkp.A, zkp.B, pk.Q)
	if C.Cmp(zkp.C) != 0 {
		return fmt.Errorf("ZKP invalid, commitment does not match A,B")
	}
	return verifyZKP(zkp, pk.System, ct.A, pk.Y, partial)
}

// The OR proof just consists of a number of proofs detailing the plaintexts
// ONE of which was the one encrypted.
type ZKPOr []*ZKP

// ProveEncyption shows that a ciphertext encrypts one of a set of values, without
// revealing which one it encrypts.
// This is actually a dlog proof extended to knowledge of one dlog OR another.
// using: https://crypto.ethz.ch/publications/files/CamSta97b.pdf
// we can construct a proof of X or Y.
// This can be extended to construct a proof of X or Y or Z or ...
// The way this works (simply) is to create the correct proof for
// the actual result, and create simulated proofs for the other results
// as all the proofs have to match on the same commitments, in order
// to simulate the "fake" proofs, we must know at least one valid proof.
// The verifier can recreate the challenge from the commitment (which we chose)
// and so verify that the proof indeed shows that one of the options was known.
//
// The encryption proof is actually a ZKP that we know the randomness used to create the ciphertext
// which means we have to keep holds of that randomness for a bit.
//
// Also to prevent ballot copying, we include some data that is unique for each encryption (but public) in the challenge
// function, meaning a copied vote will fail the ZKP validation for a different user.
func ProveEncryption(
	pk *PublicKey,
	ct *CipherText,
	plaintexts []*big.Int,
	index int,
	rnd *big.Int,
	meta []byte,
) (zkp ZKPOr) {
	// the overall proof is the slice of all the individual ones.
	zkp = make(ZKPOr, len(plaintexts))
	// first we fill in all the "fake" proofs, so we can sum the challenges.
	csum := big.NewInt(0)
	for i, pt := range plaintexts {
		if i == index {
			// we do the real one last.
			continue
		}
		zkp[i] = fakeEncZKP(pk, ct, pt)
		csum.Add(csum, zkp[i].C)
	}
	csum.Mod(csum, pk.Q)

	// now create the real proof, such with a commitment function that
	// creates a commitment that matches all the other As and Bs
	// this is a full hash over all the commitments.
	challenge := func(a, b, q *big.Int) *big.Int {
		c := zkpOrChallenge(zkp, meta, index, a, b, q)
		// but we need to subtract now our challenges from the previous fake ZKPs
		// so the sum adds up correctly.
		c.Sub(c, csum)
		c.Mod(c, q)
		return c
	}
	// now create a real ZKP, where the secret is the randomness from the encryption step.
	zkp[index] = createZKP(pk.System, pk.Y, rnd, challenge)

	return zkp
}

// create the challenge for the OR zkp, with or without
// for the create, we pass in the index and a,b values
// for the verify, we pass -1 for the index and nil for a/b
func zkpOrChallenge(zkp ZKPOr, meta []byte, index int, a, b, q *big.Int) *big.Int {
	var commit bytes.Buffer
	commit.WriteString("zkp:enc:")
	// now we add every commitment
	for i := range zkp {
		if i == index {
			// use a,b from here.
			fmt.Fprintf(&commit, "%x|%x:", a.Bytes(), b.Bytes())
		} else {
			// use a and b from the existing proof.
			fmt.Fprintf(&commit, "%x|%x:", zkp[i].A.Bytes(), zkp[i].B.Bytes())
		}
	}
	// now add the unique metadata
	commit.Write(meta)
	//fmt.Println("zkp commitment:", commit.String())
	return random.Oracle(commit.Bytes(), q)
}

func VerifyEncryptionProof(
	zkp ZKPOr,
	pk *PublicKey,
	ct *CipherText,
	possibilities []*big.Int,
	meta []byte,
) error {
	// first we should have as many proofs as possibilities.
	if len(zkp) != len(possibilities) {
		return fmt.Errorf("ZKP invalid: mistached number of proofs vs. plaintexts")
	}

	csum := big.NewInt(0)
	var err error
	betaOverM := new(big.Int)

	// all of the individual proofs should validate
	for i, z := range zkp {
		betaOverM.SetInt64(0)
		betaOverM.ModInverse(possibilities[i], pk.P)
		betaOverM.Mul(betaOverM, ct.B)
		betaOverM.Mod(betaOverM, pk.P)

		// h = public key (y)
		// G = ciphertext Alpha
		// H = beta/m
		err = verifyZKP(z, pk.System, pk.Y, ct.A, betaOverM)
		if err != nil {
			return fmt.Errorf("ZKP inner invalid at index[%d]: %w", i, err)
		}
		// sum the commitments
		csum.Add(csum, z.C)
	}
	// remember to apply the modulo
	csum.Mod(csum, pk.Q)

	// the sum of the challenges should match the calculated challenge.
	// NB we have all the commitments at this stage so we pass -1 for index
	// to be spliced in (meaning don't splice)
	calc := zkpOrChallenge(zkp, meta, -1, nil, nil, pk.Q)

	if calc.Cmp(csum) != 0 {
		return fmt.Errorf("ZKP invalid: OR proof challenge sum does not match computed challenge")
	}

	// boom! we win
	return nil
}

// To fake a ZKP we work backwards, we create a random challenge and random response
// we compute the "beta / plaintext" required for the proof and then work out the
// ca and cb that fit the system.
// Note that the fake ZKP (just like the "real" one from the Encryption), will
// NOT have a challenge that matches the A, B values.
// That has to be created separately from ALL the A,B values.
func fakeEncZKP(pk *PublicKey, ct *CipherText, pt *big.Int) *ZKP {
	betaOverM := new(big.Int).Set(pt)
	betaOverM.ModInverse(pt, pk.P)
	betaOverM.Mul(betaOverM, ct.B)
	betaOverM.Mod(betaOverM, pk.P)

	C, R := random.Int(pk.Q), random.Int(pk.Q)

	A, B, tmp := new(big.Int), new(big.Int), new(big.Int)

	// g = system.g
	// h = public key (y)
	// G = ciphertext Alpha
	// H = beta/m

	// calculate a value for A = g^R / alpha^C
	A.Exp(ct.A, C, pk.P)
	A.ModInverse(A, pk.P)
	// make tmp = g^R
	A.Mul(A, tmp.Exp(pk.G, R, pk.P))
	A.Mod(A, pk.P)

	// calculate a value for B = y^R / (beta/pt)^C
	B.Exp(betaOverM, C, pk.P)
	B.ModInverse(B, pk.P)
	// make tmp = y^R
	B.Mul(B, tmp.Exp(pk.Y, R, pk.P))
	B.Mod(B, pk.P)

	return &ZKP{A: A, B: B, C: C, R: R}
}

type PlaintextOptionsCache struct {
	system *System
	cache  map[int][]*big.Int
}

func NewPlaintextOptionsCache(s *System) *PlaintextOptionsCache {
	return &PlaintextOptionsCache{
		system: s,
		cache:  map[int][]*big.Int{},
	}
}

func (p *PlaintextOptionsCache) GetOptions(max int) []*big.Int {
	o, ok := p.cache[max]
	if !ok {
		o = make([]*big.Int, max+1)
		for i := range o {
			o[i] = big.NewInt(int64(i))
			o[i].Exp(p.system.G, o[i], p.system.P)
		}
		p.cache[max] = o
	}
	return o
}
