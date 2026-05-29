package store

import (
	"github.com/cshaizhihao/OU-UI/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&models.Agent{},
		&models.Task{},
		&models.Node{},
		&models.AuditLog{},
		&models.NodeTrafficSample{},
		&models.RoutingRule{},
		&models.LoadBalancerGroup{},
		&models.WebhookEndpoint{},
		&models.AlertEvent{},
		&models.ExternalSubscription{},
		&models.ExternalNode{},
		&models.ClashProfile{},
		&models.Tenant{},
		&models.PanelUser{},
		&models.APIKey{},
		&models.CopilotIncident{},
	); err != nil {
		return nil, err
	}
	return db, nil
}
