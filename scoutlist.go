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
	TracksMap map[spotify.ID]titleArtists
}

func main() {
	var cu clientUser
	cu.client = scoutlistAuth()
	cu.getCurrentUserID()

	//playlists := cu.getPlaylists()
	//savePlaylistsToJSON(playlistsPath, &playlists)

	//excPlaylists := loadPlaylistsFromJSON(excPlaylistsPath)
	//fmt.Println(excPlaylists)

	//excTracks := cu.getUniqueTracksFromPlaylists(&excPlaylists, nil)
	//saveTracksToGob(excTracksPath, &excTracks)
	excTracks := loadTracksFromGob(excTracksPath)
	fmt.Println(len(excTracks.TracksMap))
	//fmt.Println(excTracks)

	incPlaylists := loadPlaylistsFromJSON(incPlaylistPath)

	filteredTracks := cu.getUniqueTracksFromPlaylists(&incPlaylists, &excTracks)
	fmt.Println(len(filteredTracks.TracksMap))
	//fmt.Println(filteredTracks)

	scoutlistID := loadIDFromGob(scoutlistIDPath)
	scoutedlistID := loadIDFromGob(scoutedlistIDPath)
	scoutlistID, scoutedlistID = cu.recycleScoutlist(scoutlistID, scoutedlistID)
	saveIDToGob(scoutlistIDPath, &scoutlistID)
	saveIDToGob(scoutedlistIDPath, &scoutedlistID)
	cu.replacePlaylistWithNTracks(scoutlistID, &filteredTracks, 30)
}

func (cu *clientUser) getCurrentUserID() {
	user, err := cu.client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.userID = user.ID
}

func (cu *clientUser) getPlaylists() playlistsContainer {
	log.Println("Getting user playlists")
	offset, limit := 0, 50
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	var playlists []playlistEntry
	for total := limit; offset < total; offset += limit {
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
	var plCon playlistsContainer
	plCon.Playlists = playlists
	return plCon
}

func (cu *clientUser) getUniqueTracksFromPlaylists(
	srcPlaylists *playlistsContainer, excTracks *tracksContainer) tracksContainer {
	log.Println("Getting unique tracks from playlists...")
	var offset, limit, total int
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	fields := "items.track(id, name, artists.id), total"
	var ta titleArtists
	var uniqueTracks tracksContainer
	uniqueTracks.TracksMap = make(map[spotify.ID]titleArtists)
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
				if excTracks != nil {
					if !excTracks.contains(tr.Track.ID, &ta) {
						uniqueTracks.add(tr.Track.ID, &ta)
					}
				} else {
					uniqueTracks.add(tr.Track.ID, &ta)
				}
			}
		}
		acc += total
	}
	log.Println("Done getting unique tracks.")
	fmt.Println(acc)
	return uniqueTracks
}

func (uniqueTracks *tracksContainer) contains(tid spotify.ID, ta *titleArtists) bool {
	_, present := uniqueTracks.TracksMap[tid]
	if present == true {
		return true
	}
	for _, uta := range uniqueTracks.TracksMap {
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
					if artists[0] == uartist {
						uartists = append(uartists[:i], uartists[i+1:]...)
						artists = artists[1:]
						match = true
						break
					}
				}
			}
			if len(artists) == 0 {
				return true
			}
		}
	}
	return false
}

func (uniqueTracks *tracksContainer) add(tid spotify.ID, ta *titleArtists) {
	if uniqueTracks.contains(tid, ta) == false {
		uniqueTracks.TracksMap[tid] = *ta
	}
}

func (cu *clientUser) recycleScoutlist(
	scoutlistID spotify.ID, scoutedlistID spotify.ID) (spotify.ID, spotify.ID) {
	if scoutlistID == "" {
		return cu.createPlaylist(scoutlistID, "Scoutlist"), ""
	}
	_, err := cu.client.GetPlaylist(cu.userID, scoutlistID)
	if err != nil {
		// TODO: modify to examine # of followers & followers object
		log.Println("Scoutlist not found on Spotify")
		return cu.createPlaylist(scoutlistID, "Scoutlist"), ""
	}
	if scoutedlistID == "" {
		scoutedlistID = cu.createPlaylist(scoutedlistID, "Scoutedlist")
	} else {
		_, err := cu.client.GetPlaylist(cu.userID, scoutedlistID)
		if err != nil {
			// TODO: modify to examine # of followers & followers object
			log.Println("Scoutedlist not found on Spotify")
			scoutedlistID = cu.createPlaylist(scoutedlistID, "Scoutedlist")
		}
	}
	scoutlistTrackIDs, err := cu.getPlaylistTrackIDs(scoutlistID)
	if err != nil {
		log.Fatal(err)
	}
	if len(scoutlistTrackIDs) > 0 {
		// Maybe TODO: modify this to add only unique tracks
		_, err = cu.client.AddTracksToPlaylist(cu.userID, scoutedlistID, scoutlistTrackIDs...)
		if err != nil {
			log.Fatal(err)
		}
	}
	return scoutlistID, scoutedlistID
}

func (cu *clientUser) createPlaylist(scoutlistID spotify.ID, s string) spotify.ID {
	log.Println("Creating new", s)
	pl, err := cu.client.CreatePlaylistForUser(cu.userID, s, false)
	if err != nil {
		log.Fatal(err)
	}
	return pl.ID
}

func (cu *clientUser) getPlaylistTrackIDs(plid spotify.ID) ([]spotify.ID, error) {
	log.Println("Getting playlist track ids...")
	var trackIDs []spotify.ID
	var offset, limit, total int
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	fields := "items.track.id, total"
	for offset, limit, total = 0, 100, 100; offset < total; offset += limit {
		plTrackPage, err := cu.client.GetPlaylistTracksOpt(cu.userID, plid, &opt, fields)
		if err != nil {
			return trackIDs, err
		}
		total = plTrackPage.Total
		if trackIDs == nil {
			trackIDs = make([]spotify.ID, total)
		}
		for i, tr := range plTrackPage.Tracks {
			trackIDs[offset+i] = tr.Track.ID
		}
	}
	return trackIDs, nil
}

func (cu *clientUser) replacePlaylistWithNTracks(plid spotify.ID, trCon *tracksContainer, n int) {
	log.Println("Replacing playlist tracks...")
	size := len(trCon.TracksMap)
	if size < n {
		n = size
	}
	trackIDs := make([]spotify.ID, n)
	i := 0
	for k := range trCon.TracksMap {
		trackIDs[i] = k
		if i++; i >= n {
			break
		}
	}
	err := cu.client.ReplacePlaylistTracks(cu.userID, plid, trackIDs...)
	if err != nil {
		log.Fatal(err)
	}
}
