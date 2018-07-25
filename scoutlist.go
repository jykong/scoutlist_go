package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/zmb3/spotify"
)

type clientUser struct {
	client *spotify.Client
	userID string
}

const (
	codeTest int = 0
	userTest int = 1
)
const mode = codeTest

var cpuprofile = flag.String("cpuprofile", "cpu.prof", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "mem.prof", "write memory profile to `file`")
var lastN = flag.Int("lastN", 0, "scoutlist mode: get last N tracks per playlist")
var outN = flag.Int("outN", 30, "output N track scoutlist")

type options struct {
	lastN int
	outN  int
}

func main() {
	flag.Parse()
	if mode == codeTest && *cpuprofile != "" {
		cpuProfile()
	}

	var cu clientUser
	cu.client = scoutlistAuth()
	cu.getCurrentUserID()

	var opt options
	opt.lastN = *lastN
	opt.outN = *outN
	scoutlistUpdate(&cu, &opt)

	if mode == codeTest && *memprofile != "" {
		memProfile()
	}
}

func cpuProfile() {
	f, err := os.Create(*cpuprofile)
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()
}

func memProfile() {
	f, err := os.Create(*memprofile)
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

func (cu *clientUser) getCurrentUserID() {
	user, err := cu.client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.userID = user.ID
}
