package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"se7en-ImproveServer/server"
	"strings"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

func main() {
	Servers := server.AllServers{}
	server.MainServers = &Servers
	server.MainServers.NonStartedGames = make(map[int]*server.GameServer)
	server.MainServers.StartedGames = make(map[int]*server.GameServer)

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	r.HandleFunc("/", signInHandler)
	r.HandleFunc("/register", registerHandler)
	r.Handle("/game", checkCookieMidleWare(http.HandlerFunc(server.MainServeHTTP)))
	r.Handle("/wsgame", checkCookieMidleWare(http.HandlerFunc(server.ServeWS)))
	r.Handle("/main", checkCookieMidleWare(http.HandlerFunc(server.MainServers.ServeHTTP)))
	r.Handle("/wsmain", checkCookieMidleWare(http.HandlerFunc(server.MainServers.ServeHTTP)))
	log.Fatal(http.ListenAndServe(":8000", r))
}

func signInHandler(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("html/authorization.html"))
	tpl.Execute(w, nil)
}

func checkCookieMidleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			fmt.Println("no auth at", r.URL.Path)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		if _, ok := server.AllUsers.Users[cookie.Value]; !ok {
			cookie.Expires = time.Now().AddDate(0, 0, -1)
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, server.SessID, cookie.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("html/register.html"))
	if r.Method == http.MethodGet {
		tpl.Execute(w, nil)
	}
	if r.Method == http.MethodPost {
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, "Cant read the body", http.StatusBadRequest)
			return
		}
		name := strings.TrimPrefix(string(body), "Name=")
		if name != "" {
			if err = server.NewUser(name); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			expiration := time.Now().Add(10 * time.Hour)
			cookie := http.Cookie{
				Name:    "session_id",
				Value:   name,
				Expires: expiration,
			}
			fmt.Println(name, "new user")
			http.SetCookie(w, &cookie)
			http.Redirect(w, r, "/main", 303)
		}
	}
}
