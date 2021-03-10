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
// maintaining this file in the short term
//
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
	err := json.Unmarshal(b, &m)
	// if err != nil {
	// 	panic(err)
	// }
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
	if err != nil {
		return err
	}
	// now validate that those params are actually valid
	return s.Validate()
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

/////////////////// type ThresholdSystem ///////////////////////
func (ts *ThresholdSystem) MarshalJSON() ([]byte, error) {
	m := ts.System.toJSON()
	m["t"] = ts.T
	m["l"] = ts.L
	return json.Marshal(m)
}

func (ts *ThresholdSystem) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	var ok bool
	var x float64
	var v interface{}
	v, ok = m["t"]
	if !ok {
		return fmt.Errorf("No 't' value in JSON")
	}
	x, ok = v.(float64)
	ts.T = int(x)
	if !ok || float64(ts.T) != x {
		return fmt.Errorf("Non integer 't' value in JSON")
	}
	v, ok = m["l"]
	if !ok {
		return fmt.Errorf("No 'l' value in JSON")
	}
	x, ok = v.(float64)
	ts.L = int(x)
	if !ok || float64(ts.L) != x {
		return fmt.Errorf("Non integer 'l' value in JSON")
	}

	if ts.T < 0 || ts.L < ts.T+1 {
		return fmt.Errorf("Invalid threshold parameters in JSON")
	}
	ts.System = &System{}
	return ts.System.fromJSON(m)

}

/////////////////// type Public Key ///////////////////

func (pk *PublicKey) toJSON() map[string]interface{} {
	return map[string]interface{}{"y": toJSON(pk.Y)}
}
func (pk *PublicKey) fromJSON(m map[string]interface{}) (err error) {
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
	// should we check the pk is valid?
	// not yet as we don't have the system parameters yet
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

// we could use α and β as keys here, but ascii is simpler
func (ct *CipherText) toJSON() map[string]interface{} {
	return map[string]interface{}{
		"a": toJSON(ct.A),
		"b": toJSON(ct.B),
	}
}

func (ct *CipherText) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.toJSON())
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
		"c": toJSON(s.Ch),
		"r": toJSON(s.R),
	})
}

func (s *Signature) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	s.Ch, err = bigIntAtKey("c", m)
	if err != nil {
		return err
	}
	s.R, err = bigIntAtKey("r", m)
	return err
}

/////////////////// type ProofOkKnowledge ///////////////////

func (pok *ProofOfKnowledge) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"m": toJSON(pok.Cm),
		"c": toJSON(pok.Ch),
		"r": toJSON(pok.R),
	})
}

func (pok *ProofOfKnowledge) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	pok.Cm, err = bigIntAtKey("m", m)
	if err != nil {
		return err
	}
	pok.Ch, err = bigIntAtKey("c", m)
	if err != nil {
		return err
	}
	pok.R, err = bigIntAtKey("r", m)
	return err
}
