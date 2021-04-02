package elgamal

import (
	"fmt"
	"testing"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto"
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

func TestThreshold(t *testing.T) {
	sys := &ThresholdSystem{
		//System: DH2048modp256(),
		//System: DH1024modp160(),
		// something small
		//System: New(8),
		System: &System{P: big.NewInt(227), Q: big.NewInt(113), G: big.NewInt(69)}, // 8bit
		T:      3,                                                                  // t+1 trustees required to reconstruct
		L:      5,
	}
	// fmt.Printf("%#v\n", sys.System)

	// Phase1: keys and exponents (we will leave the signatures for now, we are playing nicely)
	fmt.Println("Phase 1: Publish Trustee Keys and Public Exponents")
	trustees := make(map[int]*PrivateParticipant, sys.L)
	exponents := map[int]crypto.BigIntSlice{}
	for i := 0; i < sys.L; i++ {
		idx := i + 1                // 1-based indices - VERY IMPORTANT
		d := big.NewInt(int64(idx)) // really dumb in the real world
		trustees[idx] = &PrivateParticipant{
			Index:     idx,
			Sys:       sys,
			Coeffs:    DeriveCoefficients(sys.System, d, sys.T),
			PublicExp: exponents,
		}
		exponents[idx] = CreateExponents(sys.System, trustees[idx].Coeffs)
	}
	// now calculate the election public key multiplicative sum of the
	// "zero" indexed exponentials.
	pk := big.NewInt(1)
	for _, data := range exponents {
		// do we need to do this in order? I don't think so.
		pk.Mul(pk, data[0])
		pk.Mod(pk, sys.P)
	}

	electionPublicKey := &PublicKey{
		System: sys.System,
		Y:      pk,
	}

	fmt.Printf("after phase1 we have the `election` public key: %v\n", electionPublicKey.toJSON())

	// Phase 2: Encrypt a the private secrets for each other trustee.
	fmt.Println("Phase 2: Trustees encrypt private secrets coefficients for each other")
	shares := map[int]map[int]*big.Int{}
	for _, ti := range trustees {
		shares[ti.Index] = map[int]*big.Int{}
		for _, tj := range trustees {
			if ti.Index == tj.Index {
				continue
			}
			shares[ti.Index][tj.Index] = ti.CreateSecretShare(tj.Index)
		}
	}

	// phase3: combine the secrets  for the final public/private key (shards)
	fmt.Println("Phase 3: Trustees can no assemble their partial keys")
	for i := 1; i <= sys.L; i++ {
		ti := trustees[i]
		// check all given shares
		for j := 1; j <= sys.L; j++ {
			if j == i {
				continue
			}
			s := shares[j]
			fmt.Println("Shares FOR i =", ti.Index, "FROM j =", j, " => ", s, ", Sji = ", s[ti.Index])
			if !ti.CheckSecretShareFrom(j, s[ti.Index]) {
				fmt.Println("SecretShareCheckFail j=", j, ", i=", ti.Index)
				panic("Secret Share Check Fail")
			}
		}
		ti.CombineSharesSharedKeys(func(j, i int) *big.Int {
			return shares[j][i]
		})

		fmt.Printf("Trustee[%d] Private Key: %v\n", ti.Index, ti.ShardKey.Secret().toJSON())
		expectedPk := sys.SimulatePublicKeyShard(ti.Index, exponents)
		fmt.Printf("PublicKey: %x\n Expected: %x\n", ti.ShardKey.Public().Y.Bytes(), expectedPk.Y.Bytes())
		if expectedPk.Y.Cmp(ti.ShardKey.Public().Y) != 0 {
			panic("shard key mismatch")
		}
	}

	fmt.Println("Ready to ecrypt and decrypt")

	// secret value.
	secretValue := int64(17)

	plain := big.NewInt(secretValue)
	//	plain.Exp(sys.G, plain, sys.P)

	fmt.Printf("PlainText[%d]: %s (%s)\n", secretValue, plain.String(), crypto.BigIntToJSON(plain))

	// encrypt a value (ignore randomness)
	r := big.NewInt(13)
	cipher := electionPublicKey.Encrypt(plain, r)

	fmt.Printf("CipherText: %v\n", cipher.toJSON())

	// partial decrypt with t random trustees.
	// using their own data.
	partials := make([]*big.Int, len(trustees)+1)
	fmt.Printf("decrypting with %d Trustees\n", len(trustees))
	for _, ti := range trustees {
		fmt.Printf("Decryption for Trustee[%d]\n", ti.Index)
		sk := ti.ShardKey.Secret()
		// gotta be 0-based in the array
		m := new(big.Int).Exp(cipher.A, sk.X, sk.P) // factor
		// m.ModInverse(m, sys.P)                      // factor^{-1}
		// m.Mul(m, cipher.B)                          // m * factor^{-1}
		// m.Mod(m, sys.P)
		partials[ti.Index] = m
		fmt.Printf("Trustee [%d] partial: %v (%s)\n", ti.Index, partials[ti.Index], crypto.BigIntToJSON(partials[ti.Index]))
	}

	fmt.Println(partials)

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
		return ThresholdDecrypt(sys.System, cipher, partials, indices)
	}

	// correct answer?
	// these indices need to be one-based now
	answers := []*big.Int{
		decrypt([]int{1, 2, 3, 4}),
		decrypt([]int{1, 2, 3, 5}),
		decrypt([]int{1, 2, 4, 5}),
		decrypt([]int{5, 2, 4, 3}),
		decrypt([]int{2, 3, 4, 5}),
		decrypt([]int{1, 2, 3, 4, 5}),
	}

	//answers[1].Add(answers[1], big.NewInt(1))

	for i, a := range answers {
		fmt.Printf("Reconstruction [%d]: %s (%s)\n", i, a.String(), crypto.BigIntToJSON(a))
	}

	for i := 0; i < len(answers)-1; i++ {
		for j := i; j < len(answers); j++ {
			// check they match!
			if i == j {
				continue
			}
			if answers[i].Cmp(answers[j]) != 0 {
				t.Logf("Answers do not all match: %s != %s", answers[i].String(), answers[j].String())
				t.Fail()
			}
		}
	}

	if answers[0].Int64() != secretValue {
		t.Logf("Incorrect decrypted result")
		t.Fail()
	}

	// // final dlog
	// dlog := DiscreteLogLookup(sys.System, 20, []*big.Int{answers[0]})
	// for i, a := range answers {
	// 	d := dlog(a)
	// 	fmt.Printf("DiscreteLog[%d] for %s: %d\n", i, a.String(), d)
	// 	if d != uint64(secretValue) {
	// 		t.Logf("DiscreteLogOfAnswer is incorrect")
	// 		t.Fail()
	// 	}
	// }

}
