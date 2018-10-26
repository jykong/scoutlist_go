package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/zmb3/spotify"
	"scoutlist"
	"scoutlist/data/s3"
)

var cpuprofile = flag.String("cpuprofile", "cpu.prof", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "mem.prof", "write memory profile to `file`")
var lastN = flag.Int("lastN", 15, "scoutlist mode: get last N tracks per playlist")
var outN = flag.Int("outN", 15, "output N track scoutlist")

func main() {
	flag.Parse()
	if s3data.ModeIsCodeTest() && *cpuprofile != "" {
		cpuProfile()
	}

	var cu scoutlist.ClientUser
	authAndSetID(&cu)

	var opt scoutlist.Options
	opt.LastN = *lastN
	opt.OutN = *outN

	sess := s3data.StartS3Session()
	updateScoutlist(sess, &cu, &opt)
	//getAndSavePlaylists(sess, &cu)

	if s3data.ModeIsCodeTest() && *memprofile != "" {
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

func getAndSavePlaylists(sess *session.Session, cu *scoutlist.ClientUser) {
	cu.Client.AutoRetry = true

	playlists := scoutlist.GetPlaylists(cu)
	s3data.SavePlaylistsToJSON(sess, s3data.GetStringConsts().PlaylistsPath, playlists)
}

func updateScoutlist(sess *session.Session, cu *scoutlist.ClientUser, opt *scoutlist.Options) {
	cu.Client.AutoRetry = true
	strCon := s3data.GetStringConsts()

	//excPlaylists := s3data.LoadPlaylistsFromJSON(sess, strCon.ExcPlaylistsPath)
	//fmt.Println(excPlaylists)

	scoutlist.StartRateLimiter()

	//excTracks := scoutlist.GetUniqueTracksFromPlaylists(cu, excPlaylists, nil, 0)
	//s3data.SaveTracksToGob(sess, strCon.ExcTracksPath, excTracks)
	excTracks := s3data.LoadTracksFromGob(sess, strCon.ExcTracksPath)
	fmt.Println(len(excTracks))
	//fmt.Println(excTracks)

	incPlaylists := s3data.LoadPlaylistsFromJSON(sess, strCon.IncPlaylistPath)

	filteredTracks := scoutlist.GetUniqueTracksFromPlaylists(cu, incPlaylists, excTracks, opt.LastN)
	fmt.Println(len(filteredTracks))
	//fmt.Println(filteredTracks)

	scoutlist.StopRateLimiter()

	scoutlistID := loadIDOrCreatePlaylist(sess, cu, strCon.ScoutlistIDPath, strCon.ScoutlistName)
	scoutedlistID := loadIDOrCreatePlaylist(sess, cu, strCon.ScoutedlistIDPath, strCon.ScoutedlistName)
	log.Println(scoutlistID)
	log.Println(scoutedlistID)

	excTracks = scoutlist.RecycleScoutlist(cu, scoutlistID, scoutedlistID, excTracks)

	s3data.SaveTracksToGob(sess, strCon.ExcTracksPath, excTracks)

	trackIDs := scoutlist.GetNTrackIDsFromTrackIDTASlice(filteredTracks, opt.OutN, true)
	//fmt.Println(trackIDs)
	scoutlist.ReplacePlaylistTracks(cu, scoutlistID, trackIDs)
}

func loadIDOrCreatePlaylist(sess *session.Session, cu *scoutlist.ClientUser,
	filePath string, name string) spotify.ID {
	plid := s3data.LoadIDFromGob(sess, filePath)
	if plid == "" {
		plid = scoutlist.CheckAndCreatePlaylist(cu, plid, name)
		s3data.SaveIDToGob(sess, filePath, &plid)
	}
	return plid
}
