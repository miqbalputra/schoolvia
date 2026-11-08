package main

import (
	"context"
	"net/http"
)

type ctxKey string

const userKey ctxKey = "user"

const shareKey ctxKey = "share_sekolah"

func contextWithShare(r *http.Request, sekolahID int) context.Context {
	return context.WithValue(r.Context(), shareKey, sekolahID)
}

func shareFromContext(r *http.Request) int {
	if v, ok := r.Context().Value(shareKey).(int); ok {
		return v
	}
	return 0
}

type User struct {
	ID        int    `json:"id"`
	Nama      string `json:"nama"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	NIP       string `json:"nip"`
	SekolahID int    `json:"sekolah_id"`
}

func contextWithUser(r *http.Request, c *Claims) context.Context {
	return context.WithValue(r.Context(), userKey, c)
}

func userFromContext(r *http.Request) *Claims {
	if v, ok := r.Context().Value(userKey).(*Claims); ok {
		return v
	}
	return nil
}

var _ = context.Background
