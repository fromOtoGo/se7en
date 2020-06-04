package server

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

type user struct {
	mu         sync.RWMutex
	client     *websocket.Conn
	Name       string
	newMsg     chan struct{}
	redirectTo chan int
	plr        player
}

type users struct {
	mu     sync.RWMutex
	client *websocket.Conn
	Users  map[string]*user
}

//AllUsers holds all users
var AllUsers = users{Users: make(map[string]*user)}

var searchingGameUsers = users{Users: make(map[string]*user)}
var inGameUsers map[string]*user = make(map[string]*user)

//NewUser creates new user or return error if exists
func NewUser(name string) error {
	AllUsers.mu.Lock()
	defer AllUsers.mu.Unlock()
	if _, ok := AllUsers.Users[name]; ok {
		return errors.New("user already exists")
	}
	channel := make(chan struct{})
	newUser := user{Name: name, newMsg: channel, redirectTo: make(chan int)}
	AllUsers.Users[name] = &newUser
	return nil
}
