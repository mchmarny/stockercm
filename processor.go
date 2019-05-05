package stockercm

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	lang "cloud.google.com/go/language/apiv1"
	"cloud.google.com/go/pubsub"
	langpb "google.golang.org/genproto/googleapis/cloud/language/v1"
)

var (
	logger    = log.New(os.Stdout, "[SENTIMENT] ", 0)
	projectID = mustEnvVar("PID", "")
	topicName = mustEnvVar("TARGET_TOPIC", "stocker-processed")

	once        sync.Once
	langClient  *lang.Client
	pubSubTopic *pubsub.Topic
)

// PubSubMessage is the payload of a Pub/Sub event
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// TextContent represents generic text event
type TextContent struct {
	Symbol    string    `json:"symbol"`
	ID        string    `json:"cid"`
	CreatedAt time.Time `json:"created"`
	Author    string    `json:"author"`
	Lang      string    `json:"lang"`
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	Magnitude float32   `json:"magnitude"`
	Score     float32   `json:"score"`
}

// ProcessorSentiment processes pubsub topic events
func ProcessorSentiment(ctx context.Context, m PubSubMessage) error {

	once.Do(func() {

		// create NLP client
		nc, err := lang.NewClient(ctx)
		if err != nil {
			logger.Fatalf("Failed to create lang client: %v", err)
		}
		langClient = nc

		// create pubsub client
		pc, err := pubsub.NewClient(ctx, projectID)
		if err != nil {
			logger.Fatalf("Error creating pubsub client: %v", err)
		}

		// Creates the new topic.
		top := pc.Topic(topicName)
		if err != nil {
			logger.Fatalf("Error creating pubsub topic: %v", err)
		}
		pubSubTopic = top

	})

	var c TextContent
	if err := json.Unmarshal(m.Data, &c); err != nil {
		logger.Fatalf("Error converting data: %v", err)
	}

	err := scoreSentiment(ctx, &c)
	if err != nil {
		logger.Fatalf("Error scoring sentiment: %v", err)
	}

	content, err := json.Marshal(c)
	if err != nil {
		logger.Fatalf("Error marshaling content: %v", err)
	}

	psResult := pubSubTopic.Publish(ctx, &pubsub.Message{
		Data: content,
	})

	// block and wait for the result
	_, err = psResult.Get(ctx)
	return err

}

func scoreSentiment(ctx context.Context, c *TextContent) error {

	result, err := langClient.AnalyzeSentiment(ctx, &langpb.AnalyzeSentimentRequest{
		Document: &langpb.Document{
			Source: &langpb.Document_Content{
				Content: c.Content,
			},
			Type: langpb.Document_PLAIN_TEXT,
		},
		EncodingType: langpb.EncodingType_UTF8,
	})
	if err != nil {
		logger.Printf("Error while scoring: %v", err)
		return err
	}

	c.Magnitude = result.DocumentSentiment.Magnitude
	c.Score = result.DocumentSentiment.Score

	return nil
}

func mustEnvVar(key, fallbackValue string) string {

	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	if fallbackValue == "" {
		logger.Fatalf("Required envvar not set: %s", key)
	}

	return fallbackValue
}
