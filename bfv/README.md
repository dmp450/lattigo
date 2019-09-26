# BFV

The package BFV is an RNS-accelerated of the Fan-Vercauteren version of the Brakerski's scale invariant homomorphic encryption scheme. It provides modular arithmetic over the integers.

## Brief description

This scheme can be used to do arithmetic over &nbsp; ![equation](https://latex.codecogs.com/gif.latex?%5Cmathbb%7BZ%7D_t%5EN)

The plaintext space and the ciphertext space share the same domain :

<p align="center">
<img src="https://latex.codecogs.com/gif.latex?%5Cmathbb%7BZ%7D_Q%5BX%5D/%28X%5EN%20&plus;%201%29">
</p>

The batch encoding of this scheme :

<p align="center">
<img src="https://latex.codecogs.com/gif.latex?%5Cmathbb%7BZ%7D_t%5EN%20%5Cleftrightarrow%20%5Cmathbb%7BZ%7D_Q%5BX%5D/%28X%5EN%20&plus;%201%29">
</p>

maps an array of integers to a polynomial with the property :

<p align="center">
<img src="https://latex.codecogs.com/gif.latex?decode%28encode%28m_1%29%20%5Cotimes%20encode%28m_2%29%29%20%3D%20m_1%20%5Codot%20m_2">
</p>


## Security parameters

![equation](https://latex.codecogs.com/gif.latex?N%20%3D%202%5E%7BlogN%7D) : the ring dimension, it should always be a power of two. This parameters has an impact on both security and performances. It should be chosen carefuly to suit the intended use of the scheme.

![equation](https://latex.codecogs.com/gif.latex?Q) : the ciphertext modulus. In this implementation it is chosen to be the product of small coprime moduli ![equation](https://latex.codecogs.com/gif.latex?q_i) of 50 to 60 bits verifying ![equation](https://latex.codecogs.com/gif.latex?q_i%20%5Cequiv%201%20%5Cmod%202N) to enable the RNS and NTT representation. This parameter has an impact on both security and performances. It is closely related to ![equation](https://latex.codecogs.com/gif.latex?N) and should be chosen carefuly to suit the intended use of the scheme.

![equation](https://latex.codecogs.com/gif.latex?%5Csigma) : the variance used for the error polynomials. This parameter is closely tied to the security of the scheme.

## Other parameters

![equation](https://latex.codecogs.com/gif.latex?P) : the extended ciphertext modulus. This modulus is used during the multiplication, and has no impact on the security. It is also the product of small coprime moduli ![equation](https://latex.codecogs.com/gif.latex?p_j) and should be chosen such that ![equation](https://latex.codecogs.com/gif.latex?Q%5Ccdot%20P%20%3E%20Q%5E2) by a small margin (~20 bits). This can be done by using one more small coprime moduli than ![equation](https://latex.codecogs.com/gif.latex?Q).

![equation](https://latex.codecogs.com/gif.latex?t) : the plaintext modulus. This parameters defines the maximum value that a plaintext coefficient can take. If a computation would lead to a higher value, this value will be reduced modulo the plaintext modulus. It can be initialized with any value, but to enable batching it must be prime and verify ![equation](https://latex.codecogs.com/gif.latex?t%20%5Cequiv%201%20%5Cmod%202N). It has no impact on the security.

## Chosing security parameters

The BFV scheme supports official parameters chosen to offer a security of 128 bits for a secret with uniform ternary distribution ![equation](https://latex.codecogs.com/gif.latex?s%20%5Cin_u%20%5C%7B-1%2C%200%2C%201%5C%7D%5EN) according to the Homomorphic Encryption Standardization (https://homomorphicencryption.org/standard/).  

Each set of parameter is defined by ![equation](https://latex.codecogs.com/gif.latex?%5C%7Blog_2%28N%29%2C%20log_2%28Q%29%2C%20%5Csigma%5C%7D) :

- {11, 54, 3.2}
- {12, 109, 3.2}
- {13, 218, 3.2}
- {14, 438, 3.2}
- {15, 881, 3.2}

Those parameters are hardcoded in the file [params.go](https://github.com/ldsec/lattigo/blob/master/bfv/params.go). By default the variance shoud always be set to 3.2 unless the user is perfectly aware of the security implication.