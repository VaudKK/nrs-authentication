package dto

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// CheckEmailResponse documents the GetUser response using lowerCamelCase fields.
type CheckEmailResponse struct {
	PaginationToken *string             `json:"paginationToken"`
	ResultMetadata  MetadataSwagger     `json:"resultMetadata"`
	Users           []CheckEmailUserDTO `json:"users"`
}

// MetadataSwagger represents operation metadata in Swagger.
type MetadataSwagger struct{}

// CheckEmailUserDTO documents a Cognito user with lowerCamelCase fields.
type CheckEmailUserDTO struct {
	Attributes           []AttributeTypeSwagger `json:"attributes"`
	Enabled              bool                   `json:"enabled"`
	MFAOptions           []MFAOptionTypeSwagger `json:"mfaOptions"`
	UserCreateDate       *time.Time             `json:"userCreateDate"`
	UserLastModifiedDate *time.Time             `json:"userLastModifiedDate"`
	UserStatus           string                 `json:"userStatus"`
	Username             *string                `json:"username"`
}

// AttributeTypeSwagger documents a Cognito attribute with lowerCamelCase fields.
type AttributeTypeSwagger struct {
	Name  *string `json:"name"`
	Value *string `json:"value"`
}

// MFAOptionTypeSwagger documents Cognito MFA details with lowerCamelCase fields.
type MFAOptionTypeSwagger struct {
	AttributeName  *string `json:"attributeName"`
	DeliveryMedium string  `json:"deliveryMedium"`
}

func MapCheckEmailResponse(output *cognitoidentityprovider.ListUsersOutput) CheckEmailResponse {
	if output == nil {
		return CheckEmailResponse{
			Users: []CheckEmailUserDTO{},
		}
	}

	users := make([]CheckEmailUserDTO, 0, len(output.Users))
	for _, user := range output.Users {
		users = append(users, mapCheckEmailUser(user))
	}

	return CheckEmailResponse{
		PaginationToken: output.PaginationToken,
		ResultMetadata:  MetadataSwagger{},
		Users:           users,
	}
}

func mapCheckEmailUser(user types.UserType) CheckEmailUserDTO {
	attributes := make([]AttributeTypeSwagger, 0, len(user.Attributes))
	for _, attribute := range user.Attributes {
		attributes = append(attributes, AttributeTypeSwagger{
			Name:  attribute.Name,
			Value: attribute.Value,
		})
	}

	mfaOptions := make([]MFAOptionTypeSwagger, 0, len(user.MFAOptions))
	for _, option := range user.MFAOptions {
		mfaOptions = append(mfaOptions, MFAOptionTypeSwagger{
			AttributeName:  option.AttributeName,
			DeliveryMedium: string(option.DeliveryMedium),
		})
	}

	return CheckEmailUserDTO{
		Attributes:           attributes,
		Enabled:              user.Enabled,
		MFAOptions:           mfaOptions,
		UserCreateDate:       user.UserCreateDate,
		UserLastModifiedDate: user.UserLastModifiedDate,
		UserStatus:           string(user.UserStatus),
		Username:             user.Username,
	}
}
