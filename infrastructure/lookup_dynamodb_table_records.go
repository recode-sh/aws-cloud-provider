package infrastructure

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	ErrNoRecodeConfigFound       = errors.New("ErrNoRecodeConfigFound")
	ErrMultipleRecodeConfigFound = errors.New("ErrMultipleRecodeConfigFound")
)

type DynamoDBRecodeConfigTableRecord struct {
	ID         string
	ConfigJSON string
}

func LookupRecodeConfigInDynamoDBTable(
	dynamoDBClient *dynamodb.Client,
) (returnedConfigJSON string, returnedError error) {

	scanResp, err := dynamoDBClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(DynamoDBRecodeConfigTableName),
	})

	if err != nil {
		var resourceNotFoundErr *types.ResourceNotFoundException

		if errors.As(err, &resourceNotFoundErr) { // Table not found
			returnedError = ErrNoRecodeConfigFound
			return
		}

		returnedError = err
		return
	}

	if scanResp.Count == 0 { // Empty table
		returnedError = ErrNoRecodeConfigFound
		return
	}

	if scanResp.Count > 1 { // Multiple rows
		returnedError = ErrMultipleRecodeConfigFound
		return
	}

	var records []DynamoDBRecodeConfigTableRecord
	err = attributevalue.UnmarshalListOfMaps(scanResp.Items, &records)

	if err != nil {
		returnedError = err
		return
	}

	returnedConfigJSON = records[0].ConfigJSON
	return
}
