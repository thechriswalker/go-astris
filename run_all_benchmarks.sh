#!/bin/bash

# 1000 voters

#./run_benchmarks 1000 2 3 | tee benchmark_1000v_2of3t.log 
#./run_benchmarks 1000 3 5 | tee benchmark_1000v_3of5t.log 
#./run_benchmarks 1000 5 7 | tee benchmark_1000v_5of7t.log 

# 10_000 voters

#./run_benchmarks 10000 2 3 | tee benchmark_10_000v_2of3t.log 
./run_benchmarks 10000 3 5 | tee benchmark_10_000v_3of5t.log 
./run_benchmarks 10000 5 7 | tee benchmark_10_000v_5of7t.log 

# 100_000 voters
./run_benchmarks 100000 2 3 | tee benchmark_100_000v_2of3t.log 
./run_benchmarks 100000 3 5 | tee benchmark_100_000v_3of5t.log 
./run_benchmarks 100000 5 7 | tee benchmark_100_000v_5of7t.log 

# 1_000_000 voters
./run_benchmarks 1000000 2 3 | tee benchmark_1_000_000v_2of3t.log 
./run_benchmarks 1000000 3 5 | tee benchmark_1_000_000v_3of5t.log 
./run_benchmarks 1000000 5 7 | tee benchmark_1_000_000v_5of7t.log 
