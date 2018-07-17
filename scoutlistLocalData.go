package main

import (
	"encoding/gob"
	"encoding/json"
	"log"
	"os"

	"github.com/zmb3/spotify"
)

const playlistsPath = "./playlists.json"
const excPlaylistsPath = "./exc_playlists.json"
const excTracksPath = "./exc_tracks.gob"
const incPlaylistPath = "./inc_playlists.json"
const scoutlistIDPath = "./scoutlist_id.gob"
const scoutedlistIDPath = "./scoutedlist_id.gob"

func savePlaylistsToJSON(filePath string, plCon *playlistsContainer) {
	var encoder *json.Encoder

	os.Remove(filePath)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	err = encoder.Encode(plCon)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("User playlists saved to", playlistsPath)
}

func loadPlaylistsFromJSON(filePath string) playlistsContainer {
	log.Println("Loading playlists from", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	var plCon playlistsContainer
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&plCon)
	if err != nil {
		log.Fatal(err)
	}
	return plCon
}

func saveTracksToGob(filePath string, tracks *tracksContainer) {
	os.Remove(filePath)
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(*tracks)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved tracks to", filePath)
}

func loadTracksFromGob(filePath string) tracksContainer {
	log.Println("Loading tracks from:", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	var tracks tracksContainer
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&tracks)
	if err != nil {
		log.Fatal(err)
	}
	return tracks
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
