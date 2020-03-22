package telegram

import (
	"sync"

	"gopkg.in/tucnak/telebot.v2"
)

// ChatStore is a in-memory map based Chat store
type ChatStore struct {
	mu    sync.RWMutex
	chats map[int64]*telebot.Chat
}

// NewChatStore creates a new ChatStore that has a instantiated map
func NewChatStore() *ChatStore {
	return &ChatStore{
		chats: make(map[int64]*telebot.Chat),
	}
}

// List all chats
func (s *ChatStore) List() ([]*telebot.Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var chats []*telebot.Chat
	for _, chat := range s.chats {
		chats = append(chats, chat)
	}
	return chats, nil
}

// Add a single Chat to the store
func (s *ChatStore) Add(chat *telebot.Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chats[chat.ID] = chat
	return nil
}

// Remove a Chat by its ID from the store
func (s *ChatStore) Remove(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.chats, id)
	return nil
}
