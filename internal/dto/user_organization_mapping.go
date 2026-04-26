package dto

import "time"

type UserOrganizationMappingResponse struct {
	ID               string    `json:"id"`
	UserEmail        string    `json:"userEmail"`
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	RoleName         string    `json:"roleName"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
