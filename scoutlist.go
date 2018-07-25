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

var cpuprofile = flag.String("cpuprofile", "cpu.prof", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "mem.prof", "write memory profile to `file`")

func main() {
	const mode = codeTest

	flag.Parse()
	if mode == codeTest && *cpuprofile != "" {
		cpuProfile()
	}

	var cu clientUser
	cu.client = scoutlistAuth()
	cu.getCurrentUserID()

	scoutlistUpdate(&cu, mode)

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
