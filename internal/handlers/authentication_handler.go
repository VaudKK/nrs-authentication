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
// @Param facility_code query string true "facility_code"
// @Param group query string true "group"
// @Success 200  {object}  []types.UserType
// @Router /get-facility-users [get]
func (h AuthenticationHandler) GetFacilityUsers(c *gin.Context) {
	facilityCode := c.Query("facility_code")

	if facilityCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing facility_code in query param"})
		return
	}

	group := c.Query("group")

	if group == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing group in query param"})
		return
	}

	response, err := h.AwsService.GetFacilityUsers(facilityCode,group)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
