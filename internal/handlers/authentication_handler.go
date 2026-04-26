package handlers

import (
	"errors"
	"net/http"
	"nrs-authentication/internal/config"
	"nrs-authentication/internal/dto"
	"nrs-authentication/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthenticationHandler struct {
	AwsService    service.AwsService
	InviteService service.InviteService
	Config        *config.Config
}

func NewAuthenticationHandler(service *service.AwsService, inviteService service.InviteService, cfg *config.Config) *AuthenticationHandler {
	return &AuthenticationHandler{
		AwsService:    *service,
		InviteService: inviteService,
		Config:        cfg,
	}
}

// @Summary Attach a role to a user
// @Description attaches a role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body dto.AttachRoleRequest true "Attach role request"
// @Success 200  {object}  dto.AttachRoleResponse
// @Router /me/attach-role [post]
func (h AuthenticationHandler) AttachRole(c *gin.Context) {
	request := dto.AttachRoleRequest{}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, _ := h.AwsService.AttachRole(request)

	c.JSON(http.StatusOK, response)
}

// @Summary Invite a user by email
// @Description Creates an invite, sends the email, and stores invite metadata.
// @Tags invites
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body dto.CreateInviteRequest true "Create invite request"
// @Security BearerAuth
// @Success 200  {object}  dto.InviteResponse
// @Failure 400  {object}  map[string]interface{}
// @Failure 401  {object}  map[string]interface{}
// @Failure 403  {object}  map[string]interface{}
// @Router /me/invites [post]
func (h AuthenticationHandler) CreateInvite(c *gin.Context) {
	_, ok := requireInviteAdmin(c, h.Config)
	if !ok {
		return
	}

	request := dto.CreateInviteRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userProfile, err := h.AwsService.GetUserProfile(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to fetch authenticated user from cognito"})
		return
	}

	hostEmail := getEmailFromCognitoAttributes(userProfile.UserAttributes)
	if hostEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "host email missing from cognito profile"})
		return
	}

	response, err := h.InviteService.CreateInvite(request, hostEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "invite": response})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary List pending invites
// @Description Returns all invites that have not been accepted.
// @Tags invites
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param organizationId query string true "Organization ID"
// @Security BearerAuth
// @Success 200  {object}  []dto.InviteResponse
// @Failure 400  {object}  map[string]interface{}
// @Failure 401  {object}  map[string]interface{}
// @Failure 403  {object}  map[string]interface{}
// @Router /me/invites/pending [get]
func (h AuthenticationHandler) ListPendingInvites(c *gin.Context) {
	_, ok := requireInviteAdmin(c, h.Config)
	if !ok {
		return
	}

	organizationID := c.Query("organizationId")
	if organizationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing organizationId in query param"})
		return
	}

	response, err := h.InviteService.ListPendingInvites(organizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Accept invite
// @Description Marks an invite as accepted.
// @Tags invites
// @Accept json
// @Produce json
// @Param inviteId path int true "Invite ID"
// @Success 200  {object}  dto.AcceptInviteResponse
// @Failure 400  {object}  dto.AcceptInviteResponse
// @Failure 404  {object}  dto.AcceptInviteResponse
// @Failure 409  {object}  dto.AcceptInviteResponse
// @Router /invites/{inviteId}/accept [get]
func (h AuthenticationHandler) AcceptInvite(c *gin.Context) {
	inviteID := c.Param("inviteId")
	if inviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inviteId"})
		return
	}

	invite, err := h.InviteService.AcceptInvite(inviteID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrInviteNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, service.ErrInviteAccepted) {
			status = http.StatusConflict
		}
		c.JSON(status, dto.AcceptInviteResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.AcceptInviteResponse{
		Success: true,
		Message: "Invite accepted",
		Invite:  invite,
	})
}

// @Summary Attach role from invite
// @Description Attaches the invited role after the invite is accepted.
// @Tags invites
// @Accept json
// @Produce json
// @Param inviteId path int true "Invite ID"
// @Success 200  {object}  dto.AttachRoleResponse
// @Failure 400  {object}  dto.AttachRoleResponse
// @Failure 404  {object}  dto.AttachRoleResponse
// @Failure 409  {object}  dto.AttachRoleResponse
// @Router /invites/{inviteId}/attach-role [post]
func (h AuthenticationHandler) AttachRoleByInvite(c *gin.Context) {
	inviteID := c.Param("inviteId")
	if inviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inviteId"})
		return
	}

	response, err := h.InviteService.AttachRoleByInviteID(inviteID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrInviteNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, service.ErrInviteNotAccepted) {
			status = http.StatusConflict
		}
		c.JSON(status, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Get users for a facility
// @Description returns a list of facility users
// @Tags users
// @Accept json
// @Produce json
// @Param facility_code query string true "facility_code"
// @Param group query string true "group"
// @Success 200  {object}  []dto.UserTypeSwagger
// @Router /me/get-facility-users [get]
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

	response, err := h.AwsService.GetFacilityUsers(facilityCode, group)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Get user by email
// @Description Returns a user by email.
// @Tags users
// @Accept json
// @Produce json
// @Param email query string true "Email"
// @Success 200  {object}  dto.ListUsersOutputSwagger  "User lookup result."
// @Router /get-user [get]
func (h AuthenticationHandler) GetUser(c *gin.Context) {
	username := c.Query("email")

	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing email in query param"})
		return
	}

	response, err := h.AwsService.GetUser(username)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
