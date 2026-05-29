package store

import (
	"github.com/cshaizhihao/OU-UI/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&models.Agent{}, &models.Task{}, &models.AuditLog{}); err != nil {
		return nil, err
	}
	return db, nil
}
