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
	playlistsPage, err := cu.client.GetPlaylistsForUser(cu.userID)
	if err != nil {
		log.Fatal(err)
	}
	//total := playlistsPage.Total
	//offset := playlistsPage.Offset
	limit := playlistsPage.Limit
	fmt.Println(playlistsPage.Total)
	for i := 0; i < limit; i++ {
		fmt.Println(playlistsPage.Playlists[i].ID, playlistsPage.Playlists[i].Name)
	}
}
