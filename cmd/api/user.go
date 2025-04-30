package main

import (
	"github.com/kweheliye/gopher-social/internal/store"
	"net/http"
)

type userKey string

const userCtx userKey = "user"

type User struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func getUserFromContext(r *http.Request) *store.User {
	user, _ := r.Context().Value(userCtx).(*store.User)
	return user
}
