package main

import (
	"github.com/kweheliye/gopher-social/internal/db"
	"github.com/kweheliye/gopher-social/internal/env"
	"github.com/kweheliye/gopher-social/internal/store"
	"log"
)

func main() {
	addr := env.GetString("DB_ADDR", "")
	conn, err := db.New(addr, 3, 3, "15m")
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	store := store.NewStorage(conn)

	db.Seed(store, conn)
}
