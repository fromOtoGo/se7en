package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"se7en-ImproveServer/bots/game"
	"se7en-ImproveServer/server"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

//Bot contains parameters of bot
type Bot struct {
	name    string
	Cookies string
}

func main() {
	var wg sync.WaitGroup
	num := 3
	wg.Add(num)
	for i := 0; i < num; i++ {
		go func() {
			MyBot := NewBot()
			Cook, err := MyBot.httpRegister()
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Second)
			MyBot.Cookies = Cook
			MyBot.wsMain()
			wg.Done()
		}()
		time.Sleep(time.Second * 2)
	}
	wg.Wait()
}

//NewBot creates new bot
func NewBot() *Bot {
	n := "Bot" + strconv.Itoa(botCount)
	fmt.Println(n)
	bot := Bot{name: (n)}

	botCount++
	return &bot
}

func (b *Bot) wsGame() {
	Header := http.Header{}
	Header.Add("Cookie", b.Cookies)
	conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8000/wsgame", Header)
	if err != nil {
		panic(err)
	}
	bot := game.NewBotGame(conn)
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"package":  "main",
				"function": "readMessage",
				"error":    err,
				"data":     "name"},
			).Error("Failed reading message")
			return
		}
		if p != nil {
			var msg map[string]interface{}
			err := json.Unmarshal(p, &msg)
			if err != nil {
				fmt.Println(err)
			}
			bot.NewMessage(msg)
			continue
		}
	}
}

func (b *Bot) httpRegister() (string, error) {
	message := b.name
	fmt.Println(bytes.NewBufferString(message))
	client :=
		&http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	resp, err := client.Post("http://127.0.0.1:8000/register", "application/json", bytes.NewBufferString(message))
	if err != nil {
		panic(err)
	}
	sc := resp.Header["Set-Cookie"][0]
	return sc, nil
}

//Games holds information about game
type Games struct {
	MaxPlayers    int      `json:"maxpl"`
	InGamePlayers *int     `json:"players"`
	Name          string   `json:"name"`
	ID            int      `json:"id,string"`
	Password      string   `json:"pass,omitempty"`
	PlayersIn     []string `json:"players_name"`
}

func unmarshalJSON(data []byte) (interface{}, error) {
	var v []Games

	err := json.Unmarshal(data, &v)
	if err != nil {
		fmt.Println(err)
	} else {
		return v[0], nil
	}

	redirect := struct {
		ConnectTo int `json:"game_id,string"`
	}{}
	err = json.Unmarshal(data, &redirect)
	if err != nil {
		fmt.Println(err)
	} else {
		return redirect, nil
	}
	// if err := json.Unmarshal(data, &v); err != nil {
	// 	fmt.Printf("Error whilde decoding %v\n", err)
	// 	return err
	// }

	return nil, errors.New("Unknows JSON type")
}

func (b *Bot) wsMain() {
	Header := http.Header{}
	Header.Add("Cookie", b.Cookies)
	conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:8000/wsmain", Header)
	if err != nil {
		panic(err)
	}
	for {

		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"package":  "main",
				"function": "readMessage",
				"error":    err,
				"data":     "name"},
			).Error("Failed reading message")
			return
		}
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err) //TODO need this?
			return
		}
		if p != nil {
			fmt.Println(string(p))
			msg, err := unmarshalJSON(p)
			if err != nil {
				fmt.Println(err)
			}
			switch msg := msg.(type) {
			case Games:
				message := server.GameJoinJSON{Name: strconv.Itoa(msg.ID)}
				fmt.Println(message)
				sendJoin(conn, message)
			case struct {
				ConnectTo int `json:"game_id,string"`
			}:
				fmt.Println(msg, "im here")
				conn.Close()
				b.wsGame()
			default:
				fmt.Println("default unknown type")
			}
			continue
		}
	}
}

var botCount int = 1

func sendJoin(client *websocket.Conn, msg server.GameJoinJSON) {
	fmt.Println("trying to connect")
	w, err := client.NextWriter(websocket.TextMessage)
	if err != nil {
		fmt.Println(err)
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
	w.Write(data)
	w.Close()
}
