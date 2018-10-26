package scoutlist

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/zmb3/spotify"
	"scoutlist/data/model"
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

var rateLimit chan int
var stopRateLimiter chan int

const initBurst int = 10

func StartRateLimiter() {
	rateLimit = make(chan int, initBurst)
	stopRateLimiter = make(chan int, 1)
	go rateLimiter(rateLimit, stopRateLimiter)
	runtime.Gosched()
}

func StopRateLimiter() {
	stopRateLimiter <- 1
}

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

func GetPlaylists(cu *ClientUser) []model.PlaylistEntry {
	log.Println("Getting user playlists")
	offset, limit := 0, 50
	var opt spotify.Options
	opt.Offset = &offset
	opt.Limit = &limit
	var playlists []model.PlaylistEntry
	for total := limit; offset < total; offset += limit {
		playlistsPage, err := cu.Client.GetPlaylistsForUserOpt(cu.UserID, &opt)
		if err != nil {
			log.Fatal(err)
		}
		if playlists == nil {
			total = playlistsPage.Total
			playlists = make([]model.PlaylistEntry, total)
		}
		var plEntry model.PlaylistEntry
		for i, pl := range playlistsPage.Playlists {
			plEntry.ID = pl.ID
			plEntry.Name = pl.Name
			playlists[offset+i] = plEntry
		}
	}
	return playlists
}

func GetUniqueTracksFromPlaylists(cu *ClientUser, srcPlaylists []model.PlaylistEntry,
	excTracks []model.TrackIDTA, lastN int) []model.TrackIDTA {
	log.Println("Getting unique tracks from playlists...")
	nPlaylists := len(srcPlaylists)
	plTracks := make(chan []model.TrackIDTA, nPlaylists)
	for i := 0; i < nPlaylists; i++ {
		if lastN > 0 {
			go func(plid spotify.ID) {
				plTracks <- fetchLastNPlaylistTracks(cu, plid, excTracks, lastN)
			}(srcPlaylists[i].ID)
		} else {
			go func(plid spotify.ID) {
				plTracks <- fetchPlaylistTracks(cu, plid, excTracks)
			}(srcPlaylists[i].ID)
		}
		runtime.Gosched()
	}
	uniqueTracks := make([]model.TrackIDTA, 0)
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

func fetchPlaylistTracks(cu *ClientUser, plid spotify.ID,
	excTracks []model.TrackIDTA) []model.TrackIDTA {
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
	uniqueTracks := make([]model.TrackIDTA, 0)
	if total == 0 {
		return uniqueTracks
	}
	nPages := total / limit
	if total%limit > 0 {
		nPages++
	}
	pgTracks := make(chan []model.TrackIDTA, nPages)
	for offset = limit; offset < total; offset += limit {
		go func(offset int) {
			pgTracks <- fetchPlaylistTracksByPage(cu, plid, offset, limit, &fields)
		}(offset)
	}
	pgTracks <- getTracksFromPage(plTrackPage.Tracks)
	var pgTrack []model.TrackIDTA
	for i := 0; i < nPages; i++ {
		pgTrack = <-pgTracks
		uniqueTracks = addUniqueTracks(uniqueTracks, pgTrack, excTracks)
	}
	//log.Println("Fetched", plid)
	return uniqueTracks
}

func fetchLastNPlaylistTracks(cu *ClientUser, plid spotify.ID,
	excTracks []model.TrackIDTA, lastN int) []model.TrackIDTA {
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
	uniqueTracks := make([]model.TrackIDTA, 0)
	if total == 0 {
		return uniqueTracks
	}
	var pgTrack []model.TrackIDTA
	for offset = (total - total%limit); offset >= 0; offset -= limit {
		if offset > 0 {
			pgTrack = fetchPlaylistTracksByPage(cu, plid, offset, limit, &fields)
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

func fetchPlaylistTracksByPage(cu *ClientUser, plid spotify.ID,
	offset int, limit int, fields *string) []model.TrackIDTA {
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

func getTracksFromPage(pageTracks []spotify.PlaylistTrack) []model.TrackIDTA {
	tracks := make([]model.TrackIDTA, len(pageTracks))
	var track model.TrackIDTA
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

func addUniqueTracks(uniqueTracks []model.TrackIDTA, srcTracks []model.TrackIDTA,
	excTracks []model.TrackIDTA) []model.TrackIDTA {
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

func tracksContain(tracks []model.TrackIDTA, newTrack *model.TrackIDTA) bool {
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

func tracksAdd(tracks []model.TrackIDTA, track *model.TrackIDTA) []model.TrackIDTA {
	if !tracksContain(tracks, track) {
		return append(tracks, *track)
	}
	return tracks
}

func CheckAndCreatePlaylist(cu *ClientUser, plid spotify.ID, plname string) spotify.ID {
	if plid == "" {
		return createPlaylist(cu, plid, plname)
	}
	_, err := cu.Client.GetPlaylist(cu.UserID, plid)
	if err != nil {
		// TODO: modify to examine # of followers & followers object
		log.Println(plname + "not found on Spotify")
		return createPlaylist(cu, plid, plname)
	}
	return plid
}

func RecycleScoutlist(cu *ClientUser, scoutlistID spotify.ID, scoutedlistID spotify.ID,
	excTracks []model.TrackIDTA) []model.TrackIDTA {
	if excTracks == nil {
		scoutlistTrackIDs, err := getPlaylistTrackIDs(cu, scoutlistID)
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
		scoutlistTracks := fetchPlaylistTracks(cu, scoutlistID, nil)
		if len(scoutlistTracks) > 0 {
			scoutlistTrackIDs := GetNTrackIDsFromTrackIDTASlice(scoutlistTracks, -1, false)
			_, err := cu.Client.AddTracksToPlaylist(cu.UserID, scoutedlistID, scoutlistTrackIDs...)
			if err != nil {
				log.Fatal(err)
			}
			excTracks = addUniqueTracks(excTracks, scoutlistTracks, nil)
		}
	}
	return excTracks
}

func createPlaylist(cu *ClientUser, scoutlistID spotify.ID, s string) spotify.ID {
	log.Println("Creating new", s)
	pl, err := cu.Client.CreatePlaylistForUser(cu.UserID, s, false)
	if err != nil {
		log.Fatal(err)
	}
	return pl.ID
}

func getPlaylistTrackIDs(cu *ClientUser, plid spotify.ID) ([]spotify.ID, error) {
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

func GetNTrackIDsFromTrackIDTASlice(tracks []model.TrackIDTA, n int, random bool) []spotify.ID {
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

func ReplacePlaylistTracks(cu *ClientUser, plid spotify.ID, trackIDs []spotify.ID) {
	log.Println("Replacing playlist tracks...")

	err := cu.Client.ReplacePlaylistTracks(cu.UserID, plid, trackIDs...)
	if err != nil {
		log.Fatal(err)
	}
}
