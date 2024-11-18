#!/bin/bash



# build up to date!
make

# options
NUM_VOTERS=${1:-1000}
THRESHOLD_TRUSTEES=${2:-5}
NUM_TRUSTEES=${3:-7}

IS_AUDIT="${4:-}"

# now create a directory (empty if necessary)
BM_DIR="benchmark_election_${NUM_VOTERS}_voters_${THRESHOLD_TRUSTEES}_of_${NUM_TRUSTEES}"

ELECTION_ID=""

if [ "${IS_AUDIT}" = "audit" ]
then
    # just run audit
    ELECTION_ID="$(ls -1 ${BM_DIR}/chain* | sed 's/.*chain_//;s/\.db//')"
    echo "Running Audit on election ${ELECTION_ID}"
else
    echo "###"
    echo "### Building Election Setup"
    echo "###"
    if [ -d ${BM_DIR} ]; then rm -r ${BM_DIR}; fi
    mkdir ${BM_DIR}

    SETUP_LOG="${BM_DIR}/000_setup.log"

    ## start the process building the genesis
    ./build/astris authority simulate --data-dir ${BM_DIR} --num-trustee ${NUM_TRUSTEES} --threshold ${THRESHOLD_TRUSTEES} | tee 

    ## now find the electionID for other processes
    ## should only be one file.
    ELECTION_ID="$(ls -1 ${BM_DIR}/chain* | sed 's/.*chain_//;s/\.db//')"

    echo "###"
    echo "### Election ID: ${ELECTION_ID}"
    echo "### Simulating Voting"
    echo "###"

    ## now simulate voting
    ./build/astris voter simulate --data-dir ${BM_DIR} --election-id ${ELECTION_ID} --num-voters ${NUM_VOTERS}


    echo "###"
    echo "### Election ID: ${ELECTION_ID}"
    echo "### Simulating Tallying"
    echo "###"
    ## now simlute the tallying
    ./build/astris trustee simulate --data-dir ${BM_DIR} --election-id ${ELECTION_ID}

fi

echo "###"
echo "### Election ID: ${ELECTION_ID}"
echo "### Auditing Chain"
echo "###"
## now the chain is built, we can run an audit
./build/astris auditor --data-dir ${BM_DIR} --election-id ${ELECTION_ID} --validate-only