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

//GameServerJSON holds info for sending
type GameServerJSON struct {
	MaxPlayers    int      `json:"maxpl"`
	InGamePlayers int      `json:"players"`
	Name          string   `json:"name"`
	ID            int      `json:"id,string"`
	Password      string   `json:"pass,omitempty"`
	PlayersIn     []string `json:"players_name"`
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
	newGame := GameServer{MaxPlayers: maximumPlayers, Name: name, ID: MainServers.nextServerID, Password: pass, table: NewTable(MainServers.nextServerID)}
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
	AllUsers.mu.Lock()
	currentUser := AllUsers.Users[stringID]
	AllUsers.mu.Unlock()
	currentUser.mu.Lock()
	currentUser.client = ws
	currentUser.mu.Unlock()
	searchingGameUsers.mu.Lock()
	searchingGameUsers.Users[stringID] = currentUser
	searchingGameUsers.mu.Unlock()
	go recieveMessage(currentUser)
	go sendGamesList(ws, stringID, currentUser)
	go refreshServerList()

}

func sendGamesList(client1 *websocket.Conn, id string, currentUser *user) {
	for {
		select {
		case gameID := <-currentUser.redirectTo:
			w, err := currentUser.client.NextWriter(websocket.TextMessage)
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
				currentUser.client.WriteJSON(errorMsg)
				return
			}
			w.Write(data)
			w.Close()
		case <-currentUser.newMsg:
			// AllUsers.Users[id].mu.Lock()
			// defer AllUsers.Users[id].mu.Unlock()
			w, err := currentUser.client.NextWriter(websocket.TextMessage)
			if err != nil {
				fmt.Println(err)
				return
			}
			GamesList := []GameServerJSON{}
			MainServers.mu.Lock()
			for _, games := range MainServers.NonStartedGames {
				GamesList = append(GamesList, GameServerJSON{ID: games.ID, InGamePlayers: 3, MaxPlayers: games.MaxPlayers, Name: games.Name, Password: games.Password, PlayersIn: games.PlayersIn})

			}

			var data []byte
			data, err = json.Marshal(GamesList)
			MainServers.mu.Unlock()
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				currentUser.client.WriteJSON(errorMsg)
				return
			}
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

func recieveMessage(currentUser *user) {
	for {
		// AllUsers.Users[id].mu.Lock()
		// defer AllUsers.Users[id].mu.Unlock()
		_, p, err := currentUser.client.ReadMessage()
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
					currentUser.client.WriteJSON(errorMsg)
					continue
				}
				NewGameServer(players, params.Name, params.Password)
				currentUser.newMsg <- struct{}{}
			}
			join := GameJoinJSON{}
			err = json.Unmarshal(p, &join)
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				currentUser.client.WriteJSON(errorMsg)
				continue
			}
			if join != (GameJoinJSON{}) {
				gameID, err := strconv.Atoi(join.Name)
				if err != nil {
					errorMsg := struct{ stringErr string }{stringErr: err.Error()}
					currentUser.client.WriteJSON(errorMsg)
					continue
				}
				MainServers.mu.Lock()
				if MainServers.NonStartedGames[gameID] != nil {
					MainServers.mu.Unlock()
					err := MainServers.NonStartedGames[gameID].table.Join(currentUser.Name)
					if err != nil {
						continue
					}
					searchingGameUsers.mu.Lock()
					delete(searchingGameUsers.Users, currentUser.Name)
					searchingGameUsers.mu.Unlock()

					refreshServerList()
					currentUser.redirectTo <- gameID
					return
				}
				MainServers.mu.Unlock()
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
