package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.uber.org/zap"
)

type Payload struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Action string `json:"action"`
	Data   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"data"`
}

var logger *zap.Logger

func initLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar o logger: %v", err)
	}
	return logger, nil
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, message := range sqsEvent.Records {
		var payload Payload

		if err := json.Unmarshal([]byte(message.Body), &payload); err != nil {
			logger.Error("Error unmarshaling message body", zap.Error(err))
			continue
		}

		logger.Info("Processing message", zap.String("id", payload.Id), zap.String("status", payload.Status), zap.String("action", payload.Action))

		if payload.Status == "approved" {
			if err := sendToLoadBalancer(payload); err != nil {
				logger.Error("Error sending to load balancer", zap.Error(err))
			}
		}
	}
	return nil
}

func sendToLoadBalancer(payload Payload) error {
	url := os.Getenv("LOAD_BALANCER_URL")

	var req *http.Request
	var err error

	switch payload.Action {
	case "POST":
		req, err = http.NewRequest("POST", url, nil)
	case "DELETE":
		req, err = http.NewRequest("DELETE", url, nil)
	default:
		return fmt.Errorf("unknown action: %s", payload.Action)
	}

	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	var err error
	logger, err = initLogger()
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		return
	}
	defer logger.Sync()

	lambda.Start(handler)
}
