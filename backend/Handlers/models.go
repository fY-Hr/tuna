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

// Profile is the public view of a user — safe to show to anyone.
type Profile struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Bio       string    `json:"bio"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// MeProfile is the owner's view of their own profile and may include
// private fields like email.
type MeProfile struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Bio       string    `json:"bio"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateProfileRequest uses pointers so "field absent" (leave
// unchanged) is distinguishable from "field set to empty" (clear it).
type UpdateProfileRequest struct {
	Bio       *string `json:"bio"`
	AvatarURL *string `json:"avatar_url"`
}
