package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	logger, err := zap.NewProduction()
	if err != nil {
		logger.Error("Erro ao inicializar o logger: %v", err)
	}
	defer logger.Sync()

	for _, message := range sqsEvent.Records {
		var payload Payload

		if err := json.Unmarshal([]byte(message.Body), &payload); err != nil {
			log.Printf("Error unmarshaling message body: %s", err)
			continue
		}

		if payload.Status == "approved" {
			if err := sendToLoadBalancer(payload); err != nil {
				log.Printf("Error sending to load balancer: %s", err)
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
	lambda.Start(handler)
}
