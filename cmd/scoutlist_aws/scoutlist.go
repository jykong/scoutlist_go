package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"

	"github.com/zmb3/spotify"
	"scoutlist"
	"scoutlist/data/s3"
)

var (
	// ErrNameNotProvided is thrown when a name is not provided
	ErrNameNotProvided = errors.New("no name was provided in the HTTP body")
)

func main() {
	lambda.Start(Handler)
}

// Handler is your Lambda function handler
// It uses Amazon API Gateway request/responses provided by the aws-lambda-go/events package,
// However you could use other event sources (S3, Kinesis etc), or JSON-decoded primitive types such as 'string'.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Processing Lambda request %s\n", request.RequestContext.RequestID)

	sess := s3data.StartS3Session()

	var cu scoutlist.ClientUser
	authAndSetID(sess, &cu)

	var opt scoutlist.Options
	opt.LastN = 15
	opt.OutN = 15

	updateScoutlist(sess, &cu, &opt)
	//getAndSavePlaylists(sess, &cu)

	/*
		// If no name is provided in the HTTP request body, throw an error
		if len(request.Body) < 1 {
			return events.APIGatewayProxyResponse{}, ErrNameNotProvided
		}
	*/

	return events.APIGatewayProxyResponse{
		Body:       "Scoutlist Recycled",
		StatusCode: 200,
	}, nil
}

func authAndSetID(sess *session.Session, cu *scoutlist.ClientUser) {
	cu.Client = scoutlist.AuthFromS3(sess)
	user, err := cu.Client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
	cu.UserID = user.ID
}

func getAndSavePlaylists(sess *session.Session, cu *scoutlist.ClientUser) {
	cu.Client.AutoRetry = true

	playlists := scoutlist.GetPlaylists(cu)
	s3data.SavePlaylistsToJSON(sess, s3data.GetStringConsts().PlaylistsPath, playlists)
}

func updateScoutlist(sess *session.Session, cu *scoutlist.ClientUser, opt *scoutlist.Options) {
	cu.Client.AutoRetry = true
	strCon := s3data.GetStringConsts()

	//excPlaylists := s3data.LoadPlaylistsFromJSON(sess, strCon.ExcPlaylistsPath)
	//fmt.Println(excPlaylists)

	scoutlist.StartRateLimiter()

	//excTracks := scoutlist.GetUniqueTracksFromPlaylists(cu, excPlaylists, nil, 0)
	//s3data.SaveTracksToGob(sess, strCon.ExcTracksPath, excTracks)
	excTracks := s3data.LoadTracksFromGob(sess, strCon.ExcTracksPath)
	fmt.Println(len(excTracks))
	//fmt.Println(excTracks)

	incPlaylists := s3data.LoadPlaylistsFromJSON(sess, strCon.IncPlaylistPath)

	filteredTracks := scoutlist.GetUniqueTracksFromPlaylists(cu, incPlaylists, excTracks, opt.LastN)
	fmt.Println(len(filteredTracks))
	//fmt.Println(filteredTracks)

	scoutlist.StopRateLimiter()

	scoutlistID := loadIDOrCreatePlaylist(sess, cu, strCon.ScoutlistIDPath, strCon.ScoutlistName)
	scoutedlistID := loadIDOrCreatePlaylist(sess, cu, strCon.ScoutedlistIDPath, strCon.ScoutedlistName)
	log.Println(scoutlistID)
	log.Println(scoutedlistID)

	excTracks = scoutlist.RecycleScoutlist(cu, scoutlistID, scoutedlistID, excTracks)

	s3data.SaveTracksToGob(sess, strCon.ExcTracksPath, excTracks)

	trackIDs := scoutlist.GetNTrackIDsFromTrackIDTASlice(filteredTracks, opt.OutN, true)
	//fmt.Println(trackIDs)
	scoutlist.ReplacePlaylistTracks(cu, scoutlistID, trackIDs)
}

func loadIDOrCreatePlaylist(sess *session.Session, cu *scoutlist.ClientUser,
	filePath string, name string) spotify.ID {
	plid := s3data.LoadIDFromGob(sess, filePath)
	if plid == "" {
		plid = scoutlist.CheckAndCreatePlaylist(cu, plid, name)
		s3data.SaveIDToGob(sess, filePath, &plid)
	}
	return plid
}
