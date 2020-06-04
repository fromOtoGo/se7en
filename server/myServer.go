package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//GameServer holds information about game
type GameServer struct {
	mu            sync.Mutex
	MaxPlayers    int      `json:"maxpl"`
	InGamePlayers *int     `json:"players"`
	Name          string   `json:"name"`
	ID            int      `json:"id,string"`
	Password      string   `json:"pass,omitempty"`
	PlayersIn     []string `json:"players_name"`
	table         *Table
}

//AllServers contains all running games adn tables in this moment
type AllServers struct {
	mu              sync.Mutex
	nextServerID    int
	idInMain        map[int]bool
	idInWhichTable  map[int]int
	NonStartedGames map[int]*GameServer
	StartedGames    map[int]*GameServer
}

//MainServers holds all running servers
var MainServers *AllServers

//NewGameServer creates new gameServer and add it to MainServer
func NewGameServer(maximumPlayers int, name string, pass string) *GameServer {
	if maximumPlayers == 0 {
		maximumPlayers = 6
	}
	newGame := GameServer{MaxPlayers: maximumPlayers, Name: name, ID: MainServers.nextServerID, Password: pass, table: NewTable()}
	MainServers.mu.Lock()
	defer MainServers.mu.Unlock()

	MainServers.NonStartedGames[MainServers.nextServerID] = &newGame
	MainServers.NonStartedGames[MainServers.nextServerID].mu.Lock()
	MainServers.NonStartedGames[MainServers.nextServerID].InGamePlayers = &MainServers.NonStartedGames[MainServers.nextServerID].table.playersCount
	MainServers.NonStartedGames[MainServers.nextServerID].mu.Unlock()
	MainServers.nextServerID++

	go refreshServerList()
	return &newGame
}

type key int

//SessID is key for context storage
const SessID key = 1

func (as *AllServers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := r.Context()
	stringID, ok := ctx.Value(SessID).(string)
	if !ok {
		return
	}
	searchingGameUsers.mu.Lock()
	searchingGameUsers.Users[stringID] = AllUsers.Users[stringID]
	searchingGameUsers.mu.Unlock()
	searchingGameUsers.Users[stringID].mu.Lock()
	searchingGameUsers.Users[stringID].client = ws
	searchingGameUsers.Users[stringID].mu.Unlock()
	go recieveMessage(ws, stringID)
	go sendGamesList(ws, stringID, AllUsers.Users[stringID].newMsg, AllUsers.Users[stringID].redirectTo)
	go refreshServerList()

}

func sendGamesList(client1 *websocket.Conn, id string, newMsg <-chan struct{}, redirect <-chan int) {
	for {
		select {
		case gameID := <-redirect:
			w, err := AllUsers.Users[id].client.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			rawID := struct {
				ConnectTo int `json:"game_id,string"`
			}{ConnectTo: gameID}
			var data []byte
			data, err = json.Marshal(rawID)
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				AllUsers.Users[id].client.WriteJSON(errorMsg)
				return
			}
			w.Write(data)
			w.Close()
		case <-newMsg:
			// AllUsers.Users[id].mu.Lock()
			// defer AllUsers.Users[id].mu.Unlock()
			w, err := AllUsers.Users[id].client.NextWriter(websocket.TextMessage)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("SENDING GAMES")

			GamesList := []*GameServer{}
			for _, games := range MainServers.NonStartedGames {
				GamesList = append(GamesList, games)
			}
			var data []byte
			data, err = json.Marshal(GamesList)
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				AllUsers.Users[id].client.WriteJSON(errorMsg)
				return
			}
			fmt.Println(string(data))
			w.Write(data)
			w.Close()
		}
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

func recieveMessage(conn1 *websocket.Conn, id string) {
	for {
		// AllUsers.Users[id].mu.Lock()
		// defer AllUsers.Users[id].mu.Unlock()
		_, p, err := AllUsers.Users[id].client.ReadMessage()
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
				// errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				// conn.WriteJSON(errorMsg)
				continue
			}
			if params != (gameJSON{}) {
				players, err := strconv.Atoi(params.MaxPlayers)
				if err != nil {
					errorMsg := struct{ stringErr string }{stringErr: err.Error()}
					AllUsers.Users[id].client.WriteJSON(errorMsg)
					continue
				}
				NewGameServer(players, params.Name, params.Password)
				AllUsers.Users[id].newMsg <- struct{}{}
			}
			join := GameJoinJSON{}
			err = json.Unmarshal(p, &join)
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				AllUsers.Users[id].client.WriteJSON(errorMsg)
				continue
			}
			if join != (GameJoinJSON{}) {
				gameID, err := strconv.Atoi(join.Name)
				if err != nil {
					errorMsg := struct{ stringErr string }{stringErr: err.Error()}
					AllUsers.Users[id].client.WriteJSON(errorMsg)
					continue
				}
				if MainServers.NonStartedGames[gameID] != nil {
					MainServers.NonStartedGames[gameID].table.Join(id)
					searchingGameUsers.mu.Lock()
					delete(searchingGameUsers.Users, id)
					searchingGameUsers.mu.Unlock()

					refreshServerList()
					AllUsers.Users[id].redirectTo <- gameID
					return
				}
			}
		}
	}

}

func sendRedirect(client *websocket.Conn, gameID int) {
	// AllUsers.Users[id].mu.Lock()
	// defer AllUsers.Users[id].mu.Unlock()
	w, err := client.NextWriter(websocket.TextMessage)
	if err != nil {
		return
	}
	rawID := struct {
		ConnectTo int `json:"game_id,string"`
	}{ConnectTo: gameID}
	var data []byte
	data, err = json.Marshal(rawID)
	if err != nil {
		errorMsg := struct{ stringErr string }{stringErr: err.Error()}
		client.WriteJSON(errorMsg)
		return
	}
	w.Write(data)
	w.Close()
}

func refreshServerList() {
	// for i := range MainServers.NonStartedGames {
	// 	if *MainServers.NonStartedGames[i].InGamePlayers == MainServers.NonStartedGames[i].MaxPlayers {
	// 		MainServers.mu.Lock()
	// 		MainServers.StartedGames[i] = MainServers.NonStartedGames[i]
	// 		MainServers.NonStartedGames[i] = nil
	// 		MainServers.mu.Unlock()
	// 	}
	// }
	searchingGameUsers.mu.Lock()
	defer searchingGameUsers.mu.Unlock()
	for i := range searchingGameUsers.Users {
		searchingGameUsers.Users[i].newMsg <- struct{}{}
	}
}
