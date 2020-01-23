package telegram

import (
	bolt "go.etcd.io/bbolt"
)

type Bolt struct {
	db *bolt.DB
}

func NewBolt(db *bolt.DB) *Bolt {
	return &Bolt{db}
}
