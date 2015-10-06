package main

import (
	"time"
)

func main() {
	config, err := readConfig("aws-dashboard.yaml")
	if err != nil {
		panic(err)
	}

	payload := Payload{Text: "AWS report for " + time.Now().String(), Channel: config.Slack.Channel, Username: config.Slack.Username, IconUrl: config.Slack.IconUrl, Attachments: make([]Attachment, 2)}
	payload.Attachments[0].Fallback = "Billing summary"
	payload.Attachments[0].Title = "Billing summary"
	payload.Attachments[0].Fields = make([]Field, 0)
	payload.Attachments[1].Fallback = "Overview`"
	payload.Attachments[1].Title = "Overview"
	payload.Attachments[1].Fields = make([]Field, 0)

	processBilling(config, &(payload.Attachments[0]))
	processOverview(config, &(payload.Attachments[1]))

	slackClient := new(SlackIncomingWebhookClient)
	slackClient.Url = config.Slack.IncomingWebhookUrl

	err = slackClient.SendMessage(payload)
	if err != nil {
		panic(err)
	}
}
