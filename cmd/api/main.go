package main

import (
	"github.com/kweheliye/gopher-social/internal/auth"
	"github.com/kweheliye/gopher-social/internal/db"
	"github.com/kweheliye/gopher-social/internal/env"
	"github.com/kweheliye/gopher-social/internal/mailer"
	store2 "github.com/kweheliye/gopher-social/internal/store"
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
				secret: env.GetString("AUTH_TOKEN_SECRET", "example"),
				exp:    time.Hour * 24 * 3, // 3 days
				iss:    "gophersocial",
			},
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
	logger.Info("Connected to database")

	store := store2.NewStorage(db)

	// Initialize mailer with fallback mechanism
	var mailClient mailer.Client
	if cfg.mail.sendGrid.apiKey != "" {
		mailClient = mailer.NewSendgrid(cfg.mail.sendGrid.apiKey, cfg.mail.fromEmail)
		logger.Info("SendGrid mailer initialized")
	}

	// If SendGrid API key is not provided or as a fallback, try MailTrap
	if mailClient == nil || cfg.mail.mailTrap.apiKey != "" {
		if cfg.mail.mailTrap.apiKey != "" {
			var mailTrapErr error
			mailTrapClient, mailTrapErr := mailer.NewMailTrapClient(cfg.mail.mailTrap.apiKey, cfg.mail.fromEmail)
			if mailTrapErr != nil {
				logger.Warnw("Failed to initialize MailTrap client", "error", mailTrapErr)
			} else {
				// If SendGrid is not configured or we want to use MailTrap as primary
				if mailClient == nil {
					mailClient = mailTrapClient
					logger.Info("MailTrap mailer initialized as primary")
				} else {
					// Store MailTrap as a fallback option that will be used in auth.go if SendGrid fails
					logger.Info("MailTrap mailer initialized as fallback")
				}
			}
		}
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

	app := &application{
		config:        cfg,
		store:         store,
		logger:        logger,
		mailer:        mailClient,
		authenticator: jwtAuthenticator,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))

}
