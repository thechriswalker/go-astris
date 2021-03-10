package elgamal

import (
	"fmt"
	"math/big"
	"testing"
)

type phase1Data struct {
	Trustee         int // same as key in the map
	SignKey         *PublicKey
	EncrKey         *PublicKey
	PublicExponents []*big.Int
	Signature       *Signature // over both keys and exponents so they all must be valid.
}

type phase2Data struct {
	Trustee       int              // same as key in the map
	PrivateShares map[int]*big.Int // we will gloss over the encrypt and sign for now
	// struct {
	// 	Trustee   int // TargetTrustee will be all but "us"
	// 	Point     *big.Int
	// 	Signature *Signature // over the point
	// } // i.e. len(private) = T-1
}

type phase3Data struct {
	Trustee          int               // same as map key
	PublicKey        *PublicKey        // trustee public key
	ProofOfKnowledge *ProofOfKnowledge //of the trustee private key
}

type publicKnowledge struct {
	Params *System
	T, L   int // thresholds

	// phase1 setup
	Phase1 map[int]*phase1Data

	// phase2 produce private shares.
	Phase2 map[int]*phase2Data

	// phase3 is the the public key generation and acknowledgement
	Phase3 map[int]*phase3Data
}

// the secret trustee data
type trustee struct {
	ID     int
	Public *publicKnowledge // ref to all the public knowledge

	derivationSecret *big.Int

	// keys
	keys *DerivedKeys

	// election shared key data
	coefficients []*big.Int

	sharedKey *KeyPair
}

// separated to force no "private" knowledge
func (t *trustee) deriveCoefficients() {
	t.coefficients = deriveCoefficients(t.Public.Params, t.derivationSecret, t.Public.T)
}
func (t *trustee) deriveKeys() {
	t.keys = DeriveKeys(t.Public.Params, t.derivationSecret)
}
func (t *trustee) Exponents() []*big.Int {
	exponents := make([]*big.Int, len(t.coefficients))
	for i, c := range t.coefficients {
		exponents[i] = new(big.Int).Exp(t.Public.Params.G, c, t.Public.Params.P)
	}
	return exponents
}
func (t *trustee) calculateSecretFor(j int) *big.Int {
	bigJ := big.NewInt(int64(j))
	// we can recreate the polynomial by working backwards from t to 0
	s := big.NewInt(0)
	for n := t.Public.T; n >= 0; n-- {
		// each power is multiplied (to exponentiate) and the next (lower) coefficient is added
		s.Mul(s, bigJ)
		s.Add(s, t.coefficients[n])
		s.Mod(s, t.Public.Params.P)
	}
	return s
}
func (t *trustee) calculateSharedKeys() {
	// private key is SIGMA from j=1->L s_ji
	x := big.NewInt(0)
	for j := 0; j < t.Public.L; j++ {
		if j == t.ID {
			// this is us. we do this ourselves.
			x.Add(x, t.calculateSecretFor(j))
		} else {
			// add the secret share from trustee j for us
			x.Add(x, t.Public.Phase2[j].PrivateShares[t.ID])
		}
		x.Mod(x, t.Public.Params.Q)
	}
	t.sharedKey = keypairForSecret(t.Public.Params, x)
}

func TestThreshold(t *testing.T) {

	// Authority generates public data
	election := &publicKnowledge{
		Params: DH2048modp256(),
		T:      2, // t+1 trustees required to reconstruct
		L:      5,
		Phase1: map[int]*phase1Data{},
		Phase2: map[int]*phase2Data{},
		Phase3: map[int]*phase3Data{},
	}

	// Phase1: keys and exponents (we will leave the signatures for now, we are playing nicely)
	fmt.Println("Phase 1: Publish Trustee Keys and Public Exponents")
	trustees := make([]*trustee, election.L)
	for i := range trustees {
		trustees[i] = &trustee{
			ID:               i,
			Public:           election,
			derivationSecret: big.NewInt(int64(i)), // really dumb in the real world
		}
		trustees[i].deriveKeys()
		trustees[i].deriveCoefficients()

		election.Phase1[i] = &phase1Data{
			Trustee:         i,
			SignKey:         trustees[i].keys.Sig.Public(),
			EncrKey:         trustees[i].keys.Enc.Public(),
			PublicExponents: trustees[i].Exponents(),
		}
	}
	// now calculate the election public key multiplicative sum of the
	// "zero" indexed exponentials.
	pk := big.NewInt(1)
	for _, data := range election.Phase1 {
		// do we need to do this in order? I don't think so.
		pk.Mul(pk, data.PublicExponents[0])
		pk.Mod(pk, election.Params.P)
	}
	electionPublicKey := &PublicKey{
		System: election.Params,
		Y:      pk,
	}

	fmt.Printf("after phase1 we have the `election` public key: %v\n", electionPublicKey.toJSON())

	// Phase 2: Encrypt a the private secrets for each other trustee.
	fmt.Println("Phase 2: Trustees encrypt private secrets coefficients for each other")
	for i := range trustees {
		shares := map[int]*big.Int{}
		election.Phase2[i] = &phase2Data{
			Trustee:       i,
			PrivateShares: shares,
		}
		for j := range trustees {
			if i == j {
				continue
			}
			shares[j] = trustees[i].calculateSecretFor(j)
		}
	}

	// phase3: combine the secrets  for the final public/private key (shards)
	fmt.Println("Phase 3: Trustees can no assemble their partial keys")
	for i := range trustees {
		trustees[i].calculateSharedKeys()
		election.Phase3[i] = &phase3Data{
			Trustee:   i,
			PublicKey: trustees[i].sharedKey.Public(),
		}
		fmt.Printf("Trustee[%d] Private Key: %v\n", i, trustees[i].sharedKey.Secret().toJSON())
	}

	fmt.Println("Ready to ecrypt and decrypt")

	// secret value.
	plain := big.NewInt(1337)
	fmt.Printf("PlainText: %s (%s)\n", plain.Text(10), toJSON(plain))

	// encrypt a value (ignore randomness)
	cipher := electionPublicKey.Encrypt(plain, nil)

	fmt.Printf("CipherText: %v\n", cipher.toJSON())

	// partial decrypt with t random trustees.
	// using their own data.
	partials := make([]*big.Int, len(trustees))
	for i, t := range trustees {
		// @todo ZKP here to prove decryption
		sk := t.sharedKey.Secret()
		partials[i] = new(big.Int).Exp(cipher.A, sk.X, sk.P)
		fmt.Printf("Trustee [%d] partial: %s\n", i, toJSON(partials[i]))
	}

	// final decryption.
	// we only actually want T+1 trustees, not L.
	// we should really produce a number of decryptions from all valid
	// subsets of T+1 trustees but for this we will.
	// in fact we simply ought to choose n subsets such that
	// as many trustee's partials are used as possible.
	// i.e. in this case we have 5 honest trustees.
	// we need to cover all of them so
	// [0,1,2], [2,3,4] means that data from all of them is considered.
	//
	// choose explicitly...

	// decryption is ct.Beta * (SIGMA(j){ c_j^LI_j}) ^-1
	// that is multiplicative sum over our `j` which
	// are the chosen t+1 indexes
	// c_j is the partial decryption from trustee j
	// and LI_j is the langrange coefficient j from
	// the set.
	decrypt := func(indices []int) *big.Int {
		// lets calculate the sigma first.
		sigma := big.NewInt(1)
		for _, j := range indices {
			cj := partials[j]
			lij := lagrange(indices, j, election.Params.Q)
			raised := new(big.Int).Exp(cj, lij, election.Params.P)
			sigma.Mul(sigma, raised)
			sigma.Mod(sigma, election.Params.P)
		}

		// now inverse and multiply.
		sigma.ModInverse(sigma, election.Params.P)
		sigma.Mul(sigma, cipher.B)
		return sigma.Mod(sigma, election.Params.P)
	}

	// correct answer?

	answers := []*big.Int{
		decrypt([]int{0, 1, 2}),
		decrypt([]int{0, 1, 4}),
		decrypt([]int{1, 2, 3}),
		decrypt([]int{2, 3, 4}),
	}

	//answers[1].Add(answers[1], big.NewInt(1))

	for i, a := range answers {
		fmt.Printf("Reconstruction [%d]: %s (%s)\n", i, a.Text(10), toJSON(a))
	}

	for i := 0; i < len(answers)-1; i++ {
		for j := i; j < len(answers); j++ {
			// check they match!
			if i == j {
				continue
			}
			if answers[i].Cmp(answers[j]) != 0 {
				t.Logf("Answers do not all match: %s != %s", answers[i].Text(10), answers[j].Text(10))
				t.Fail()
			}
		}
	}
}
