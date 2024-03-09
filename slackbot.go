package psbotfunc

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"go.uber.org/zap"
)

type CallbackFunc func(*SlackBot, context.Context, slackevents.EventsAPIEvent) error

type SlackBot struct {
	Logger *zap.Logger

	Token  string
	Secret string
	Api    *slack.Client

	Callback CallbackFunc
}

func NewSlackBot(logger *zap.Logger, token, secret string, callback CallbackFunc) *SlackBot {
	return &SlackBot{
		Logger:   logger,
		Token:    token,
		Secret:   secret,
		Api:      slack.New(token),
		Callback: callback,
	}
}

func (s *SlackBot) HandleRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("read request failed: %w", err)
	}
	defer r.Body.Close()

	// リクエストの検証
	sv, err := slack.NewSecretsVerifier(r.Header, s.Secret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("validate request failed: %w", err)
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("validate request failed: %w", err)
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return fmt.Errorf("validate request failed: %w", err)
	}

	// eventをパース
	event, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("parse event failed: %w", err)
	}

	// URLVerification eventをhandle（EventAPI有効化時に叩かれる）
	if event.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("marshal json failed: %w", err)
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
		return nil
	}

	// Eventのハンドリング
	if event.Type == slackevents.CallbackEvent {
		err := s.Callback(s, ctx, event)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("handle callback failed: %w", err)
		}
	}

	s.Logger.Info("handle finished.")
	return nil
}
