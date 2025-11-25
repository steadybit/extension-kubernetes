package ai

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	bedrock "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go/logging"
)

type Client struct {
	BR *bedrock.Client
}

func NewAIClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("eu-central-1"),
		config.WithLogger(logging.Nop{}),
	)
	if err != nil {
		return nil, err
	}

	// Configure retry attempts and HTTP timeout on the shared config
	cfg.RetryMaxAttempts = 3
	cfg.HTTPClient = &http.Client{
		Timeout: 90 * time.Second,
	}

	br := bedrock.NewFromConfig(cfg)
	return &Client{BR: br}, nil
}
