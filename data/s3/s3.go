package s3data

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"

	"github.com/zmb3/spotify"
	"scoutlist/data/model"
)

// Test Modes
const (
	codeTest int = 0
	userTest int = 1
)

var mode = userTest

// StringConsts pre-defined filepaths
type StringConsts struct {
	PlaylistsPath     string
	ExcPlaylistsPath  string
	ExcTracksPath     string
	IncPlaylistPath   string
	ScoutlistIDPath   string
	ScoutedlistIDPath string
	ScoutlistName     string
	ScoutedlistName   string
}

type playlistsStruct struct {
	Playlists []model.PlaylistEntry
}

type tracksStruct struct {
	Tracks []model.TrackIDTA
}

// ModeIsCodeTest Check if mode is set to code test
func ModeIsCodeTest() bool {
	return mode == codeTest
}

// GetStringConsts Get pre-defined filepaths
func GetStringConsts() StringConsts {
	var strCon StringConsts
	var pathPrefix string
	var scoutlistPrefix string
	switch mode {
	case codeTest:
		pathPrefix = "code_test_data/"
		scoutlistPrefix = "Test"
	case userTest:
		pathPrefix = "user_test_data/"
		scoutlistPrefix = ""
	default:
		pathPrefix = ""
		scoutlistPrefix = ""
	}
	strCon.PlaylistsPath = pathPrefix + "playlists.json"
	strCon.ExcPlaylistsPath = pathPrefix + "exc_playlists.json"
	strCon.ExcTracksPath = pathPrefix + "exc_tracks.gob"
	strCon.IncPlaylistPath = pathPrefix + "inc_playlists.json"
	strCon.ScoutlistIDPath = pathPrefix + "scoutlist_id.gob"
	strCon.ScoutedlistIDPath = pathPrefix + "scoutedlist_id.gob"
	strCon.ScoutlistName = scoutlistPrefix + "Scoutlist"
	strCon.ScoutedlistName = scoutlistPrefix + "Scoutedlist"
	return strCon
}

const bucket string = "jykong-webtest"

// StartS3Session Start AWS S3 session to us-east-1
func StartS3Session() *session.Session {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	return sess
}

// SavePlaylistsToJSON ...
func SavePlaylistsToJSON(sess *session.Session, filePath string,
	playlists []model.PlaylistEntry) {

	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "\t")
	plStruct := playlistsStruct{
		playlists,
	}
	err := encoder.Encode(plStruct)
	if err != nil {
		log.Fatal(err)
	}

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
		Body:   buf,
	})
	if err != nil {
		log.Fatalf("Unable to upload %q to %q, %v", filePath, bucket, err)
	}
	log.Printf("Successfully uploaded %q to %q\n", filePath, bucket)
}

// LoadPlaylistsFromJSON ...
func LoadPlaylistsFromJSON(sess *session.Session, filePath string) []model.PlaylistEntry {
	downloader := s3manager.NewDownloader(sess)

	buf := aws.NewWriteAtBuffer([]byte{})

	numBytes, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(filePath),
		})
	if err != nil {
		log.Fatalf("Unable to download item %q, %v", filePath, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	plStruct := playlistsStruct{}
	err = decoder.Decode(&plStruct)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Downloaded %s\n%d bytes\n", filePath, numBytes)
	return plStruct.Playlists
}

// SaveTracksToGob ...
func SaveTracksToGob(sess *session.Session, filePath string, tracks []model.TrackIDTA) {
	buf := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buf)
	trStruct := tracksStruct{
		tracks,
	}
	err := encoder.Encode(trStruct)
	if err != nil {
		log.Fatal(err)
	}

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
		Body:   buf,
	})
	if err != nil {
		log.Fatalf("Unable to upload %q to %q, %v", filePath, bucket, err)
	}
	log.Printf("Successfully uploaded %q to %q\n", filePath, bucket)
}

// LoadTracksFromGob ...
func LoadTracksFromGob(sess *session.Session, filePath string) []model.TrackIDTA {
	downloader := s3manager.NewDownloader(sess)

	buf := aws.NewWriteAtBuffer([]byte{})

	numBytes, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(filePath),
		})
	if err != nil {
		log.Fatalf("Unable to download item %q, %v", filePath, err)
	}

	trStruct := tracksStruct{}
	decoder := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
	err = decoder.Decode(&trStruct)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Downloaded %s\n%d bytes\n", filePath, numBytes)
	return trStruct.Tracks
}

// SaveIDToGob ...
func SaveIDToGob(sess *session.Session, filePath string, spid *spotify.ID) {
	buf := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(*spid)
	if err != nil {
		log.Fatal(err)
	}

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
		Body:   buf,
	})
	if err != nil {
		log.Fatalf("Unable to upload %q to %q, %v", filePath, bucket, err)
	}
	log.Printf("Successfully uploaded %q to %q\n", filePath, bucket)
}

// LoadIDFromGob ...
func LoadIDFromGob(sess *session.Session, filePath string) spotify.ID {
	downloader := s3manager.NewDownloader(sess)

	buf := aws.NewWriteAtBuffer([]byte{})

	numBytes, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(filePath),
		})
	if err != nil {
		log.Fatalf("Unable to download item %q, %v", filePath, err)
	}
	var spid spotify.ID
	decoder := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
	err = decoder.Decode(&spid)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Downloaded %s\n%d bytes\n", filePath, numBytes)
	return spid
}
