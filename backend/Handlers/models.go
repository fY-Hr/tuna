package Handlers

import (
	"time"
)

type Wave struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	IsArchive bool      `json:"is_archive"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Reaction struct {
	Value int16 `json:"value"`
}
