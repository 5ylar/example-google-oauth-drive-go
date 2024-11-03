package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  os.Getenv("REDIRECT_URL"),
	ClientID:     os.Getenv("CLIENT_ID"),
	ClientSecret: os.Getenv("CLIENT_SECRET"),
	Scopes:       strings.Split(os.Getenv("SCOPES"), ","),
	Endpoint:     google.Endpoint,
}

var tokenFile = os.Getenv("TOKEN_FILE")

func main() {

	token, err := tokenFromFile(tokenFile)

	if err != nil {
		panic(err)
	}

	client := googleOauthConfig.Client(context.Background(), token)

	ctx := context.Background()

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		panic(err)
	}

	// ------- upload ------

	file, err := os.Open("a_10mb.csv")

	if err != nil {
		panic(err)
	}

	info, err := file.Stat()

	if err != nil {
		panic(err)
	}

	defer file.Close()

	targetFolder := "1LxEjCsKJV9eQNiDbkRSrWVRlIUn5kIP_"

	// file metadata
	f := &drive.File{
		Name:    info.Name(),
		Parents: []string{targetFolder},
	}

	_, err = srv.Files.
		Create(f).
		Context(ctx).
		Media(file).
		ProgressUpdater(func(now, size int64) {
			fmt.Printf("%d, %d\r", now, size)
		}).
		Do()

	if err != nil {
		panic(err)
	}

	// ------- list ------

	r, err := srv.Files.
		List().
		Context(ctx).
		Q(fmt.Sprintf("'%s' in parents", targetFolder)).
		PageSize(100).
		Fields("nextPageToken, files(id, name)").
		Do()

	if err != nil {
		panic(err)
	}

	for _, i := range r.Files {
		fmt.Printf("%s (%s)\n", i.Name, i.Id)
	}
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	var token oauth2.Token

	err = json.NewDecoder(f).Decode(&token)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

func saveToken(token *oauth2.Token) error {
	f, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return err
	}

	defer f.Close()

	err = json.NewEncoder(f).Encode(token)

	if err != nil {
		return err
	}

	return nil
}
