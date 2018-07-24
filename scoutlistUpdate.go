package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/zmb3/spotify"
)

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
type trackIDta struct {
	ID spotify.ID
	ta titleArtists
}

func scoutlistUpdate(cu *clientUser) {
	cu.client.AutoRetry = true

	//playlists := cu.getPlaylists()
	//savePlaylistsToJSON(playlistsPath, &playlists)

	excPlaylists := loadPlaylistsFromJSON(excPlaylistsPath)
	//fmt.Println(excPlaylists)

	excTracks := cu.getUniqueTracksFromPlaylistsAsync(&excPlaylists, nil)
	fmt.Println(len(excTracks))
	//excTracks := cu.getUniqueTracksFromPlaylists(&excPlaylists, nil)
	//saveTracksToGob(excTracksPath, &excTracks)
	//excTracks := loadTracksFromGob(excTracksPath)
	//fmt.Println(len(excTracks.TracksMap))
	//fmt.Println(excTracks)

	incPlaylists := loadPlaylistsFromJSON(incPlaylistPath)

	filteredTracks := cu.getUniqueTracksFromPlaylistsAsync(&incPlaylists, excTracks)
	fmt.Println(len(filteredTracks))
	//filteredTracks := cu.getUniqueTracksFromPlaylists(&incPlaylists, &excTracks)
	//fmt.Println(len(filteredTracks.TracksMap))
	//fmt.Println(filteredTracks)

	scoutlistID := loadIDFromGob(scoutlistIDPath)
	scoutedlistID := loadIDFromGob(scoutedlistIDPath)
	scoutlistID, scoutedlistID = cu.recycleScoutlist(scoutlistID, scoutedlistID)
	saveIDToGob(scoutlistIDPath, &scoutlistID)
	saveIDToGob(scoutedlistIDPath, &scoutedlistID)
	//trackIDs := getNTrackIDsFromTracksContainer(&filteredTracks, 30)
	trackIDs := getNTrackIDsFromTrackIDtaSlice(filteredTracks, 30)
	cu.replacePlaylistTracks(scoutlistID, trackIDs)
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
					if !excTracks.contains(&tr.Track.ID, &ta) {
						uniqueTracks.add(&tr.Track.ID, &ta)
					}
				} else {
					uniqueTracks.add(&tr.Track.ID, &ta)
				}
			}
		}
		acc += total
	}
	log.Println("Done getting unique tracks.")
	fmt.Println(acc)
	return uniqueTracks
}

const initBurst int = 7

func rateLimiter(limiter chan int, stop chan int) {
	for i := 0; i < initBurst; i++ {
		limiter <- 1
	}
	for range time.Tick(77 * time.Millisecond) {
		select {
		case <-stop:
			return
		default:
			limiter <- 1
		}
	}
}

func (cu *clientUser) getUniqueTracksFromPlaylistsAsync(
	srcPlaylists *playlistsContainer, excTracks []trackIDta) []trackIDta {
	log.Println("Getting unique tracks from playlists...")
	ratelimiter := make(chan int, initBurst)
	stopRateLimiter := make(chan int, 1)
	go rateLimiter(ratelimiter, stopRateLimiter)
	runtime.Gosched()

	plTracks := make(chan []trackIDta, len(srcPlaylists.Playlists))
	for _, pl := range srcPlaylists.Playlists {
		go func(plid spotify.ID) {
			plTracks <- cu.fetchPlaylistTracks(ratelimiter, plid, excTracks)
		}(pl.ID)
		runtime.Gosched()
	}
	uniqueTracks := make([]trackIDta, 0)
	acc := 0
	for range srcPlaylists.Playlists {
		srcTracks := <-plTracks
		uniqueTracks = addUniqueTracks(uniqueTracks, srcTracks, excTracks)
		acc += len(srcTracks)
	}
	stopRateLimiter <- 1
	log.Println("Done getting unique tracks.")
	fmt.Println(acc)
	return uniqueTracks
}

func (cu *clientUser) fetchPlaylistTracks(ratelimiter chan int, plid spotify.ID,
	excTracks []trackIDta) []trackIDta {
	offset, limit := 0, 100
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	fields := "items.track(id, name, artists.id), total"
	<-ratelimiter
	plTrackPage, err := cu.client.GetPlaylistTracksOpt(cu.userID, plid, &opt, fields)
	if err != nil {
		log.Fatal(err)
	}
	total := plTrackPage.Total
	nPages := total / limit
	if total%limit > 0 {
		nPages++
	}
	pgTracks := make(chan []trackIDta, nPages)
	for offset = limit; offset < total; offset += limit {
		go func(offset int) {
			pgTracks <- cu.fetchPlaylistTracksByPage(ratelimiter, plid, offset, limit, &fields)
		}(offset)
	}
	pgTracks <- getTracksFromPage(plTrackPage.Tracks)
	uniqueTracks := make([]trackIDta, 0)
	var pgTrack []trackIDta
	for i := 0; i < nPages; i++ {
		pgTrack = <-pgTracks
		uniqueTracks = addUniqueTracks(uniqueTracks, pgTrack, excTracks)
	}
	//log.Println("Fetched", plid)
	return uniqueTracks
}

func (cu *clientUser) fetchPlaylistTracksByPage(ratelimiter chan int,
	plid spotify.ID, offset int, limit int, fields *string) []trackIDta {
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	<-ratelimiter
	plTrackPage, err := cu.client.GetPlaylistTracksOpt(cu.userID, plid, &opt, *fields)
	if err != nil {
		log.Fatal(err)
	}
	return getTracksFromPage(plTrackPage.Tracks)
}

func getTracksFromPage(pageTracks []spotify.PlaylistTrack) []trackIDta {
	tracks := make([]trackIDta, len(pageTracks))
	var track trackIDta
	for i, tr := range pageTracks {
		track.ta.Title = tr.Track.Name
		track.ta.Artists = nil
		for _, ar := range tr.Track.Artists {
			track.ta.Artists = append(track.ta.Artists, ar.Name)
		}
		track.ID = tr.Track.ID
		tracks[i] = track
	}
	return tracks
}

func addUniqueTracks(uniqueTracks []trackIDta, srcTracks []trackIDta,
	excTracks []trackIDta) []trackIDta {
	const interval = 7000
	acc := 0
	n := len(uniqueTracks)
	if excTracks != nil {
		for _, track := range srcTracks {
			if !tracksContain(excTracks, &track) {
				uniqueTracks = tracksAdd(uniqueTracks, &track)
			}
			acc += n
			if acc > interval {
				acc -= interval
				runtime.Gosched()
			}
		}
	} else {
		for _, track := range srcTracks {
			uniqueTracks = tracksAdd(uniqueTracks, &track)
			acc += n
			if acc > interval {
				acc -= interval
				runtime.Gosched()
			}
		}
	}
	return uniqueTracks
}

func tracksContain(tracks []trackIDta, newTrack *trackIDta) bool {
	for i := 0; i < len(tracks); i++ {
		if tracks[i].ID == newTrack.ID {
			return true
		}
		if tracks[i].ta.Title == newTrack.ta.Title {
			nArtists := len(tracks[i].ta.Artists)
			if nArtists != len(newTrack.ta.Artists) {
				continue
			}
			match := true
			for _, artist := range tracks[i].ta.Artists {
				match = false
				for _, ntArtist := range newTrack.ta.Artists {
					if artist == ntArtist {
						match = true
						break
					}
				}
				if match == false {
					break
				}
			}
			if match == true {
				return true
			}
		}
	}
	return false
}

func tracksAdd(tracks []trackIDta, track *trackIDta) []trackIDta {
	if !tracksContain(tracks, track) {
		return append(tracks, *track)
	}
	return tracks
}

func (uniqueTracks *tracksContainer) contains(tid *spotify.ID, ta *titleArtists) bool {
	_, present := uniqueTracks.TracksMap[*tid]
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

func (uniqueTracks *tracksContainer) add(tid *spotify.ID, ta *titleArtists) {
	if uniqueTracks.contains(tid, ta) == false {
		uniqueTracks.TracksMap[*tid] = *ta
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

func getNTrackIDsFromTracksContainer(trCon *tracksContainer, n int) []spotify.ID {
	size := len(trCon.TracksMap)
	if size < n {
		n = size
	}
	trackIDs := make([]spotify.ID, n)
	i := 0
	for k := range trCon.TracksMap {
		if i < n {
			trackIDs[i] = k
			i++
		} else {
			break
		}
	}
	return trackIDs
}

func getNTrackIDsFromTrackIDtaSlice(tracks []trackIDta, n int) []spotify.ID {
	size := len(tracks)
	if size < n {
		n = size
	}
	trackIDs := make([]spotify.ID, n)
	for i, tr := range tracks {
		if i < n {
			trackIDs[i] = tr.ID
		} else {
			break
		}
	}
	return trackIDs
}

func (cu *clientUser) replacePlaylistTracks(plid spotify.ID, trackIDs []spotify.ID) {
	log.Println("Replacing playlist tracks...")

	err := cu.client.ReplacePlaylistTracks(cu.userID, plid, trackIDs...)
	if err != nil {
		log.Fatal(err)
	}
}
