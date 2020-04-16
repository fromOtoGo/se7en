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

// var GameTable = NewTable()
var allTables = make([]*Table, 0, 5)

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
		"trump":  allTables[0].trump,
		"cards":  allTables[0].players[num].currentCards,
		"player": allTables[0].onTable,
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
		cookie, err = r.Cookie("table")
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
			allTables[0].Join(name)

			expiration := time.Now().Add(10 * time.Hour)
			cookie := http.Cookie{
				Name:    "session_id",
				Value:   name,
				Expires: expiration,
			}
			http.SetCookie(w, &cookie)

			cookie = http.Cookie{
				Name:    "table",
				Value:   "0",
				Expires: expiration,
			}
			http.SetCookie(w, &cookie)

			http.Redirect(w, r, "/game", 303)
		}

	}
}

type viewData struct {
	Name string
}

//var data ViewData

func gameHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Cookie("session_id")

	playerID, err := strconv.Atoi(id.Value)
	if err != nil {
		panic(err)
	}
	data := viewData{Name: allTables[0].players[playerID].id}
	tpl := template.Must(template.ParseFiles("html/game.html"))
	// if r.Method == http.MethodGet {
	tpl.Execute(w, data)
	// }
}
func main() {

	allTables = append(allTables, NewTable())
	go func() {
		for {
			for i := range allTables {
				if allTables[i].playersCount == 2 {
					allTables[i].Start()
					allTables = append(allTables, NewTable())
				}
			}
		}
	}()

	rc := mux.NewRouter()
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	rc.HandleFunc("/game", gameHandler)
	withCheckHandler := checkCookieMidleWare(rc)

	r.HandleFunc("/", yourHandler)
	r.HandleFunc("/register", registerHandler)
	r.HandleFunc("/wsgame", wsGameHandler)
	r.Handle("/game", withCheckHandler)

	log.Fatal(http.ListenAndServe(":8000", r))
}

func wsGameHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in hand")

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	id, _ := r.Cookie("session_id")
	table, _ := r.Cookie("table")
	playerID, err := strconv.Atoi(id.Value)
	if err != nil {
		panic(err)
	}
	playerTable, err := strconv.Atoi(table.Value)
	if err != nil {
		panic(err)
	}
	go readMessage(ws, playerTable, playerID)
	go sendNewMsgNotifications(ws, playerID)

}

func readMessage(conn *websocket.Conn, table int, num int) {
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

			if data["bet"] != nil {
				if allTables[table].players[num].betFlag == true {
					allTables[table].players[num].betFlag = false
					betStr := data["bet"].(string)
					bet, err := strconv.Atoi(betStr)
					if err != nil {
						panic("w8ing int")
					}
					allTables[table].inputCh <- bet
				}
			}
			if data["card_number"] != nil {
				if allTables[table].players[num].turnFlag == true {
					allTables[table].players[num].turnFlag = false
					turnFloat := data["card_number"].(float64)
					card := int(turnFloat)
					allTables[table].inputCh <- card
				}
			}
			if data["bet"] != nil {
				if allTables[table].players[num].jokerFlag == true {
					allTables[table].players[num].jokerFlag = false
					betStr := data["bet"].(string)
					bet, err := strconv.Atoi(betStr)
					if err != nil {
						panic("w8ing int")
					}
					allTables[table].inputCh <- bet
				}
			}
		}

	}

}
