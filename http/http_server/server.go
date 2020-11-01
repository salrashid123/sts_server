package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
)

var ()

const (
	allowedToken = "iamthewalrus"
)

func gethandler(w http.ResponseWriter, r *http.Request) {

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf(string(requestDump))

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	splitToken := strings.Split(authHeader, "Bearer")
	if len(splitToken) == 2 {
		reqToken := strings.TrimSpace(splitToken[1])
		if reqToken != allowedToken {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	} else {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	fmt.Fprint(w, fmt.Sprintf("ok"))
}

func main() {

	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path("/").HandlerFunc(gethandler)
	var server *http.Server
	server = &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	http2.ConfigureServer(server, &http2.Server{})
	fmt.Println("Starting Server..")
	err := server.ListenAndServe()
	fmt.Printf("Unable to start Server %v", err)

}
