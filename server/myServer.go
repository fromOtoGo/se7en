package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//GameServer holds information about game
type GameServer struct {
	MaxPlayers    int    `json:"maxpl"`
	InGamePlayers int    `json:"players"`
	Name          string `json:"name"`
	Password      string `json:"pass,omitempty"`
	table         *Table
}

//AllServers contains all running games adn tables in this moment
type AllServers struct {
	nextServerID    int
	idInMain        map[int]bool
	idInWhichTable  map[int]int
	nonStartedGames map[int]*GameServer
	startedGames    map[int]*GameServer
	mu              sync.RWMutex
}

//MainServers holds all running servers
var MainServers *AllServers

// MainServers.nonStartedGames = make(map[int]*GameServer)

//NewGameServer creates new gameServer and add it to MainServer
func NewGameServer(maximumPlayers int, name string, pass string) *GameServer {
	if maximumPlayers == 0 {
		maximumPlayers = 6
	}
	newGame := GameServer{MaxPlayers: maximumPlayers, Name: name, Password: pass, table: NewTable()}
	MainServers.mu.Lock()
	MainServers.nonStartedGames[MainServers.nextServerID] = &newGame
	MainServers.nextServerID++
	MainServers.mu.Unlock()
	return &newGame
}

type key int

const sessID key = 1

func (ms *AllServers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rout := mux.NewRouter()
	rout.HandleFunc("/main", handlerMain)
	rout.HandleFunc("/wsmain", handlerWSMain)
	rout.ServeHTTP(w, r)
}

func handlerMain(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("html/main.html"))
	tpl.Execute(w, nil)
}

func handlerWSMain(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in ws main handler")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("WS CONNECTED")
	ctx := r.Context()
	stringID, ok := ctx.Value(sessID).(string)
	if !ok {
		return
	}
	id, err := strconv.Atoi(stringID)
	if err != nil {
		panic(err)
	}
	go recieveMessage(ws, id)
	go sendGamesList(ws, id, AllPlayers[id].newMsg)
	AllPlayers[id].newMsg <- struct{}{}

}

func sendGamesList(client *websocket.Conn, id int, newMsg <-chan struct{}) {
	for {
		<-newMsg
		w, err := client.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		GamesList := []GameServer{}
		for _, games := range MainServers.nonStartedGames {
			GamesList = append(GamesList, *games)
		}
		var data []byte
		data, err = json.Marshal(GamesList)
		if err != nil {
			panic(err)
		}
		w.Write(data)
		w.Close()
	}
}

type gameJSON struct {
	Name       string `json:"name"`
	Password   string `json:"pass"`
	MaxPlayers string `json:"max_players"`
}

//GameJoinJSON ....
type GameJoinJSON struct {
	Name string `json:"join"`
}

func recieveMessage(conn *websocket.Conn, id int) {
	for {

		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// if err := conn.WriteMessage(messageType, p); err != nil {
		// 	log.Println(err)
		// 	return
		// }
		if p != nil {
			params := gameJSON{}
			err := json.Unmarshal(p, &params)
			if err != nil {
				panic(err)
			}
			fmt.Println(params)
			players, err := strconv.Atoi(params.MaxPlayers)
			if err != nil {
				panic(err)
			}
			NewGameServer(players, params.Name, params.Password)
			AllPlayers[id].newMsg <- struct{}{}
		}

	}

}
