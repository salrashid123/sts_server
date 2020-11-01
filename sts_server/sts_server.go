package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"net/http"

	"log"
	"net/http/httputil"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
)

type contextKey string

const contextEventKey contextKey = "event"

// https://www.rfc-editor.org/rfc/rfc8693.html#section-2.2.1
type TokenResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type,omitempty"`
	ExpiresIn       int64  `json:"expires_in,omitempty"`
	Scope           string `json:"scope,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
}

// support standard TokenTypes
const (
	AccessToken  string = "urn:ietf:params:oauth:token-type:access_token"
	RefreshToken string = "urn:ietf:params:oauth:token-type:refresh_token"
	IDToken      string = "urn:ietf:params:oauth:token-type:id_token"
	SAML1        string = "urn:ietf:params:oauth:token-type:saml1"
	SAML2        string = "urn:ietf:params:oauth:token-type:saml2"
	JWT          string = "urn:ietf:params:oauth:token-type:jwt"
)

const (
	inboundPassphrase  = "iamtheeggman"
	outboundPassphrase = "iamthewalrus"
)

var (
	httpport   = flag.String("httpport", ":8080", "httpport")
	tsAudience = flag.String("tsAudience", "https://foo.bar", "Audience value for the TokenService")
	// support standard TokenTypes
	tokenTypes = []string{AccessToken, RefreshToken, IDToken, SAML1, SAML2, JWT}
)

type stsRequest struct {
	GrantType        string `json:"grant_type"`
	Resource         string `json:"resource,omitempty"`
	Audience         string `json:"audience,omitempty"`
	Scope            string `json:"scope,omitempty"`
	RequestTokenType string `json:"requested_token_type,omitempty"`
	SubjectToken     string `json:"subject_token"`
	SubjectTokenType string `json:"subject_token_type"`
	ActorToken       string `json:"actor_token,omitempty"`
	ActorTokenType   string `json:"actor_token_type,omitempty"`
}

func isValidTokenType(str string) bool {
	for _, a := range tokenTypes {
		if a == str {
			return true
		}
	}
	return false
}

func eventsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			fmt.Printf("Error Reading Request: %v", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		fmt.Printf(fmt.Sprintf("Request Dump: %s\n", string(requestDump)))
		event := &stsRequest{}

		contentType := r.Header.Get("Content-type")

		switch {
		case contentType == "application/json":
			err := json.NewDecoder(r.Body).Decode(event)
			if err != nil {
				fmt.Printf("Could Not parse application/json payload: %v", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		case contentType == "application/x-www-form-urlencoded":
			err := r.ParseForm()
			if err != nil {
				fmt.Printf("Could not parse application/x-www-form-urlencode Form: %v", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			v := r.Form

			event = &stsRequest{
				GrantType:        v.Get("grant_type"),
				Resource:         v.Get("resource"),
				Audience:         v.Get("audience"),
				Scope:            v.Get("scope"),
				SubjectToken:     v.Get("subject_token"),
				SubjectTokenType: v.Get("subject_token_type"),
				ActorToken:       v.Get("actor_token"),
				ActorTokenType:   v.Get("actor_token_type"),
			}
		default:
			fmt.Printf("Invalid Content Type [%s]", contentType)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextEventKey, *event)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func verifyAuthToken(ctx context.Context, rawToken string) bool {
	return true
}

func tokenhandlerpost(w http.ResponseWriter, r *http.Request) {

	val := r.Context().Value(contextKey("event")).(stsRequest)
	fmt.Printf("  %v\n", val)

	if val.GrantType == "" || val.SubjectToken == "" || val.SubjectTokenType == "" {

		fmt.Printf("Invalid Request Payload Headers: \n %v\n", val)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidTokenType(val.SubjectTokenType) {
		fmt.Printf("Invalid subject_token_type: %s", val.SubjectTokenType)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if val.ActorTokenType != "" && !isValidTokenType(val.ActorTokenType) {
		log.Printf("Invalid actor_token_type: %s", val.ActorTokenType)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if val.SubjectToken != inboundPassphrase {
		log.Printf("Provided subjectToken is invalid: Got: [%s]  Want [%s]", val.SubjectToken, inboundPassphrase)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	p := &TokenResponse{
		AccessToken:     outboundPassphrase,
		IssuedTokenType: AccessToken,
		TokenType:       "Bearer",
		ExpiresIn:       int64(60),
	}
	fmt.Printf("Response Data: %v", p)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store")

	err := json.NewEncoder(w).Encode(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Could not marshall JSON to output %v", err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
}

func main() {
	flag.Parse()

	router := mux.NewRouter()
	router.Path("/token").Methods(http.MethodPost).HandlerFunc(tokenhandlerpost)
	var server *http.Server
	server = &http.Server{
		Addr:    *httpport,
		Handler: eventsMiddleware(router),
	}
	http2.ConfigureServer(server, &http2.Server{})
	fmt.Println("Starting Server..")
	err := server.ListenAndServe()
	fmt.Printf("Unable to start Server %v", err)

}
