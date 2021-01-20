package astris

import "net"

type config struct {
	maxPeers       int    // the max peers to connect to
	httpListen     string // the web interface listen address
	grpcListen     string // the local GRPC listen address
	grpcPublic     string // the publically accessible peer address
	blockchainFile string // the sqlite3 database file
}

// ConfigOption is used to configure Astris
type ConfigOption interface {
	apply(c *config)
}

type cf func(*config)

func (fn cf) apply(c *config) {
	fn(c)
}

var _ ConfigOption = cf(func(c *config) {})

// WithMaxPeers controls the number of peers we will connect to
// n > 0 => n is the maximum number of peers we will seek out
// n = 0 => don't try to connect to any other peers.
// n < 0 => illegal
func WithMaxPeers(n int) cf {
	if n < 0 {
		panic("WithMaxPeers called with a negative int")
	}
	return func(c *config) {
		c.maxPeers = n
	}
}

// WithHTTP configures the http listen address for the web interface
func WithHTTP(addr string) cf {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		panic("WithHTTP called with invalid host:port")
	}
	return func(c *config) {
		c.httpListen = addr
	}
}

// WithGRPC configures the grpc listen address
func WithGRPC(addr string) cf {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		panic("WithGRPC called with invalid host:port")
	}
	return func(c *config) {
		c.grpcListen = addr
	}
}

// WithPublicGRPC sets the publically accessible address for the
// node.
// If not set, the GRPC listen address is used.
// If the server is behind a NAT or a reverse proxy then this
// will need to be set, probably with a FQDN.
func WithPublicGRPC(addr string) cf {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		panic("WithPublicGRPC called with invalid host:port")
	}
	return func(c *config) {
		c.grpcPublic = addr
	}
}

// WithBlockchainFile sets the path to the sqlite database storing the persistent data.
func WithBlockchainFile(file string) cf {
	return func(c *config) {
		c.blockchainFile = file
	}
}
