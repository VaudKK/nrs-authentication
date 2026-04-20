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
	GetFacilityUsers(string, string) ([]types.UserType, error)
	GetUser(string) (*cognitoidentityprovider.ListUsersOutput, error)
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

func (s *awsService) GetUser(email string) (*cognitoidentityprovider.ListUsersOutput, error) {

	ctx := context.TODO()

	userPoolId := s.Config.CognitoUserPoolId
	filter := "email = \"" + email + "\""
	limit := int32(1)

	response, err := client.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId: &userPoolId,
		Filter:     &filter,
		Limit:      &limit,
	})
	if err != nil {
		s.Log.WithError(err).Error("Error while fetching user")
		return nil, err
	}
	return response, nil
}

func (s *awsService) GetFacilityUsers(facilityCode, role string) ([]types.UserType, error) {

	ctx := context.TODO()

	userPoolId := s.Config.CognitoUserPoolId

	var err error
	var out *cognitoidentityprovider.ListUsersInGroupOutput
	var outAll *cognitoidentityprovider.ListUsersOutput

	var users []types.UserType

	switch role {
	case "doctor":
		out, err = getUsersByGroup("DOCTOR", userPoolId)

	case "nurse":
		out, err = getUsersByGroup("NURSE", userPoolId)

	case "hospital_admin":
		out, err = getUsersByGroup("HOSPITAL_ADMIN", userPoolId)

	default:
		outAll, err = client.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
			UserPoolId: &userPoolId,
		})
	}

	if err != nil {
		s.Log.WithError(err).Error("Error while fetching facility users")
		return []types.UserType{}, nil
	}

	if role == "doctor" || role == "nurse" || role == "hospital_admin" {
		users = filterAndAppendUsers(out.Users, facilityCode)
	} else {
		users = filterAndAppendUsers(outAll.Users, facilityCode)
	}

	if users == nil {
		return []types.UserType{}, nil
	}

	// attach group
	users = appendGroupName(userPoolId, s.Log, users)

	return users, nil
}

func appendGroupName(userPoolId string, logger *logrus.Logger, users []types.UserType) []types.UserType {
	// attach group
	for i, user := range users {

		response, err := getUsersGroup(*user.Username, userPoolId)

		if err == nil {
			attrName := "groups"
			attrValue := ""

			for i, group := range response.Groups {
				attrValue += *group.GroupName

				if i < len(response.Groups)-1 {
					attrValue += ","
				}
			}

			users[i].Attributes = append(users[i].Attributes, types.AttributeType{
				Name:  &attrName,
				Value: &attrValue,
			})
		} else {
			logger.WithError(err).Error("Error attaching group")
		}
	}

	return users
}

func getUsersByGroup(groupName, userPoolId string) (*cognitoidentityprovider.ListUsersInGroupOutput, error) {
	ctx := context.TODO()
	return client.ListUsersInGroup(ctx, &cognitoidentityprovider.ListUsersInGroupInput{
		UserPoolId: &userPoolId,
		GroupName:  &groupName,
	})
}

func getUsersGroup(username, userPoolId string) (*cognitoidentityprovider.AdminListGroupsForUserOutput, error) {
	ctx := context.TODO()
	return client.AdminListGroupsForUser(ctx, &cognitoidentityprovider.AdminListGroupsForUserInput{
		UserPoolId: &userPoolId,
		Username:   &username,
	})
}

func filterAndAppendUsers(out []types.UserType, facilityCode string) []types.UserType {
	var users []types.UserType
	for _, user := range out {
		for _, attr := range user.Attributes {
			if *attr.Name == "custom:facility_code" && *attr.Value == facilityCode {
				users = append(users, user)
			}
		}
	}

	return users
}
