package dto

import "time"

type CreateInviteRequest struct {
	TargetEmail      string `json:"targetEmail" binding:"required,email"`
	RoleName         string `json:"roleName" binding:"required,min=3,max=100"`
	OrganizationID   string `json:"organizationId" binding:"required,min=1,max=100"`
	OrganizationName string `json:"organizationName" binding:"required,min=1,max=255"`
}

type InviteResponse struct {
	ID               string    `json:"id"`
	TargetEmail      string    `json:"targetEmail"`
	RoleName         string    `json:"roleName"`
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	Sent             bool      `json:"sent"`
	Accepted         bool      `json:"accepted"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type AcceptInviteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Invite  any    `json:"invite,omitempty"`
}
