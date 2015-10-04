package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type Attachment struct {
	Fallback   string  `json:"fallback"`
	Color      string  `json:"color"`
	Pretext    string  `json:"pretext"`
	AuthorName string  `json:"author_name"`
	AuthorLink string  `json:"author_link"`
	AuthorIcon string  `json:"author_icon"`
	Title      string  `json:"title"`
	TitleLink  string  `json:"title_link"`
	Text       string  `json:"text"`
	Fields     []Field `json:"fields"`
	ImageUrl   string  `json:"image_url"`
	ThumbUrl   string  `json:"thumb_url"`
}

type Payload struct {
	Text        string       `json:"text"`
	Username    string       `json:"username"`
	IconUrl     string       `json:"icon_url"`
	IconEmoji   string       `json:"icon_emoji"`
	Channel     string       `json:"channel"`
	Attachments []Attachment `json:"attachments"`
}

type SlackIncomingWebhookClient struct {
	Url string
}

func (client *SlackIncomingWebhookClient) SendMessage(payload Payload) (err error) {
	encodedJsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(client.Url, "application/json", bytes.NewReader(encodedJsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	result := string(body)
	if result != "ok" {
		return errors.New(result)
	}

	return nil
}
