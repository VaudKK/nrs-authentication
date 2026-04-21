package dto

// ListUsersOutputSwagger documents the GetUser response using UpperCamelCase fields.
type ListUsersOutputSwagger struct {
	PaginationToken *string            `json:"PaginationToken"`
	ResultMetadata  MetadataSwagger    `json:"ResultMetadata"`
	Users           []UserTypeSwagger  `json:"Users"`
}

// MetadataSwagger represents operation metadata in Swagger.
type MetadataSwagger struct{}

// UserTypeSwagger documents a Cognito user with UpperCamelCase fields.
type UserTypeSwagger struct {
	Attributes           []AttributeTypeSwagger `json:"Attributes"`
	Enabled              bool                   `json:"Enabled"`
	MFAOptions           []MFAOptionTypeSwagger `json:"MFAOptions"`
	UserCreateDate       *string                `json:"UserCreateDate"`
	UserLastModifiedDate *string                `json:"UserLastModifiedDate"`
	UserStatus           string                 `json:"UserStatus"`
	Username             *string                `json:"Username"`
}

// AttributeTypeSwagger documents a Cognito attribute with UpperCamelCase fields.
type AttributeTypeSwagger struct {
	Name  *string `json:"Name"`
	Value *string `json:"Value"`
}

// MFAOptionTypeSwagger documents Cognito MFA details with UpperCamelCase fields.
type MFAOptionTypeSwagger struct {
	AttributeName  *string `json:"AttributeName"`
	DeliveryMedium string  `json:"DeliveryMedium"`
}
