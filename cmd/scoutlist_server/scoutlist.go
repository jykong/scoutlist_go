package main

import (
	"fmt"
	"log"
	"net/http"

	"scoutlist"
)

var cu scoutlist.ClientUser

func main() {
	mode := scoutlist.CodeTest // change this to switch between code & user tests
	scoutlist.SetMode(mode)

	cu.Client = scoutlist.Auth()
	getCurrentUserID(cu)

	handleRequests()
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")

	var opt scoutlist.Options
	opt.LastN = 15
	opt.OutN = 15
	scoutlist.Update(&cu, &opt)
}

func handleRequests() {
	http.HandleFunc("/", homePage)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func getCurrentUserID(cu scoutlist.ClientUser) {
	user, err := cu.Client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.UserID = user.ID
}
