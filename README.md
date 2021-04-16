# Astris e-voting

This is a proof of concept software implementation of the Astris e-voting scheme, designed for a minimal trust voting platform as part of an MSc project.

Uses `github.com/ncw/gmp` for large integer support. To build you will need `libgmp-dev` or equivalent.

#### Performance

We need the performance to be _reasonable_ such that this could be used for a large scale election. I simulated a 5 trustee, 1 million voter election. The ballots were cast randomly (hence the fairly even spread of votes). Then the `auditor` mode of the election validates the entire chain to ensure the cryptographic integrity of the data and computes the result.
This was performed on an Intel i7-8550U clocked at 4.000GHz. The CPU has 4 cores (8 threads) but there has been no effort to parallelize any operations, so it is effectively bound by single core performance.

```
$ ./run.sh auditor validate --data-dir=sim --election-id=AAB6zJrFJZK_Bx0aWhxQDsp7-jSj2swiewTWk-m5ikk
make: 'build/astris' is up to date.
11:40:19.001 INF Astris Voting license=GPLv3+ protocol=1.0 version=v0.0.0
11:40:19.002 INF Starting Election Auditor Process
11:45:52.546 INF Chain reverse pass complete, starting forward validation pass1_ms=321513.684706
2000015 / 2000015 [-----------------------------------------------------------------------------------------------------------------------------------------------] 100.00% 93 p/s
17:42:25.635 INF Chain forward pass complete pass2_ms=21393092.132859 total_ms=21714605.817565
17:42:25.636 INF Blockchain Validation Success chain=AAB6zJrFJZK_Bx0aWhxQDsp7-jSj2swiewTWk-m5ikk depth=2000015 head=LHw6TS0DreYljVpvQf7n1vJsxdMA0x2SOSKSBB8m568 ms=21714605.817565
{
  "NumVoters": 1000000,
  "VoterTurnout": 1000000,
  "NumRepeatVotes": 0,
  "TalliesSubmitted": 5,
  "TalliesRequired": 3,
  "Results": [
    {
      "Candidate": "C1",
      "Count": 199597
    },
    {
      "Candidate": "C2",
      "Count": 199497
    },
    {
      "Candidate": "C3",
      "Count": 199759
    },
    {
      "Candidate": "C4",
      "Count": 200484
    },
    {
      "Candidate": "C5",
      "Count": 199684
    }
  ]
}
```

The full validation pass took ~21,715,000ms (just over 6 hours - the first time I did this it was closer to 10hours, but then I switched out the Go "math/big" BigInt for a wrapper around the GNU GMP library and performance increased significantly).
There has been little optimisation and everything runs serially, so further improvement is certainly possible. This should scale fairly linearly with respect to voters, as vote registration/ballot blocks outnumber all others massively. In this example we have 2,000,015 blocks and 15 are not voter blocks.

