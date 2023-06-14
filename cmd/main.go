package main

import (
	"context"
	"fmt"
	"github.com/ku/slack-command-responder/responder"
	"github.com/slack-go/slack"
	"net/url"
	"os"
)

func main() {
	err := _main()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func _main() error {
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	r := responder.NewSlackCommandResponder(&responder.SlackCommandResponderConfig{
		SigningSecret: signingSecret,
		HTTPAddr:      "localhost:3000",
		Responder:     llmResponder,
		Executor:      NewBashResponder().Handle,
	})
	return r.Start()
}

func makeRunnableTextBlock(text string) *slack.SectionBlock {
	runBtnText := slack.NewTextBlockObject("plain_text", "Run", true, false)
	runBtnEle := slack.NewButtonBlockElement("btnid", text, runBtnText)
	tb := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("```%s```", text), false, false)
	return slack.NewSectionBlock(tb, nil, slack.NewAccessory(runBtnEle))
}

func echoResponder(ctx context.Context, vals url.Values) ([]slack.MsgOption, error) {
	text := vals.Get("text")

	return []slack.MsgOption{
		slack.MsgOptionBlocks(
			makeRunnableTextBlock(text),
		),
	}, nil
}
