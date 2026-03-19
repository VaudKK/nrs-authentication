package service

import (
	"context"
	"nrs-authentication/internal/config"
	"nrs-authentication/internal/dto"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/sirupsen/logrus"
)

type AwsService interface {
	AttachRole(dto.AttachRoleRequest) (dto.AttachRoleResponse,error)
}

type awsService struct {
	Config *config.Config
	Log    *logrus.Logger
}


func NewAwsService(config *config.Config, log *logrus.Logger) AwsService{
	return &awsService{
		Config: config,
		Log: log,
	}
}

func (s *awsService) AttachRole(request dto.AttachRoleRequest) (dto.AttachRoleResponse,error){

	ctx := context.TODO()

	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		s.Log.WithError(err).Error("Unable  to load sdk config")
		return dto.AttachRoleResponse{
			Success: false,
			Message: "Could not load configs",
		},err
	}

	client := cognitoidentityprovider.NewFromConfig(cfg)

	userPoolId := s.Config.CognitoUserPoolId

	_, err = client.AdminAddUserToGroup(ctx,&cognitoidentityprovider.AdminAddUserToGroupInput{
		UserPoolId: &userPoolId,
		Username: &request.Username,
		GroupName: &request.GroupName,
	})

	if err != nil {
		s.Log.WithError(err).Error("Error while adding user to group")
		return dto.AttachRoleResponse{
			Success: false,
			Message: err.Error(),
		},err
	}

	return dto.AttachRoleResponse{
		Success: true,
		Message: "Success",
	},nil
}
