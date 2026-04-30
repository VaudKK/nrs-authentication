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
		if errors.Is(err, service.ErrOrganizationNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "invite": response})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary List organization members
// @Description Returns the members of an organization for authorized admin roles.
// @Tags organizations
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param organizationId query string true "Organization ID"
// @Security BearerAuth
// @Success 200  {object}  []dto.UserOrganizationMappingResponse
// @Failure 400  {object}  map[string]interface{}
// @Failure 401  {object}  map[string]interface{}
// @Failure 403  {object}  map[string]interface{}
// @Router /me/organizations/members [get]
func (h AuthenticationHandler) ListOrganizationMembers(c *gin.Context) {
	_, ok := requireInviteAdmin(c, h.Config)
	if !ok {
		return
	}

	organizationID := c.Query("organizationId")
	if organizationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing organizationId in query param"})
		return
	}

	response, err := h.InviteService.ListOrganizationMembers(organizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
// @Param inviteId path string true "Invite ID"
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

// @Summary List authenticated user's organizations
// @Description Returns the organizations the authenticated user belongs to.
// @Tags organizations
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Security BearerAuth
// @Success 200  {object}  []dto.UserOrganizationMappingResponse
// @Failure 400  {object}  map[string]interface{}
// @Failure 401  {object}  map[string]interface{}
// @Router /me/organizations [get]
func (h AuthenticationHandler) ListMyOrganizations(c *gin.Context) {
	_, ok := requireAuthenticated(c, h.Config)
	if !ok {
		return
	}

	userProfile, err := h.AwsService.GetUserProfile(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to fetch authenticated user from cognito"})
		return
	}

	userEmail := getEmailFromCognitoAttributes(userProfile.UserAttributes)
	if userEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user email missing from cognito profile"})
		return
	}

	response, err := h.InviteService.ListUserOrganizationsByEmail(userEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Attach role from invite
// @Description Attaches the invited role after the invite is accepted.
// @Tags invites
// @Accept json
// @Produce json
// @Param inviteId path string true "Invite ID"
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

// @Summary Get user by email
// @Description Fetches a user from Cognito by email. Primarily for testing purposes.
// @Tags users
// @Accept json
// @Produce json
// @Param email query string true "User email"
// @Success 200  {object}  dto.CheckEmailResponse
// @Failure 400  {object}  map[string]interface{}
// @Router /check-email [get]
func (h AuthenticationHandler) CheckEmail(c *gin.Context) {

	email := c.Query("email")

	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter is required"})
		return
	}

	response, err := h.AwsService.GetUser(email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MapCheckEmailResponse(response))
}
