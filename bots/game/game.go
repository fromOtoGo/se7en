package game

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//BotGame ...
type BotGame struct {
	trump        string
	cardsOnTable []string
	botCards     []string
	position     int
	turn         int
	isBet        bool
	players      int
	conn         *websocket.Conn
	mu           sync.RWMutex
}

//NewBotGame creates game logic for game
func NewBotGame(conn *websocket.Conn) *BotGame {
	Game := BotGame{}
	Game.conn = conn
	Game.players = 3
	return &Game
}

func (bg *BotGame) start() {
	for {

	}
}

//NewMessage ...
func (bg *BotGame) NewMessage(msg map[string]interface{}) {
	bg.mu.Lock()
	for key, val := range msg {
		switch key {
		case "trump":
			bg.trump = val.(string)
		case "player":
			if val != nil {
				interType := val.([]interface{})
				var cards []string
				for i := range interType {
					cards = append(cards, interType[i].(string))
				}
				bg.cardsOnTable = cards
			} else {
				bg.cardsOnTable = nil
			}
		case "cards":
			if val != nil {
				interType := val.([]interface{})
				var cards []string
				for i := range interType {
					cards = append(cards, interType[i].(string))
				}
				bg.botCards = cards
			} else {
				bg.botCards = nil
			}
		case "position":
			bg.position = int(val.(float64))
		case "turn":
			bg.turn = int(val.(float64))
		case "isBet":
			bg.isBet = val.(bool)
		case "game_over":
			fmt.Println("the game is over I go off")
			bg.conn.Close()
		default:
		}
	}
	bg.mu.Unlock()

	// time.Sleep(time.Millisecond * 100)
	if bg.turn == bg.position {
		fmt.Println(bg)
		if len(bg.botCards) > 0 {
			if bg.isBet {
				bg.send([]byte(`{"bet":"1"}`))
			} else {
				if len(bg.botCards) == 1 {
					msg := sendCardJSON{CardNumber: 0}
					data, _ := json.Marshal(msg)
					bg.send(data)
					fmt.Printf("0 sended by%v card %v\n", bg.position, bg.botCards[0])
					return
				}
				// time.Sleep(time.Millisecond * 100)
				if len(bg.cardsOnTable) == 0 {
					if bg.botCards[0] == "♠1" { //if joker send code
						msg := sendCardJSON{CardNumber: 0, Joker: 0}
						data, _ := json.Marshal(msg)
						bg.send(data)
						fmt.Printf("1 sended by%v card %v\n", bg.position, bg.botCards[0])
					} else {
						msg := sendCardJSON{CardNumber: 0}
						data, _ := json.Marshal(msg)
						bg.send(data)
						fmt.Printf("2 sended by%v card %v\n", bg.position, bg.botCards[0])
					}
				} else {
					pos := bg.turn
					if pos >= bg.players {
						pos = 0
					}
					for i := 0; i < bg.players; i++ {
						if len(bg.cardsOnTable[pos]) < 1 { //fix it
							if len(bg.cardsOnTable[pos]) > 2 {
								break
							}
							pos++
							if pos >= bg.players {
								pos = 0
							}
						}
					}
					if len(bg.cardsOnTable[pos]) < 1 {
						fmt.Println("same bug")
						return
					}
					suit := bg.cardsOnTable[pos][0:3]
					for i := range bg.botCards {
						if bg.botCards[i] == "♠1" {
							msg := sendCardJSON{CardNumber: i, Joker: 0}
							data, _ := json.Marshal(msg)
							bg.send(data)
							fmt.Printf("4 sended by%v card %v\n", bg.position, bg.botCards[i])
							return

						}
					}
					for i := range bg.botCards {
						if bg.botCards[i][0:3] == suit {
							msg := sendCardJSON{CardNumber: i}
							data, _ := json.Marshal(msg)
							bg.send(data)
							fmt.Printf("5 sended by%v card %v\n", bg.position, bg.botCards[i])
							return
						}
					}
					for i := range bg.botCards {
						if bg.botCards[i][0:3] == bg.trump[0:3] {
							msg := sendCardJSON{CardNumber: i}
							data, _ := json.Marshal(msg)
							bg.send(data)
							fmt.Printf("6 sended by%v card %v\n", bg.position, bg.botCards[i])
							return
						}
					}
					msg := sendCardJSON{CardNumber: 0}
					data, _ := json.Marshal(msg)
					bg.send(data)
					fmt.Printf("7 sended by%v card %v\n", bg.position, bg.botCards[0])
				}
			}
		}
	}
}

type sendCardJSON struct {
	CardNumber int `json:"card_number"`
	Joker      int `json:"joker"`
}

func (bg *BotGame) send(message []byte) {
	time.Sleep(time.Millisecond * 10)
	w, err := bg.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		fmt.Println(err)
		return
	}
	w.Write(message)
	w.Close()
}
