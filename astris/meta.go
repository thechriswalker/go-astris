package astris

// These variables will be linked in at build time
// and are to do with the build/source
var (
	BuildDate string
	Commit    string
	Version   string
)

// This is the protocol version of the election scheme.
// Just in case we change things along the way.
// It is encoded into the genesis block and so the election ID
var (
	AstrisProtocolVersion = "1.0"
	AstrisWorkLevel       = 16 // 2 bytes worth
)
