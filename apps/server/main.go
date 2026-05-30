package main

import (
	"log"

	"github.com/cshaizhihao/OU-UI/internal/config"
	ouserver "github.com/cshaizhihao/OU-UI/internal/server"
	"github.com/cshaizhihao/OU-UI/internal/store"
)

func main() {
	cfg := config.LoadServer()

	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	ouserver.StartBackgroundJobs(cfg, db)
	router := ouserver.NewRouter(cfg, db)

	log.Printf("OU-UI server listening on %s", cfg.ListenAddr())
	if cfg.TLSEnabled() {
		log.Printf("HTTPS enabled with secure path %s", cfg.SecurePath)
		log.Fatal(router.RunTLS(cfg.ListenAddr(), cfg.TLSCertFile, cfg.TLSKeyFile))
	}

	log.Printf("HTTP enabled with secure path %s", cfg.SecurePath)
	log.Fatal(router.Run(cfg.ListenAddr()))
}
