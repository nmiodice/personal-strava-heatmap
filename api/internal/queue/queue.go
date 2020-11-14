package queue

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-queue-go/azqueue"
)

type QueueService interface {
	Enqueue(ctx context.Context, msgs ...interface{}) ([]string, error)
}

type AzureStorageQueue struct {
	messagesURL *azqueue.MessagesURL
}

func NewAzureStorageQueue(ctx context.Context, queueName, accountName, accountKey string) (*AzureStorageQueue, error) {
	primaryURLRaw := fmt.Sprintf("https://%s.queue.core.windows.net", accountName)
	primaryURL, err := url.Parse(primaryURLRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %v: %v", primaryURLRaw, err)
	}

	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*primaryURL, p)
	queueURL := serviceURL.NewQueueURL(queueName)
	messagesURL := queueURL.NewMessagesURL()

	return &AzureStorageQueue{
		messagesURL: &messagesURL,
	}, nil
}

func (as AzureStorageQueue) Enqueue(ctx context.Context, msgs ...interface{}) ([]string, error) {
	messageIDs := []string{}
	for _, msg := range msgs {
		bytes, err := json.Marshal(msg)
		if err != nil {
			return messageIDs, err
		}

		asString := base64.StdEncoding.EncodeToString(bytes)
		res, err := as.messagesURL.Enqueue(ctx, asString, 0, 0)
		if err != nil {
			return messageIDs, err
		}

		messageIDs = append(messageIDs, res.MessageID.String())
	}

	return messageIDs, nil
}
