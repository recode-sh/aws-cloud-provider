package infrastructure

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func UpdateRecodeConfigInDynamoDBTable(
	dynamoDBClient *dynamodb.Client,
	configID string,
	configJSON string,
) error {

	configRecord := DynamoDBRecodeConfigTableRecord{
		ID:         configID,
		ConfigJSON: configJSON,
	}

	marshaledConfigRecord, err := attributevalue.MarshalMap(configRecord)

	if err != nil {
		return err
	}

	_, err = dynamoDBClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(DynamoDBRecodeConfigTableName),
		Item:      marshaledConfigRecord,
	})

	return err
}
