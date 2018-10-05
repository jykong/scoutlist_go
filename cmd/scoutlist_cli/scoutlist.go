package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"scoutlist"
)

var cpuprofile = flag.String("cpuprofile", "cpu.prof", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "mem.prof", "write memory profile to `file`")
var lastN = flag.Int("lastN", 15, "scoutlist mode: get last N tracks per playlist")
var outN = flag.Int("outN", 15, "output N track scoutlist")

func main() {
	flag.Parse()
	mode := scoutlist.CodeTest // change this to switch between code & user tests
	scoutlist.SetMode(mode)
	if mode == scoutlist.CodeTest && *cpuprofile != "" {
		cpuProfile()
	}

	var cu scoutlist.ClientUser
	cu.Client = scoutlist.Auth()
	getCurrentUserID(cu)

	var opt scoutlist.Options
	opt.LastN = *lastN
	opt.OutN = *outN
	scoutlist.Update(&cu, &opt)

	if mode == scoutlist.CodeTest && *memprofile != "" {
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

func getCurrentUserID(cu scoutlist.ClientUser) {
	user, err := cu.Client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.UserID = user.ID
}
