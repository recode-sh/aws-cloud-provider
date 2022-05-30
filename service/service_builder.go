package service

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/recode-sh/aws-cloud-provider/userconfig"
	"github.com/recode-sh/recode/entities"
)

//go:generate mockgen -destination ../mocks/user_config_resolver.go -package mocks -mock_names UserConfigResolver=UserConfigResolver github.com/recode-sh/aws-cloud-provider/service UserConfigResolver
type UserConfigResolver interface {
	Resolve() (*userconfig.Config, error)
}

//go:generate mockgen -destination ../mocks/service_config_loader.go -package mocks -mock_names ConfigLoader=ServiceConfigLoader github.com/recode-sh/aws-cloud-provider/service ConfigLoader
type UserConfigLoader interface {
	Load(userConfig *userconfig.Config) (aws.Config, error)
}

type UserConfigValidator interface {
	Validate(userConfig *userconfig.Config) error
}

type Builder struct {
	userConfigResolver  UserConfigResolver
	userConfigValidator UserConfigValidator
	userConfigLoader    UserConfigLoader
}

func NewBuilder(
	userConfigResolver UserConfigResolver,
	userConfigValidator UserConfigValidator,
	userConfigLoader UserConfigLoader,
) Builder {

	return Builder{
		userConfigResolver:  userConfigResolver,
		userConfigValidator: userConfigValidator,
		userConfigLoader:    userConfigLoader,
	}
}

func (b Builder) Build() (entities.CloudService, error) {
	userConfig, err := b.userConfigResolver.Resolve()

	if err != nil {
		return nil, err
	}

	if err := b.userConfigValidator.Validate(userConfig); err != nil {
		return nil, err
	}

	AWSSDKConfig, err := b.userConfigLoader.Load(userConfig)

	if err != nil {
		return nil, err
	}

	AWSService := NewAWS(AWSSDKConfig)

	return AWSService, nil
}
