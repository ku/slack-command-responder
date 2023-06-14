package main

import (
	"context"
	"fmt"
	"github.com/ku/slack-command-responder/llm"
	"github.com/slack-go/slack"
	"os"
	"strings"
)

func llmResponder(ctx context.Context, text string) ([]slack.Block, error) {
	c := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"), func() ([]byte, error) {
		return os.ReadFile("prompt.txt")
	})
	resp, err := c.Completion(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to Complete: %w", err)
	}
	return commandBlocksFromResponse(resp.GetText()), nil
}

func commandBlocksFromResponse(rawText string) []slack.Block {
	var blocks []slack.Block
	//Replace the ampersand, &, with &amp;
	//Replace the less-than sign, < with &lt;
	//Replace the greater-than sign, > with &gt;
	// https://api.slack.com/reference/surfaces/formatting#escaping
	unescapedTexet := strings.ReplaceAll(rawText, "&amp;", "&")
	fields := strings.Split(unescapedTexet, "```")
	for n, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if (n % 2) == 0 {
			blocks = append(blocks, textSection(field))
		} else {
			b := makeRunnableTextBlock(field)
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func textSection(rawText string) slack.Block {
	tb := slack.NewTextBlockObject("mrkdwn", rawText, false, false)
	b := slack.NewSectionBlock(tb, nil, nil)
	return b
}
