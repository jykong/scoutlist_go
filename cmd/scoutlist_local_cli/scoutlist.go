package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/zmb3/spotify"
	"scoutlist"
	"scoutlist/data/local"
)

var cpuprofile = flag.String("cpuprofile", "cpu.prof", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "mem.prof", "write memory profile to `file`")
var lastN = flag.Int("lastN", 15, "scoutlist mode: get last N tracks per playlist")
var outN = flag.Int("outN", 15, "output N track scoutlist")

func main() {
	flag.Parse()
	if localdata.ModeIsCodeTest() && *cpuprofile != "" {
		cpuProfile()
	}

	var cu scoutlist.ClientUser
	authAndSetID(&cu)

	var opt scoutlist.Options
	opt.LastN = *lastN
	opt.OutN = *outN
	//updateScoutlist(&cu, &opt)
	getAndSavePlaylists(&cu)

	if localdata.ModeIsCodeTest() && *memprofile != "" {
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

func authAndSetID(cu *scoutlist.ClientUser) {
	cu.Client = scoutlist.Auth()
	user, err := cu.Client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.UserID = user.ID
}

func getAndSavePlaylists(cu *scoutlist.ClientUser) {
	cu.Client.AutoRetry = true

	playlists := scoutlist.GetPlaylists(cu)
	localdata.SavePlaylistsToJSON(localdata.GetStringConsts().PlaylistsPath, playlists)
}

func updateScoutlist(cu *scoutlist.ClientUser, opt *scoutlist.Options) {
	cu.Client.AutoRetry = true
	strCon := localdata.GetStringConsts()

	//excPlaylists := loadPlaylistsFromJSON(strCon.ExcPlaylistsPath)
	//fmt.Println(excPlaylists)

	scoutlist.StartRateLimiter()

	//excTracks := cu.getUniqueTracksFromPlaylists(rateLimit, excPlaylists, nil, 0)
	//saveTracksToGob(strCon.ExcTracksPath, excTracks)
	excTracks := localdata.LoadTracksFromGob(strCon.ExcTracksPath)
	fmt.Println(len(excTracks))
	//fmt.Println(excTracks

	incPlaylists := localdata.LoadPlaylistsFromJSON(strCon.IncPlaylistPath)

	filteredTracks := scoutlist.GetUniqueTracksFromPlaylists(cu, incPlaylists, excTracks, opt.LastN)
	fmt.Println(len(filteredTracks))
	//fmt.Println(filteredTracks)

	scoutlist.StopRateLimiter()

	scoutlistID := loadIDOrCreatePlaylist(cu, strCon.ScoutlistIDPath, strCon.ScoutlistName)
	scoutedlistID := loadIDOrCreatePlaylist(cu, strCon.ScoutedlistIDPath, strCon.ScoutedlistName)

	excTracks = scoutlist.RecycleScoutlist(cu, scoutlistID, scoutedlistID, excTracks)

	localdata.SaveTracksToGob(strCon.ExcTracksPath, excTracks)

	trackIDs := scoutlist.GetNTrackIDsFromTrackIDTASlice(filteredTracks, opt.OutN, true)
	//fmt.Println(trackIDs)
	scoutlist.ReplacePlaylistTracks(cu, scoutlistID, trackIDs)
}

func loadIDOrCreatePlaylist(cu *scoutlist.ClientUser, filePath string, name string) spotify.ID {
	plid := localdata.LoadIDFromGob(filePath)
	if plid == "" {
		plid = scoutlist.CheckAndCreatePlaylist(cu, plid, name)
		localdata.SaveIDToGob(filePath, &plid)
	}
	return plid
}
