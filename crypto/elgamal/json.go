package elgamal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
)

// Having this file is a bit of a shame.
//
// Go natively JSON encodes big.Int values as json numbers, like: `{"n": 100 }` which is fair enough
// when the numbers are small, but will cause interop problems when they get bigger. Also, the decimal
// representation is massive when the numbers are 2048bit integers.
//
// Go offers no control over the JSON representation natively unless we create our own type and that is more
// trouble.
//
// So for now we have this file which explicitly defines the behaviour of json marshalling and unmarshalling
// of these types, converting the big.Ints to base64url encoded strings of the bytes. We can do this
// because all the strings will be >= 0, and big.Int.Bytes() is the absolute value (unsigned big-endian bytes)
//
// Ideally I would prefer to use reflection and just tweak the big.Int fields, but again that is more complex than
// maintaining this file.
//
// actually we could probably implement a custom map[string]*big.Int with custom marshalling
// and use mapstruct (or whatever it is called) to do the transformation to a "Key" or "System"

/* https://play.golang.org/p/7jf-hGHNH1K
package main

import (
	"math/big"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"

)

type IntMap map[string]*big.Int

func (m IntMap) MarshalJSON() ([]byte, error) {
	mm := map[string]string {}
	for k, v := range m {
		mm[k] = v.Text(62)
	}
	return json.Marshal(mm)
}

type X struct {
 P *big.Int
Q *big.Int
}



func main() {
	j :=	json.NewEncoder(os.Stdout)
	x := &X{ P: big.NewInt(123456789), Q: big.NewInt(987654321) }
	xm := IntMap{}
	fmt.Println(x)
	j.Encode(x)
	err := mapstructure.Decode(x, &xm)
	fmt.Println("mapstrucute.Decode", err)
	fmt.Println(xm)
	j.Encode(xm)

}

*/

/////////////////// Helpers ///////////////////

func bigIntAtKey(k string, m map[string]interface{}) (*big.Int, error) {
	v, ok := m[k]
	if !ok {
		return nil, fmt.Errorf("No field '%s' in JSON object", k)
	}
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("Invalid type at field '%s' (expecting string, got %s)", k, reflect.TypeOf(v).Kind())
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("Invalid type at field '%s' (expecting unpadded base64url encoded data)", k)
	}
	i := new(big.Int).SetBytes(b)
	return i, nil
}

func toJSON(x *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(x.Bytes())
}

func getMap(b []byte) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	err := json.Unmarshal(b, m)
	return m, err
}

/////////////////// type System ///////////////////

func (s *System) toJSON() map[string]interface{} {
	return map[string]interface{}{
		"p": toJSON(s.P),
		"q": toJSON(s.Q),
		"g": toJSON(s.G),
	}
}
func (s *System) fromJSON(m map[string]interface{}) (err error) {
	// it should have P Q G
	s.P, err = bigIntAtKey("p", m)
	if err != nil {
		return err
	}
	s.Q, err = bigIntAtKey("q", m)
	if err != nil {
		return err
	}
	s.G, err = bigIntAtKey("g", m)
	return err
}

func (s *System) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toJSON())
}

func (s *System) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	return s.fromJSON(m)
}

/////////////////// type Public Key ///////////////////

func (pk *PublicKey) toJSON() map[string]interface{} {
	// don't include this data in the JSON encoding
	// we must add it after JSON decoding
	// m := pk.System.toJSON()
	//m["y"] = toJSON(pk.Y)
	//return m
	return map[string]interface{}{"y": toJSON(pk.Y)}
}
func (pk *PublicKey) fromJSON(m map[string]interface{}) (err error) {
	// don't encode this
	// if err = pk.System.fromJSON(m); err != nil {
	// 	return err
	// }
	pk.Y, err = bigIntAtKey("y", m)
	return err
}

func (pk *PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(pk.toJSON())
}

func (pk *PublicKey) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	return pk.fromJSON(m)
}

/////////////////// type Secret Key ///////////////////

func (sk *SecretKey) toJSON() map[string]interface{} {
	m := sk.PublicKey.toJSON()
	m["x"] = toJSON(sk.X)
	return m
}
func (sk *SecretKey) fromJSON(m map[string]interface{}) (err error) {
	if err = sk.PublicKey.fromJSON(m); err != nil {
		return err
	}
	sk.X, err = bigIntAtKey("x", m)
	return err
}

func (sk *SecretKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(sk.toJSON())
}

func (sk *SecretKey) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	return sk.fromJSON(m)
}

/////////////////// type CipherText ///////////////////

func (ct *CipherText) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"a": toJSON(ct.A),
		"b": toJSON(ct.B),
	})
}

func (ct *CipherText) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	ct.A, err = bigIntAtKey("a", m)
	if err != nil {
		return err
	}
	ct.B, err = bigIntAtKey("b", m)
	return err
}

/////////////////// type Signature ///////////////////

func (s *Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"ch": toJSON(s.Ch),
		"r":  toJSON(s.R),
	})
}

func (s *Signature) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	s.Ch, err = bigIntAtKey("ch", m)
	if err != nil {
		return err
	}
	s.R, err = bigIntAtKey("r", m)
	return err
}

/////////////////// type ProofOkKnowledge ///////////////////

func (pok *ProofOfKnowledge) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"cm": toJSON(pok.Cm),
		"ch": toJSON(pok.Ch),
		"r":  toJSON(pok.R),
	})
}

func (pok *ProofOfKnowledge) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	pok.Cm, err = bigIntAtKey("cm", m)
	if err != nil {
		return err
	}
	pok.Ch, err = bigIntAtKey("ch", m)
	if err != nil {
		return err
	}
	pok.R, err = bigIntAtKey("r", m)
	return err
}
