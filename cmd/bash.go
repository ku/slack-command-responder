package main

import (
	"context"
	"fmt"
	"github.com/slack-go/slack"
	"io"
	"os/exec"
)

type BashResponder struct {
}

func NewBashResponder() *BashResponder {
	return &BashResponder{}
}

func (b *BashResponder) Handle(ctx context.Context, script string) ([]slack.Block, error) {
	cmd := exec.Command("/bin/bash")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	defer stdin.Close()

	go func() {
		io.WriteString(stdin, script)
		stdin.Close()
	}()

	out, err := cmd.CombinedOutput()

	placeholder := slack.NewTextBlockObject("plain_text", "You can edit and run it again", false, false)

	ib := slack.NewPlainTextInputBlockElement(placeholder, "rerun-block-input")
	ib.Multiline = true
	ib.InitialValue = script
	lb := slack.NewTextBlockObject("plain_text", "Command executed", false, false)
	//	el := slack.NewPlainTextInputBlockElement(ib, "input-script")

	htb := slack.NewTextBlockObject("plain_text", "Output", false, false)

	return []slack.Block{
		slack.NewInputBlock("executed-script-block", lb, nil, ib),
		slack.NewActionBlock("action-block",
			slack.NewButtonBlockElement("rerun-button-action", "Run", slack.NewTextBlockObject("plain_text", "Run again", true, false)),
		),
		slack.NewHeaderBlock(htb),
		textSection(string(out)),
	}, nil
}
