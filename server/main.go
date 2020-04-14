package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var GameTable = NewTable()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func sendNewMsgNotifications(client *websocket.Conn, num int) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		w, err := client.NextWriter(websocket.TextMessage)
		if err != nil {
			ticker.Stop()
			break
		}

		msg := newMessage(num)
		w.Write(msg)
		w.Close()

		<-ticker.C
	}
}

func newMessage(num int) []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"trump":  GameTable.trump,
		"cards":  GameTable.players[num].currentCards,
		"player": GameTable.onTable,
	})
	return data
}

var tpl = template.Must(template.ParseFiles("html/authorization.html"))

func yourHandler(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, nil)

}

func checkCookieMidleWare(next http.Handler) http.Handler {
	fmt.Println("check cookie")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		fmt.Println(cookie)
		if err != nil {
			fmt.Println("no auth at", r.URL.Path)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
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
			panic(err)
		}
		name := strings.TrimPrefix(string(body), "Name=")
		if name != "" {
			fmt.Println(string(body))
			GameTable.Join(name)

			expiration := time.Now().Add(10 * time.Hour)
			cookie := http.Cookie{
				Name:    "session_id",
				Value:   name,
				Expires: expiration,
			}
			http.SetCookie(w, &cookie)
			http.Redirect(w, r, "/game", 303)
		}

	}
}

type ViewData struct {
	Name string
}

//var data ViewData

func gameHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Cookie("session_id")
	playerID, err := strconv.Atoi(id.Value)
	if err != nil {
		panic(err)
	}
	data := ViewData{Name: GameTable.players[playerID].id}
	tpl := template.Must(template.ParseFiles("html/game.html"))
	// if r.Method == http.MethodGet {
	tpl.Execute(w, data)
	// }
}
func main() {

	go func() {
		for {
			if GameTable.playersCount == 2 {
				GameTable.Start()
				break
			}

		}
	}()

	rc := mux.NewRouter()
	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	//r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	rc.HandleFunc("/game", gameHandler)
	withCheckHandler := checkCookieMidleWare(rc)

	r.HandleFunc("/", yourHandler)
	r.HandleFunc("/register", registerHandler)
	r.HandleFunc("/wsgame", wsGameHandler)
	r.Handle("/game", withCheckHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}

func wsGameHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in hand")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	id, _ := r.Cookie("session_id")
	playerID, err := strconv.Atoi(id.Value)
	if err != nil {
		panic(err)
	}
	go readMessage(ws, GameTable.playersCount)
	go sendNewMsgNotifications(ws, playerID)

}

func readMessage(conn *websocket.Conn, num int) {
	for {

		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}
		if p != nil {
			var data map[string]interface{}
			err := json.Unmarshal(p, &data)
			if err != nil {
				panic(err)
			}
			fmt.Println(data)
		}

	}

}
