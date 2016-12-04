package main

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"crypto/sha256"
	"flag"
	"encoding/hex"
	"encoding/binary"
	"strconv"
	"strings"
	"runtime"
	"sync"
	"github.com/losalamos/rdrand"
	"github.com/orcaman/concurrent-map"
)

/* Goal is to test speed of various methods of generating random numbers.
Here we compare /dev/urandom, SHA256, and Intel's RDRAND instruction.
*/

var hash bool
var intel bool
var count int
var debug bool
var randbin string

func init() {
	flag.BoolVar(&hash, "hash", false, "Sets state of CSPRNG generation; false is /dev/urandom, true is SHA256")
	flag.BoolVar(&intel, "intel", false, "Sets state of CSPRNG generation; false is /dev/urandom, true is RDRAND")
	flag.IntVar(&count, "count", 100000, "# of random numbers generated; defaults to 1e5")
	flag.BoolVar(&debug, "debug", false, "Will print array states if true")
	flag.StringVar(&randbin, "randbin", "/home/jonathan/gocall", "Path to the C binary that makes RDRAND calls.")
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

func rdrandcall() string {
	rand := rdrand.Uint64()
	randstr := hexconv(rand)
	return randstr
}

func hexconv (inp uint64) string {
	inpstr := strconv.FormatUint(inp, 16)
	capstr := strings.ToUpper(inpstr)
	return capstr
}

func main() {
	flag.Parse()
	coutmap := cmap.New()
	ceil := new(big.Int).SetUint64(1 << 63)
	seed, _ := rand.Int(rand.Reader, ceil)
	bseed := seed.Bytes()
	bseedslice := bseed[:]
	nonce := uint64(0)
	noncearr := make([]byte, 8)
	nproc := runtime.NumCPU()
	runtime.GOMAXPROCS(nproc)
	if debug == true {
		fmt.Println("Flag settings are")
		fmt.Println("hash", hash, "rdrand", intel, "count", count, "debug", debug)
		fmt.Println("Seed is")
		fmt.Println(seed)
		fmt.Println("nproc is")
		fmt.Println(nproc)
	}
	mode := 0
	if hash == true {
		mode = 2
	} else {
		if intel == true {
			if rdrand.Available() == true {
				mode = 1
			} else {
				fmt.Println("No RDRAND support!")
				fmt.Println("Switching to /dev/urandom instead.")
				mode = 0
			}
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
	var waiter sync.WaitGroup
	tcount := count/nproc
	for n := 0; n < nproc; n++ {
		waiter.Add(1)
		go func () {
			defer waiter.Done()
			tnonce := 0
			switch mode {
			case 2:
				for i := 0; i < tcount; i++ {
					newhash := shahash(nonce, noncearr, bseedslice)
					coutmap.Set(newhash, tnonce)
					nonce += 1
					tnonce += 1
				}
			case 1:
				for i := 0; i < tcount; i++ {
					rnd := rdrandcall()
					coutmap.Set(rnd, tnonce)
					tnonce += 1
				}
			case 0:
				for i := 0; i < tcount; i++ {
					randstr := urandomcall(ceil)
					coutmap.Set(randstr, tnonce)
					tnonce += 1
				}
			}
			if debug == true {
				fmt.Println("nonce is")
				fmt.Println(tnonce)
			}
		}()
	}
	waiter.Wait()
	if coutmap.Count() == count {
		fmt.Println("Desired count was met!")
	}
	fmt.Println(coutmap.Count())
}
