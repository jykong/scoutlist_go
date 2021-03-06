package scoutlist

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"net/http"
	"os"

	"github.com/pkg/browser"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const savedAuthTokenFile = "./auth_token.gob"
const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadPrivate)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

// Auth authentication
func Auth() *spotify.Client {
	_, err := os.Stat(savedAuthTokenFile)
	if os.IsNotExist(err) {
		return authFromBrowser()
	}
	return authFromFile()
}

// AuthFromS3 Authenticate from S3
func AuthFromS3(sess *session.Session) *spotify.Client {
	authTokenFilepath := os.Getenv("auth_token_filepath")
	if authTokenFilepath == "" {
		log.Fatal("Could not find auth_token_filepath environment variable.")
	}
	tok := s3loadTokenFromGob(sess, authTokenFilepath)
	client := new(spotify.Client)
	*client = auth.NewClient(tok)
	return client
}

// LoadTokenFromGob ...
func s3loadTokenFromGob(sess *session.Session, filePath string) *oauth2.Token {
	bucket := os.Getenv("bucket")
	if bucket == "" {
		log.Fatal("Could not find bucket environment variable.")
	}

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
	var tok = new(oauth2.Token)
	decoder := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
	err = decoder.Decode(&tok)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Downloaded %s\n%d bytes\n", filePath, numBytes)
	return tok
}

func authFromBrowser() *spotify.Client {
	log.Println("Opening http server...")
	srv := &http.Server{}
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	log.Println("Opening auth URL via browser")
	url := auth.AuthURL(state)
	browser.OpenURL(url)

	// wait for auth to complete
	client := <-ch
	log.Println("Auth complete.")

	log.Println("...Closing http server")
	err := srv.Shutdown(nil)
	if err != nil {
		log.Fatal("http server shutdown error:", err)
	}

	return client
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := new(spotify.Client)
	*client = auth.NewClient(tok)
	saveTokenToFile(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}

func saveTokenToFile(t *oauth2.Token) {
	os.Remove(savedAuthTokenFile)

	file, err := os.Create(savedAuthTokenFile)
	if err != nil {
		log.Println(err)
		return
	}

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(*t)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Auth token saved to", savedAuthTokenFile)
}

func authFromFile() *spotify.Client {
	log.Println("Loading auth from from file...")
	var tok oauth2.Token
	file, err := os.Open(savedAuthTokenFile)
	if err != nil {
		log.Fatal(err)
	}

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&tok)

	client := new(spotify.Client)
	*client = auth.NewClient(&tok)
	log.Println("Auth complete.")
	return client
}
