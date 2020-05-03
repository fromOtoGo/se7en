package main

import "errors"

type user struct {
	Name   string
	newMsg chan struct{}
	plr    player
}

//AllUsers holds all users
var AllUsers map[string]*user = make(map[string]*user)
var searchingGameUsers map[string]*user = make(map[string]*user)
var inGameUsers map[string]*user = make(map[string]*user)

//NewUser creates new user or return error if exists
func NewUser(name string) error {
	if _, ok := AllUsers[name]; ok {
		return errors.New("user already exists")
	}
	channel := make(chan struct{})
	newUser := user{Name: name, newMsg: channel}
	AllUsers[name] = &newUser
	return nil
}
