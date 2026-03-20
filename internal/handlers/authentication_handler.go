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

func NewAuthenticationHandler(service *service.AwsService) *AuthenticationHandler {
	return &AuthenticationHandler{
		AwsService: *service,
	}
}

// @Summary Attach a role to a user
// @Description attaches a role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Success 200  {object}  dto.AttachRoleResponse
// @Router /attach-role [post]
func (h AuthenticationHandler) AttachRole(c *gin.Context) {
	request := dto.AttachRoleRequest{}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, _ := h.AwsService.AttachRole(request)

	c.JSON(http.StatusOK, response)
}

// @Summary Get users for a facility
// @Description returns a list of facility users
// @Tags users
// @Accept json
// @Produce json
// @Param facility_id query string true "facility_id"
// @Param group query string true "group"
// @Success 200  {object}  []types.UserType
// @Router /get-facility-users [get]
func (h AuthenticationHandler) GetFacilityUsers(c *gin.Context) {
	facilityId := c.Query("facility_id")

	if facilityId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing facility_id in query param"})
		return
	}

	group := c.Query("group")

	if group == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing group in query param"})
		return
	}

	response, err := h.AwsService.GetFacilityUsers(facilityId,group)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
