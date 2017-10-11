package main

import (
	"encoding/json"
	"fmt"

	"github.com/docker/libkv/store"
	"github.com/tucnak/telebot"
)

const telegramUsersDirectory = "telegram/users"

// UserStore writes the users to a libkv store backend
type UserStore struct {
	kv store.Store
}

// NewUserStore from a filename and loading the contents if there is
func NewUserStore(kv store.Store) (*UserStore, error) {
	return &UserStore{kv: kv}, nil
}

// Len returns the users count
func (s *UserStore) Len() int {
	list, err := s.kv.List(telegramUsersDirectory)
	if err != nil {
		return -1
	}
	return len(list)
}

// List all users as slice to range over
func (s *UserStore) List() []telebot.User {
	kvPairs, err := s.kv.List(telegramUsersDirectory)
	if err != nil {
		return nil
	}

	var users []telebot.User
	for _, kv := range kvPairs {
		var u telebot.User
		if err := json.Unmarshal(kv.Value, &u); err != nil {
			break
		}
		users = append(users, u)
	}

	return users
}

// Add a telebot User to the store
func (s *UserStore) Add(u telebot.User) error {
	b, err := json.Marshal(u)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%d", telegramUsersDirectory, u.ID)

	return s.kv.Put(key, b, nil)
}

// Remove a telebot User from the store
func (s *UserStore) Remove(u telebot.User) error {
	key := fmt.Sprintf("%s/%d", telegramUsersDirectory, u.ID)
	return s.kv.Delete(key)
}
