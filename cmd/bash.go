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

func (b *BashResponder) Handle(ctx context.Context, block string) ([]slack.Block, error) {
	cmd := exec.Command("/bin/bash")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	defer stdin.Close()

	go func() {
		io.WriteString(stdin, block)
		stdin.Close()
	}()

	out, err := cmd.CombinedOutput()

	return []slack.Block{textSection(string(out))}, nil
}
