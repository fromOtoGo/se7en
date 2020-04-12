package main

import (
	"fmt"
	"strconv"
	"time"
)

type Player struct {
	id           string
	currentCards []string
	bet          int
	score        int
}

type Table struct {
	cards            [36]string
	onTable          []string
	playersCount     int
	firstTurn        int
	currentTurn      int
	players          []Player
	maxCardsToPlayer int
	trump            string
}

func NewTable() *Table {
	newT := Table{}
	// newT.players = make(map[string]Player)
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
	fmt.Println("Table created")
	return &newT
}

func (t *Table) Join(pID string) {
	fmt.Println("Join", pID)
	t.playersCount++
	t.players = append(t.players, Player{id: pID})
}

func (t *Table) Start() {
	if t.playersCount > 1 {
		t.maxCardsToPlayer = 36 / t.playersCount
		//start new server and redirect
	}
	for i := 0; i < t.maxCardsToPlayer; i++ {
		t.Round(i + 1)
	}
	for i := t.maxCardsToPlayer; i > 0; i-- {
		t.Round(i)
	}
	for id := range t.players {
		fmt.Println(t.players[id].id, t.players[id].score)
	}
}

func (t *Table) Round(round int) {
	fmt.Println()
	fmt.Print("ROUND ", round, " ")
	fmt.Println("First turn:", t.players[t.firstTurn].id)
	roundScore := make(map[string]int)
	for key := range roundScore {
		roundScore[key] = 0
	}
	t.trump = ""
	time.Sleep(1 * time.Second)

	if round <= t.maxCardsToPlayer {
		count := 0
		for i := range t.players {
			t.players[i].currentCards = t.cards[count*round : count*round+round]
			count++
		}
	}

	t.currentTurn = t.firstTurn
	t.firstTurn++
	if t.firstTurn == t.playersCount {
		t.firstTurn = 0
	}
	t.trump = t.players[0].currentCards[0]
	time.Sleep(1 * time.Second)
	for len(t.players[0].currentCards) > 0 {
		for i := 0; i < t.playersCount; i++ {
			t.onTable = append(t.onTable, t.DropCard())
		}
		whosTurn := t.players[t.currentTurn].id
		whoWin := t.WhoGetTheTable()
		roundScore[whoWin]++

		fmt.Printf("%s turn. Cards on TABLE: %s. Trump:%s. WINNER:%s\n", whosTurn, t.onTable, t.trump, whoWin)
		time.Sleep(1 * time.Second)
		t.onTable = nil

	}
	for id := range t.players {
		t.players[id].score += roundScore[t.players[id].id]
	}
}

func (t *Table) DropCard() (toTable string) {
	toTable = t.players[t.currentTurn].currentCards[0]
	t.players[t.currentTurn].currentCards = t.players[t.currentTurn].currentCards[1:]
	t.currentTurn++
	if t.currentTurn == t.playersCount {
		t.currentTurn = 0
	}
	return
}

func (t *Table) WhoGetTheTable() (id string) {
	maxCard := t.onTable[0]
	maxIndex := 0
	for i := 1; i < len(t.onTable); i++ {
		currentCard := t.onTable[i]
		if maxCard[0:3] != currentCard[0:3] {
			if currentCard[0:3] != t.trump[0:3] {
				continue
			}
			maxCard = currentCard
			maxIndex = i
		} else {
			if maxCard < currentCard {
				maxCard = currentCard
				maxIndex = i
			}
		}

	}

	winIDIndex := t.currentTurn + maxIndex
	if winIDIndex >= t.playersCount {
		winIDIndex -= t.playersCount
	}
	id = t.players[winIDIndex].id
	t.currentTurn = winIDIndex

	return
}
