package astris

import (
	"fmt"
	"time"

	big "github.com/ncw/gmp"

	"github.com/thechriswalker/go-astris/crypto"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
)

type ElectionState struct {
	// NB the state overlay is not implemented yet
	prev *ElectionState // this is so we can recurse backwards, making an "overlay state" from an existing one.

	// the block ID that we must "revalidate" from to Clone this state.
	// every time we reach a checkpoint, we clone the state and update this field
	// with the ID before the clone. We will need a "Chain" instance to clone the data though.
	// I really want the ability to `maybe = chain.Clone(); maybe.Add(block); maybe.Commit()` or something.

	// meaning we have to rewind til the `prev_id == checkpoint` and therefore we can then move forward again.
	// find the chain at checkpoint and then bring it forward to the point
	checkpoint ID

	// now the actual state, this state is used by the chain validator to see if the next block is valid.

	// it never changes so we can copy the pointer to each new state
	immutableSetupData *PayloadElectionSetup

	// the cryptosystem
	system *elgamal.ThresholdSystem

	// this is a lookup to enable time-to-phase checks.
	phaseForTime []*absoluteTimes
	exponents    map[int]crypto.BigIntSlice

	// all the trustee encrypted shares (without the proofs/etc)
	trusteeShares map[int]map[int]*elgamal.CipherText
	haveAllShares bool

	// trusteePublic is a map of the public key shards for each trustee.
	// we validate them on the way in.
	trusteePublic map[int]*elgamal.PublicKey
	haveAllPublic bool

	electionPublicKey *elgamal.PublicKey

	// this is the list of valid voters, with their public keys and the block ids of their
	// last vote
	voters         map[string]*VoterState
	discardedVotes uint64 // count discarded votes

	// this can only be worked out AFTER the votecasting is over.
	// at that point we can find the votes we need to tally them
	localTallies []*elgamal.CipherText

	// once the partial results come in for each candidate we can add them here.
	// note this is a double-slice, the first index is the trustee, and the second
	// is the partials for the candidates.
	resultPartials map[int]*PayloadPartialTally
	finalTallys    []Tally
}

func newElectionState(data *PayloadElectionSetup) *ElectionState {
	return &ElectionState{
		immutableSetupData: data,
		//exponents:          make(map[int]crypto.BigIntSlice, len(data.Trustees)),
		trusteeShares: make(map[int]map[int]*elgamal.CipherText, len(data.Trustees)),
		trusteePublic: make(map[int]*elgamal.PublicKey, len(data.Trustees)),
		voters:        make(map[string]*VoterState, 2048), //who knows how big this will be
		//localTallies:   make([]*elgamal.CipherText, len(data.Candidates)),
		resultPartials: make(map[int]*PayloadPartialTally, len(data.Trustees)),
	}
}

func (es *ElectionState) ElectionPublicKey() *elgamal.PublicKey {
	if es.immutableSetupData == nil {
		return nil
	}
	if es.electionPublicKey == nil {
		pk := big.NewInt(1)
		for _, t := range es.immutableSetupData.Trustees {
			// do we need to do this in order? I don't think so.
			pk.Mul(pk, t.Exponents[0])
			pk.Mod(pk, es.immutableSetupData.Params.P)
		}
		es.electionPublicKey = &elgamal.PublicKey{
			System: es.immutableSetupData.Params,
			Y:      pk,
		}
	}
	return es.electionPublicKey
}

type VoterState struct {
	key  *big.Int
	vote []*elgamal.CipherText // the last vote.
}

func (es *ElectionState) Checkpoint() *ElectionState {
	// Clone without rewind, mark new checkpoint state
	return nil
}

func (es *ElectionState) Cslone() *ElectionState {
	// allow clone of empty state
	if es == nil {
		return nil
	}
	// rewind, copy, fastforward
	// clone := &ElectionState{
	// 	immutableSetupData: es.immutableSetupData,
	// }
	return nil
}

// depending on the time, we should know which "phase" and therefore which
// payload we are expecting.
func (es *ElectionState) GetPhaseForTime(t time.Time) int {
	if es.phaseForTime == nil {
		es.phaseForTime = make([]*absoluteTimes, 4)
		t := es.immutableSetupData.Timing
		es.phaseForTime[0] = t.ParameterConfirmation.ToAbsolute(t.Timezone)
		es.phaseForTime[1] = t.VoterRegistration.ToAbsolute(t.Timezone)
		es.phaseForTime[2] = t.VoteCasting.ToAbsolute(t.Timezone)
		es.phaseForTime[3] = t.TallyDecryption.ToAbsolute(t.Timezone)
	}
	for idx, times := range es.phaseForTime {
		if t.After(times.start) && t.Before(times.end) {
			return idx + 1 //1 based phases
		}
	}
	return -1 // invalid phase

}

func (es *ElectionState) HasAllTrusteeShares() bool {
	if !es.haveAllShares {
		es.haveAllShares = len(es.trusteeShares) == len(es.immutableSetupData.Trustees)
	}
	return es.haveAllShares
}

func (es *ElectionState) HasAllTrusteePublic() bool {
	if !es.haveAllPublic {
		es.haveAllPublic = len(es.trusteePublic) == len(es.immutableSetupData.Trustees)
	}
	return es.haveAllPublic
}

func (es *ElectionState) ValidateShardKey(index int, pk *elgamal.PublicKey) error {
	// we need a map of exponents.
	if es.exponents == nil {
		E := map[int]crypto.BigIntSlice{}
		for _, trst := range es.immutableSetupData.Trustees {
			//fmt.Println("Exponent Map for Index:", trst.Index, "Exponents:", trst.Exponents)
			E[trst.Index] = trst.Exponents
		}
		es.exponents = E
	}
	if es.system == nil {
		es.system = &elgamal.ThresholdSystem{
			System: es.immutableSetupData.Params,
			L:      len(es.immutableSetupData.Trustees),
			T:      es.immutableSetupData.TrusteesRequired - 1,
		}
	}
	expectedPk := es.system.SimulatePublicKeyShard(index, es.exponents)

	if pk.Y.Cmp(expectedPk.Y) != 0 {
		return fmt.Errorf("Calculated Public Key does not match given")
	}
	return nil

}

func (es *ElectionState) AddPartialTally(pt *PayloadPartialTally) {
	es.resultPartials[pt.Index] = pt
}

type ElectionStats struct {
	NumVoters        uint64  // registered
	VoterTurnout     uint64  // how many voters voter
	NumRepeatVotes   uint64  // discarded votes due to repeat voting
	TalliesSubmitted int     // number of partial tallies submitted
	TalliesRequired  int     // number of partial tallies required
	Results          []Tally // the final result
}

type Tally struct {
	Candidate string
	Count     uint64
}

func (es *ElectionState) countVotes() (unique uint64, dupes uint64) {
	for _, v := range es.voters {
		if v.vote != nil {
			unique++
		}
	}
	dupes = es.discardedVotes
	return
}

func (es *ElectionState) GetLocalTally() []*elgamal.CipherText {
	if es.localTallies == nil {
		es.localTallies = make([]*elgamal.CipherText, len(es.immutableSetupData.Candidates))
		for i := 0; i < len(es.immutableSetupData.Candidates); i++ {
			es.localTallies[i] = &elgamal.CipherText{}
		}
		sys := es.system.System
		i := 0
		for _, v := range es.voters {
			i++
			//	fmt.Printf("counting votes for voter %d: %v\n", i, v.vote)
			if v.vote != nil {
				for c, ev := range v.vote {
					es.localTallies[c] = es.localTallies[c].Mul(sys, ev)
				}
			}
		}
		//fmt.Println("Encrypted Tallies:", es.localTallies)
	}

	return es.localTallies
}

func (es *ElectionState) combineTallies(maxVotes uint64) []Tally {
	if es.finalTallys != nil {
		return es.finalTallys
	}
	if len(es.resultPartials) < es.immutableSetupData.TrusteesRequired {
		return nil
	}

	// we need to combine the tallys.
	// we will produce local tallies during the validation phase for the
	// partial tally inputs, so this is a naive combination of the pieces.
	tallies := make([]Tally, len(es.immutableSetupData.Candidates))
	exponentials := make([]*big.Int, len(tallies))
	for ci, ct := range es.localTallies {
		// computing for candidate ci
		// gather the partial decryptions
		// in 1-based trustee index.
		// it will be sparse
		partials := make([]*big.Int, len(es.immutableSetupData.Trustees)+1)
		indices := make([]int, 0, es.immutableSetupData.TrusteesRequired)
		for idx, resp := range es.resultPartials {
			partials[idx] = resp.Decrypted[ci]
			// create the list of trustee indices as we go
			if len(indices) < es.immutableSetupData.TrusteesRequired {
				indices = append(indices, idx)
			}
		}
		//fmt.Printf("candidate[%d] using trustees: %#v\nFor partials: %#v\n", ci+1, indices, partials)

		exponentials[ci] = elgamal.ThresholdDecrypt(
			es.immutableSetupData.Params,
			ct,
			partials,
			indices,
		)

	}

	dlog := elgamal.DiscreteLogLookup(es.immutableSetupData.Params, maxVotes, exponentials)

	for ci, exp := range exponentials {
		//fmt.Printf("Candidate[%d]: expvote: %s\n", ci+1, exp.String())
		tallies[ci] = Tally{
			Candidate: es.immutableSetupData.Candidates[ci],
			Count:     dlog(exp),
		}
	}

	es.finalTallys = tallies
	return es.finalTallys
}

// returns nil if not complete
func (es *ElectionState) GetResult() *ElectionStats {
	unique, dupes := es.countVotes()
	stats := &ElectionStats{
		NumVoters:        uint64(len(es.voters)),
		VoterTurnout:     unique,
		NumRepeatVotes:   dupes,
		TalliesSubmitted: len(es.resultPartials),
		TalliesRequired:  es.immutableSetupData.TrusteesRequired,
		Results:          es.combineTallies(unique),
	}
	return stats
}

// // we might need a mutex in here...
// type stringSet map[string]struct{}

// func (ss stringSet) Add(s string) {
// 	ss[s] = struct{}{}
// }
// func (ss stringSet) Has(s string) bool {
// 	_, ok := ss[s]
// 	return ok
// }
// func (ss stringSet) Remove(s string) {
// 	delete(ss, s)
// }
