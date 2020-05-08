package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
)

//NextPlayersID corresponds next new players ID
var NextPlayersID int = 0

//PlayersDB all created players
var PlayersDB map[int]*player = make(map[int]*player)

type player struct {
	id           int
	name         string
	currentCards []string
	bet          int
	score        int
	newMsg       chan struct{}
	sendScore    chan struct{}
	betFlag      bool
	turnFlag     bool
	jokerFlag    bool
	inputCh      chan int
	err          error
	tablePTR     *Table
}

//Table is struct of game
type Table struct {
	cards            [36]string
	onTable          []string
	playersCount     int
	currentRound     int
	cardsOnRound     int
	firstTurn        int
	currentTurn      int
	players          []*player
	maxCardsToPlayer int
	trump            string
	ScoreChart       [][]struct {
		Bet int
		Got int
	}

	mu sync.RWMutex
}

//AllPlayers contains all online players
// var AllPlayers = map[int]*player{}

var gameTables = make([]*Table, 0, 5)
var nonStartedTables = map[string]*Table{}

//NewPlayer Creates new player to DB
func newPlayer(name string) *player {
	AllUsers[name].plr = player{}
	AllUsers[name].plr.name = name
	AllUsers[name].plr.inputCh = make(chan int)
	AllUsers[name].plr.newMsg = make(chan struct{})
	AllUsers[name].plr.err = nil
	return &AllUsers[name].plr
}

//NewTable create new game table
func NewTable() *Table {
	newT := Table{}
	number := 0
	for suit := 0; suit < 4; suit++ {
		for nom := 0; nom < 9; nom++ {
			switch suit {
			case 0:
				newT.cards[number] = "♣" + strconv.Itoa(nom)
			case 1:
				newT.cards[number] = "♦" + strconv.Itoa(nom)
			case 2:
				newT.cards[number] = "♠" + strconv.Itoa(nom)
			case 3:
				newT.cards[number] = "♥" + strconv.Itoa(nom)
			default:
			}
			number++
		}
	}
	newT.firstTurn = 1
	fmt.Println("Table created")
	gameTables = append(gameTables, &newT)
	go func() {
		for {
			if len(newT.players) == 2 {
				newT.Start()
				break
			}
		}
	}()
	return &newT
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("html/main.html"))
	tpl.Execute(w, nil)
}

//ServeWS ...
func ServeWS(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in hand")

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	id, _ := r.Cookie("session_id")
	fmt.Println(id)
	go readMessage(ws, id.Value)
	go sendCards(ws, id.Value)

	AllUsers[id.Value].plr.newMsg <- struct{}{}
	time.Sleep(100 * time.Millisecond)
	AllUsers[id.Value].plr.newMsg <- struct{}{}
}

func (t *Table) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("IN SERVE")
	id, _ := r.Cookie("session_id")
	if AllUsers[id.Value].plr.tablePTR == nil {
		http.Redirect(w, r, "/main", http.StatusFound)
		return
	}
	data := struct{ Name int }{Name: 7}
	tpl := template.Must(template.ParseFiles("html/game.html"))
	tpl.Execute(w, data)
}

//Join adds player to table
func (t *Table) Join(name string) {
	t.playersCount++

	t.players = append(t.players, newPlayer(name))
	AllUsers[name].plr.tablePTR = t
}

//Start starts the game
func (t *Table) Start() {
	for i := range t.players {
		t.players[i].id = i
	}

	rounds := 30 // should be precalculated amount of rounds
	for round := 0; round < rounds; round++ {
		t.ScoreChart[round] = make([]struct {
			Bet int
			Got int
		}, rounds)
		for i := range t.players {
			t.ScoreChart[round][i] = struct {
				Bet int
				Got int
			}{}
		}
	}
	t.sendScore()
	t.maxCardsToPlayer = 36 / t.playersCount

	for i := 0; i < t.maxCardsToPlayer; i++ {
		t.round(i + 1)
	}
	for i := 0; i < len(t.players); i++ {
		t.round(t.maxCardsToPlayer)
	}
	for i := t.maxCardsToPlayer; i > 0; i-- {
		t.round(i)
	}
	t.blindRound()
	for id := range t.players {
		fmt.Println(t.players[id].id, t.players[id].score)
	}
}

func (t *Table) round(round int) {
	t.currentRound++
	roundScore := make(map[int]int)
	for key := range roundScore {
		roundScore[key] = 0
	}
	t.trump = ""

	for i := range t.cards {
		j := rand.Intn(i + 1)
		t.cards[i], t.cards[j] = t.cards[j], t.cards[i]
	}

	if round <= t.maxCardsToPlayer {
		count := 0
		for i := range t.players {
			t.players[i].currentCards = t.cards[count*round : count*round+round]
			sort.SliceStable(t.players[i].currentCards, func(x, y int) bool { return t.players[i].currentCards[x] > t.players[i].currentCards[y] })
			count++
			fmt.Println(t.players[i])
		}
	}

	t.trump = t.cards[35]
	t.currentTurn = t.firstTurn
	t.firstTurn++

	if t.firstTurn == t.playersCount {
		t.firstTurn = 0
	}

	t.refreshCards()

	for i := range t.players {
		turn := t.currentTurn + i
		if turn >= t.playersCount {
			turn = 0
		}
		t.getBet(turn, t.players[turn].inputCh)
		t.refreshCards()
		t.sendScore()
	}

	for len(t.players[0].currentCards) > 0 {
		t.refreshCards()
		t.sendScore()
		firstCardIndex := t.currentTurn
		t.onTable = make([]string, t.playersCount)
		for i := 0; i < t.playersCount; i++ {
			card := ""
			var cardIndex int
			for !t.cardPermissionToTable(t.onTable[firstCardIndex], card, t.currentTurn) {
				cardIndex = t.dropCard(t.currentTurn)
				if t.players[t.currentTurn].currentCards[cardIndex] == "♠1" {
					var err error
					for {
						card, err = t.whatJokerMeans(t.currentTurn)
						if err != nil {
							t.mu.Lock()
							t.players[t.currentTurn].err = err
							t.mu.Unlock()
							continue
						}
						break
					}
					break
				} else {
					card = t.players[t.currentTurn].currentCards[cardIndex]
				}
			}
			t.players[t.currentTurn].currentCards = append(t.players[t.currentTurn].currentCards[:cardIndex], t.players[t.currentTurn].currentCards[cardIndex+1:]...)
			t.onTable[t.currentTurn] = card
			t.currentTurn++
			if t.currentTurn == t.playersCount {
				t.currentTurn = 0
			}
			t.refreshCards()
			t.sendScore()
		}
		whosTurn := t.players[t.currentTurn].name
		whoWin := t.whoGetTheTable()
		roundScore[whoWin]++

		fmt.Printf("%s turn. Cards on TABLE: %s. Trump:%s. WINNER:%s\n", whosTurn, t.onTable, t.trump, t.players[whoWin].name)
		time.Sleep(1 * time.Second)
		t.onTable = nil

	}
	t.calculateScore(roundScore)
}

func (t *Table) getBet(player int, bet <-chan int) {
	t.mu.Lock()
	t.players[player].betFlag = true
	t.mu.Unlock()
	t.players[player].bet = <-bet
	t.ScoreChart[t.currentRound][player].Bet = t.players[player].bet
}

func (t *Table) dropCard(player int) (cardIndex int) {

	t.mu.Lock()
	t.players[player].turnFlag = true
	t.mu.Unlock()
	cardIndex = <-t.players[player].inputCh
	return
}

func (t *Table) whoGetTheTable() (id int) {
	maxCard := t.onTable[t.currentTurn]
	maxIndex := t.currentTurn
	for i := 0; i < len(t.onTable); i++ {
		index := t.currentTurn + i
		if index >= t.playersCount {
			index -= t.playersCount
		}
		currentCard := t.onTable[index]
		if maxCard[0:3] != currentCard[0:3] {
			if currentCard[0:3] != t.trump[0:3] {
				continue
			}
			maxCard = currentCard
			maxIndex = index
		} else {
			if maxCard < currentCard {
				maxCard = currentCard
				maxIndex = index
			}
		}
	}
	winIDIndex := maxIndex
	id = t.players[winIDIndex].id
	t.currentTurn = winIDIndex
	t.ScoreChart[t.currentRound][winIDIndex].Got++
	return
}

func (t *Table) cardPermissionToTable(suit string, card string, id int) (ok bool) {
	if card == "" {
		return false
	}
	if suit == "" { //if first card
		return true
	}

	suit = suit[0:3]
	if card[0:3] == suit { //if same suit
		return true
	}
	trumpSuit := t.trump[0:3]
	for i := range t.players[id].currentCards { //check for non having suit
		if t.players[id].currentCards[i][0:3] == suit {
			return false
		}
	}
	if card[0:3] == trumpSuit { //if trump ok
		return true
	}
	for i := range t.players[id].currentCards { //check for non having trumps
		if t.players[id].currentCards[i][0:3] == trumpSuit {
			return false
		}
	}
	return true
}

func (t *Table) whatJokerMeans(player int) (string, error) {
	t.mu.Lock()
	t.players[player].jokerFlag = true
	t.mu.Unlock()
	joker := <-t.players[player].inputCh //0-maxTrump, 1-♥maxHeart, 2-♦maxDiamond, 3-♣maxClub, 4-♠maxSpade, 5-♥minHeart, 6-♦minDiamond, 7-♣minClub, 8-♠minSpade, 9-minOnTable
	switch joker {
	case 0:
		return (t.trump[0:3] + "9"), nil
	case 1:
		if len(t.onTable) != 0 {
			return "", errors.New("Should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			fmt.Println(len(t.players[player].currentCards), t.cardsOnRound)
			return "", errors.New("Should be first card of round")
		}
		return "♥9", nil
	case 2:
		if len(t.onTable) != 0 {
			return "", errors.New("Should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New("Should be first card of round")
		}
		return "♦9", nil
	case 3:
		if len(t.onTable) != 0 {
			return "", errors.New("Should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New("Should be first card of round")
		}
		return "♣9", nil
	case 4:
		if len(t.onTable) != 0 {
			return "", errors.New("Should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New("Should be first card of round")
		}
		return "♠9", nil
	case 5:
		return "♥", nil
	case 6:
		return "♦", nil
	case 7:
		return "♣", nil
	case 8:
		return "♠", nil
	case 9:
		return "♠", nil //prob any lowest, do not allow to start with it
	default:
		return "", errors.New("wrong joker code")
	}
}

func sendCards(client *websocket.Conn, name string) {
	for {
		select {
		case <-AllUsers[name].plr.newMsg:
			w, err := client.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			var cardsWithShift []string
			if len(AllUsers[name].plr.tablePTR.onTable) != 0 {
				shift := AllUsers[name].plr.id
				cardsWithShift = append(AllUsers[name].plr.tablePTR.onTable[shift:],
					AllUsers[name].plr.tablePTR.onTable[:shift]...)
			}
			data, err := json.Marshal(map[string]interface{}{
				"trump":  AllUsers[name].plr.tablePTR.trump,
				"cards":  AllUsers[name].plr.currentCards,
				"player": cardsWithShift,
				"error":  AllUsers[name].plr.err,
			})
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				client.WriteJSON(errorMsg)
				w.Close()
				continue
			}
			AllUsers[name].plr.tablePTR.mu.Lock()
			AllUsers[name].plr.err = nil
			AllUsers[name].plr.tablePTR.mu.Unlock()
			w.Write(data)
			w.Close()
		case <-AllUsers[name].plr.sendScore:
			w, err := client.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			var names []string
			var totalScore []int
			for i := range AllUsers[name].plr.tablePTR.players {
				names = append(names, AllUsers[name].plr.tablePTR.players[i].name)
				totalScore = append(totalScore, AllUsers[name].plr.tablePTR.players[i].score)
			}
			data, err := json.Marshal(map[string]interface{}{
				"names":      names,
				"totalScore": totalScore,
				"scoreChart": AllUsers[name].plr.tablePTR.ScoreChart,
			})
			if err != nil {
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				client.WriteJSON(errorMsg)
			}
			w.Write(data)
			w.Close()
		}
	}
}

func (t *Table) refreshCards() {
	for i := range t.players {
		t.players[i].newMsg <- struct{}{}
	}
}

func readMessage(conn *websocket.Conn, name string) {
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
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				conn.WriteJSON(errorMsg)
				continue
			}

			if data["bet"] != nil {
				if AllUsers[name].plr.betFlag == true {
					AllUsers[name].plr.betFlag = false
					betStr := data["bet"].(string)
					bet, err := strconv.Atoi(betStr)
					if err != nil {
						errorMsg := struct{ stringErr string }{stringErr: err.Error()}
						conn.WriteJSON(errorMsg)
						continue
					}
					AllUsers[name].plr.inputCh <- bet
				}
			}
			if data["card_number"] != nil {
				if AllUsers[name].plr.turnFlag == true {
					AllUsers[name].plr.turnFlag = false
					turnFloat := data["card_number"].(float64)
					card := int(turnFloat)
					AllUsers[name].plr.inputCh <- card
				}
			}
			if data["joker"] != nil {
				if AllUsers[name].plr.jokerFlag == true {
					AllUsers[name].plr.jokerFlag = false
					jokerFloat := data["joker"].(float64)
					AllUsers[name].plr.inputCh <- int(jokerFloat)
				}
			}
		}
	}
}

func (t *Table) blindRound() {
	t.currentRound++
	roundScore := make(map[int]int)
	for key := range roundScore {
		roundScore[key] = 0
	}
	t.trump = ""
	t.currentTurn = t.firstTurn
	t.firstTurn++

	if t.firstTurn == t.playersCount {
		t.firstTurn = 0
	}

	t.refreshCards()

	suits := make([]string, 18)
	for i := 0; i < t.maxCardsToPlayer; i++ {
		suits = append(suits, "suit")
	}
	for _, i := range t.players {
		i.currentCards = suits
	}
	t.refreshCards()

	for i := range t.players {
		turn := t.currentTurn + i
		if turn >= t.playersCount {
			turn = 0
		}
		t.getBet(turn, t.players[turn].inputCh)
		t.refreshCards()
	}

	for i := range t.cards {
		j := rand.Intn(i + 1)
		t.cards[i], t.cards[j] = t.cards[j], t.cards[i]
	}

	count := 0
	for i := range t.players {
		t.players[i].currentCards = t.cards[count*t.maxCardsToPlayer : count*t.maxCardsToPlayer+t.maxCardsToPlayer]
		sort.SliceStable(t.players[i].currentCards, func(x, y int) bool { return t.players[i].currentCards[x] > t.players[i].currentCards[y] })
		count++
		fmt.Println(t.players[i])
	}

	t.trump = t.cards[35]

	for len(t.players[0].currentCards) > 0 {
		t.refreshCards()
		firstCardIndex := t.currentTurn
		t.onTable = make([]string, t.playersCount)
		for i := 0; i < t.playersCount; i++ {
			card := ""
			var cardIndex int
			for !t.cardPermissionToTable(t.onTable[firstCardIndex], card, t.currentTurn) {
				cardIndex = t.dropCard(t.currentTurn)
				if t.players[t.currentTurn].currentCards[cardIndex] == "♠1" {
					var err error
					for {
						card, err = t.whatJokerMeans(t.currentTurn)
						if err != nil {
							t.mu.Lock()
							t.players[t.currentTurn].err = err
							t.mu.Unlock()
							continue
						}
						break
					}
					break
				} else {
					card = t.players[t.currentTurn].currentCards[cardIndex]
				}
			}
			t.players[t.currentTurn].currentCards = append(t.players[t.currentTurn].currentCards[:cardIndex], t.players[t.currentTurn].currentCards[cardIndex+1:]...)
			t.onTable[t.currentTurn] = card
			t.currentTurn++
			if t.currentTurn == t.playersCount {
				t.currentTurn = 0
			}
			t.refreshCards()
		}
		whosTurn := t.players[t.currentTurn].name
		whoWin := t.whoGetTheTable()
		roundScore[whoWin]++

		fmt.Printf("%s turn. Cards on TABLE: %s. Trump:%s. WINNER:%s\n", whosTurn, t.onTable, t.trump, t.players[whoWin].name)
		time.Sleep(1 * time.Second)
		t.onTable = nil

	}
	t.calculateScore(roundScore)
}

func (t *Table) calculateScore(roundScore map[int]int) {
	for id := range t.players {
		difference := roundScore[t.players[id].id] - t.players[id].bet
		if difference < 0 {
			t.players[id].score += difference * 10
		} else if difference == 0 {
			if t.players[id].bet == 0 {
				t.players[id].score += 5
			} else {
				t.players[id].score += t.players[id].bet * 10
			}
		} else {
			t.players[id].score += roundScore[t.players[id].id]
		}
	}
}

func (t *Table) sendScore() {
	for _, pl := range t.players {
		pl.sendScore <- struct{}{}
	}
}
