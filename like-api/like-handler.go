package main

/*
해당 함수는 좋아요 기능을 위해 구현
기능: 좋아요 개수 확인, 좋아요 개수 +1
*/

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"net/http"
	"os"
)

type response struct {
	Status bool `json:"status"`
	Code int `json:"code"`
	LikeCount int `json:"like_count"`
}

type likeInfo struct {
	ID string `json:"id"`
	LikeCount int `json:"like_count"`
}

// DB Setting
var dbConnect = dynamodb.New(session.New(), aws.NewConfig().WithRegion("ap-northeast-2"))
var tableName = "wallet-teaser"
var tableKey = "id"
var itemName = "likes"

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

func responseFunc(status bool, code int, likeCount int) (response, error) {
	return response{Status:status, Code:code, LikeCount:likeCount}, nil
}

func responseAPIGatewayRes(status bool, code int, likeCount int) (events.APIGatewayProxyResponse, error) {
	js, err := json.Marshal(response{Status:status, Code:code, LikeCount:likeCount})

	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body: string(js),
	}, nil
}

// HTTPMethod가 GET일 경우 LikeCount 정보 반환
func getLikeCount() (int, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			tableKey: {
				S: aws.String(itemName),
			},
		},

	}

	result, err := dbConnect.GetItem(input)

	if err != nil {
		return -1, err
	}

	if result.Item == nil {
		return -1, err
	}

	res := new(likeInfo)
	err = dynamodbattribute.UnmarshalMap(result.Item, res)


	if err != nil {
		return -1, err
	}

	return res.LikeCount, nil

}

// HTTPMethod가 PUT일 경우 LikeCount + 1 후 변경 값 반환
func updateLikeCount() (int, error) {

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val": {
				N: aws.String("1"),
			},
		},
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			tableKey: {
				S: aws.String(itemName),
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set like_count = like_count + :val"),
	}

	updatedResult, err := dbConnect.UpdateItem(input)
	if err != nil {
		return -1, err
	}

	if updatedResult.Attributes == nil {
		return -1, err
	}

	res := new(likeInfo)
	err = dynamodbattribute.UnmarshalMap(updatedResult.Attributes, res)

	if err != nil {
		return -1, err
	}

	return res.LikeCount, nil

}

func handleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)  {
	fmt.Println("HTTPMethode : ", request.HTTPMethod)

	var likeCount int
	if request.HTTPMethod == "GET" {
		likeCountResult, err := getLikeCount()
		if err != nil {
			return responseAPIGatewayRes(false, 0, -1)
		}

		likeCount = likeCountResult
	}

	if request.HTTPMethod == "PUT" {
		updateLikeCount, err := updateLikeCount()
		if err != nil {
			return responseAPIGatewayRes(false, 0, -1)
		}

		likeCount = updateLikeCount
	}

	return responseAPIGatewayRes(true, 0, likeCount)
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	errorLogger.Println(err.Error())

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}


func main()  {
	lambda.Start(handleRequest)
}