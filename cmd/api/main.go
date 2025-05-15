package main

import (
	"github.com/go-redis/redis/v8"
	"github.com/kweheliye/gopher-social/internal/auth"
	"github.com/kweheliye/gopher-social/internal/db"
	"github.com/kweheliye/gopher-social/internal/env"
	"github.com/kweheliye/gopher-social/internal/mailer"
	"github.com/kweheliye/gopher-social/internal/ratelimiter"
	store2 "github.com/kweheliye/gopher-social/internal/store"
	"github.com/kweheliye/gopher-social/internal/store/cache"
	"go.uber.org/zap"
	"time"
)

const (
	version = "0.0.2"
)

//	@title			GopherSocial API
//	@description	API for GopherSocial, a social network for gohpers
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath					/v1
//
// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description
func main() {
	cfg := config{
		apiURL:      env.GetString("EXTERNAL_URL", "localhost:8080"),
		addr:        env.GetString("ADDR", ":8080"),
		frontendURL: env.GetString("FRONTEND_URL", "http://localhost:3000"),
		env:         env.GetString("ENV", "dev"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", ""),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		redisCfg: redisConfig{
			addr:    env.GetString("REDIS_ADDR", "localhost:6379"),
			pw:      env.GetString("REDIS_PW", ""),
			db:      env.GetInt("REDIS_DB", 0),
			enabled: env.GetBool("REDIS_ENABLED", true),
		},
		mail: mailConfig{
			exp:       time.Hour * 24 * 3, // 3 days
			fromEmail: env.GetString("FROM_EMAIL", ""),
			sendGrid: sendGridConfig{
				apiKey: env.GetString("SENDGRID_API_KEY", ""),
			},
			mailTrap: mailTrapConfig{
				apiKey: env.GetString("MAILTRAP_API_KEY", ""),
			},
		},
		auth: authConfig{
			basic: basicConfig{
				user: env.GetString("AUTH_BASIC_USER", ""),
				pass: env.GetString("AUTH_BASIC_PASS", ""),
			},
			token: tokenConfig{
				secret: env.GetString("AUTH_TOKEN_SECRET", "test"),
				exp:    time.Hour * 24 * 3, // 3 days
				iss:    "gophersocial",
			},
		},
		rateLimiter: ratelimiter.Config{
			RequestsPerTimeFrame: env.GetInt("RATELIMITER_REQUESTS_COUNT", 20),
			TimeFrame:            time.Second * 5,
			Enabled:              env.GetBool("RATE_LIMITER_ENABLED", true),
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
		logger.Panic(err)
	}

	defer db.Close()
	logger.Info("database connection pool established")

	// Cache
	var rdb *redis.Client
	if cfg.redisCfg.enabled {
		rdb = cache.NewRedisClient(cfg.redisCfg.addr, cfg.redisCfg.pw, cfg.redisCfg.db)
		logger.Info("redis cache connection established")
		defer rdb.Close()
	}

	// Rate limiter
	rateLimiter := ratelimiter.NewFixedWindowLimiter(
		cfg.rateLimiter.RequestsPerTimeFrame,
		cfg.rateLimiter.TimeFrame,
	)

	var mailClient mailer.Client
	if cfg.mail.sendGrid.apiKey != "" {
		mailClient = mailer.NewSendgrid(cfg.mail.sendGrid.apiKey, cfg.mail.fromEmail)
		logger.Info("SendGrid mailer initialized")
	}

	if mailClient == nil {
		logger.Warn("No mail client configured, emails will not be sent")
	}

	// Authenticator
	jwtAuthenticator := auth.NewJWTAuthenticator(
		cfg.auth.token.secret,
		cfg.auth.token.iss,
		cfg.auth.token.iss,
	)

	store := store2.NewStorage(db)
	cacheStorage := cache.NewRedisStorage(rdb)

	app := &application{
		config:        cfg,
		store:         store,
		logger:        logger,
		mailer:        mailClient,
		authenticator: jwtAuthenticator,
		cacheStorage:  cacheStorage,
		rateLimiter:   rateLimiter,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))

}
