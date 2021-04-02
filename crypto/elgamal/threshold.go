package elgamal

import (
	"bytes"
	"fmt"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto"
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

// ThresholdSystem is the distributed decryption, public encryption scheme
// with T+1 participants required to decrypt.
type ThresholdSystem struct {
	*System
	T int // T+1 participants required to decrypt
	L int // Total number of participants
}

// generate the public key shard for a trustee from the published exponents of all trustees.
// assuming 1-based trustees.
// the private key will be $x_i = \sum\limit{i=0}{t} S_{ji}$ the public key is $g^{x_i}
// $g^{S_{ji}} = \prod\limt{k=0}{t} (E_{jk})^{i^k}$  so $y_i = \prod\limit{i=0}{t} g^{S_{ji}}
// So we work out $y_i$ by calculating $g^{S_{ji}}$ for each index and multiplying them.
func (ts *ThresholdSystem) SimulatePublicKeyShard(i int, E map[int]crypto.BigIntSlice) *PublicKey {
	Y, I := big.NewInt(1), big.NewInt(int64(i))
	gSji, iK, x := new(big.Int), new(big.Int), new(big.Int)

	for j := 1; j <= ts.L; j++ {
		//	fmt.Println("Simulation: j =", j, ", i =", i)
		gSji.SetInt64(1)
		iK.SetInt64(1)
		for k := 0; k <= ts.T; k++ {
			// fmt.Println("k =", k)
			// fmt.Println("Ej = ", E[j])
			// fmt.Println("calculating Ejk^i^k, Ejk =", E[j][k], ", i^k =", iK)
			// $E_{jk}^{i^k}$
			x.Exp(E[j][k], iK, ts.P)
			// $g^{S_{ji}} * x$ for the product
			gSji.Mul(gSji, x)
			gSji.Mod(gSji, ts.P)
			// raise $i^k$ another power by multiplying by I
			iK.Mul(iK, I)
			iK.Mod(iK, ts.Q)
		}
		Y.Mul(Y, gSji)
		Y.Mod(Y, ts.P)
	}

	pk := &PublicKey{
		System: ts.System,
		Y:      Y,
	}
	if err := pk.Validate(); err != nil {
		panic(err)
	}
	return pk
}

func CreateExponents(s *System, coeffs []*big.Int) []*big.Int {
	exponents := make([]*big.Int, len(coeffs))
	for i, c := range coeffs {
		exponents[i] = new(big.Int).Exp(s.G, c, s.P)
	}
	return exponents
}

func (pp *PrivateParticipant) CreateSecretShare(j int) *big.Int {
	bigJ := big.NewInt(int64(j))
	// we can recreate the polynomial by working backwards from t to 0
	Sij := big.NewInt(0)
	for n := pp.Sys.T; n >= 0; n-- {
		// each power is multiplied (to exponentiate) and the next (lower) coefficient is added
		Sij.Mul(Sij, bigJ)
		Sij.Add(Sij, pp.Coeffs[n])
		Sij.Mod(Sij, pp.Sys.Q)
	}
	//fmt.Printf("SecretShareFor i=%d,j=%d Sij=%v\n", pp.Index, j, Sij)
	return Sij
}

type PrivateParticipant struct {
	Sys       *ThresholdSystem
	Index     int
	Coeffs    crypto.BigIntSlice
	PublicExp map[int]crypto.BigIntSlice
	ShardKey  *KeyPair
}

func (pp *PrivateParticipant) CheckSecretShareFrom(j int, Sji *big.Int) bool {
	bigI := big.NewInt(int64(pp.Index))
	calc, iK := big.NewInt(1), big.NewInt(1)
	tmp := new(big.Int)
	exponents := pp.PublicExp[j]
	//fmt.Println("exponents", exponents)
	for k := 0; k <= pp.Sys.T; k++ {
		tmp.Exp(exponents[k], iK, pp.Sys.P)
		calc.Mul(calc, tmp)
		calc.Mod(calc, pp.Sys.P)
		// raise $i^k$ another power by multiplying by I
		iK.Mul(iK, bigI)
		iK.Mod(iK, pp.Sys.Q)
	}
	// make tmp into G^Sji from the share.
	//fmt.Println("G", pp.Sys.G, "Sji", Sji, "P", pp.Sys.P, ", tmp=", tmp)
	tmp.Exp(pp.Sys.G, Sji, pp.Sys.P)
	//fmt.Println("calculated g^Sji =", calc, ", expected: ", tmp)
	return tmp.Cmp(calc) == 0
}

func (pp *PrivateParticipant) CombineSharesSharedKeys(shareFn func(j, i int) *big.Int) {
	// private key is SIGMA from j=1->L s_ji
	x := big.NewInt(0)
	for j := 1; j <= pp.Sys.L; j++ {
		if j == pp.Index {
			// this is us. we do this ourselves.
			x.Add(x, pp.CreateSecretShare(j))
		} else {
			// add the secret share from trustee j for us
			x.Add(x, shareFn(j, pp.Index))
		}
		x.Mod(x, pp.Sys.Q)
	}
	pp.ShardKey = keypairForSecret(pp.Sys.System, x)
}

// Perform the partial decryption.
func (pp *PrivateParticipant) PartialDecrypt(ct *CipherText) *big.Int {
	// fmt.Printf("%#v\n", pp.ShardKey.Secret())
	// fmt.Printf("%#v\n", pp.Sys.System)
	// fmt.Printf("%#v\n", ct)
	return new(big.Int).Exp(ct.A, pp.ShardKey.Secret().X, pp.Sys.P)
}

// mod-inverse of all the factors except the current index
func lagrange(indices []int, index int, modulus *big.Int) (r *big.Int) {
	r = new(big.Int).Set(bigOne)
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

func DeriveCoefficients(params *System, secret *big.Int, t int) []*big.Int {
	// t+1 is the number of coefficients we want.
	// secret is our random secret that gives our deterministic "random"
	// coefficients and keys.
	// OK so we just want the coefficients.
	coefficients := make([]*big.Int, t+1) // t+1 coefficients i.e. c0 + c1*x + c2 * x^2 ... + ct * x^t

	buf := &bytes.Buffer{}

	for i := range coefficients {
		buf.Reset()
		fmt.Fprintf(
			buf,
			"coef|%x|%d|%x|%d",
			params.P.Bytes(),
			t,
			secret.Bytes(),
			i,
		)
		coefficients[i] = random.Oracle(buf.Bytes(), params.Q)
	}
	return coefficients
}

// decryption is ct.Beta * (SIGMA(j){ c_j^LI_j}) ^-1
// that is multiplicative sum over our `j` which
// are the chosen t+1 indexes
// c_j is the partial decryption from trustee j
// and LI_j is the langrange coefficient j from
// the set.
// NB partials will be sparse and indices must 1-based trustee indices
func ThresholdDecrypt(s *System, ct *CipherText, partials []*big.Int, indices []int) *big.Int {
	// lets calculate the sigma first.
	// fmt.Printf("Decrypt: A=%s B=%s\n", ct.A, ct.B)
	// fmt.Printf("Decrypt: Prtials=%v\n", partials)
	// fmt.Printf("Decrypt: indices=%v\n", indices)
	sigma := big.NewInt(1)
	for _, j := range indices {

		cj := partials[j]
		//fmt.Printf("c_%d = %s\n", j, cj)
		lij := lagrange(indices, j, s.Q)
		//fmt.Printf("LI_%d = %s\n", j, lij)
		raised := new(big.Int).Exp(cj, lij, s.P)
		//fmt.Printf("c_%d^LI_%d = %s\n", j, j, raised)
		sigma.Mul(sigma, raised)
		sigma.Mod(sigma, s.P)
		//fmt.Printf("Sigma[j=%d] %s\n", j, sigma)
	}

	// now inverse and multiply.
	sigma.ModInverse(sigma, s.P)
	//fmt.Printf("Sigma^-1: %s\n", sigma)
	sigma.Mul(sigma, ct.B)
	sigma.Mod(sigma, s.P)
	//fmt.Printf("Sigma^-1*beta: %s\n", sigma)
	return sigma
}
