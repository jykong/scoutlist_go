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

type playlistEntry struct {
	ID   spotify.ID `json:"id"`
	Name string     `json:"name"`
}

type playlistsContainer struct {
	Playlists []playlistEntry `json:"items"`
}

type titleArtists struct {
	Title   string
	Artists []string
}

type tracksContainer struct {
	tracksMap map[spotify.ID]titleArtists
}

const playlistsPath = "./playlists.json"
const excPlaylistsPath = "./exc_playlists.json"

func main() {
	var cu clientUser
	cu.client = scoutlistAuth()
	cu.getCurrentUserID()

	//var playlists playlistsContainer
	//cu.getPlaylists(&playlists)
	//savePlaylists(playlistsPath, &playlists)

	var excPlaylists playlistsContainer
	loadPlaylists(excPlaylistsPath, &excPlaylists)
	//fmt.Println(excPlaylists)

	var excTracks tracksContainer
	cu.getUniqueTracksFromPlaylists(&excPlaylists, &excTracks)
	fmt.Println(len(excTracks.tracksMap))
	//fmt.Println(excTracks)
}

func (cu *clientUser) getCurrentUserID() {
	user, err := cu.client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.userID = user.ID
}

func (cu *clientUser) getPlaylists(plCon *playlistsContainer) {
	log.Println("Getting user playlists")
	offset, limit := 0, 50

	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit

	var total int
	var playlists []playlistEntry

	for total = limit; offset < total; offset += limit {
		playlistsPage, err := cu.client.GetPlaylistsForUserOpt(cu.userID, &opt)
		if err != nil {
			log.Fatal(err)
		}
		if playlists == nil {
			total = playlistsPage.Total
			playlists = make([]playlistEntry, 0, total)
		}
		var plEntry playlistEntry
		for _, pl := range playlistsPage.Playlists {
			plEntry.ID = pl.ID
			plEntry.Name = pl.Name
			playlists = append(playlists, plEntry)
		}
		//for i, pl := range playlistsPage.Playlists {
		//	fmt.Printf("%03d) %s %s\n", offset+i, pl.ID, pl.Name)
		//}
	}

	plCon.Playlists = playlists
}

func savePlaylists(filePath string, plCon *playlistsContainer) {
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

func loadPlaylists(filePath string, plCon *playlistsContainer) {
	log.Println("Loading playlists from", filePath)
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

func (cu *clientUser) getUniqueTracksFromPlaylists(
	srcPlaylists *playlistsContainer, uniqueTracks *tracksContainer) {

	log.Println("Getting unique tracks from playlists...")
	var offset, limit int

	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit

	fields := "items.track(id, name, artists.id), total"

	var total int
	var ta titleArtists

	uniqueTracks.tracksMap = make(map[spotify.ID]titleArtists)

	acc := 0
	for _, pl := range srcPlaylists.Playlists {
		for offset, limit, total = 0, 100, 100; offset < total; offset += limit {
			plTrackPage, err := cu.client.GetPlaylistTracksOpt(cu.userID, pl.ID, &opt, fields)
			if err != nil {
				log.Fatal(err)
			}
			total = plTrackPage.Total
			for _, tr := range plTrackPage.Tracks {
				ta.Title = tr.Track.Name
				ta.Artists = nil
				for _, ar := range tr.Track.Artists {
					ta.Artists = append(ta.Artists, ar.Name)
				}
				uniqueTracks.add(tr.Track.ID, &ta)
			}
		}
		acc += total
	}
	log.Println("Done getting unique tracks.")
	fmt.Println(acc)
}

func (uniqueTracks *tracksContainer) contains(tid spotify.ID, ta *titleArtists) bool {
	_, present := uniqueTracks.tracksMap[tid]
	if present == true {
		return true
	}
	for _, uta := range uniqueTracks.tracksMap {
		if ta.Title == uta.Title {
			nArtists := len(ta.Artists)
			if nArtists != len(uta.Artists) {
				continue
			}
			uartists := make([]string, nArtists)
			artists := make([]string, nArtists)
			copy(uartists, uta.Artists)
			copy(artists, ta.Artists)
			for match := true; match == true; {
				match = false
				for i, uartist := range uartists {
					for j, artist := range artists {
						if artist == uartist {
							uartists = append(uartists[:i], uartists[i+1:]...)
							artists = append(artists[:j], artists[j+1:]...)
							match = true
							break
						}
					}
					if match {
						break
					}
				}
			}
			if len(uartists) == 0 && len(artists) == 0 {
				return true
			}
		}
	}
	return false
}

func (uniqueTracks *tracksContainer) add(tid spotify.ID, ta *titleArtists) {
	if uniqueTracks.contains(tid, ta) == false {
		uniqueTracks.tracksMap[tid] = *ta
	}
}
