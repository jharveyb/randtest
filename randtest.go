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
	"io"
	"sync"
	"github.com/losalamos/rdrand"
	"github.com/orcaman/concurrent-map"
	gorand "github.com/tmthrgd/go-rand"
)

/* 
Goal is to test speed of various methods of generating random numbers.
Here we compare /dev/urandom, SHA256, and Intel's RDRAND instruction.

Missing AES-NI because my machine doesn't support it.
*/

var hash string
var intel bool
var count int
var debug bool

func init() {
	flag.StringVar(&hash, "hash", "none", "Sets source of random numbers; 'sha' is SHA256, 'cha' is ChaCha20")
	flag.BoolVar(&intel, "intel", false, "Sets source of random numbers; false is /dev/urandom, true is RDRAND")
	flag.IntVar(&count, "count", 100000, "# of random numbers generated; defaults to 100000")
	flag.BoolVar(&debug, "debug", false, "Will print array states if true")
	}

// All RNG calls return strings; overhead varies across methods.

// Computes SHA256( Seed || Nonce)
func shahash(nonce uint64, bseedslice []byte, shacache [32]byte, hashstr string) string {
	binary.BigEndian.PutUint64(bseedslice[8:16], nonce)
	shacache = sha256.Sum256(bseedslice)
	hashstr = hex.EncodeToString(shacache[:])
	return hashstr
}

func chahash(ChaRead io.Reader, chacache []byte, hashstr string) string {
	ChaRead.Read(chacache)
	hashstr = hex.EncodeToString(chacache[:])
	return hashstr
	}

// Go's crypto/rand Reader, used here, uses /dev/urandom
func urandomcall(ceil *big.Int, randintstr string) string {
	randint, _ := rand.Int(rand.Reader, ceil)
	brandint := randint.Bytes()
	randintstr = hex.EncodeToString(brandint[:])
	return randintstr
}

// Size of output here 1/4 that of other 2 RNG functions
func rdrandcall(randstr string) string {
	rand := rdrand.Uint64()
	randstr = hexconv(rand)
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
	noncearr := make([]byte, 8)
	bseedslice = append(bseedslice, noncearr...)
	fmt.Println(bseedslice)
	fmt.Println(bseedslice[8:16])
	nproc := runtime.NumCPU()
	runtime.GOMAXPROCS(nproc)
	if debug == true {
		fmt.Println("Flag settings are")
		fmt.Println("hash", hash, "rdrand", intel, "count", count, "debug", debug)
		fmt.Println("Seed is")
		fmt.Println(seed)
		fmt.Println(bseedslice)
		fmt.Println("nproc is")
		fmt.Println(nproc)
	}
	modemap := map[int]string{
		3:"ChaCha20",2:"SHA256",1:"RDRAND",0:"/dev/urandom/",
	}
	mode := 0
	switch hash {
	case "cha":
		mode = 3
	case "sha":
		mode = 2
	case "none":
		if intel == true {
			if rdrand.Available() == true {
				mode = 1
			} else {
				fmt.Println("No RDRAND support!")
				fmt.Println("Using /dev/urandom.")
			}
		}
	}
	if debug == true {
		fmt.Println("Mode is")
		fmt.Println(mode)
		fmt.Println(modemap[mode])
	}
	var waiter sync.WaitGroup
	tcount := count/nproc
	waiter.Add(nproc)
	for n := 0; n < nproc; n++ {
		go func (n, tcount int) {
			defer waiter.Done()
			tnonce := 0
			var nonce uint64
			nonce = uint64(n*tcount)
			var randstring string
			var cachestr string
			hashcache := make([]byte, 32)
			var shacache [32]byte
			ChaRead, err := gorand.New(nil)
			if err != nil {
				fmt.Println("ChaRead init. failed!")
				fmt.Println("Switching to /dev/urandom.")
			}
			// Tested both switch-in-for & vice-versa; no noticable
			// speed difference with go 1.7.4 linux/amd64
			for i := 0; i < tcount; i++ {
				switch mode {
				case 3:
					randstring = chahash(ChaRead, hashcache, cachestr)
				case 2:
					randstring = shahash(nonce, bseedslice, shacache, cachestr)
				case 1:
					randstring = rdrandcall(cachestr)
				case 0:
					randstring = urandomcall(ceil, cachestr)
				}
				coutmap.Set(randstring, tnonce)
				nonce += 1
				tnonce += 1
			}
			if debug == true {
				fmt.Println("nonce is")
				fmt.Println(tnonce)
			}
		}(n, tcount)
	}
	waiter.Wait()
	if coutmap.Count() == count {
		fmt.Println("Desired count was met!")
	}
	fmt.Println(coutmap.Count())
}
