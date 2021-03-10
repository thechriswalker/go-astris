package astris

import (
	"encoding/json"
	"io"
	"net"
)

type Node interface {
	// update our known peers list that we have a sighting of this peer
	PeerSeen(addr net.Addr)
}

// CanonicalJSONEncoder is a helper to write out canonically encoded json.
// it sorts map keys and removes all extraneous whitespace.
// useful to creating canonical representations of data, and
// also to check hashes against in-memory objects (rather than
// pre-existing byte slices)
func CanonicalJSONEncoder(out io.Writer) func(v interface{}) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "")
	enc.SetEscapeHTML(false)
	return func(v interface{}) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var t interface{}
		err = json.Unmarshal(b, &t)
		if err != nil {
			return err
		}
		// t is map[string]interface instead of struct, so the keys will be sorted.
		return enc.Encode(t)
	}
}
