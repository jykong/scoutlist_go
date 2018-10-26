package model

import "github.com/zmb3/spotify"

type PlaylistEntry struct {
	ID   spotify.ID `json:"id"`
	Name string     `json:"name"`
}
type TitleArtists struct {
	Title   string
	Artists []string
}
type TrackIDTA struct {
	ID spotify.ID
	TA TitleArtists
}
