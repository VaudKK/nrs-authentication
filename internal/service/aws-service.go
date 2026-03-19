package service

import (
	"context"
	"errors"
	"nrs-authentication/internal/config"
	"nrs-authentication/internal/dto"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/sirupsen/logrus"
)

var client *cognitoidentityprovider.Client

type AwsService interface {
	AttachRole(dto.AttachRoleRequest) (dto.AttachRoleResponse, error)
	GetFacilityUsers(string) ([]types.UserType, error)
}

type awsService struct {
	Config    *config.Config
	AwsConfig aws.Config
	Log       *logrus.Logger
}

func NewAwsService(config *config.Config, log *logrus.Logger, awsConfig aws.Config) AwsService {

	client = cognitoidentityprovider.NewFromConfig(awsConfig)

	return &awsService{
		Config:    config,
		Log:       log,
		AwsConfig: awsConfig,
	}
}

func (s *awsService) AttachRole(request dto.AttachRoleRequest) (dto.AttachRoleResponse, error) {

	ctx := context.TODO()

	userPoolId := s.Config.CognitoUserPoolId

	_, err := client.AdminAddUserToGroup(ctx, &cognitoidentityprovider.AdminAddUserToGroupInput{
		UserPoolId: &userPoolId,
		Username:   &request.Username,
		GroupName:  &request.GroupName,
	})

	if err != nil {

		var userNotFound *types.UserNotFoundException
		var notAuthorized *types.NotAuthorizedException
		var invalidParam *types.InvalidParameterException

		errMessage := ""

		switch {
		case errors.As(err, &userNotFound):
			errMessage = "User not found"
		case errors.As(err, &notAuthorized):
			errMessage = "Not authorized"
		case errors.As(err, &invalidParam):
			errMessage = "Invalid parameter"
		default:
			errMessage = "Error while adding user to group"
		}

		s.Log.WithError(err).Error(errMessage)
		return dto.AttachRoleResponse{
			Success: false,
			Message: errMessage,
		}, err
	}

	return dto.AttachRoleResponse{
		Success: true,
		Message: "Success",
	}, nil
}

func (s *awsService) GetFacilityUsers(facilityId string) ([]types.UserType, error) {

	ctx := context.TODO()

	userPoolId := s.Config.CognitoUserPoolId

	out, err := client.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId:      &userPoolId,
		AttributesToGet: []string{"custom:facility_id"},
	})

	if err != nil {
		s.Log.WithError(err).Error("Error while fetching facility users")
		return []types.UserType{}, nil
	}

	var users []types.UserType

	for _, user := range out.Users {
		for _, attr := range user.Attributes {
			if *attr.Name == "custom:facility_id" && *attr.Value == facilityId {
				users = append(users, user)
			}
		}
	}

	if users == nil {
		return []types.UserType{}, nil
	}

	return users, nil
}
