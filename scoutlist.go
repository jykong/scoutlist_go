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
}

func (cu *clientUser) getCurrentUserID() {
	user, err := cu.client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.userID = user.ID
}

type playlistIDName struct {
	ID   spotify.ID `json:"ID"`
	Name string     `json:"Name"`
}

func (cu *clientUser) getPlaylists(filePath string) {
	log.Println("Getting user playlists")
	offset, limit := 0, 50

	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit

	var encoder *json.Encoder
	var playlists []playlistIDName

	if filePath != "" {
		os.Remove(filePath)

		file, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		encoder = json.NewEncoder(file)
	}

	for total := limit; offset < total; offset += limit {
		playlistsPage, err := cu.client.GetPlaylistsForUserOpt(cu.userID, &opt)
		if err != nil {
			log.Fatal(err)
		}
		total = playlistsPage.Total

		if encoder != nil {
			if playlists == nil {
				playlists = make([]playlistIDName, 0, total)
			}
			var plIDName playlistIDName
			for _, pl := range playlistsPage.Playlists {
				plIDName.ID = pl.ID
				plIDName.Name = pl.Name
				playlists = append(playlists, plIDName)
			}
		} else {
			for i, pl := range playlistsPage.Playlists {
				fmt.Printf("%03d) %s %s\n", offset+i, pl.ID, pl.Name)
			}
		}
	}

	if encoder != nil {
		encoder.SetIndent("", "\t")
		err := encoder.Encode(playlists)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("User playlists saved to", playlistsPath)
	}
}
