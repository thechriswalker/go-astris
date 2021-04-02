package elgamal

import (
	"encoding/json"
	"fmt"
	"reflect"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto"
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
	return crypto.BigIntFromJSON(s)
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
		"p": crypto.BigIntToJSON(s.P),
		"q": crypto.BigIntToJSON(s.Q),
		"g": crypto.BigIntToJSON(s.G),
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

func (pk *PublicKey) toJSON() map[string]string {
	return map[string]string{
		"y": crypto.BigIntToJSON(pk.Y),
	}
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

func (sk *SecretKey) toJSON() map[string]string {
	m := sk.PublicKey.toJSON()
	m["x"] = crypto.BigIntToJSON(sk.X)
	return m
}
func (sk *SecretKey) fromJSON(m map[string]interface{}) (err error) {
	sk.PublicKey = &PublicKey{}
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
func (ct *CipherText) toJSON() map[string]string {
	return map[string]string{
		"a": crypto.BigIntToJSON(ct.A),
		"b": crypto.BigIntToJSON(ct.B),
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
	return json.Marshal(map[string]string{
		"c": crypto.BigIntToJSON(s.C),
		"r": crypto.BigIntToJSON(s.R),
	})
}

func (s *Signature) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	s.C, err = bigIntAtKey("c", m)
	if err != nil {
		return err
	}
	s.R, err = bigIntAtKey("r", m)
	return err
}

/////////////////// type ZKP ///////////////////
func (zkp *ZKP) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"a": crypto.BigIntToJSON(zkp.A),
		"b": crypto.BigIntToJSON(zkp.B),
		"c": crypto.BigIntToJSON(zkp.C),
		"r": crypto.BigIntToJSON(zkp.R),
	})
}

func (zkp *ZKP) UnmarshalJSON(b []byte) error {
	m, err := getMap(b)
	if err != nil {
		return err
	}
	zkp.A, err = bigIntAtKey("a", m)
	if err != nil {
		return err
	}
	zkp.B, err = bigIntAtKey("b", m)
	if err != nil {
		return err
	}
	zkp.C, err = bigIntAtKey("c", m)
	if err != nil {
		return err
	}
	zkp.R, err = bigIntAtKey("r", m)
	return err
}

///////////////// type DerivedKeys ///////////////////////////
// just store the Private Keys.
func (dk *DerivedKeys) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]*SecretKey{
		"sig": dk.Sig.Secret(),
		"enc": dk.Enc.Secret(),
	})
}

func (dk *DerivedKeys) UnmarshalJSON(b []byte) error {
	dk.Sig = &KeyPair{}
	dk.Enc = &KeyPair{}
	m := map[string]*SecretKey{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	sig, ok := m["sig"]
	if !ok {
		return fmt.Errorf("No signing key in derivedkeys")
	}
	dk.Sig.sk = sig
	enc, ok := m["enc"]
	if !ok {
		return fmt.Errorf("No encrpytion key in derivedkeys")
	}
	dk.Enc.sk = enc
	return nil
}

func (dk *DerivedKeys) ReSystem(s *System) {
	dk.System = s
	// replace the keypairs is the easiest way
	dk.Sig = keypairForSecret(s, dk.Sig.sk.X)
	dk.Enc = keypairForSecret(s, dk.Enc.sk.X)
}

////////////////// type KeyPair /////////////////////
func (kp *KeyPair) MarshalJSON() ([]byte, error) {
	return json.Marshal(kp.sk)
}

func (kp *KeyPair) UnmarshalJSON(b []byte) error {
	kp.sk = &SecretKey{}
	return json.Unmarshal(b, kp.sk)

}
