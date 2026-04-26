package service

import (
	"errors"
	"fmt"
	"nrs-authentication/internal/config"
	"nrs-authentication/internal/dto"
	"nrs-authentication/internal/mailer"
	"nrs-authentication/internal/model"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ErrInviteNotFound       = errors.New("invite not found")
	ErrInviteNotAccepted    = errors.New("invite has not been accepted")
	ErrInviteAccepted       = errors.New("invite has already been accepted")
	ErrOrganizationNotFound = errors.New("organization not found")
)

type InviteService interface {
	CreateInvite(dto.CreateInviteRequest, string) (dto.InviteResponse, error)
	ListPendingInvites(string) ([]dto.InviteResponse, error)
	ListUserOrganizationsByEmail(string) ([]dto.UserOrganizationMappingResponse, error)
	ListOrganizationMembers(string) ([]dto.UserOrganizationMappingResponse, error)
	AcceptInvite(string) (dto.InviteResponse, error)
	AttachRoleByInviteID(string) (dto.AttachRoleResponse, error)
}

type inviteService struct {
	db         *gorm.DB
	log        *logrus.Logger
	mailer     mailer.Mailer
	config     *config.Config
	awsService AwsService
}

type inviteTemplateData struct {
	Name         string
	Host         string
	Organization string
	JoinLink     string
}

type organizationRecord struct {
	ID   string `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

func NewInviteService(cfg *config.Config, log *logrus.Logger, awsService AwsService) (InviteService, error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		sslMode := cfg.PostgresSSLMode
		if sslMode == "" {
			sslMode = "disable"
		}

		if cfg.PostgresHost == "" || cfg.PostgresUser == "" || cfg.PostgresDB == "" || cfg.PostgresPort == 0 {
			return nil, errors.New("postgres configuration is incomplete")
		}

		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.PostgresHost,
			cfg.PostgresPort,
			cfg.PostgresUser,
			cfg.PostgresPassword,
			cfg.PostgresDB,
			sslMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Invite{}, &model.UserOrganizationMapping{}); err != nil {
		return nil, err
	}

	m := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPSender, log)

	return &inviteService{
		db:         db,
		log:        log,
		mailer:     m,
		config:     cfg,
		awsService: awsService,
	}, nil
}

func (s *inviteService) CreateInvite(request dto.CreateInviteRequest, hostEmail string) (dto.InviteResponse, error) {
	organization, err := s.findOrganizationByID(request.OrganizationID)
	if err != nil {
		return dto.InviteResponse{}, err
	}

	invite := model.Invite{
		TargetEmail:      strings.TrimSpace(strings.ToLower(request.TargetEmail)),
		HostEmail:        strings.TrimSpace(strings.ToLower(hostEmail)),
		RoleName:         strings.TrimSpace(request.RoleName),
		OrganizationID:   strings.TrimSpace(organization.ID),
		OrganizationName: strings.TrimSpace(organization.Name),
	}

	if err := s.db.Create(&invite).Error; err != nil {
		return dto.InviteResponse{}, err
	}

	joinLink := s.buildInviteJoinLink(invite.ID)
	data := inviteTemplateData{
		Name:         invite.TargetEmail,
		Host:         invite.HostEmail,
		Organization: invite.OrganizationName,
		JoinLink:     joinLink,
	}

	sendErr := s.mailer.Send(invite.TargetEmail, "invite.tmpl", data, false)
	if sendErr == nil {
		invite.Sent = true
		if err := s.db.Save(&invite).Error; err != nil {
			return dto.InviteResponse{}, err
		}
	} else {
		s.log.WithError(sendErr).Error("Error while sending invite email")
	}

	return mapInvite(invite), sendErr
}

func (s *inviteService) ListPendingInvites(organizationID string) ([]dto.InviteResponse, error) {
	var invites []model.Invite
	if err := s.db.Where("accepted = ? AND organization_id = ?", false, strings.TrimSpace(organizationID)).Order("created_at desc").Find(&invites).Error; err != nil {
		return nil, err
	}

	response := make([]dto.InviteResponse, 0, len(invites))
	for _, invite := range invites {
		response = append(response, mapInvite(invite))
	}

	return response, nil
}

func (s *inviteService) ListUserOrganizationsByEmail(userEmail string) ([]dto.UserOrganizationMappingResponse, error) {
	var mappings []model.UserOrganizationMapping
	if err := s.db.Where("user_email = ?", strings.TrimSpace(strings.ToLower(userEmail))).Order("organization_name asc").Find(&mappings).Error; err != nil {
		return nil, err
	}

	response := make([]dto.UserOrganizationMappingResponse, 0, len(mappings))
	for _, mapping := range mappings {
		response = append(response, mapUserOrganizationMapping(mapping))
	}

	return response, nil
}

func (s *inviteService) ListOrganizationMembers(organizationID string) ([]dto.UserOrganizationMappingResponse, error) {
	var mappings []model.UserOrganizationMapping
	if err := s.db.Where("organization_id = ?", strings.TrimSpace(organizationID)).Order("user_email asc").Find(&mappings).Error; err != nil {
		return nil, err
	}

	response := make([]dto.UserOrganizationMappingResponse, 0, len(mappings))
	for _, mapping := range mappings {
		response = append(response, mapUserOrganizationMapping(mapping))
	}

	return response, nil
}

func (s *inviteService) AcceptInvite(inviteID string) (dto.InviteResponse, error) {
	invite, err := s.findInvite(inviteID)
	if err != nil {
		return dto.InviteResponse{}, err
	}

	if invite.Accepted {
		return dto.InviteResponse{}, ErrInviteAccepted
	}

	invite.Accepted = true
	if err := s.db.Save(&invite).Error; err != nil {
		return dto.InviteResponse{}, err
	}

	return mapInvite(invite), nil
}

func (s *inviteService) AttachRoleByInviteID(inviteID string) (dto.AttachRoleResponse, error) {
	invite, err := s.findInvite(inviteID)
	if err != nil {
		return dto.AttachRoleResponse{Success: false, Message: err.Error()}, err
	}

	if !invite.Accepted {
		return dto.AttachRoleResponse{Success: false, Message: ErrInviteNotAccepted.Error()}, ErrInviteNotAccepted
	}

	user, err := s.awsService.GetUser(invite.TargetEmail)
	if err != nil {
		return dto.AttachRoleResponse{Success: false, Message: "user lookup failed"}, err
	}

	if len(user.Users) == 0 || user.Users[0].Username == nil || strings.TrimSpace(*user.Users[0].Username) == "" {
		return dto.AttachRoleResponse{Success: false, Message: "invited user not found"}, errors.New("invited user not found")
	}

	response, err := s.awsService.AttachRole(dto.AttachRoleRequest{
		Username:  strings.TrimSpace(*user.Users[0].Username),
		GroupName: invite.RoleName,
	})
	if err != nil {
		return response, err
	}

	if err := s.upsertUserOrganizationMapping(invite); err != nil {
		s.log.WithError(err).Error("Error while saving user organization mapping")
		return dto.AttachRoleResponse{
			Success: false,
			Message: "role attached but failed to save organization mapping",
		}, err
	}

	return response, nil
}

func (s *inviteService) findInvite(inviteID string) (model.Invite, error) {
	var invite model.Invite
	if err := s.db.Where("id = ?", strings.TrimSpace(inviteID)).First(&invite).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Invite{}, ErrInviteNotFound
		}
		return model.Invite{}, err
	}
	return invite, nil
}

func (s *inviteService) buildInviteJoinLink(inviteID string) string {
	base := s.config.InviteURL
	if base == "" {
		base = fmt.Sprintf("http://localhost:%s", s.config.Port)
	}

	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s", base, inviteID)
}

func (s *inviteService) findOrganizationByID(organizationID string) (organizationRecord, error) {
	var organization organizationRecord
	err := s.db.Table("organizations").
		Select("id", "name").
		Where("id = ?", strings.TrimSpace(organizationID)).
		First(&organization).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return organizationRecord{}, ErrOrganizationNotFound
		}
		return organizationRecord{}, err
	}

	return organization, nil
}

func mapInvite(invite model.Invite) dto.InviteResponse {
	return dto.InviteResponse{
		ID:               invite.ID,
		TargetEmail:      invite.TargetEmail,
		RoleName:         invite.RoleName,
		OrganizationID:   invite.OrganizationID,
		OrganizationName: invite.OrganizationName,
		Sent:             invite.Sent,
		Accepted:         invite.Accepted,
		CreatedAt:        invite.CreatedAt,
		UpdatedAt:        invite.UpdatedAt,
	}
}

func mapUserOrganizationMapping(mapping model.UserOrganizationMapping) dto.UserOrganizationMappingResponse {
	return dto.UserOrganizationMappingResponse{
		ID:               mapping.ID,
		UserEmail:        mapping.UserEmail,
		OrganizationID:   mapping.OrganizationID,
		OrganizationName: mapping.OrganizationName,
		RoleName:         mapping.RoleName,
		Active:           mapping.Active,
		CreatedAt:        mapping.CreatedAt,
		UpdatedAt:        mapping.UpdatedAt,
	}
}

func (s *inviteService) upsertUserOrganizationMapping(invite model.Invite) error {
	userEmail := strings.TrimSpace(strings.ToLower(invite.TargetEmail))
	organizationID := strings.TrimSpace(invite.OrganizationID)

	var mapping model.UserOrganizationMapping
	err := s.db.Where("user_email = ? AND organization_id = ?", userEmail, organizationID).First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			mapping = model.UserOrganizationMapping{
				UserEmail:        userEmail,
				OrganizationID:   organizationID,
				OrganizationName: strings.TrimSpace(invite.OrganizationName),
				RoleName:         strings.TrimSpace(invite.RoleName),
				Active:           true,
			}
			return s.db.Create(&mapping).Error
		}

		return err
	}

	mapping.OrganizationName = strings.TrimSpace(invite.OrganizationName)
	mapping.RoleName = strings.TrimSpace(invite.RoleName)
	mapping.Active = true

	return s.db.Save(&mapping).Error
}
