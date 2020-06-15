package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

//NextPlayersID corresponds next new players ID
var NextPlayersID int = 0

//PlayersDB all created players
var PlayersDB map[int]*player = make(map[int]*player)

type player struct {
	mu           sync.RWMutex
	client       *websocket.Conn
	id           int
	name         string
	currentCards []string
	bet          int
	score        int
	newMsg       chan struct{}
	sendScore    chan struct{}
	sendEnd      chan struct{}
	betFlag      bool
	turnFlag     bool
	jokerFlag    bool
	inputCh      chan int
	jokerChan    chan int
	err          error
	tablePTR     *Table
}

//Table is struct of game
type Table struct {
	mu               sync.Mutex
	file             *os.File
	cards            [36]string
	onTable          []string
	playersCount     int
	currentRound     int
	cardsOnRound     int
	firstTurn        int
	currentTurn      int
	id               int
	players          []*player
	maxCardsToPlayer int
	trump            string
	gameNotEnd       bool
	ScoreChart       [][]struct {
		Bet int
		Got int
	}
}

func (t *Table) Write(s string) {
	t.file.WriteString(s)
}

//AllPlayers contains all online players
// var AllPlayers = map[int]*player{}

var gameTables = make([]*Table, 0, 5)
var nonStartedTables = map[string]*Table{}

//NewPlayer Creates new player to DB
func newPlayer(name string) *player {
	AllUsers.mu.Lock()
	defer AllUsers.mu.Unlock()
	AllUsers.Users[name].mu.Lock()
	defer AllUsers.Users[name].mu.Unlock()
	AllUsers.Users[name].plr = player{}
	AllUsers.Users[name].plr.mu.Lock()
	defer AllUsers.Users[name].plr.mu.Unlock()
	AllUsers.Users[name].plr.name = name
	AllUsers.Users[name].plr.inputCh = make(chan int)
	AllUsers.Users[name].plr.jokerChan = make(chan int)
	AllUsers.Users[name].plr.newMsg = make(chan struct{})
	AllUsers.Users[name].plr.sendScore = make(chan struct{})
	AllUsers.Users[name].plr.sendEnd = make(chan struct{})
	AllUsers.Users[name].plr.err = nil
	return &AllUsers.Users[name].plr
}

func (t *Table) getPlrsInGame() (count int) {
	t.mu.Lock()
	count = t.playersCount
	t.mu.Unlock()
	return
}

//NewTable create new game table
func NewTable(ID int) *Table {
	newT := Table{id: ID}
	newT.file, _ = os.Create(strconv.Itoa(newT.id) + ".txt")
	newT.mu.Lock()
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
	newT.gameNotEnd = true
	newT.firstTurn = 1
	gameTables = append(gameTables, &newT)
	newT.mu.Unlock()

	log.Infof("Table %v created on %v", newT.id, time.Now().UTC().Format("Jan_2 2006 15:04:05"))
	return &newT
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("html/main.html"))
	tpl.Execute(w, nil)
}

//ServeWS ...
func ServeWS(w http.ResponseWriter, r *http.Request) {

	id, _ := r.Cookie("session_id")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "main",
			"function": "ServeWS",
			"error":    err,
			"data":     id},
		).Error("Failed upgrade to websocket")
	}
	var wg sync.WaitGroup
	wg.Add(2)
	AllUsers.mu.Lock()
	currentUser := AllUsers.Users[id.Value]
	AllUsers.mu.Unlock()
	currentUser.plr.mu.Lock()
	currentUser.plr.client = ws
	currentUser.plr.mu.Unlock()
	go currentUser.plr.readMessage()
	go currentUser.plr.SendCards()
	currentUser.plr.newMsg <- struct{}{}
	// time.Sleep(100 * time.Millisecond)
	// AllUsers.Users[id.Value].plr.newMsg <- struct{}{}
	currentUser.plr.sendScore <- struct{}{}
	wg.Wait()
	currentUser.plr.mu.Lock()
	currentUser.plr.client = nil
	currentUser.plr.mu.Unlock()
}

//MainServeHTTP ...y
func MainServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Cookie("session_id")
	AllUsers.mu.Lock()
	defer AllUsers.mu.Unlock()
	if AllUsers.Users[id.Value].plr.tablePTR == nil {
		http.Redirect(w, r, "/main", http.StatusFound)
		return
	}
	data := struct{ Name int }{Name: 7}
	tpl := template.Must(template.ParseFiles("html/game.html"))
	tpl.Execute(w, data)
}

//Join adds player to table
func (t *Table) Join(name string) error {
	t.mu.Lock()

	if len(t.players) < 3 {
		t.playersCount++
		t.players = append(t.players, newPlayer(name))
	} else {
		return errors.New("Game was already started")
	}
	count := t.playersCount
	id := t.id

	t.mu.Unlock()
	AllUsers.mu.Lock()
	AllUsers.Users[name].plr.tablePTR = t
	AllUsers.mu.Unlock()
	if count == 3 {
		t.Write("SHOULD START\n")
		go func() {
			MainServers.mu.Lock()
			delete(MainServers.NonStartedGames, id)
			MainServers.mu.Unlock()
			t.Start()
			t.mu.Lock()
			for _, pl := range t.players {
				pl.mu.Lock()
				pl.tablePTR = nil
				pl.mu.Unlock()
			}
			t.mu.Unlock()
		}()
	}
	return nil
}

func (t *Table) addNewRoindInChart(round int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ScoreChart = append(t.ScoreChart, make([]struct {
		Bet int
		Got int
	}, t.playersCount))

	for i := range t.players {
		t.ScoreChart[round-1][i] = struct {
			Bet int
			Got int
		}{}
	}
}

//Start starts the game
func (t *Table) Start() {
	t.Write("GAME STARTED\n")
	t.mu.Lock()
	for i := range t.players {
		t.players[i].id = i
	}

	t.ScoreChart = make([][]struct {
		Bet int
		Got int
	}, 0, 18)

	// t.maxCardsToPlayer = 36 / t.playersCount
	t.maxCardsToPlayer = 6
	t.mu.Unlock()
	t.sendScore()
	time.Sleep(1 * time.Second)
	// go t.sendEverySecInfo()
	for i := 0; i < 6; i++ {
		t.Write(fmt.Sprintln("round", i))
		t.round(i + 1)
	}
	t.file.Close()
	os.Remove(strconv.Itoa(t.id) + ".txt")
	// for i := 0; i < t.maxCardsToPlayer; i++ {
	// 	t.round(i + 1)
	// }
	// for i := 0; i < len(t.players); i++ {
	// 	t.round(t.maxCardsToPlayer)
	// }
	// for i := t.maxCardsToPlayer; i > 0; i-- {
	// 	t.round(i)
	// }
	time.Sleep(20 * time.Second)
	t.sendScore()
	t.sendEndOfGame()
	t.mu.Lock()
	t.gameNotEnd = false
	t.mu.Unlock()

	log.Infof("Game over of table %v on %v", t.id, time.Now().UTC().Format("Jan_2 2006 15:04:05"))
}

func (t *Table) round(round int) {
	t.mu.Lock()
	t.currentRound++
	t.mu.Unlock()
	t.addNewRoindInChart(t.currentRound)
	roundScore := make(map[int]int)
	for key := range roundScore {
		roundScore[key] = 0
	}
	t.mu.Lock()
	t.trump = ""
	rand.Seed(time.Now().UnixNano())

	func(cards [36]string) {
		for i := range cards {
			j := rand.Intn(i + 1)
			cards[i], cards[j] = cards[j], cards[i]
		}

		if round <= t.maxCardsToPlayer {
			count := 0
			for i := range t.players {
				t.players[i].currentCards = cards[count*round : count*round+round]
				sort.SliceStable(t.players[i].currentCards, func(x, y int) bool { return t.players[i].currentCards[x] > t.players[i].currentCards[y] })
				count++
			}
		}
		t.trump = cards[35]
	}(t.cards)

	t.currentTurn = t.firstTurn
	t.firstTurn++

	if t.firstTurn == t.playersCount {
		t.firstTurn = 0
	}
	t.mu.Unlock()
	t.refreshCards()

	for range t.players {

		t.getBet(t.currentTurn, t.players[t.currentTurn].inputCh)
		t.refreshCards()
		t.mu.Lock()
		t.currentTurn++
		if t.currentTurn >= t.playersCount {
			t.currentTurn = 0
		}
		t.mu.Unlock()
	}

	for len(t.players[0].currentCards) > 0 {
		t.sendScore()
		t.refreshCards()
		firstCardIndex := t.currentTurn
		t.mu.Lock()
		t.onTable = make([]string, t.playersCount)
		t.mu.Unlock()
		for i := 0; i < t.playersCount; i++ {
			t.Write("NEXT CARD\n")
			card := ""
			var cardIndex int
			for !t.cardPermissionToTable(t.onTable[firstCardIndex], card, t.currentTurn) {
				if card != "" {
					t.Write(fmt.Sprintf("wrong card %s\n", card))
					time.Sleep(time.Millisecond * 10)
				}
				t.refreshCards()
				t.sendScore()
				card, cardIndex = t.dropCard(t.currentTurn)
			}
			t.Write("CARD IS OK\n")
			t.mu.Lock()
			t.players[t.currentTurn].mu.Lock()
			t.players[t.currentTurn].currentCards = append(t.players[t.currentTurn].currentCards[:cardIndex], t.players[t.currentTurn].currentCards[cardIndex+1:]...)
			t.players[t.currentTurn].mu.Unlock()
			t.onTable[t.currentTurn] = card
			t.currentTurn++
			if t.currentTurn == t.playersCount {
				t.currentTurn = 0
			}
			t.mu.Unlock()
			t.sendScore()
			t.refreshCards()

		}
		whoWin := t.whoGetTheTable()
		roundScore[whoWin]++
		time.Sleep(800 * time.Millisecond)
		t.mu.Lock()
		t.onTable = nil
		t.mu.Unlock()
		t.Write("next iter")
	}
	t.Write("round edn\n")
	t.calculateScore(roundScore)
}

func (t *Table) getBet(player int, bet <-chan int) {
	t.mu.Lock()
	t.players[player].betFlag = true
	t.mu.Unlock()
	t.sendScore()
	t.Write("waiting bet\n")
	temp := <-bet
	t.Write("GOT BET\n")
	t.mu.Lock()
	t.players[player].bet = temp
	t.ScoreChart[t.currentRound-1][player].Bet = t.players[player].bet
	t.mu.Unlock()
}

func (t *Table) dropCard(player int) (card string, cardIndex int) {

	//t.mu.Lock()
	t.players[player].mu.Lock()
	t.players[player].turnFlag = true
	t.players[player].mu.Unlock()
	//t.mu.Unlock()
	t.sendScore()    //
	t.refreshCards() //
	t.Write(fmt.Sprintln("waiting card", t.players[player].turnFlag))
	go func() {
		time.Sleep(time.Millisecond * 100)
		t.sendScore()    //
		t.refreshCards() //
	}()
	cardIndex = <-t.players[player].inputCh
	t.Write(fmt.Sprintln("got card"))
	for cardIndex >= len(t.players[player].currentCards) {
		t.mu.Lock()
		t.players[player].turnFlag = true
		t.mu.Unlock()
		t.Write(fmt.Sprintln("bad index", t.players[player].turnFlag))
		time.Sleep(time.Millisecond * 100)
		t.sendScore()    //
		t.refreshCards() //
		cardIndex = <-t.players[player].inputCh
		t.Write(fmt.Sprintln("Got new"))
	}
	t.Write(fmt.Sprintln("check joker"))
	select {
	case jokerType := <-t.players[player].jokerChan:

		if t.players[player].currentCards[cardIndex] == "♠1" {
			card, _ = t.whatJokerMeans(player, jokerType)
		} else {
			card = t.players[player].currentCards[cardIndex]
		}

	default:
		if t.players[player].currentCards[cardIndex] == "♠1" {
			card, _ = t.whatJokerMeans(player, <-t.players[player].jokerChan)
		} else {
			card = t.players[player].currentCards[cardIndex]
		}
	}
	t.Write(fmt.Sprintln("checked"))
	return
}

func (t *Table) whoGetTheTable() (id int) {
	t.mu.Lock()
	defer t.mu.Unlock()
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
	t.ScoreChart[t.currentRound-1][winIDIndex].Got++
	return
}

func (t *Table) cardPermissionToTable(suit string, card string, id int) (ok bool) {
	if card == "" {
		return false
	}

	if card == "♠1" {
		return true
	}

	if card == "♥9" || card == "♦9" || card == "♣9" || card == "♠9" {
		return true
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
		if (t.players[id].currentCards[i][0:3] == suit) && (t.players[id].currentCards[i] != "♠1") {
			return false
		}
	}
	if card[0:3] == trumpSuit { //if trump ok
		return true
	}
	for i := range t.players[id].currentCards { //check for non having trumps
		if (t.players[id].currentCards[i][0:3] == trumpSuit) && (t.players[id].currentCards[i] != "♠1") {
			return false
		}
	}
	return true
}

func (t *Table) whatJokerMeans(player int, joker int) (string, error) {
	//0-maxTrump, 1-♥maxHeart, 2-♦maxDiamond, 3-♣maxClub, 4-♠maxSpade, 5-♥minHeart, 6-♦minDiamond, 7-♣minClub, 8-♠minSpade, 9-minOnTable
	switch joker {
	case 0:
		return (t.trump[0:3] + "9"), nil
	case 1:
		if len(t.onTable) != 0 {
			return "", errors.New(strconv.Itoa(joker) + " should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New(strconv.Itoa(joker) + " should be first card of round")
		}
		return "♥9", nil
	case 2:
		if len(t.onTable) != 0 {
			return "", errors.New(strconv.Itoa(joker) + " should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New(strconv.Itoa(joker) + " should be first card of round")
		}
		return "♦9", nil
	case 3:
		if len(t.onTable) != 0 {
			return "", errors.New(strconv.Itoa(joker) + " should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New(strconv.Itoa(joker) + " should be first card of round")
		}
		return "♣9", nil
	case 4:
		if len(t.onTable) != 0 {
			return "", errors.New(strconv.Itoa(joker) + " should be first turn")
		}
		if len(t.players[player].currentCards) != t.cardsOnRound {
			return "", errors.New(strconv.Itoa(joker) + " should be first card of round")
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
		if len(t.onTable) != 0 {
			return "", errors.New(strconv.Itoa(joker) + "can't be handled. Choose suit")
		}
		return "0000", nil
	default:
		return "", errors.New("wrong joker code")
	}
}

func (p *player) SendCards() {
	for {
		select {
		case <-p.newMsg:
			p.mu.Lock()
			plTable := p.tablePTR
			p.mu.Unlock()
			w, err := p.client.NextWriter(websocket.TextMessage)
			if err != nil {
				log.WithFields(log.Fields{
					"package":  "main",
					"function": "sendCards",
					"error":    err,
					"data":     p.name},
				).Error("Failed sending cards message to websocket")
				return
			}
			var cardsWithShift []string
			plTable.mu.Lock()
			if len(plTable.onTable) != 0 {
				shift := p.id
				cardsWithShift = append(p.tablePTR.onTable[shift:],
					p.tablePTR.onTable[:shift]...)
			}
			data, err := json.Marshal(map[string]interface{}{
				"trump":  p.tablePTR.trump,
				"cards":  p.currentCards,
				"player": cardsWithShift,
			})
			plTable.mu.Unlock()
			if err != nil {
				log.WithFields(log.Fields{
					"package":  "main",
					"function": "sendCards",
					"error":    err,
					"data":     p.name},
				).Error("Failed converting cards message to JSON")

				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				p.client.WriteJSON(errorMsg)
				w.Close()
				continue

			}
			w.Write(data)
			w.Close()
		case <-p.sendScore:
			p.mu.Lock()
			p.tablePTR.mu.Lock()
			w, err := p.client.NextWriter(websocket.TextMessage)
			if err != nil {
				log.WithFields(log.Fields{
					"package":  "main",
					"function": "sendCards",
					"error":    err,
					"data":     p.name},
				).Error("Failed sending score message to websocket")
				return
			}
			var names []string
			var totalScore []int
			for i := range p.tablePTR.players {
				names = append(names, p.tablePTR.players[i].name)
				totalScore = append(totalScore, p.tablePTR.players[i].score)
			}
			data, err := json.Marshal(map[string]interface{}{
				"position":   p.id,
				"turn":       p.tablePTR.currentTurn,
				"isBet":      p.betFlag,
				"names":      names,
				"totalScore": totalScore,
				"scoreChart": p.tablePTR.ScoreChart,
			})
			if err != nil {
				log.WithFields(log.Fields{
					"package":  "main",
					"function": "sendCards",
					"error":    err,
					"data":     p.name},
				).Error("Failed converting score message to JSON")
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				p.client.WriteJSON(errorMsg)
			}
			w.Write(data)
			w.Close()
			p.mu.Unlock()
			p.tablePTR.mu.Unlock()
		case <-p.sendEnd:
			p.mu.Lock()
			p.tablePTR.mu.Lock()
			w, err := p.client.NextWriter(websocket.TextMessage)
			if err != nil {
				log.WithFields(log.Fields{
					"package":  "main",
					"function": "sendCards",
					"error":    err,
					"data":     p.name},
				).Error("Failed sending score message to websocket")
				p.mu.Unlock()
				p.tablePTR.mu.Unlock()
				return
			}
			var names []string
			var totalScore []int
			for i := range p.tablePTR.players {
				names = append(names, p.tablePTR.players[i].name)
				totalScore = append(totalScore, p.tablePTR.players[i].score)
			}
			data, err := json.Marshal(map[string]interface{}{
				"game_over": 1,
			})
			if err != nil {
				log.WithFields(log.Fields{
					"package":  "main",
					"function": "sendCards",
					"error":    err,
					"data":     p.name},
				).Error("Failed converting end of game message to JSON")
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				p.client.WriteJSON(errorMsg)
			}
			w.Write(data)
			w.Close()
			p.mu.Unlock()
			p.tablePTR.mu.Unlock()
			p.client.Close()
			return
		}
	}

}

func (p *player) readMessage() {
	for {
		_, msg, err := p.client.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"package":  "main",
				"function": "readMessage",
				"error":    err,
				"data":     p.name},
			).Error("Failed reading message")
			return
		}
		// if err := p.client.WriteMessage(messageType, msg); err != nil {
		// 	log.Println(err) //TODO need this?
		// 	return
		// }
		if p != nil {
			var data map[string]interface{}
			err := json.Unmarshal(msg, &data)
			if err != nil {
				if err != nil {
					log.WithFields(log.Fields{
						"package":  "main",
						"function": "readMessage",
						"error":    err,
						"data":     p.name},
					).Error("Failed unmarshalling JSON")
					return
				}
				errorMsg := struct{ stringErr string }{stringErr: err.Error()}
				p.client.WriteJSON(errorMsg)
				continue
			}

			if data["bet"] != nil {
				if p.betFlag == true {
					betStr := data["bet"].(string)
					bet, err := strconv.Atoi(betStr)
					if err != nil {
						log.WithFields(log.Fields{
							"package":  "main",
							"function": "readMessage",
							"error":    err,
							"data":     bet},
						).Warning("Bet should be int")
						errorMsg := struct{ stringErr string }{stringErr: err.Error()}
						p.client.WriteJSON(errorMsg)
						continue
					}
					p.betFlag = false
					p.inputCh <- bet
				}
			}
			if data["card_number"] != nil {
				p.tablePTR.Write(fmt.Sprintln(data["card_number"]))

				p.mu.Lock()
				p.tablePTR.Write(fmt.Sprintln(p.turnFlag))
				if p.turnFlag == true {
					p.turnFlag = false
					p.mu.Unlock()
					turnFloat := data["card_number"].(float64)
					card := int(turnFloat)
					p.tablePTR.Write("send to channel\n")
					p.inputCh <- card
					if data["joker"] != nil {
						jokerFloat := data["joker"].(float64)
						p.jokerChan <- int(jokerFloat)
					}
				} else {
					p.mu.Unlock()
				}
			}

		}
	}
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

func (t *Table) refreshCards() {
	for i := range t.players {
		t.players[i].newMsg <- struct{}{}
	}
}

func (t *Table) sendScore() {
	for _, pl := range t.players {
		pl.sendScore <- struct{}{}
	}
}

func (t *Table) sendEndOfGame() {
	for _, pl := range t.players {
		pl.sendEnd <- struct{}{}
	}
}

// "♥" "♦" "♣" "♠"
