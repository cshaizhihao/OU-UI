package server

import (
	"time"

	"github.com/cshaizhihao/OU-UI/internal/config"
	"github.com/cshaizhihao/OU-UI/internal/providers"
	"gorm.io/gorm"
)

func StartBackgroundJobs(cfg config.ServerConfig, db *gorm.DB) {
	h := Handler{cfg: cfg, db: db, registry: providers.DefaultRegistry()}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		h.runMaintenanceSweep()
		for range ticker.C {
			h.runMaintenanceSweep()
		}
	}()
}
