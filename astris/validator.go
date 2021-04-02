package astris

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/thechriswalker/go-astris/blockchain"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
	"github.com/thechriswalker/go-astris/timezone"
)

var _ blockchain.BlockValidator = (*ElectionValidator)(nil)

// should have a "loose" mode for when I trust that the data is good
// and just want to catch up to state.
// In that mode we don't validate signatures or ZKPs

type ElectionValidator struct {
	electionID ID
	// some internal switches. note that we have no mutex, so right now, no concurrent access.
	isRealtime bool // are we validating in realtime? or validating a chain that already exists.
	// for the ZKPs
	plaintextOptions *elgamal.PlaintextOptionsCache
	// the payload hash validation
	payloadValidator blockchain.PayloadHashValidator
	// worklevel for PoW
	workLevel int

	// state so far
	state *ElectionState

	// if true, do less checks (faster, but not fully secure)
	// I use it in the simulator to get state up to date when I have validated it previously
	// it only affects voter registration and casting as that is usually orders of magnitude
	// more blocks than anything else.
	// NB it's exposed so we can modify it "on the fly"
	LooseMode bool
}

func NewElectionValidator(id ID) *ElectionValidator {
	return &ElectionValidator{
		electionID: id,
		workLevel:  AstrisWorkLevel, // start with this
	}
}

func (ev *ElectionValidator) WorkLevel() int {
	return ev.workLevel
}

func (ev *ElectionValidator) System() *elgamal.System {
	return ev.state.immutableSetupData.Params
}

func (ev *ElectionValidator) GetResult() *ElectionStats {
	return ev.state.GetResult()
}

func (ev *ElectionValidator) GetLocalTally() []*elgamal.CipherText {
	return ev.state.GetLocalTally()
}

func (ev *ElectionValidator) ElectionPublicKey() *elgamal.PublicKey {
	return ev.state.ElectionPublicKey()
}

func (ev *ElectionValidator) GetNumTrustees() int {
	return len(ev.state.immutableSetupData.Trustees)
}

// func (ev *electionValidator) Clone() blockchain.BlockValidator {
// 	return &electionValidator{
// 		isRealtime: ev.isRealtime,
// 		state:      ev.state.Clone(),
// 	}
// }

func checkCanonical(payload interface{}, blk *blockchain.Block) bool {
	return CanonicalJSON.HashCheck(payload, blk.Header.PayloadHash[:])
}

// We have to pass in a when
func (ev *ElectionValidator) Validate(blk *blockchain.Block) (err error) {
	start := time.Now()
	defer func() {
		log.Debug().
			Str("block", blk.Header.ID.String()).
			Int("depth", int(blk.Header.Depth)).
			Dur("ms", time.Now().Sub(start)).
			Err(err).
			Msg("Block Validation")
	}()
	// validate the next block on this chain.
	// all blocks must pass the payload hash validation
	if err = blockchain.PayloadHashValidator(0).Validate(blk); err != nil {
		return
	}
	// if our state is `nil` then this should be the genesis block
	if ev.state == nil {
		err = ev.checkGenesis(blk)
		return
	}
	// if not then we need to check based on the timestamp of the block
	// we should have a timestamp > the previous and based on the timing
	// data when should this happen.
	// But when we retro-spectively validate a chain, we have to trust
	// that the timing is OK. This could be a downfall in the protocol...
	// either way, we need to know the current time to validate whether this
	// block is A) a reasonable timestamp, B) in the correct phase.
	if ev.isRealtime {
		// @todo: then NOW is the election phase
		// we must validate the block time is close enough to our local time to be valid to be added to the chain
		// it should also be greater than that of the previous block.
	}
	// OK let's validate the next block by type.
	// first we find the current phase.
	switch ev.state.GetPhaseForTime(blk.Header.GetTime()) {
	case 1: // parameter confirmation, if we have all the initial payloads, this can be the second round.
		if !ev.state.HasAllTrusteeShares() {
			err = ev.checkTrusteeShares(blk)
			return
		}
		if !ev.state.HasAllTrusteePublic() {
			err = ev.checkTrusteePublic(blk)
			return
		}
		err = fmt.Errorf("Already have all trustee data, no more valid blocks for this phase")
		return
	case 2: // voter registration, only those blocks are valid.
		err = ev.checkVoterRegistration(blk)
		return
	case 3: // vote casting, only cast votes are valid.
		err = ev.checkVoteCast(blk)
		return
	case 4: // tally decryptions only
		err = ev.checkPartialTally(blk)
		return
	default:
		// not in a valid time to add any blocks.
		fmt.Printf("%v\n", blk.Header.GetTime())
		err = fmt.Errorf("Block Timestamp is part of a valid election phase")
		return
	}

}

// share from index J to index I, i.e. $S_{ji}$
func (ev *ElectionValidator) GetEncryptedSecretShare(j, i int) *elgamal.CipherText {
	//fmt.Printf("%#v\n", ev.state.trusteeShares)
	return ev.state.trusteeShares[j][i]
}

// Checks the Genesis Block
func (ev *ElectionValidator) checkGenesis(blk *blockchain.Block) error {
	// the PREV should be 0's, and depth 0
	if !blk.Header.IsGenesis() {
		return fmt.Errorf("Expecting Genesis Block")
	}
	// the time should be less than or equal to now.
	t := blk.Header.GetTime()
	if time.Now().Before(t) {
		return fmt.Errorf("Genesis Block timestamp is in the future: %s", t.Format(time.RFC3339))
	}
	if blk.Header.PayloadHint != uint8(HintElectionSetup) {
		return fmt.Errorf("Genesis block has invalid payload hint: expected %d, got %d", HintElectionSetup, blk.Header.PayloadHint)
	}
	//fmt.Println(">>>>>>>>>>>>>>>\n", string(blk.Payload), ">>>>>>>>>>>>>>>>>")
	// we are interested in the payload.
	setupData := &PayloadElectionSetup{}
	if err := json.Unmarshal(blk.Payload, setupData); err != nil {
		return fmt.Errorf("unable to unmarshal genesis block payload: %w", err)
	}
	if !checkCanonical(setupData, blk) {
		return fmt.Errorf("Genesis block is not Canonically Encoded")
	}
	if blk.Header.ID != ev.electionID {
		return fmt.Errorf("Genesis block ID does not match given election ID")
	}

	// now we must check the payload. It has it's own validation function.
	return ev.validateSetupDataPayload(setupData, t, blk.Header.ID)
}

// Validate the data in the object and if valid, update our validator state.
func (ev *ElectionValidator) validateSetupDataPayload(data *PayloadElectionSetup, minTime time.Time, id ID) error {
	if data.Version != AstrisProtocolVersion {
		return fmt.Errorf("Unknown Astris Protocol Version: %s (expected %s)", data.Version, AstrisProtocolVersion)
	}
	// technically name can be empty, so we will not validate it.
	// block difficulty must be in the range [0, id size in bits = 32 * 8 = 256)
	if data.Difficulty > 255 {
		return fmt.Errorf("Block Difficulty out of range")
	}
	// check the parameters.
	if err := data.Params.Validate(); err != nil {
		return fmt.Errorf("Encryption Params Invalid: %w", err)
	}
	// candidates. there should be more than one, and maxChoices should be in the range [1,len(candidates)]
	lc := len(data.Candidates)
	if lc < 2 {
		return fmt.Errorf("Must be at least 2 candidates, only %d present", lc)
	}
	if data.MaxChoices < 1 || data.MaxChoices > lc {
		return fmt.Errorf("Max choices must be between 1 and the number of candidates (%d): got %d", lc, data.MaxChoices)
	}
	// now trustees, we demand at least 3 trustees and at least 2 as the thresold.
	lt := len(data.Trustees)
	if lt < 3 {
		return fmt.Errorf("Must be at least 3 trustees, only %d present", lt)
	}
	if data.TrusteesRequired < 2 || data.TrusteesRequired > lt {
		return fmt.Errorf("Trustees required must be between 2 and the number of trustees (%d): got %d", lt, data.TrusteesRequired)
	}
	// now we need to validate each trustee.
	// we should also check for uniqueness of signing and encryption
	// keys, and exponents.
	unqSigKey := make(map[string]struct{}, lt)
	unqEncKey := make(map[string]struct{}, lt)
	unqExponents := make(map[string]struct{}, lt)
	for i, trustee := range data.Trustees {
		idx := i + 1 // trustees use 1-based indexing
		if trustee == nil {
			return fmt.Errorf("Trustee[%d] data missing", idx)
		}
		sig := trustee.SigKey.String()
		if _, ok := unqSigKey[sig]; ok {
			return fmt.Errorf("Trustee[%d] has a previously seen signing key", idx)
		} else {
			unqSigKey[sig] = struct{}{}
		}
		enc := trustee.EncKey.String()
		if _, ok := unqEncKey[enc]; ok {
			return fmt.Errorf("Trustee[%d] has a previously seen encryption key", idx)
		} else {
			unqEncKey[enc] = struct{}{}
		}
		exp := fmt.Sprintf("%v", trustee.Exponents)
		if _, ok := unqExponents[exp]; ok {
			return fmt.Errorf("Trustee[%d] has a previously seen public exponent set", idx)
		} else {
			unqExponents[exp] = struct{}{}
		}
		// OK, uniqueness check done.
		// the unmarshalling will not have added the "System" to the keys
		trustee.SigKey.System = data.Params
		trustee.EncKey.System = data.Params
		// nor the "index"
		trustee.Index = idx
		// now validate keys
		if err := trustee.SigKey.Validate(); err != nil {
			return fmt.Errorf("Trustee[%d] has an invalid signing key: %w", idx, err)
		}
		if err := trustee.EncKey.Validate(); err != nil {
			return fmt.Errorf("Trustee[%d] has an invalid encryption key: %w", idx, err)
		}
		// validate the PoK of the secret key
		if err := trustee.EncKey.VerifyProof(trustee.EncProof); err != nil {
			return fmt.Errorf("Trustee[%d] has an invalid PoK of encryption key: %w", idx, err)
		}
		// validate the signature
		if err := trustee.SigKey.Verify(trustee, trustee.Signature); err != nil {
			fmt.Printf("%#v\n", trustee.Signature)
			return fmt.Errorf("Trustee[%d] has a invalid signature: %w", idx, err)
		}
		// all good. phew
	}

	// now validate the registrar info.
	if data.Registrar == nil {
		return fmt.Errorf("Registrar data missing")
	}
	// the registrar URL should be a valid http or https URL.
	rURL, err := url.Parse(data.Registrar.RegistrationURL)
	if err != nil {
		return fmt.Errorf("Registrar URL invalid: %w", err)
	}
	if rURL.Scheme != "http" && rURL.Scheme != "https" {
		return fmt.Errorf("Registrar URL not HTTP(S) endpoint got: %s", rURL.Scheme)
	}
	// validate key
	data.Registrar.SigKey.System = data.Params
	if err := data.Registrar.SigKey.Validate(); err != nil {
		return fmt.Errorf("Registrar Siging Key invalid: %w", err)
	}
	// validate signature.
	if err := data.Registrar.SigKey.Verify(data.Registrar, data.Registrar.Signature); err != nil {
		return fmt.Errorf("Registrar Data signature error: %w", err)
	}

	// now validate timing data.
	if data.Timing == nil {
		return fmt.Errorf("Timing Data missing")
	}
	if !timezone.IsValidZone(data.Timing.Timezone) {
		return fmt.Errorf("Timing data timezone is invalid: %s", data.Timing.Timezone)
	}
	// now we need to validate all sections are non-overlapping
	// and that they all start _after_ the genesis block timestamp.
	var t time.Time
	t, err = checkPhase(data.Timing.ParameterConfirmation, data.Timing.Timezone, minTime)
	if err != nil {
		return fmt.Errorf("Timing for ParameterConfirmation invalid: %w", err)
	}
	t, err = checkPhase(data.Timing.VoterRegistration, data.Timing.Timezone, t)
	if err != nil {
		return fmt.Errorf("Timing for VoterRegistration invalid: %w", err)
	}
	t, err = checkPhase(data.Timing.VoteCasting, data.Timing.Timezone, t)
	if err != nil {
		return fmt.Errorf("Timing for VoteCasting invalid: %w", err)
	}
	t, err = checkPhase(data.Timing.TallyDecryption, data.Timing.Timezone, t)
	if err != nil {
		return fmt.Errorf("Timing for TallyDecryption invalid: %w", err)
	}

	// wow, we all all good add the setupdata to state.
	ev.state = newElectionState(data)
	ev.plaintextOptions = elgamal.NewPlaintextOptionsCache(data.Params)
	ev.workLevel = int(data.Difficulty)
	return nil
}

func checkPhase(bounds *TimeBounds, zone string, min time.Time) (time.Time, error) {
	if bounds == nil {
		return min, fmt.Errorf("Bounds missing")
	}
	t1, err := bounds.Opens.ToTime(zone)
	if err != nil {
		return min, fmt.Errorf("Opens timespec invalid: %w", err)
	}
	t2, err := bounds.Closes.ToTime(zone)
	if err != nil {
		return min, fmt.Errorf("Closes timespec invalid: %w", err)
	}
	if t1.Before(min) {
		return min, fmt.Errorf("Opens time is too early")
	}
	if t2.Before(t1) {
		return min, fmt.Errorf("Closes time is before opens time")
	}
	return t2, nil
}

// this block should contain the trustee Shares payload.
func (ev *ElectionValidator) checkTrusteeShares(blk *blockchain.Block) error {
	if blk.Header.PayloadHint != uint8(HintTrusteeShares) {
		return fmt.Errorf("Expecting a TrusteeShares block (%d) got %d (%s)", HintTrusteeShares, blk.Header.PayloadHint, PayloadHint(blk.Header.PayloadHint))
	}
	s := &PayloadTrusteeShares{}
	if err := json.Unmarshal(blk.Payload, s); err != nil {
		return fmt.Errorf("Error unmarshalling Trustee Shares payload: %w", err)
	}
	if !checkCanonical(s, blk) {
		return fmt.Errorf("Trustee Shares payload was not canonically encoded")
	}
	// we need to "check this"
	idx := s.Index - 1
	if idx < 0 || idx >= len(ev.state.immutableSetupData.Trustees) {
		return fmt.Errorf("Alleged Trustee %d is invalid", s.Index)
	}
	trustee := ev.state.immutableSetupData.Trustees[idx]
	if trustee == nil {
		return fmt.Errorf("Alleged Trustee %d state is unknown", s.Index)
	}

	// OK, we should not have seen this trustee before.
	if _, ok := ev.state.trusteeShares[s.Index]; ok {
		// we have seen this one before.
		return fmt.Errorf("Trustee[%d] has already added their shares", s.Index)
	}

	expectedLength := len(ev.state.immutableSetupData.Trustees) - 1
	maxIndex := expectedLength + 1 // 1-based for trustees
	if len(s.Shares) != expectedLength {
		return fmt.Errorf("Trustee[%d] shares length is not correct: expecting %d got %d", s.Index, expectedLength, len(s.Shares))
	}
	shares := map[int]*elgamal.CipherText{}
	// now we process each share.
	for _, encshare := range s.Shares {
		// we expected OK to be true and curr to be nil
		curr, _ := shares[encshare.Recipient]
		if curr != nil {
			return fmt.Errorf("Trustee[%d] provided share for duplicate recipient %d", s.Index, encshare.Recipient)
		}
		if encshare.Recipient < 1 || encshare.Recipient > maxIndex || encshare.Recipient == s.Index {
			// this is not a valid recipient.
			return fmt.Errorf("Trustee[%d] provided invalid share recipient %d", s.Index, encshare.Recipient)
		}
		// OK we can check the share.
		encshare.Sender = s.Index
		if err := trustee.SigKey.Verify(encshare, encshare.Signature); err != nil {
			return fmt.Errorf("Trustee[%d] invalid signature for share[%d]: %w", s.Index, encshare.Recipient, err)
		}
		// share is valid. add to list
		shares[encshare.Recipient] = encshare.Point
	}
	// block is good, update the state to show we have the shares from this trustee.
	ev.state.trusteeShares[s.Index] = shares
	// all good
	return nil
}

func (ev *ElectionValidator) checkTrusteePublic(blk *blockchain.Block) error {
	if blk.Header.PayloadHint != uint8(HintTrusteePublic) {
		return fmt.Errorf("Expecting a TrusteePublic block (%d) got %d (%s)", HintTrusteePublic, blk.Header.PayloadHint, PayloadHint(blk.Header.PayloadHint))
	}
	s := &PayloadTrusteePublic{}
	if err := json.Unmarshal(blk.Payload, s); err != nil {
		return fmt.Errorf("Error unmarshalling Trustee Public payload: %w", err)
	}
	if !checkCanonical(s, blk) {
		return fmt.Errorf("Trustee Public payload was not canonically encoded")
	}
	// we need to "check this"
	idx := s.Index - 1 // indices in the data structures are 1-based
	if idx < 0 || idx >= len(ev.state.immutableSetupData.Trustees) {
		return fmt.Errorf("Alleged Trustee %d is invalid", s.Index)
	}
	s.ShardKey.System = ev.System()
	trustee := ev.state.immutableSetupData.Trustees[idx]
	if trustee == nil {
		return fmt.Errorf("Alleged Trustee %d state is unknown", s.Index)
	}
	// have we recieved this data from the trustee yet?
	if _, ok := ev.state.trusteePublic[s.Index]; ok {
		return fmt.Errorf("Trustee[%d] has already added their public data", s.Index)
	}
	// ok, we need to validate the public data, but we will check the signatures first.
	if err := trustee.SigKey.Verify(s, s.Signature); err != nil {
		return fmt.Errorf("Trustee[%d] Public data signature error: %w", s.Index, err)
	}
	// and the PoK - this is the important one as anyone could calculate the public key
	if err := s.ShardKey.VerifyProof(s.ShardProof); err != nil {
		return fmt.Errorf("Trustee[%d] Public Shard Key PoK error: %w", s.Index, err)
	}
	// OK, now we need to validate the shard key is what we expect
	if err := ev.state.ValidateShardKey(s.Index, s.ShardKey); err != nil {
		return fmt.Errorf("Trustee[%d] Shard Key value incorrect: %w", s.Index, err)
	}

	// all good add the key and continue
	ev.state.trusteePublic[s.Index] = s.ShardKey
	return nil
}

func (ev *ElectionValidator) checkVoterRegistration(blk *blockchain.Block) error {
	if blk.Header.PayloadHint != uint8(HintVoterReg) {
		return fmt.Errorf("Expecting a VoterRegistration block (%d) got %d (%s)", HintVoterReg, blk.Header.PayloadHint, PayloadHint(blk.Header.PayloadHint))
	}
	vr := &PayloadVoterRegistration{}
	if err := json.Unmarshal(blk.Payload, vr); err != nil {
		return fmt.Errorf("Error unmarshalling Voter Registration payload: %w", err)
	}
	if !checkCanonical(vr, blk) {
		return fmt.Errorf("VoterRegistration payload was not canonically encoded")
	}

	// verify that we haven't seen this voter before. We actually take a sha256 as hex string to perform the mapping.
	vr.voterHash = sha256Hex(vr.VoterId)

	if _, ok := ev.state.voters[vr.voterHash]; ok {
		// we have seen this one.
		return fmt.Errorf("Voter already registered: %s", vr.VoterId)
	}

	// check key is valid.
	vr.SigningKey.System = ev.state.immutableSetupData.Params
	if err := vr.SigningKey.Validate(); err != nil {
		return fmt.Errorf("Voter Signing Key is invalid: %w", err)
	}

	// check signatures unless in loose mode
	if !ev.LooseMode {
		// check registrar signature first
		if err := ev.state.immutableSetupData.Registrar.SigKey.VerifySignature(vr.RSignature, vr.RSigMessage()); err != nil {
			return fmt.Errorf("Registrar Signature on voter registration invalid: %w", err)
		}
		if err := vr.SigningKey.VerifySignature(vr.VSignature, vr.VSigMessage()); err != nil {
			return fmt.Errorf("VoterSignature Signature on voter registration invalid: %w", err)
		}
	}

	// they all validate. grand.
	ev.state.voters[vr.voterHash] = &VoterState{
		key: vr.SigningKey.Y,
	}

	return nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func (ev *ElectionValidator) checkVoteCast(blk *blockchain.Block) error {
	if blk.Header.PayloadHint != uint8(HintBallot) {
		return fmt.Errorf("Expecting a CastVote block (%d) got %d (%s)", HintBallot, blk.Header.PayloadHint, PayloadHint(blk.Header.PayloadHint))
	}
	cv := &PayloadCastVote{}
	if err := json.Unmarshal(blk.Payload, cv); err != nil {
		return fmt.Errorf("Error unmarshalling CastVote payload: %w", err)
	}
	if !checkCanonical(cv, blk) {
		return fmt.Errorf("CastVote payload was not canonically encoded")
	}

	// voter must exist
	cv.voterHash = sha256Hex(cv.VoterId)
	v, ok := ev.state.voters[cv.voterHash]
	if !ok {
		return fmt.Errorf("unknown voterid in cast vote: %s", cv.VoterId)
	}
	pk := &elgamal.PublicKey{
		System: ev.state.immutableSetupData.Params,
		Y:      v.key,
	}

	if !ev.LooseMode {
		// check signature first.
		if err := pk.Verify(cv, cv.Signature); err != nil {
			return fmt.Errorf("Voter Signature on CastVote is invalid: %w", err)
		}

		// now check the proofs.
		// start with each of the individual proofs.
		// we encrypt $g^v$ for a vote $v$
		zeroOrOne := ev.plaintextOptions.GetOptions(1)
		// we also homomorphically sum the ciphertexts (well multiply)
		// so we have a ciphertext for the final proof.
		var ctSum *elgamal.CipherText
		for i, zkp := range cv.Proofs {
			if err := elgamal.VerifyEncryptionProof(zkp, ev.ElectionPublicKey(), cv.Votes[i], zeroOrOne, []byte(cv.voterHash)); err != nil {
				fmt.Println(pk.Y)
				return fmt.Errorf("ZeroKnowledgeProof for EncryptedVote[%d] is invalid: %w", i+1, err)
			}
			ctSum = ctSum.Mul(pk.System, cv.Votes[i])
		}

		// now the overall proof.

		// the options will be cached, as they will be used repeatedly.
		options := ev.plaintextOptions.GetOptions(ev.state.immutableSetupData.MaxChoices)
		if err := elgamal.VerifyEncryptionProof(cv.Proof, ev.ElectionPublicKey(), ctSum, options, []byte(cv.voterHash)); err != nil {
			return fmt.Errorf("ZeroKnowledgeProof of max choices invalid: %w", err)
		}
	}
	// ok it all looks good.
	// if the voter has voted before, add that id to the discarded block list
	if v.vote != nil {
		ev.state.discardedVotes++
	}
	// the vote should now be the encrypoted voted.
	v.vote = cv.Votes
	return nil
}
func (ev *ElectionValidator) checkPartialTally(blk *blockchain.Block) error {
	if blk.Header.PayloadHint != uint8(HintPartialTally) {
		return fmt.Errorf("Expecting a PartialTally block (%d) got %d (%s)", HintBallot, blk.Header.PayloadHint, PayloadHint(blk.Header.PayloadHint))
	}

	pt := &PayloadPartialTally{}
	if err := json.Unmarshal(blk.Payload, pt); err != nil {
		return fmt.Errorf("Error unmarshalling PartialTally payload: %w", err)
	}
	if !checkCanonical(pt, blk) {
		return fmt.Errorf("PartialTally payload was not canonically encoded")
	}

	// we need to "check this"
	idx := pt.Index - 1
	if idx < 0 || idx >= len(ev.state.immutableSetupData.Trustees) {
		return fmt.Errorf("Alleged Trustee %d is invalid", pt.Index)
	}
	trustee := ev.state.immutableSetupData.Trustees[idx]
	if trustee == nil {
		return fmt.Errorf("Alleged Trustee %d state is unknown", pt.Index)
	}

	// check the trustee hasn't already add their tally.
	if _, ok := ev.state.resultPartials[pt.Index]; ok {
		return fmt.Errorf("Trustee[%d] has already submitted a partial decryption", pt.Index)
	}

	// firstly tallies,decrypted and proofs should be candidate len.
	nCandidates := len(ev.state.immutableSetupData.Candidates)
	if len(pt.Tallies) != nCandidates {
		return fmt.Errorf("PartialTally tally count incorrect: expected %d, got %d", nCandidates, len(pt.Tallies))
	}
	if len(pt.Decrypted) != nCandidates {
		return fmt.Errorf("PartialTally decrypted count incorrect: expected %d, got %d", nCandidates, len(pt.Decrypted))
	}
	if len(pt.Proofs) != nCandidates {
		return fmt.Errorf("PartialTally proof count incorrect: expected %d, got %d", nCandidates, len(pt.Proofs))
	}

	// @TODO: check the tallies match our local count.
	local := ev.state.GetLocalTally()
	for i := range pt.Tallies {
		if !pt.Tallies[i].Equals(local[i]) {
			return fmt.Errorf("Given Partial Tally does not match our local Tally")
		}
	}

	// now the signature
	if err := trustee.SigKey.Verify(pt, pt.Signature); err != nil {
		return fmt.Errorf("PartialTally signature invalid: %w", err)
	}

	// now the proofs
	shardKey := ev.state.trusteePublic[trustee.Index]
	for i, zkp := range pt.Proofs {
		if err := elgamal.VerifyPartialDecryptionProof(zkp, shardKey, pt.Tallies[i], pt.Decrypted[i]); err != nil {
			return fmt.Errorf("ZKP of correct decryption for candiate %d, is incorrect: %w", i+1, err)
		}
	}
	// All good
	ev.state.AddPartialTally(pt)
	return nil
}
