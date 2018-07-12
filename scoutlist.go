package main

import (
	"fmt"
	"log"

	"github.com/zmb3/spotify"
)

func main() {
	var client *spotify.Client
	client = scoutlistAuth()
	// use the client to make calls that require authorization
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
}
