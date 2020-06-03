package server

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

type user struct {
	mu     sync.RWMutex
	client *websocket.Conn
	Name   string
	newMsg chan struct{}
	plr    player
}

type users struct {
	mu    sync.RWMutex
	Users map[string]*user
}

//AllUsers holds all users
var AllUsers = users{Users: make(map[string]*user)}

var searchingGameUsers = users{Users: make(map[string]*user)}
var inGameUsers map[string]*user = make(map[string]*user)

//NewUser creates new user or return error if exists
func NewUser(name string) error {
	if _, ok := AllUsers.Users[name]; ok {
		return errors.New("user already exists")
	}
	channel := make(chan struct{})
	newUser := user{Name: name, newMsg: channel}
	AllUsers.Users[name] = &newUser
	return nil
}
