package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/zmb3/spotify"
)

type clientUser struct {
	client *spotify.Client
	userID string
}

const playlistsPath = "./playlists.json"

func main() {
	var cu clientUser
	cu.client = scoutlistAuth()
	cu.getCurrentUserID()

	cu.getPlaylists(playlistsPath)
	var excPlaylists playlistsContainer
	cu.loadPlaylists(playlistsPath, &excPlaylists)
	fmt.Println(excPlaylists)
}

func (cu *clientUser) getCurrentUserID() {
	user, err := cu.client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.userID = user.ID
}

type playlistEntry struct {
	ID   spotify.ID `json:"id"`
	Name string     `json:"name"`
}

type playlistsContainer struct {
	Playlists []playlistEntry `json:"items"`
}

func (cu *clientUser) getPlaylists(filePath string) {
	log.Println("Getting user playlists")
	offset, limit := 0, 50

	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit

	var encoder *json.Encoder
	var total int
	var playlists []playlistEntry

	if filePath != "" {
		os.Remove(filePath)

		file, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		encoder = json.NewEncoder(file)
	}

	for total = limit; offset < total; offset += limit {
		playlistsPage, err := cu.client.GetPlaylistsForUserOpt(cu.userID, &opt)
		if err != nil {
			log.Fatal(err)
		}
		total = playlistsPage.Total

		if encoder != nil {
			if playlists == nil {
				playlists = make([]playlistEntry, 0, total)
			}
			var plEntry playlistEntry
			for _, pl := range playlistsPage.Playlists {
				plEntry.ID = pl.ID
				plEntry.Name = pl.Name
				playlists = append(playlists, plEntry)
			}
		} else {
			for i, pl := range playlistsPage.Playlists {
				fmt.Printf("%03d) %s %s\n", offset+i, pl.ID, pl.Name)
			}
		}
	}

	if encoder != nil {
		var plCon playlistsContainer
		plCon.Playlists = playlists
		encoder.SetIndent("", "\t")
		err := encoder.Encode(plCon)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("User playlists saved to", playlistsPath)
	}
}

func (cu *clientUser) loadPlaylists(filePath string, plCon *playlistsContainer) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&plCon)
	if err != nil {
		log.Fatal(err)
	}
}
