package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/thathaneydude/go-tenable"
	"net/http"
)

func BuildSlackText(event go_tenable.Event) string {
	var ResponseText string
	switch event.Action {
	case "user.impersonation.start":
		ResponseText = fmt.Sprintf("\"%v\" has started impersonating user \"%v\"", event.Actor.Name, event.Target.Name)
	case "user.impersonation.end":
		ResponseText = fmt.Sprintf("\"%v\" has finished impersonating user \"%v\"", event.Actor.Name, event.Target.Name)
	case "user.create":
		if event.Target.Name != "" {
			ResponseText = fmt.Sprintf("\"%v\" has created user \"%v\"", event.Actor.Name, event.Target.Name)
		} else {
			ResponseText = fmt.Sprintf("\"%v\" has created a user", event.Actor.Name)
		}

	case "user.delete":
		ResponseText = fmt.Sprintf("\"%v\" has deleted user \"%v\"", event.Actor.Name, event.Target.Name)
	case "user.update":
		if event.Actor.Name == event.Target.Name {
			ResponseText = fmt.Sprintf("\"%v\" has updated their account", event.Actor.Name)
		} else {
			ResponseText = fmt.Sprintf("\"%v\" has updated user \"%v\"", event.Actor.Name, event.Target.Name)
		}
	}
	log.Debug(fmt.Sprintf("Message being sent to Slack: %v\n", ResponseText))
	return ResponseText
}

type SlackClient struct {
	client     *http.Client
	WebHookURL string
}

type SlackMessage struct {
	text string
}

func (slack *SlackClient) SendMessage(msg *SlackMessage) http.Response {
	log.Debug(fmt.Sprintf("Sending message to Slack WebHook: %v\n", msg.text))
	bodyMap := map[string]interface{}{
		"text": msg.text,
	}
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest("POST", slack.WebHookURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Error(fmt.Sprintf("Unable to build Slack request: %v\n", err))
	}
	resp, err := slack.client.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("Unable to send Slack request: %v\n", err))
	}
	return *resp

}

func NewSlackClient(WebHookURL string) SlackClient {
	client := &http.Client{}
	slackClient := &SlackClient{
		client,
		WebHookURL,
	}

	return *slackClient
}
