package psbotfunc

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/slack-go/slack/slackevents"
)

type SlackMessage struct {
	Blocks []struct {
		BlockID  string `json:"block_id"`
		Elements []struct {
			Elements []struct {
				Type   string `json:"type"`
				UserID string `json:"user_id"`
			} `json:"elements"`
			Type string `json:"type"`
		} `json:"elements"`
		Type string `json:"type"`
	} `json:"blocks"`
	Channel      string `json:"channel"`
	DisplayAsBot bool   `json:"display_as_bot"`
	EventTs      string `json:"event_ts"`
	Files        []struct {
		Created            int    `json:"created"`
		DisplayAsBot       bool   `json:"display_as_bot"`
		Editable           bool   `json:"editable"`
		ExternalType       string `json:"external_type"`
		FileAccess         string `json:"file_access"`
		Filetype           string `json:"filetype"`
		HasRichPreview     bool   `json:"has_rich_preview"`
		ID                 string `json:"id"`
		IsExternal         bool   `json:"is_external"`
		IsPublic           bool   `json:"is_public"`
		IsStarred          bool   `json:"is_starred"`
		MediaDisplayType   string `json:"media_display_type"`
		Mimetype           string `json:"mimetype"`
		Mode               string `json:"mode"`
		Name               string `json:"name"`
		OriginalH          int    `json:"original_h"`
		OriginalW          int    `json:"original_w"`
		Permalink          string `json:"permalink"`
		PermalinkPublic    string `json:"permalink_public"`
		PrettyType         string `json:"pretty_type"`
		PublicURLShared    bool   `json:"public_url_shared"`
		Size               int    `json:"size"`
		Timestamp          int    `json:"timestamp"`
		Title              string `json:"title"`
		URLPrivate         string `json:"url_private"`
		URLPrivateDownload string `json:"url_private_download"`
		User               string `json:"user"`
		UserTeam           string `json:"user_team"`
		Username           string `json:"username"`
	} `json:"files"`
	Text   string `json:"text"`
	Ts     string `json:"ts"`
	Type   string `json:"type"`
	Upload bool   `json:"upload"`
	User   string `json:"user"`
}

func ConverToMessage(event slackevents.EventsAPIEvent, message *SlackMessage) error {
	data, ok := event.Data.(*slackevents.EventsAPICallbackEvent)
	if !ok {
		return errors.New("convert to slackevents.EventsAPICallbackEvent failed")
	}
	err := json.Unmarshal(*data.InnerEvent, message)
	if err != nil {
		return fmt.Errorf("handle callback failed: %w", err)
	}
	return nil
}
