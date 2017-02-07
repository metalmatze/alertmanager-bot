package main

import (
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/tucnak/telebot"
	yaml "gopkg.in/yaml.v2"
)

const fileMode = 0600

// UserStore writes the users to a file for persistence
type UserStore struct {
	mu    sync.Mutex
	file  string
	users map[int]telebot.User
}

// NewUserStore from a filename and loading the contents if there is
func NewUserStore(file string) (*UserStore, error) {
	store := &UserStore{
		file:  file,
		users: make(map[int]telebot.User),
	}

	// If file for storing not present create it
	_, err := os.Stat(store.file)
	if err != nil {
		_, err := os.Create(store.file)
		if err != nil {
			return nil, err
		}
		log.Printf("created %s", store.file)
	}

	usersBytes, err := ioutil.ReadFile(store.file)
	if err != nil {
		return store, err
	}

	store.mu.Lock()
	if err := yaml.Unmarshal(usersBytes, &store.users); err != nil {
		return store, err
	}
	store.mu.Unlock()

	return store, nil
}

// Len returns the users count
func (s *UserStore) Len() int {
	return len(s.users)
}

// List all users as slice to range over
func (s *UserStore) List() []telebot.User {
	var users []telebot.User
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

// Add a telebot User to the store and write the current users to disk
func (s *UserStore) Add(u telebot.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users[u.ID] = u

	out, err := yaml.Marshal(s.users)
	if err != nil {
		return err
	}
	ioutil.WriteFile(s.file, out, fileMode)

	return nil
}

// Remove a telebot User from the store and write the current users to disk
func (s *UserStore) Remove(u telebot.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, u.ID)

	out, err := yaml.Marshal(s.users)
	if err != nil {
		return err
	}

	ioutil.WriteFile(s.file, out, fileMode)

	return nil
}
