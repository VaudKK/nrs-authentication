package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserOrganizationMapping struct {
	ID               string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserEmail        string    `gorm:"size:255;not null;index:idx_user_org,unique" json:"userEmail"`
	OrganizationID   string    `gorm:"size:100;not null;index:idx_user_org,unique" json:"organizationId"`
	OrganizationName string    `gorm:"size:255;not null" json:"organizationName"`
	RoleName         string    `gorm:"size:100;not null" json:"roleName"`
	Active           bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (m *UserOrganizationMapping) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}

	return nil
}
