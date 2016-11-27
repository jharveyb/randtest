package main

// #cgo CFLAGS: -std=c11
// 
// unsigned int rand;
//
// int rdrand64 (unsigned int *rand) 
// {
//     unsigned char result;
//
//     __asm__ ("rdrand %0; setc %1" : "=r" (*rand), "=qm" (result));
//     return (unsigned int) result;
// }
import "C"
import (
	"fmt"
	"crypto/rand"
	"math/big"
	"crypto/sha256"
	"flag"
	"encoding/hex"
	"encoding/binary"
	"os/exec"
)

/* Goal is to test speed of various methods of generating random numbers.
Here we compare /dev/urandom, SHA256, and Intel's RDRAND instruction.

TODO: Properly implement the setting of randpath using environment vars.
*/

/*

Here lies my attampt at using cgo vs. external C binary...
Currently compiling, but not being random...

rand := C.rdrand64(&C.rand)
fmt.Printf("rand is of type %T\n", rand)
fmt.Println(int(rand))
*/

var hash bool
var rdrand bool
var count int
var debug bool
var randbin string

func init() {
	flag.BoolVar(&hash, "hash", false, "Sets state of CSPRNG generation; false is /dev/urandom, true is SHA256")
	flag.BoolVar(&rdrand, "rdrand", false, "Sets state of CSPRNG generation; false is /dev/urandom, true is RDRAND")
	flag.IntVar(&count, "count", 100000, "# of random numbers generated; defaults to 1e5")
	flag.BoolVar(&debug, "debug", false, "Will print array states if true")
	flag.StringVar(&randbin, "randbin", "/home/jonathan/gocall", "Path to the C binary that makes RDRAND calls.")
	}

func debugmsg(outsize int, nonce uint64) {
	fmt.Println("map size is")
	fmt.Println(outsize)
	fmt.Println("nonce is")
	fmt.Println(nonce)
	if int(nonce) == outsize {
		fmt.Println("Desired count was met!")
	}
}

func shahash(nonce uint64, noncearr, bseedslice []byte) string {
	binary.BigEndian.PutUint64(noncearr, nonce)
	plain := append(bseedslice, noncearr...)
	hash := sha256.Sum256(plain)
	hashstr := hex.EncodeToString(hash[:])
	return hashstr
}

func urandomcall(ceil *big.Int) string {
	randint, _ := rand.Int(rand.Reader, ceil)
	brandint := randint.Bytes()
	randintstr := hex.EncodeToString(brandint[:])
	return randintstr
}

// Need to update to stream stdout vs. return one value
// And match buflength here to that in the C binary (100000)
func rdrandcall(canal chan string) {
	rand64 := "DEADBEEF"
	randbuf, _ := exec.Command(randbin).Output()
	randbuf, err := exec.Command(randbin).Output()
	if err != nil {
		// Should handle this properly
		fmt.Println("RDRAND Fail!")
		canal <-"DEADBEEF"
	} else {
		rand64 = string(randbuf)
		canal <-rand64
	}
}

func main() {
	flag.Parse()
	outmap := make(map[string]uint64)
	ceil := new(big.Int).SetUint64(1 << 63)
	seed, _ := rand.Int(rand.Reader, ceil)
	bseed := seed.Bytes()
	bseedslice := bseed[:]
	nonce := uint64(0)
	noncearr := make([]byte, 8)
	/*
	nprocmap := make(map[string]int)
	for x := 0; x < 64; x++ {
		nprocmap[string(x)] = x
	}
	nprocbuf, _ := exec.Command("nproc").Output()
	nprocstr := string(nprocbuf[0])
	nproc := nprocmap[nprocstr]
	fmt.Println(nprocmap)
	*/
	if debug == true {
		fmt.Println("Flag settings are")
		fmt.Println("hash", hash, "rdrand", rdrand, "count", count, "debug", debug)
		fmt.Println("Seed is")
		fmt.Println(seed)
		/*
		fmt.Println("nproc is")
		fmt.Println(nproc)
		*/
	}
	mode := 0
	if hash == true {
		mode = 2
	} else {
		if rdrand == true {
			mode = 1
		}
	}
	modemap := map[int]string{
		2:"SHA256",1:"RDRAND",0:"/dev/urandom/",
	}
	if debug == true {
		fmt.Println("Mode is")
		fmt.Println(mode)
		modename := modemap[mode]
		fmt.Println(modename)
	}
	// Chose to have loop in switch vs. switch in loop b/c assuming
	// that this is faster. Would be less redundant code if done 
	// with switch inside the loop though.
	switch mode {
	case 2:
		for i := 0; i < count; i++ {
			newhash := shahash(nonce, noncearr, bseedslice)
			outmap[newhash] = nonce
			nonce += 1
		}
		if debug == true {
			debugmsg(len(outmap), nonce)
		}
	case 1:
		// buflength here must match the setting in the C binary
		buflength := 10000
		canal := make(chan string, buflength)
		// Should change to $(nproc) threads & buffer the C binary
		// or stream from stdout there, & restart when exhausted
		for i := 0; i < count; i++ {
			go rdrandcall(canal)
		}
		for {
			rdrand64 := <-canal
			outmap[rdrand64] = nonce
			nonce += 1
			if len(outmap) == count {
				break
			}
		}
		if debug == true {
			debugmsg(len(outmap), nonce)
		}
	case 0:
		for i := 0; i < count; i++ {
			randstr := urandomcall(ceil)
			outmap[randstr] = nonce
			nonce += 1
		}
		if debug == true {
			debugmsg(len(outmap), nonce)
		}
	}
}
