package main

import (
	"fmt"
	"log"

	"github.com/zmb3/spotify"
)

type clientUser struct {
	client *spotify.Client
	userID string
}

func main() {
	var cu clientUser
	cu.client = scoutlistAuth()
	cu.getCurrentUserID()

	cu.getPlaylists("")
}

func (cu *clientUser) getCurrentUserID() {
	user, err := cu.client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.userID = user.ID
}

func (cu *clientUser) getPlaylists(file string) {
	var opt spotify.Options
	offset, limit := 0, 50
	opt.Offset = &offset
	opt.Limit = &limit

	for total := limit; offset < total; offset += limit {
		playlistsPage, err := cu.client.GetPlaylistsForUserOpt(cu.userID, &opt)
		if err != nil {
			log.Println(err)
		}

		for i, pl := range playlistsPage.Playlists {
			fmt.Printf("%03d) %s %s\n", offset+i, pl.ID, pl.Name)
		}
		total = playlistsPage.Total
	}
}
