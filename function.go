package psbotfunc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	// framework "github.com/GoogleCloudPlatform/functions-framework-go"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"go.uber.org/zap"
)

// func init() {
// 	framework.functions.HTTP("Pokemonsleep bot", PokemonSleepFoods)
// }

func PokemonSleepFoods(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	token := os.Getenv("SLACK_AUTH_TOKEN")
	secrets := os.Getenv("SLACK_SIGNING_SECRETS")
	foodConf := os.Getenv("POKEMONSLEEP_FOODS_JSON_PATH")
	cookConf := os.Getenv("POKEMONSLEEP_COOKS_JSON_PATH")

	logger, err := zap.NewProduction()
	if err != nil {
		logger.Error("failed init logger.", zap.Error(err))
		return
	}
	defer logger.Sync()

	slackbot := NewSlackBot(logger, token, secrets, func(s *SlackBot, ctx context.Context, event slackevents.EventsAPIEvent) error {
		s.Logger.Info("callback")
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			var message SlackMessage
			err := ConverToMessage(event, &message)
			if err != nil {
				return fmt.Errorf("handle callback failed: %w", err)
			}

			if len(message.Files) == 0 {
				_, _, err = s.Api.PostMessage(ev.Channel, slack.MsgOptionText("画像を添付してください", false))
				if err != nil {
					return fmt.Errorf("handle callback failed: %w", err)
				}
				return nil
			}

			psclient, err := NewClient(ctx, s.Token, foodConf, cookConf, s.Logger)
			if err != nil {
				return fmt.Errorf("init PokemonSleep Client failed: %w", err)
			}
			result, err := psclient.GetResultText(ctx, message.Text, message.Files[0].Filetype, message.Files[0].URLPrivateDownload, message.Files[0].OriginalW, message.Files[0].OriginalH)
			if err != nil {
				return fmt.Errorf("failed to get result text: %w", err)
			}

			for _, text := range result {
				_, _, err = s.Api.PostMessage(ev.Channel, slack.MsgOptionText(text, false), slack.MsgOptionTS(message.Ts))
				if err != nil {
					return fmt.Errorf("handle callback failed: %w", err)
				}
			}
		default:
			return errors.New("unknown event")
		}
		return nil
	})

	err = slackbot.HandleRequest(ctx, w, r)
	if err != nil {
		logger.Error("failed handle request.", zap.Error(err))
		return
	}
}
