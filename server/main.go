package main

import (
	"encoding/json"
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

var GameTable = NewTable()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func sendNewMsgNotifications(client *websocket.Conn) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		w, err := client.NextWriter(websocket.TextMessage)
		if err != nil {
			ticker.Stop()
			break
		}

		msg := newMessage()
		w.Write(msg)
		w.Close()

		<-ticker.C
	}
}

func newMessage() []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"trump":  GameTable.trump,
		"cards":  GameTable.players[0].currentCards,
		"player": GameTable.onTable,
	})
	fmt.Println(string(data))
	return data
}

var tpl = template.Must(template.ParseFiles("html/authorization.html"))

func YourHandler(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, nil)

}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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
		if strings.TrimPrefix(string(body), "Name=") != "" {
			fmt.Println(string(body))
			GameTable.Join(strings.TrimPrefix(string(body), "Name="))
			http.Redirect(w, r, "/game", 303)
		}

	}
}

type ViewData struct {
	Name string
}

//var data ViewData

func GameHandler(w http.ResponseWriter, r *http.Request) {
	data := ViewData{Name: GameTable.players[GameTable.playersCount-1].id}
	tpl := template.Must(template.ParseFiles("html/game.html"))
	// if r.Method == http.MethodGet {
	tpl.Execute(w, data)
	// }
}
func main() {

	//GameTable.Join("Sten")
	// GameTable.Join("Mila")
	// GameTable.Join("Irma")
	// GameTable.Join("Helga")
	// GameTable.Join("John")
	// GameTable.Start()
	go func() {
		for {
			if GameTable.playersCount == 2 {
				GameTable.Start()
				break
			}

		}
	}()

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	//r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	r.HandleFunc("/game", GameHandler)
	r.HandleFunc("/", YourHandler)
	r.HandleFunc("/register", RegisterHandler)
	r.HandleFunc("/wsgame", wsGameHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}

func wsGameHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in hand")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	go readMessage(ws)
	go sendNewMsgNotifications(ws)

}

func readMessage(conn *websocket.Conn) {
	fmt.Println("INPUT MESSAGE")
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
			var data interface{}
			err := json.Unmarshal(p, &data)
			fmt.Println(string(p))
			if err != nil {
				panic(err)
			}
			fmt.Println(data)

		}
	}
}
