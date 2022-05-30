package service

import (
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/recode-sh/aws-cloud-provider/infrastructure"
	"github.com/recode-sh/recode/entities"
	"github.com/recode-sh/recode/stepper"
)

func (a *AWS) CreateRecodeConfigStorage(
	stepper stepper.Stepper,
) error {

	dynamoDBClient := dynamodb.NewFromConfig(a.sdkConfig)

	stepper.StartTemporaryStep("Creating a DynamoDB table to store Recode's data")

	err := infrastructure.CreateDynamoDBTableForRecodeConfig(
		dynamoDBClient,
	)

	if err != nil && errors.Is(err, infrastructure.ErrRecodeConfigTableAlreadyExists) {
		return nil
	}

	return err
}

func (a *AWS) LookupRecodeConfig(
	stepper stepper.Stepper,
) (*entities.Config, error) {

	dynamoDBClient := dynamodb.NewFromConfig(a.sdkConfig)

	configJSON, err := infrastructure.LookupRecodeConfigInDynamoDBTable(
		dynamoDBClient,
	)

	if err != nil {

		if errors.Is(err, infrastructure.ErrNoRecodeConfigFound) {
			// No config table or no records.
			return nil, entities.ErrRecodeNotInstalled
		}

		return nil, err
	}

	var recodeConfig *entities.Config
	err = json.Unmarshal([]byte(configJSON), &recodeConfig)

	if err != nil {
		return nil, err
	}

	return recodeConfig, nil
}

func (a *AWS) SaveRecodeConfig(
	stepper stepper.Stepper,
	config *entities.Config,
) error {

	configJSON, err := json.Marshal(config)

	if err != nil {
		return err
	}

	dynamoDBClient := dynamodb.NewFromConfig(a.sdkConfig)

	return infrastructure.UpdateRecodeConfigInDynamoDBTable(
		dynamoDBClient,
		config.ID,
		string(configJSON),
	)
}

func (a *AWS) RemoveRecodeConfigStorage(
	stepper stepper.Stepper,
) error {

	dynamoDBClient := dynamodb.NewFromConfig(a.sdkConfig)

	stepper.StartTemporaryStep("Removing the DynamoDB table used to store Recode's data")

	return infrastructure.RemoveDynamoDBTableForRecodeConfig(
		dynamoDBClient,
	)
}
