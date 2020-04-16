package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
)

type player struct {
	id           string
	currentCards []string
	bet          int
	score        int
	betFlag      bool
	turnFlag     bool
	jokerFlag    bool
}

//Table is struct of game
type Table struct {
	cards            [36]string
	onTable          []string
	playersCount     int
	firstTurn        int
	currentTurn      int
	players          []player
	maxCardsToPlayer int
	trump            string
	inputCh          chan int
	mu               sync.RWMutex
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
	newT.inputCh = make(chan int)
	fmt.Println("Table created")
	return &newT
}

//Join adds player to table
func (t *Table) Join(pID string) {
	fmt.Println("Join", pID)
	t.playersCount++
	t.players = append(t.players, player{id: pID})
}

//Start starts the game
func (t *Table) Start() {
	if t.playersCount > 1 {
		t.maxCardsToPlayer = 36 / t.playersCount
		//start new server and redirect
	}
	for i := 0; i < t.maxCardsToPlayer; i++ {
		t.round(i + 1)
	}
	for i := t.maxCardsToPlayer; i > 0; i-- {
		t.round(i)
	}
	for id := range t.players {
		fmt.Println(t.players[id].id, t.players[id].score)
	}
}

func (t *Table) round(round int) {
	fmt.Println()
	fmt.Print("ROUND ", round, " ")
	fmt.Println("First turn:", t.players[t.firstTurn].id)
	roundScore := make(map[string]int)
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
		}
	}

	t.trump = t.cards[35]
	t.currentTurn = t.firstTurn
	t.firstTurn++

	if t.firstTurn == t.playersCount {
		t.firstTurn = 0
	}

	for i := range t.players {
		turn := t.currentTurn + i
		if turn >= t.playersCount {
			turn = 0
		}
		t.getBet(turn, t.inputCh)
	}

	for len(t.players[0].currentCards) > 0 {
		firstCardIndex := t.currentTurn
		t.onTable = make([]string, t.playersCount)
		for i := 0; i < t.playersCount; i++ {
			card := ""
			var cardIndex int
			for !t.cardPermissionToTable(t.onTable[firstCardIndex], card, t.currentTurn) {
				cardIndex = t.dropCard(t.currentTurn)
				if t.players[t.currentTurn].currentCards[cardIndex] == "♠1" {
					card = t.whatJokerMeans(t.currentTurn)
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
		}
		whosTurn := t.players[t.currentTurn].id
		whoWin := t.whoGetTheTable()
		roundScore[whoWin]++

		fmt.Printf("%s turn. Cards on TABLE: %s. Trump:%s. WINNER:%s\n", whosTurn, t.onTable, t.trump, whoWin)
		time.Sleep(1 * time.Second)
		t.onTable = nil

	}
	for id := range t.players {
		difference := roundScore[t.players[id].id] - t.players[id].bet
		if difference < 0 {
			t.players[id].score += difference * 10
		} else if difference == 0 {
			if t.players[id].bet == 0 {
				t.players[id].score += 5
			} else {
				t.players[id].score += difference * 10
			}
		} else {
			t.players[id].score += roundScore[t.players[id].id]
		}
	}
}

func (t *Table) getBet(player int, bet <-chan int) {
	t.mu.Lock()
	t.players[player].betFlag = true
	t.mu.Unlock()
	t.players[player].bet = <-bet
}

func (t *Table) dropCard(player int) (cardIndex int) {

	t.mu.Lock()
	t.players[player].turnFlag = true
	t.mu.Unlock()
	cardIndex = <-t.inputCh
	return
}

func (t *Table) whoGetTheTable() (id string) {
	maxCard := t.onTable[t.currentTurn]
	maxIndex := t.currentTurn
	for i := 1; i < len(t.onTable); i++ {
		index := t.currentTurn + i
		if index >= t.playersCount {
			index = 0
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

func (t *Table) whatJokerMeans(player int) string {
	t.mu.Lock()
	t.players[player].jokerFlag = true
	t.mu.Unlock()
	joker := <-t.inputCh //0-maxTrump, 1-♥maxHeart, 2-♦maxDiamond, 3-♣maxClub, 4-♠maxSpade, 5-♥minHeart, 6-♦minDiamond, 7-♣minClub, 8-♠minSpade
	switch joker {
	case 0:
		return (t.trump[0:3] + "9")
	case 1:
		return "♥9"
	case 2:
		return "♦9"
	case 3:
		return "♣9"
	case 4:
		return "♠9"
	case 5:
		return "♥"
	case 6:
		return "♦"
	case 7:
		return "♣"
	case 8:
		return "♠"
	default:
		// panic("bad joker")
	}
	return ""
}
