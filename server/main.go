package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func main() {
	Servers := AllServers{}
	MainServers = &Servers
	MainServers.nonStartedGames = make(map[int]*GameServer)
	MainServers.startedGames = make(map[int]*GameServer)

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	r.HandleFunc("/", signInHandler)
	r.HandleFunc("/register", registerHandler)
	r.Handle("/game", checkCookieMidleWare(NewTable()))
	r.Handle("/wsgame", checkCookieMidleWare(http.HandlerFunc(ServeWS)))
	r.Handle("/main", checkCookieMidleWare(http.HandlerFunc(MainServers.ServeHTTP)))
	r.Handle("/wsmain", checkCookieMidleWare(http.HandlerFunc(MainServers.ServeHTTP)))
	log.Fatal(http.ListenAndServe(":8000", r))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
		if _, ok := AllUsers[cookie.Value]; !ok {
			cookie.Expires = time.Now().AddDate(0, 0, -1)
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, sessID, cookie.Value)
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
			if err = NewUser(name); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			expiration := time.Now().Add(10 * time.Hour)
			cookie := http.Cookie{
				Name:    "session_id",
				Value:   name,
				Expires: expiration,
			}
			http.SetCookie(w, &cookie)
			http.Redirect(w, r, "/main", 303)
		}
	}
}
