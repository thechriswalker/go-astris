package astris

import (
	"bytes"
	"fmt"
	"time"

	"github.com/thechriswalker/go-astris/blockchain"
	"github.com/thechriswalker/go-astris/crypto"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
)

//go:generate stringer -type=PayloadHint
type PayloadHint uint8

const (
	HintUnknown       PayloadHint = 0
	HintElectionSetup PayloadHint = 1
	HintTrusteeShares PayloadHint = 2
	HintTrusteePublic PayloadHint = 3
	HintVoterReg      PayloadHint = 4
	HintBallot        PayloadHint = 5
	HintPartialTally  PayloadHint = 6
)

func NewBlockBase(hint PayloadHint, payload interface{}) (*blockchain.Block, error) {
	var buf bytes.Buffer
	hash, err := CanonicalJSON.EncodeAndHash(&buf, nil, payload)
	if err != nil {
		return nil, err
	}
	blk := &blockchain.Block{
		Header: &blockchain.BlockHeader{
			PayloadHash: SliceToID(hash),
			PayloadHint: uint8(hint),
		},
		Payload: buf.Bytes(),
	}
	return blk, nil
}

// PayloadElectionSetup this is want is loaded into the config file
// We can only add so much at a time, we need input from the previous
type PayloadElectionSetup struct {
	Version          string          `json:"protocolVersion"`
	Name             string          `json:"name"`
	Difficulty       uint            `json:"blockDifficulty"`
	Params           *elgamal.System `json:"encryptionSharedParams"`
	TrusteesRequired int             `json:"trusteesRequired"`
	Candidates       []string        `json:"candidates"`
	MaxChoices       int             `json:"maxChoices"` // how many candidates we are allowed to vote for.
	Trustees         []*TrusteeSetup `json:"trustees"`
	Registrar        *RegistrarSetup `json:"registrar"`
	Timing           *TimingInfo     `json:"timing"`
}

type TrusteeSetup struct {
	Index     int                       `json:"-"`    // this should be added to the setup to make it possible to use the "SignatureMessage" function
	Name      string                    `json:"name"` // this up to the trustee, and is included in the signature to prevent re-use by others
	SigKey    *elgamal.PublicKey        `json:"verificationKey"`
	EncKey    *elgamal.PublicKey        `json:"encryptionKey"`
	EncProof  *elgamal.ProofOfKnowledge `json:"encryptionProof"`
	Exponents crypto.BigIntSlice        `json:"publicExponents"`
	Signature *elgamal.Signature        `json:"signature"`
}

func (t *TrusteeSetup) SignatureMessage() []byte {
	// trustee:%d(index):%s(encKey.Y):%s(publicExponents.join(":"))
	var m bytes.Buffer
	fmt.Fprintf(&m, "trustee:%d:%s:%x", t.Index, t.Name, t.EncKey.Y.Bytes())
	// now all the exponents
	for _, ex := range t.Exponents {
		fmt.Fprintf(&m, ":%x", ex.Bytes())
	}
	return m.Bytes()
}

var _ elgamal.Signable = (*TrusteeSetup)(nil)

type RegistrarSetup struct {
	Name            string             `json:"name"`
	SigKey          *elgamal.PublicKey `json:"verificationKey"`
	RegistrationURL string             `json:"registrationURL"`
	Signature       *elgamal.Signature `json:"signature"`
}

func (r *RegistrarSetup) SignatureMessage() []byte {
	return nil
}

var _ elgamal.Signable = (*RegistrarSetup)(nil)

// like RFC3339, but without TZ INFO!
const TimeSpecFormat = `2006-01-02T15:04:05`

// As we are dealing with dates in the future, we must use wall clock times
// and keep hold of the timezone. Best to avoid ambiguous times as much as possible...
type TimeSpec string

// ToTime converts the spec into a point in time. Note that if the time is in the future
// it is possible that the this function will return a different value closer to the time
// (timezones change)
func (ts TimeSpec) ToTime(zone string) (time.Time, error) {
	// parse the time!
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(TimeSpecFormat, string(ts), loc)
}

type absoluteTimes struct {
	start time.Time
	end   time.Time
}

// NB we must have check the timezone exists, or this will silently fail.
func (tb *TimeBounds) ToAbsolute(zone string) *absoluteTimes {
	start, _ := tb.Opens.ToTime(zone)
	end, _ := tb.Closes.ToTime(zone)
	return &absoluteTimes{
		start: start,
		end:   end,
	}
}

type TimeBounds struct {
	Opens  TimeSpec `json:"opens"`
	Closes TimeSpec `json:"closes"`
}

type TimingInfo struct {
	Timezone              string      `json:"timeZone"`
	ParameterConfirmation *TimeBounds `json:"parameterConfirmation"`
	VoterRegistration     *TimeBounds `json:"voterRegistration"`
	VoteCasting           *TimeBounds `json:"voteCasting"`
	TallyDecryption       *TimeBounds `json:"tallyDecryption"`
}

type PayloadTrusteeShares struct {
	Index  int               `json:"trusteeIndex"` // sender trustee (i) 1-based
	Shares []*EncryptedShare `json:"shares"`       // one for each trustee
}

// EncryptedShare is the private data for the other trustees
type EncryptedShare struct {
	Sender    int                 `json:"-"`         // omit this in the JSON representation, we will fill it from the parent "index"
	Recipient int                 `json:"recipient"` // recipient trustee (j), the array will have one less entry that trustees
	Point     *elgamal.CipherText `json:"point"`     // the point data encrypted with the recipient's public encryption key
	Signature *elgamal.Signature  `json:"signature"` // a signature over the data with the participants signing key
}

var _ elgamal.Signable = (*EncryptedShare)(nil)

func (es *EncryptedShare) SignatureMessage() []byte {
	s := fmt.Sprintf("share:%d:%d:%x:%x", es.Sender, es.Recipient, es.Point.A.Bytes(), es.Point.B.Bytes())
	return []byte(s)
}

// PayloadTrusteePublic is the acknowledgement of the validity of the shares from the other
// participants and the publishing of the public key shard and a pok of the shard of decryption key
type PayloadTrusteePublic struct {
	Index      int                       `json:"trusteeIndex"` // the trustee index 1 <= L
	ShardKey   *elgamal.PublicKey        `json:"shardKey"`     // the shard of the public key
	ShardProof *elgamal.ProofOfKnowledge `json:"shardPoK"`     // pok of the decryption part of the public key
	Signature  *elgamal.Signature        `json:"signature"`    // a signature over the data to authenticate it
}

var _ elgamal.Signable = (*PayloadTrusteePublic)(nil)

func (tp *PayloadTrusteePublic) SignatureMessage() []byte {
	s := fmt.Sprintf("shard:%d:%x", tp.Index, tp.ShardKey.Y.Bytes())
	return []byte(s)
}

type PayloadVoterRegistration struct {
	VoterId    string             `json:"voterId"` //only the registrar knows the mapping
	voterHash  string             // the sha256 lowercased hex version of the VoterId
	SigningKey *elgamal.PublicKey `json:"verificationKey"`
	RSignature *elgamal.Signature `json:"registrarSig"` // from registrar
	VSignature *elgamal.Signature `json:"voterSig"`     // from voter
}

func (vr *PayloadVoterRegistration) VSigMessage() []byte {
	if vr.voterHash == "" {
		vr.voterHash = sha256Hex(vr.VoterId)
	}
	s := fmt.Sprintf("voter:v:%s:%x", vr.voterHash, vr.RSignature.R.Bytes())
	return []byte(s)
}
func (vr *PayloadVoterRegistration) RSigMessage() []byte {
	if vr.voterHash == "" {
		vr.voterHash = sha256Hex(vr.VoterId)
	}
	s := fmt.Sprintf("voter:r:%s:%x", vr.voterHash, vr.SigningKey.Y.Bytes())
	return []byte(s)
}

type PayloadCastVote struct {
	VoterId   string                `json:"voterId"`
	voterHash string                // the sha256 lowercased hex version of the VoterId
	Votes     []*elgamal.CipherText `json:"votes"`     // the encrypted votes
	Proofs    []elgamal.ZKPOr       `json:"proofs"`    // the individual proofs
	Proof     elgamal.ZKPOr         `json:"proof"`     // this is the disjoint overall proof
	Signature *elgamal.Signature    `json:"signature"` // from voter private key
}

func (cv *PayloadCastVote) SignatureMessage() []byte {
	if cv.voterHash == "" {
		cv.voterHash = sha256Hex(cv.VoterId)
	}
	var m bytes.Buffer
	fmt.Fprintf(&m, "ballot:%s", cv.voterHash)
	for _, v := range cv.Votes {
		fmt.Fprintf(&m, "|%x:%x", v.A.Bytes(), v.B.Bytes())
	}
	return m.Bytes()
}

type PayloadPartialTally struct {
	Index     int                   `json:"trusteeIndex"`
	Tallies   []*elgamal.CipherText `json:"tallies"`   // in candidate order
	Decrypted crypto.BigIntSlice    `json:"decrypted"` // in candidate order
	Proofs    []*elgamal.ZKP        `json:"proofs"`    // in candidate order
	Signature *elgamal.Signature    `json:"signature"`
}

func (pt *PayloadPartialTally) SignatureMessage() []byte {
	var m bytes.Buffer
	fmt.Fprintf(&m, "tally:%d", pt.Index)
	for _, t := range pt.Tallies {
		fmt.Fprintf(&m, ":%x|%x", t.A.Bytes(), t.B.Bytes())
	}
	for _, d := range pt.Decrypted {
		fmt.Fprintf(&m, ":%x", d.Bytes())
	}
	return m.Bytes()
}
