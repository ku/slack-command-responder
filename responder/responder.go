package responder

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/slack-go/slack"
	"io"
	"log"
	"net/http"
	"net/url"
)

type SlackCommandResponderConfig struct {
	SigningSecret string
	HTTPAddr      string
	Responder     func(ctx context.Context, text string) ([]slack.Block, error)
	Executor      func(ctx context.Context, text string) ([]slack.Block, error)
}

type SlackCommandResponder struct {
	conf *SlackCommandResponderConfig
}

func (r *SlackCommandResponder) Start() error {
	http.HandleFunc("/interactivity", wrap(r.interactivityHandler))
	http.HandleFunc("/slash-command", wrap(r.commandHandler))
	return http.ListenAndServe(r.conf.HTTPAddr, nil)

}

func (r *SlackCommandResponder) interactivityHandler(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	if err := r.verifySignature(req, body); err != nil {
		return nil, err
	}

	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}
	payload := vals.Get("payload")
	var icb slack.InteractionCallback
	if err := json.Unmarshal([]byte(payload), &icb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	ctx := context.Background()
	for _, ba := range icb.ActionCallback.BlockActions {
		script := ba.Value

		tb := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("```%s```", script), false, false)
		cmdBlock := slack.NewSectionBlock(tb, nil, nil)

		//		ctxBlock := slack.NewContextBlock("cb", tb)

		res, err := r.conf.Executor(ctx, script)

		blocks := []slack.Block{}
		blocks = append(blocks, cmdBlock)
		blocks = append(blocks, res...)

		opts := append([]slack.MsgOption{
			slack.MsgOptionResponseURL(icb.ResponseURL, "in_channel"),
			slack.MsgOptionBlocks(blocks...),
		})

		if err := r.respondToResponseURL(ctx, ba.InitialChannel, opts, err); err != nil {
			log.Printf("failed to respond to response url: %w", err.Error())
		}
		break
	}

	return nil, nil
}

func (r *SlackCommandResponder) respondAsync(channelID, responseURL, text string) {
	ctx := context.Background()
	blocks, err := r.conf.Responder(ctx, text)

	opts := []slack.MsgOption{
		slack.MsgOptionReplaceOriginal(responseURL),
		slack.MsgOptionBlocks(blocks...),
	}

	if err := r.respondToResponseURL(ctx, channelID, opts, err); err != nil {
		log.Printf("failed to respond to response url: %w", err.Error())
	}

}

func (r *SlackCommandResponder) commandHandler(w http.ResponseWriter, req *http.Request) (any, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	if err := r.verifySignature(req, body); err != nil {
		return nil, err
	}

	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	channelID := vals.Get("channel_id")
	responseURL := vals.Get("response_url")
	text := vals.Get("text")

	go r.respondAsync(channelID, responseURL, text)

	tb := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("processing your request now..."), false, false)
	section := slack.NewSectionBlock(tb, nil, nil)

	var resp blocks
	resp.Blocks = append(resp.Blocks, section)

	return resp, err
}

type blocks struct {
	Blocks []interface{} `json:"blocks"`
}

func NewSlackCommandResponder(conf *SlackCommandResponderConfig) *SlackCommandResponder {
	return &SlackCommandResponder{
		conf: conf,
	}
}

func wrap(f func(w http.ResponseWriter, r *http.Request) (interface{}, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := f(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		switch v := resp.(type) {
		case string:
			w.Header().Set("content-type", "text")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(v))
		case []byte:
			w.Header().Set("content-type", "text")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(v)
		default:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}
		if err != nil {
			log.Printf("%s %s: %s", r.Method, r.URL.Path, err.Error())
		}
	}
}

func (r *SlackCommandResponder) verifySignature(req *http.Request, body []byte) error {
	sv, err := slack.NewSecretsVerifier(req.Header, r.conf.SigningSecret)
	if err != nil {
		return err
	}
	if _, err := sv.Write(body); err != nil {
		return err
	}
	if err := sv.Ensure(); err != nil {
		return err
	}
	return nil
}

func (r *SlackCommandResponder) respondToResponseURL(ctx context.Context, channel string, opts []slack.MsgOption, err error) error {

	if err != nil {
		opts = append(opts, slack.MsgOptionText(fmt.Sprintf("❌ failed: %s", err.Error()), false))
	}

	c := slack.New("")
	if _, _, err := c.PostMessageContext(ctx, channel, opts...); err != nil {
		log.Printf(err.Error())
	}
	return nil

}
