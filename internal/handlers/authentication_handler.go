package handlers

import (
	"net/http"
	"nrs-authentication/internal/dto"
	"nrs-authentication/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthenticationHandler struct {
	AwsService service.AwsService
}


func NewAuthenticationHandler(service *service.AwsService) *AuthenticationHandler{
	return &AuthenticationHandler{
		AwsService: *service,
	}
}


func (h AuthenticationHandler) AttachRole(c *gin.Context){
	request := dto.AttachRoleRequest{}


	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest,gin.H{"error": err.Error()})
		return
	}

	response, _ := h.AwsService.AttachRole(request)

	c.JSON(http.StatusOK,response)
}