package main

import (
	"net/http"
	"se7en-ImproveServer/server"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

func TestMain(t *testing.T) {
	Servers := server.AllServers{}
	server.MainServers = &Servers
	server.MainServers.NonStartedGames = make(map[int]*server.GameServer)
	server.MainServers.StartedGames = make(map[int]*server.GameServer)

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	r.HandleFunc("/", signInHandler)
	r.HandleFunc("/register", registerHandler)
	r.Handle("/game", checkCookieMidleWare(server.NewTable()))
	r.Handle("/wsgame", checkCookieMidleWare(http.HandlerFunc(server.ServeWS)))
	r.Handle("/main", checkCookieMidleWare(http.HandlerFunc(server.MainServers.ServeHTTP)))
	r.Handle("/wsmain", checkCookieMidleWare(http.HandlerFunc(server.MainServers.ServeHTTP)))
	log.Fatal(http.ListenAndServe(":8000", r))
}
