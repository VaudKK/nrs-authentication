package dto

// attach role request
type AttachRoleRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=100"`
	GroupName string `json:"groupName" binding:"required,min=3,max=100"`
}

// attach role response
type AttachRoleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
