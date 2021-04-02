package crypto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	big "github.com/ncw/gmp"
)

func BigIntToJSON(x *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(x.Bytes())
}
func BigIntFromJSON(s string) (*big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("Expecting unpadded base64url encoded data, got: %s", s)
	}
	return new(big.Int).SetBytes(b), nil
}

// slice of *big.Int s
type BigIntSlice []*big.Int

// func (s BigIntSlice) String() string {
// 	strs := make([]string, len(s))
// 	for i, n := range s {
// 		strs[i] = BigIntToJSON(n)
// 	}
// 	return fmt.Sprintf("bigints:%s", strings.Join(strs, "|"))
// }

func (s BigIntSlice) MarshalJSON() ([]byte, error) {
	strs := make([]string, len(s))
	for i, n := range s {
		strs[i] = BigIntToJSON(n)
	}
	return json.Marshal(strs)
}

func (s *BigIntSlice) UnmarshalJSON(b []byte) error {
	var strs []string
	if err := json.Unmarshal(b, &strs); err != nil {
		return err
	}
	bs := make(BigIntSlice, len(strs))
	for i := range strs {
		n, err := BigIntFromJSON(strs[i])
		if err != nil {

		}
		bs[i] = n
	}
	*s = bs
	return nil
}
