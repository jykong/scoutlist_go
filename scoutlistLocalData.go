package scoutlist

import (
	"encoding/gob"
	"encoding/json"
	"log"
	"os"

	"github.com/zmb3/spotify"
)

// Test Modes
const (
	CodeTest int = 0
	UserTest int = 1
)

var mode = CodeTest

type stringConsts struct {
	PlaylistsPath     string
	ExcPlaylistsPath  string
	ExcTracksPath     string
	IncPlaylistPath   string
	ScoutlistIDPath   string
	ScoutedlistIDPath string
	ScoutlistName     string
	ScoutedlistName   string
}

type playlistsStruct struct {
	Playlists []playlistEntry
}
type tracksStruct struct {
	Tracks []trackIDTA
}

// SetMode sets the mode
func SetMode(newMode int) {
	mode = newMode
}

func getStringConsts() stringConsts {
	var strCon stringConsts
	var pathPrefix string
	var scoutlistPrefix string
	switch mode {
	case CodeTest:
		pathPrefix = "../../code_test_data/"
		scoutlistPrefix = "Test"
	case UserTest:
		pathPrefix = "../../user_test_data/"
		scoutlistPrefix = ""
	default:
		pathPrefix = ""
		scoutlistPrefix = ""
	}
	strCon.PlaylistsPath = pathPrefix + "playlists.json"
	strCon.ExcPlaylistsPath = pathPrefix + "exc_playlists.json"
	strCon.ExcTracksPath = pathPrefix + "exc_tracks.gob"
	strCon.IncPlaylistPath = pathPrefix + "inc_playlists.json"
	strCon.ScoutlistIDPath = pathPrefix + "scoutlist_id.gob"
	strCon.ScoutedlistIDPath = pathPrefix + "scoutedlist_id.gob"
	strCon.ScoutlistName = scoutlistPrefix + "Scoutlist"
	strCon.ScoutedlistName = scoutlistPrefix + "Scoutedlist"
	return strCon
}

func savePlaylistsToJSON(filePath string, playlists []playlistEntry) {
	os.Remove(filePath)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	plStruct := playlistsStruct{
		playlists,
	}
	err = encoder.Encode(plStruct)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("User playlists saved to", filePath)
}

func loadPlaylistsFromJSON(filePath string) []playlistEntry {
	log.Println("Loading playlists from", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(file)
	plStruct := playlistsStruct{}
	err = decoder.Decode(&plStruct)
	if err != nil {
		log.Fatal(err)
	}
	return plStruct.Playlists
}

func saveTracksToGob(filePath string, tracks []trackIDTA) {
	os.Remove(filePath)
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	trStruct := tracksStruct{
		tracks,
	}
	err = encoder.Encode(trStruct)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved tracks to", filePath)
}

func loadTracksFromGob(filePath string) []trackIDTA {
	log.Println("Loading tracks from:", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	trStruct := tracksStruct{}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&trStruct)
	if err != nil {
		log.Fatal(err)
	}
	return trStruct.Tracks
}

func saveIDToGob(filePath string, spid *spotify.ID) {
	os.Remove(filePath)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(*spid)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved ID to", filePath)
}

func loadIDFromGob(filePath string) spotify.ID {
	log.Println("Loading ID from:", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Println(err)
		return ""
	}
	var spid spotify.ID
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&spid)
	if err != nil {
		log.Fatal(err)
	}
	return spid
}
