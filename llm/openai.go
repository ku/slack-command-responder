package llm

import (
	"context"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	prompt func() ([]byte, error)
}

type completionResponse struct {
	resp *openai.ChatCompletionResponse
}

func (o *completionResponse) GetText() string {
	if len(o.resp.Choices) == 0 {
		return ""
	}

	return o.resp.Choices[0].Message.Content
}

func NewOpenAIClient(apiKey string, prompt func() ([]byte, error)) *Client {
	return &Client{
		client: openai.NewClient(apiKey),
		prompt: prompt,
	}
}

func (c *Client) Name() string {
	return "openai"
}

func (c *Client) Completion(ctx context.Context, txt string) (*completionResponse, error) {
	var resp openai.ChatCompletionResponse
	var err error

	prompt, err := c.prompt()
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	msgs := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: string(prompt),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: txt,
		},
	}

	resp, err = c.client.CreateChatCompletion(
		ctx, openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Messages:    msgs,
			Temperature: 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}
	return &completionResponse{resp: &resp}, nil
}
