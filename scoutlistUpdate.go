package scoutlist

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/zmb3/spotify"
)

// ClientUser Spotify Client and UserID
type ClientUser struct {
	Client *spotify.Client
	UserID string
}

// Options options
type Options struct {
	LastN int
	OutN  int
}
type playlistEntry struct {
	ID   spotify.ID `json:"id"`
	Name string     `json:"name"`
}
type titleArtists struct {
	Title   string
	Artists []string
}
type trackIDTA struct {
	ID spotify.ID
	TA titleArtists
}

const initBurst int = 10

func rateLimiter(limiter chan int, stop chan int) {
	for i := 0; i < initBurst; i++ {
		limiter <- 1
	}
	for range time.Tick(76 * time.Millisecond) {
		select {
		case <-stop:
			return
		default:
			limiter <- 1
		}
	}
}

// Update update
func Update(cu *ClientUser, opt *Options) {
	cu.Client.AutoRetry = true
	strCon := getStringConsts()

	//playlists := cu.getPlaylists()
	//savePlaylistsToJSON(strCon.PlaylistsPath, playlists)

	//excPlaylists := loadPlaylistsFromJSON(strCon.ExcPlaylistsPath)
	//fmt.Println(excPlaylists)

	rateLimit := make(chan int, initBurst)
	stopRateLimiter := make(chan int, 1)
	go rateLimiter(rateLimit, stopRateLimiter)
	runtime.Gosched()

	//excTracks := cu.getUniqueTracksFromPlaylists(rateLimit, excPlaylists, nil, 0)
	//saveTracksToGob(strCon.ExcTracksPath, excTracks)
	excTracks := loadTracksFromGob(strCon.ExcTracksPath)
	fmt.Println(len(excTracks))
	//fmt.Println(excTracks)

	incPlaylists := loadPlaylistsFromJSON(strCon.IncPlaylistPath)

	filteredTracks := cu.getUniqueTracksFromPlaylists(rateLimit, incPlaylists, excTracks, opt.LastN)
	fmt.Println(len(filteredTracks))
	//fmt.Println(filteredTracks)

	stopRateLimiter <- 1

	scoutlistID := loadIDFromGob(strCon.ScoutlistIDPath)
	scoutedlistID := loadIDFromGob(strCon.ScoutedlistIDPath)
	scoutlistID = cu.checkAndCreatePlaylist(scoutlistID, strCon.ScoutlistName)
	scoutedlistID = cu.checkAndCreatePlaylist(scoutedlistID, strCon.ScoutedlistName)
	saveIDToGob(strCon.ScoutlistIDPath, &scoutlistID)
	saveIDToGob(strCon.ScoutedlistIDPath, &scoutedlistID)

	excTracks = cu.recycleScoutlist(scoutlistID, scoutedlistID, excTracks)
	saveTracksToGob(strCon.ExcTracksPath, excTracks)
	trackIDs := getNTrackIDsFromTrackIDTASlice(filteredTracks, opt.OutN, true)
	//fmt.Println(trackIDs)
	cu.replacePlaylistTracks(scoutlistID, trackIDs)
}

func (cu *ClientUser) getPlaylists() []playlistEntry {
	log.Println("Getting user playlists")
	offset, limit := 0, 50
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	var playlists []playlistEntry
	for total := limit; offset < total; offset += limit {
		playlistsPage, err := cu.Client.GetPlaylistsForUserOpt(cu.UserID, &opt)
		if err != nil {
			log.Fatal(err)
		}
		if playlists == nil {
			total = playlistsPage.Total
			playlists = make([]playlistEntry, total)
		}
		var plEntry playlistEntry
		for i, pl := range playlistsPage.Playlists {
			plEntry.ID = pl.ID
			plEntry.Name = pl.Name
			playlists[offset+i] = plEntry
		}
	}
	return playlists
}

func (cu *ClientUser) getUniqueTracksFromPlaylists(rateLimit chan int,
	srcPlaylists []playlistEntry, excTracks []trackIDTA, lastN int) []trackIDTA {
	log.Println("Getting unique tracks from playlists...")
	nPlaylists := len(srcPlaylists)
	plTracks := make(chan []trackIDTA, nPlaylists)
	for i := 0; i < nPlaylists; i++ {
		if lastN > 0 {
			go func(plid spotify.ID) {
				plTracks <- cu.fetchLastNPlaylistTracks(rateLimit, plid, excTracks, lastN)
			}(srcPlaylists[i].ID)
		} else {
			go func(plid spotify.ID) {
				plTracks <- cu.fetchPlaylistTracks(rateLimit, plid, excTracks)
			}(srcPlaylists[i].ID)
		}
		runtime.Gosched()
	}
	uniqueTracks := make([]trackIDTA, 0)
	acc := 0
	for i := 0; i < nPlaylists; i++ {
		srcTracks := <-plTracks
		uniqueTracks = addUniqueTracks(uniqueTracks, srcTracks, excTracks)
		acc += len(srcTracks)
	}
	log.Println("Done getting unique tracks.")
	fmt.Println(acc)
	return uniqueTracks
}

func (cu *ClientUser) fetchPlaylistTracks(rateLimit chan int, plid spotify.ID,
	excTracks []trackIDTA) []trackIDTA {
	offset, limit := 0, 100
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	fields := "items.track(id, name, artists.id), total"
	if rateLimit != nil {
		<-rateLimit
	}
	plTrackPage, err := cu.Client.GetPlaylistTracksOpt(cu.UserID, plid, &opt, fields)
	if err != nil {
		log.Fatal(err)
	}
	total := plTrackPage.Total
	uniqueTracks := make([]trackIDTA, 0)
	if total == 0 {
		return uniqueTracks
	}
	nPages := total / limit
	if total%limit > 0 {
		nPages++
	}
	pgTracks := make(chan []trackIDTA, nPages)
	for offset = limit; offset < total; offset += limit {
		go func(offset int) {
			pgTracks <- cu.fetchPlaylistTracksByPage(rateLimit, plid, offset, limit, &fields)
		}(offset)
	}
	pgTracks <- getTracksFromPage(plTrackPage.Tracks)
	var pgTrack []trackIDTA
	for i := 0; i < nPages; i++ {
		pgTrack = <-pgTracks
		uniqueTracks = addUniqueTracks(uniqueTracks, pgTrack, excTracks)
	}
	//log.Println("Fetched", plid)
	return uniqueTracks
}

func (cu *ClientUser) fetchLastNPlaylistTracks(rateLimit chan int, plid spotify.ID,
	excTracks []trackIDTA, lastN int) []trackIDTA {
	offset, limit := 0, 100
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	fields := "items.track(id, name, artists.id), total"
	if rateLimit != nil {
		<-rateLimit
	}
	plTrackPage, err := cu.Client.GetPlaylistTracksOpt(cu.UserID, plid, &opt, fields)
	if err != nil {
		log.Fatal(err)
	}
	total := plTrackPage.Total
	uniqueTracks := make([]trackIDTA, 0)
	if total == 0 {
		return uniqueTracks
	}
	var pgTrack []trackIDTA
	for offset = (total - total%limit); offset >= 0; offset -= limit {
		if offset > 0 {
			pgTrack = cu.fetchPlaylistTracksByPage(rateLimit, plid, offset, limit, &fields)
		} else {
			pgTrack = getTracksFromPage(plTrackPage.Tracks)
		}
		uniqueTracks = addUniqueTracks(uniqueTracks, pgTrack, excTracks)
		if len(uniqueTracks) >= lastN {
			break
		}
	}
	//log.Println("Fetched", plid)
	return uniqueTracks
}

func (cu *ClientUser) fetchPlaylistTracksByPage(rateLimit chan int,
	plid spotify.ID, offset int, limit int, fields *string) []trackIDTA {
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	if rateLimit != nil {
		<-rateLimit
	}
	plTrackPage, err := cu.Client.GetPlaylistTracksOpt(cu.UserID, plid, &opt, *fields)
	if err != nil {
		log.Fatal(err)
	}
	return getTracksFromPage(plTrackPage.Tracks)
}

func getTracksFromPage(pageTracks []spotify.PlaylistTrack) []trackIDTA {
	tracks := make([]trackIDTA, len(pageTracks))
	var track trackIDTA
	for i, tr := range pageTracks {
		track.TA.Title = tr.Track.Name
		track.TA.Artists = nil
		for _, ar := range tr.Track.Artists {
			track.TA.Artists = append(track.TA.Artists, ar.Name)
		}
		track.ID = tr.Track.ID
		tracks[i] = track
	}
	return tracks
}

func addUniqueTracks(uniqueTracks []trackIDTA, srcTracks []trackIDTA,
	excTracks []trackIDTA) []trackIDTA {
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

func tracksContain(tracks []trackIDTA, newTrack *trackIDTA) bool {
	for i := 0; i < len(tracks); i++ {
		if tracks[i].ID == newTrack.ID {
			return true
		}
		if tracks[i].TA.Title == newTrack.TA.Title {
			nArtists := len(tracks[i].TA.Artists)
			if nArtists != len(newTrack.TA.Artists) {
				continue
			}
			match := true
			for _, artist := range tracks[i].TA.Artists {
				match = false
				for _, ntArtist := range newTrack.TA.Artists {
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

func tracksAdd(tracks []trackIDTA, track *trackIDTA) []trackIDTA {
	if !tracksContain(tracks, track) {
		return append(tracks, *track)
	}
	return tracks
}

func (cu *ClientUser) checkAndCreatePlaylist(plid spotify.ID, plname string) spotify.ID {
	if plid == "" {
		return cu.createPlaylist(plid, plname)
	}
	_, err := cu.Client.GetPlaylist(cu.UserID, plid)
	if err != nil {
		// TODO: modify to examine # of followers & followers object
		log.Println(plname + "not found on Spotify")
		return cu.createPlaylist(plid, plname)
	}
	return plid
}

func (cu *ClientUser) recycleScoutlist(scoutlistID spotify.ID, scoutedlistID spotify.ID,
	excTracks []trackIDTA) []trackIDTA {
	if excTracks == nil {
		scoutlistTrackIDs, err := cu.getPlaylistTrackIDs(scoutlistID)
		if err != nil {
			log.Fatal(err)
		}
		if len(scoutlistTrackIDs) > 0 {
			_, err = cu.Client.AddTracksToPlaylist(cu.UserID, scoutedlistID, scoutlistTrackIDs...)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		scoutlistTracks := cu.fetchPlaylistTracks(nil, scoutlistID, nil)
		if len(scoutlistTracks) > 0 {
			scoutlistTrackIDs := getNTrackIDsFromTrackIDTASlice(scoutlistTracks, -1, false)
			_, err := cu.Client.AddTracksToPlaylist(cu.UserID, scoutedlistID, scoutlistTrackIDs...)
			if err != nil {
				log.Fatal(err)
			}
			excTracks = addUniqueTracks(excTracks, scoutlistTracks, nil)
		}
	}
	return excTracks
}

func (cu *ClientUser) createPlaylist(scoutlistID spotify.ID, s string) spotify.ID {
	log.Println("Creating new", s)
	pl, err := cu.Client.CreatePlaylistForUser(cu.UserID, s, false)
	if err != nil {
		log.Fatal(err)
	}
	return pl.ID
}

func (cu *ClientUser) getPlaylistTrackIDs(plid spotify.ID) ([]spotify.ID, error) {
	log.Println("Getting playlist track ids...")
	var trackIDs []spotify.ID
	var offset, limit, total int
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	fields := "items.track.id, total"
	for offset, limit, total = 0, 100, 100; offset < total; offset += limit {
		plTrackPage, err := cu.Client.GetPlaylistTracksOpt(cu.UserID, plid, &opt, fields)
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

func getNTrackIDsFromTrackIDTASlice(tracks []trackIDTA, n int, random bool) []spotify.ID {
	size := len(tracks)
	if size < n || n == -1 {
		n = size
	}
	trackIDs := make([]spotify.ID, n)
	if random {
		s := rand.NewSource(time.Now().UnixNano())
		r := rand.New(s)
		var alreadyGen bool
		genNums := make([]int, n)
		for i := 0; i < n; {
			alreadyGen = false
			rn := r.Intn(size)
			for j := 0; j < i; j++ {
				if rn == genNums[j] {
					alreadyGen = true
					break
				}
			}
			if !alreadyGen {
				genNums[i] = rn
				trackIDs[i] = tracks[rn].ID
				i++
			}
		}
	} else {
		for i, tr := range tracks {
			if i < n {
				trackIDs[i] = tr.ID
			} else {
				break
			}
		}
	}
	return trackIDs
}

func (cu *ClientUser) replacePlaylistTracks(plid spotify.ID, trackIDs []spotify.ID) {
	log.Println("Replacing playlist tracks...")

	err := cu.Client.ReplacePlaylistTracks(cu.UserID, plid, trackIDs...)
	if err != nil {
		log.Fatal(err)
	}
}
