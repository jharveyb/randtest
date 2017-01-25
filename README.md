# Randtest

Allows for quick testing of the various ways to generate cryptographically secure random numbers.

Currently supports:

* /dev/urandom
* RDRAND instruction (Intel only)
* SHA256
* ChaCha20

Note that, for GNU/Linux systems with kernel version 4.8 and later, /dev/urandom is now a ChaCha20-based CRNG, which is significantly faster than the old /dev/urandom. More info here: https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/commit/?id=818e607b57c94ade9824dad63a96c2ea6b21baf3

The default RNG is /dev/urandom; while it is safe to use both the "hash" and "intel" arguments, "hash" takes precedence when choosing an RNG.

Run "randtest --help" or read the source to learn more about command line arguments.

On my machine (Linux 4.8.0, 64-bit, go 1.7.4), I found ChaCha20 to be the fastest method, followed by SHA256, /dev/urandom/, and then RDRAND.

Known Issues & Quirks:

* The RDRAND calls only generate 64 bits of entropy, while all other calls generate 256 bits. Adjust accordingly when measuring speed.
* All methods have overhead related to converting their output to a string before storing; this overhead varies across the methods.
* SHA256 calls end slightly before all output is stored; this can be seen with the debug flag.

Relevant info concerning the speed of cryptographic functions - https://cryptopp.com/benchmarks.html

Feel free to make PRs or contact me directly!
