package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Invite struct {
	ID               string    `gorm:"type:uuid;primaryKey" json:"id"`
	TargetEmail      string    `gorm:"size:255;not null;index" json:"targetEmail"`
	HostEmail        string    `gorm:"size:255;not null" json:"hostEmail"`
	RoleName         string    `gorm:"size:100;not null" json:"roleName"`
	OrganizationID   string    `gorm:"size:100;not null;index" json:"organizationId"`
	OrganizationName string    `gorm:"size:255;not null" json:"organizationName"`
	Sent             bool      `gorm:"not null;default:false" json:"sent"`
	Accepted         bool      `gorm:"not null;default:false" json:"accepted"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (i *Invite) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}

	return nil
}
