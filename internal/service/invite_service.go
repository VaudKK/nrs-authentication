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
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteNotAccepted = errors.New("invite has not been accepted")
	ErrInviteAccepted    = errors.New("invite has already been accepted")
)

type InviteService interface {
	CreateInvite(dto.CreateInviteRequest, string) (dto.InviteResponse, error)
	ListPendingInvites(string) ([]dto.InviteResponse, error)
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

	if err := db.AutoMigrate(&model.Invite{}); err != nil {
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
	invite := model.Invite{
		TargetEmail:      strings.TrimSpace(strings.ToLower(request.TargetEmail)),
		HostEmail:        strings.TrimSpace(strings.ToLower(hostEmail)),
		RoleName:         strings.TrimSpace(request.RoleName),
		OrganizationID:   strings.TrimSpace(request.OrganizationID),
		OrganizationName: strings.TrimSpace(request.OrganizationName),
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

	return s.awsService.AttachRole(dto.AttachRoleRequest{
		Username:  strings.TrimSpace(*user.Users[0].Username),
		GroupName: invite.RoleName,
	})
}

func (s *inviteService) findInvite(inviteID string) (model.Invite, error) {
	var invite model.Invite
	if err := s.db.First(&invite, inviteID).Error; err != nil {
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
