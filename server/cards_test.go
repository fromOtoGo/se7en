package main

import (
	"testing"
)

type CalcScoreTest struct {
	Bet    int
	Score  int
	Result int
}

func TestCalcScore(t *testing.T) {
	Tbl := Table{}
	plr := player{name: "test"}
	Tbl.players = append(Tbl.players, &plr)
	cases := []CalcScoreTest{
		CalcScoreTest{
			Bet:    0,
			Score:  0,
			Result: 5,
		},
		CalcScoreTest{
			Bet:    0,
			Score:  8,
			Result: 8,
		},
		CalcScoreTest{
			Bet:    1,
			Score:  0,
			Result: -10,
		},
		CalcScoreTest{
			Bet:    1,
			Score:  1,
			Result: 10,
		},
		CalcScoreTest{
			Bet:    1,
			Score:  2,
			Result: 2,
		},
	}

	for caseNum, item := range cases {
		Tbl.players[0].score = 0
		Tbl.players[0].bet = item.Bet
		var score = make(map[int]int)
		score[0] = item.Score
		Tbl.calculateScore(score)
		if Tbl.players[0].score != item.Result {
			t.Errorf("Test failed in %v case. Got %v, expected %v",
				caseNum, Tbl.players[0].score, item.Result)
		}
	}
}

type JokerTest struct {
	Joker     int
	FirstTurn bool
	FirstCard bool
	Result    string
	ResultErr bool
}

func TestJokerMean(t *testing.T) {
	Tbl := Table{}
	plr := player{name: "test"}
	Tbl.players = append(Tbl.players, &plr)
	Tbl.trump = "♥5"

	cases := []JokerTest{
		JokerTest{
			Joker:     0,
			FirstTurn: false,
			Result:    "♥9",
		},
		JokerTest{
			Joker:     1,
			FirstTurn: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     1,
			FirstTurn: true,
			FirstCard: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     1,
			FirstTurn: true,
			FirstCard: true,
			ResultErr: false,
			Result:    "♥9",
		},
		JokerTest{
			Joker:     2,
			FirstTurn: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     2,
			FirstTurn: true,
			FirstCard: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     2,
			FirstTurn: true,
			FirstCard: true,
			ResultErr: false,
			Result:    "♦9",
		},
		JokerTest{
			Joker:     3,
			FirstTurn: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     3,
			FirstTurn: true,
			FirstCard: false,
			ResultErr: true,
		},
		JokerTest{ //case 10
			Joker:     3,
			FirstTurn: true,
			FirstCard: true,
			ResultErr: false,
			Result:    "♣9",
		},
		JokerTest{
			Joker:     4,
			FirstTurn: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     4,
			FirstTurn: true,
			FirstCard: false,
			ResultErr: true,
		},
		JokerTest{
			Joker:     4,
			FirstTurn: true,
			FirstCard: true,
			ResultErr: false,
			Result:    "♠9",
		},
		JokerTest{
			Joker:  5,
			Result: "♥",
		},
		JokerTest{
			Joker:  6,
			Result: "♦",
		},
		JokerTest{
			Joker:  7,
			Result: "♣",
		},
		JokerTest{
			Joker:  8,
			Result: "♠",
		},
		JokerTest{
			Joker:  9,
			Result: "♠",
		},
		JokerTest{
			Joker:     10,
			ResultErr: true,
		},
	}
	Tbl.players[0].inputCh = make(chan int)
	for caseNum, item := range cases {
		Tbl.players[0].currentCards = make([]string, 0)
		Tbl.players[0].currentCards = append(Tbl.players[0].currentCards, "♠2")
		if !item.FirstTurn {
			Tbl.onTable = append(Tbl.onTable, "♥3")
		} else {
			Tbl.onTable = make([]string, 0)
		}
		if !item.FirstCard {
			Tbl.cardsOnRound = 5
		} else {
			Tbl.cardsOnRound = 1
		}
		go func() {
			Tbl.players[0].inputCh <- item.Joker
		}()
		jok, err := Tbl.whatJokerMeans(0)
		if err == nil {
			if jok != item.Result {
				t.Errorf("Test failed in %v case. Got %v, expected %v",
					caseNum, jok, item.Result)
			}
		} else {
			if !item.ResultErr {
				t.Errorf("Test failed in %v case. Non expected error %v", caseNum, err)
			}
		}
	}
}

type Perm struct {
	Cards  []string
	Suit   string
	Card   string
	Result bool
}

func TestCardPerm(t *testing.T) {
	Tbl := Table{}
	plr1 := player{name: "test1"}
	Tbl.players = append(Tbl.players, &plr1)

	Tbl.trump = "♠5"

	cases := []Perm{
		Perm{},
		Perm{
			Card:   "♠1",
			Result: true,
		},
		Perm{
			Card:   "♣1",
			Suit:   "♣",
			Result: true,
		},
		Perm{
			Card:   "♣1",
			Suit:   "♦",
			Cards:  []string{"♣1", "♣2", "♦1"},
			Result: false,
		},
		Perm{
			Card:   "♠1",
			Suit:   "♦",
			Cards:  []string{"♣1", "♣2", "♠1"},
			Result: true,
		},
		Perm{
			Card:   "♣1",
			Suit:   "♦",
			Cards:  []string{"♣1", "♣2", "♠1"},
			Result: false,
		},
		Perm{
			Card:   "♣1",
			Suit:   "♦",
			Cards:  []string{"♣1", "♣2", "♠1"},
			Result: false,
		},
		Perm{
			Card:   "♣1",
			Suit:   "♦",
			Cards:  []string{"♣1", "♣2", "♥1"},
			Result: true,
		},
	}

	for caseNum, item := range cases {
		Tbl.players[0].currentCards = item.Cards
		if Tbl.cardPermissionToTable(item.Suit, item.Card, 0) != item.Result {
			t.Errorf("case %v, got: %v, expeceted %v", caseNum, !item.Result, item.Result)
		}
	}
}

type whoGet struct {
	turn   int
	table  []string
	result int
}

func TestWhoGetTable(t *testing.T) {
	Tbl := Table{}
	plr1 := player{name: "test1"}
	Tbl.players = append(Tbl.players, &plr1)
	plr2 := player{name: "test2"}
	Tbl.players = append(Tbl.players, &plr2)
	plr3 := player{name: "test3"}
	Tbl.players = append(Tbl.players, &plr3)
	plr4 := player{name: "test4"}
	Tbl.players = append(Tbl.players, &plr4)
	Tbl.playersCount = 4
	Tbl.trump = "♠5"

	cases := []whoGet{
		whoGet{
			turn:   0,
			table:  []string{"♥1", "♦1", "♠1", "♣1"},
			result: 2,
		},
	}

	for caseNum, item := range cases {
		Tbl.onTable = item.table
		Tbl.currentTurn = item.turn
		winner := Tbl.whoGetTheTable()
		if item.result != winner {
			t.Errorf("case %v, got: %v, expeceted %v", caseNum, winner, item.result)
		}
	}

}

// ♥ ♦ ♣ ♠
