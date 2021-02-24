# crypto

Requirements from our encryption system for the main voting

- distributed threshold key, public key known, secret key unknown to all
- probablistic encryption
- homomorphic addition of ciphertexts
- proof of knowledge of secret key
- proof of encryption range (i.e. encrypted value is between a min, max)
- proof of partial decryption

Considering the pros and cons of each, I choose to implement the ElGamal
as the only con is the lack of post-quantum resistance.

In the fullness of time, I hope that the encryption layer can be revisited
and updated as the technology changes. The fact that we need to keep the data
on a block chain means that we want to keep the data to a reasonable size

## Pallier

- [N] distributed threshold key, public key known, secret key unknown to all
- [y] probablistic encryption
- [y] homomorphic addition of ciphertexts
- [~] verifiable encryption / decryption

pros:

- fairly simple
- homomorphic addition

cons:

- DLP so no post-quantum resistance
- no well-known shared key generation scheme.

## Lattice (dBFV)

- [y] distributed threshold key, public key known, secret key unknown to all
- [y] probablistic encryption
- [~] homomorphic addition of ciphertexts

pros:

- post-quantum encryption

cons:

- limited number of homomorphic operations not suitable for large scale elections
- large ciphertexts

## ElGamal

- [y] distributed threshold key, public key known, secret key unknown to all
- [y] probablistic encryption
- [~] homomorphic addition of ciphertexts (with Exponential ElGamal)
- [y] proof of knowledge of secret key
- [y] proof of encryption range (i.e. encrypted value is between a min, max)
- [y] proof of partial decryption

pros:

- Simple
- can be adapted for homomorphic addition easily
- distributed threshold key generation
- reasonable sized ciphertexts

cons:

- DLP based crypto not considered post-quantum which could reveal votes later
