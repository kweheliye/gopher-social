package main

import (
	"github.com/kweheliye/gopher-social/internal/db"
	"github.com/kweheliye/gopher-social/internal/env"
	store2 "github.com/kweheliye/gopher-social/internal/store"
	"go.uber.org/zap"
	"log"
)

const (
	version = "0.0.1"
)

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
		env:  env.GetString("ENV", "dev"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", ""),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
	}

	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// Main Database
	db, err := db.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		log.Panic(err)
	}

	defer db.Close()
	log.Println("Connected to database")

	store := store2.NewStorage(db)

	app := &application{
		config: cfg,
		store:  store,
		logger: logger,
	}

	mux := app.mount()
	log.Fatal(app.run(mux))

}
