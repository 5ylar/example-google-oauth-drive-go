package main

import (
	"context"
	// "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:8080"),
		Handler: NewHandler(),
	}

	err := server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		customState := r.FormValue("custom-state")

		if strings.TrimSpace(customState) == "" {
			slog.Error("[callback] empty custom state")

			w.WriteHeader(422)
			w.Write([]byte("empty custom state"))

			return
		}

		oauthState := base64.URLEncoding.EncodeToString([]byte(customState))

		u := googleOauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

		http.Redirect(w, r, u, http.StatusTemporaryRedirect)
	})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {

		slog.Info("[callback] requested")

		state := r.FormValue("state")

		stateb, err := base64.URLEncoding.DecodeString(state)

		if err != nil {
			slog.Error("[callback] decoding state failed")

			w.WriteHeader(422)
			w.Write([]byte("decoding state failed"))

			return
		}

		slog.Info("[callback] state", slog.String("state value", string(stateb)))

		code := r.FormValue("code")

		if strings.TrimSpace("code") == "" {
			slog.Error("[callback] empty code")

			w.WriteHeader(422)
			w.Write([]byte("empty code"))

			return
		}

		token, err := googleOauthConfig.Exchange(context.Background(), code)

		if err != nil {
			slog.Error(
				"[callback] exchange token failed",
				slog.String("message", err.Error()),
			)

			w.WriteHeader(500)
			w.Write([]byte("exchange token failed"))

			return
		}

		err = saveToken(token)

		if err != nil {
			slog.Error(
				"[callback] saving token failed",
				slog.String("message", err.Error()),
			)

			w.WriteHeader(500)
			w.Write([]byte("saving token failed"))

			return
		}

		w.WriteHeader(200)
	})

	return mux
}

func generateStateOauthCookie(val string) string {
	state := base64.URLEncoding.EncodeToString([]byte(val))

	return state
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
