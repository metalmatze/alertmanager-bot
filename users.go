package main

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/tucnak/telebot"
	yaml "gopkg.in/yaml.v2"
)

type UserStore struct {
	mu    sync.RWMutex
	file  string
	users map[int]telebot.User
}

func NewUserStore(file string) error {
	store := UserStore{
		file:  file,
		users: make(map[int]telebot.User),
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	usersBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	store.mu.Lock()
	if err := yaml.Unmarshal(usersBytes, &store.users); err != nil {
		return err
	}
	store.mu.Unlock()

	return nil
}

func (s *UserStore) Add(u telebot.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[u.ID] = u

	return nil
}

func (s *UserStore) Delete(uID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, uID)

	return nil
}
