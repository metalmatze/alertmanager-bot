package telegram

import (
	bolt "go.etcd.io/bbolt"
)

// Bolt implements a telegram specific storage
type Bolt struct {
	db *bolt.DB
}

// NewBolt creates the new telegram specific storage
func NewBolt(db *bolt.DB) *Bolt {
	return &Bolt{db}
}
