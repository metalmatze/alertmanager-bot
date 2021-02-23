package telegram

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/docker/libkv/store"
	"gopkg.in/tucnak/telebot.v2"
)

// ChatStore writes the users to a libkv store backend.
type ChatStore struct {
	kv             store.Store
	storeKeyPrefix string
}

// NewChatStore stores telegram chats in the provided kv backend.
func NewChatStore(kv store.Store, storeKeyPrefix string) (*ChatStore, error) {
	return &ChatStore{kv: kv, storeKeyPrefix: storeKeyPrefix}, nil
}

// List all chats saved in the kv backend.
func (s *ChatStore) List() ([]*telebot.Chat, error) {
	kvPairs, err := s.kv.List(s.storeKeyPrefix)
	if err != nil {
		return nil, err
	}

	var chats []*telebot.Chat
	for _, kv := range kvPairs {
		var c *telebot.Chat
		if err := json.Unmarshal(kv.Value, &c); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}

	return chats, nil
}

// Get a specific chat by its ID.
func (s *ChatStore) Get(id telebot.ChatID) (*telebot.Chat, error) {
	key := fmt.Sprintf("%s/%d", s.storeKeyPrefix, id)
	kv, err := s.kv.Get(key)
	if err != nil {
		if errors.Is(err, store.ErrKeyNotFound) {
			return nil, ChatNotFoundErr
		}
		return nil, err
	}
	var c *telebot.Chat
	err = json.Unmarshal(kv.Value, &c)
	return c, err
}

// Add a telegram chat to the kv backend.
func (s *ChatStore) Add(c *telebot.Chat) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%d", s.storeKeyPrefix, c.ID)

	return s.kv.Put(key, b, nil)
}

// Remove a telegram chat from the kv backend.
func (s *ChatStore) Remove(c *telebot.Chat) error {
	key := fmt.Sprintf("%s/%d", s.storeKeyPrefix, c.ID)
	return s.kv.Delete(key)
}
