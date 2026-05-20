package main

import (
	"AuthService/internal/config"
	"AuthService/internal/database"
	"AuthService/internal/handler"
	"AuthService/internal/repository/postgres"
	"AuthService/internal/service"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatal("connection to postgres failed: ", err)
	}
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal("closing database failed: ", err)
		}
	}(db)

	database.RunMigrations(cfg.DB)

	userRepo := postgres.NewUserRepo(db)
	tokenRepo := postgres.NewRefreshTokenRepo(db)

	tokenSvc := service.NewTokenService(
		cfg.JWT.Secret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	authSvc := service.NewAuthService(userRepo, tokenRepo, tokenSvc)

	h := handler.New(authSvc)

	log.Printf("Server starting on :%s", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, h.Router()); err != nil {
		log.Fatal("server:", err)
	}
}
