package localdata

import (
	"encoding/gob"
	"encoding/json"
	"log"
	"os"

	"github.com/zmb3/spotify"
	"scoutlist/data/model"
)

// Test Modes
const (
	codeTest int = 0
	userTest int = 1
)

var mode = codeTest

// StringConsts pre-defined filepaths
type StringConsts struct {
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
	Playlists []model.PlaylistEntry
}

type tracksStruct struct {
	Tracks []model.TrackIDTA
}

// ModeIsCodeTest Check if mode is set to code test
func ModeIsCodeTest() bool {
	return mode == codeTest
}

// GetStringConsts Get pre-defined filepaths
func GetStringConsts() StringConsts {
	var strCon StringConsts
	var pathPrefix string
	var scoutlistPrefix string
	switch mode {
	case codeTest:
		pathPrefix = "../../code_test_data/"
		scoutlistPrefix = "Test"
	case userTest:
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

// SavePlaylistsToJSON ...
func SavePlaylistsToJSON(filePath string, playlists []model.PlaylistEntry) {
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

// LoadPlaylistsFromJSON ...
func LoadPlaylistsFromJSON(filePath string) []model.PlaylistEntry {
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

// SaveTracksToGob ...
func SaveTracksToGob(filePath string, tracks []model.TrackIDTA) {
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

// LoadTracksFromGob ...
func LoadTracksFromGob(filePath string) []model.TrackIDTA {
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

// SaveIDToGob ...
func SaveIDToGob(filePath string, spid *spotify.ID) {
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

// LoadIDFromGob ...
func LoadIDFromGob(filePath string) spotify.ID {
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
