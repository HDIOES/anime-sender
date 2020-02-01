package main

import (
	"encoding/json"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

//TelegramService struct
type TelegramService struct {
	HTTPGateway *HTTPGateway
	Settings    *Settings
}

func (ts *TelegramService) receiveNotification(msg *nats.Msg) {
	if notification, notificationErr := decodeNotification(msg); notificationErr != nil {
		HandleError(notificationErr)
	} else {
		switch notification.Type {
		case "startCommand":
			{
				if err := ts.sendStartMessage(notification); err != nil {
					HandleError(err)
				}
			}
		case "animesCommand":
			{
				if err := ts.sendAnimesMessage(notification); err != nil {
					HandleError(err)
				}
			}
		case "subscriptionsCommand":
			{
				if err := ts.sendSubscriptionsMessage(notification); err != nil {
					HandleError(err)
				}
			}
		case "defaultCommand":
			{
				if err := ts.sendDefaultMessage(notification); err != nil {
					HandleError(err)
				}
			}
		case "setWebhookNotification":
			{
				if err := ts.sendSetWebhookMessage(notification); err != nil {
					HandleError(err)
				}
			}
		}
	}
}

func decodeNotification(msg *nats.Msg) (*Notification, error) {
	notification := &Notification{}
	unmarshalErr := json.Unmarshal(msg.Data, notification)
	if unmarshalErr != nil {
		return nil, errors.WithStack(unmarshalErr)
	}
	return notification, nil
}

//Notification struct
type Notification struct {
	TelegramID int64    `json:"telegramId"`
	Type       string   `json:"type"`
	Text       string   `json:"text"`
	Animes     []string `json:"animes"`
}

func (ts *TelegramService) sendStartMessage(notification *Notification) error {
	sendMessage := SendMessage{
		Text:   notification.Text,
		ChatID: notification.TelegramID,
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", sendMessage)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) sendAnimesMessage(notification *Notification) error {
	sendMessage := SendMessage{
		ChatID:      notification.TelegramID,
		Text:        notification.Text,
		ReplyMarkup: &ReplyKeyboardMarkup{},
	}
	count := len(notification.Animes)
	sendMessage.ReplyMarkup.Keyboard = make([][]KeyboardButton, count)
	for i := 0; i < count; i++ {
		sendMessage.ReplyMarkup.Keyboard[i] = make([]KeyboardButton, 1)
		sendMessage.ReplyMarkup.Keyboard[i][0] = KeyboardButton{
			Text: notification.Animes[i],
		}
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", sendMessage)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) sendSubscriptionsMessage(notification *Notification) error {
	sendMessage := SendMessage{
		ChatID:      notification.TelegramID,
		Text:        notification.Text,
		ReplyMarkup: &ReplyKeyboardMarkup{},
	}
	count := len(notification.Animes)
	sendMessage.ReplyMarkup.Keyboard = make([][]KeyboardButton, count)
	for i := 0; i < count; i++ {
		sendMessage.ReplyMarkup.Keyboard[i] = make([]KeyboardButton, 1)
		sendMessage.ReplyMarkup.Keyboard[i][0] = KeyboardButton{
			Text: notification.Animes[i],
		}
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", sendMessage)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) sendDefaultMessage(notification *Notification) error {
	sendMessage := SendMessage{
		ChatID: notification.TelegramID,
		Text:   notification.Text,
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", sendMessage)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) sendSetWebhookMessage(notification *Notification) error {
	file, err := os.Open(ts.Settings.PathToPublicKey)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()
	parameters := make(map[string]interface{}, 2)
	//write certificate
	parameters["certificate"] = file
	//write url
	parameters["url"] = ts.Settings.WebhookURL
	httpStatus, resErr := ts.HTTPGateway.PostWithApplicationForm(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/setWebhook", parameters)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

//SendMessage struct
type SendMessage struct {
	ChatID      int64                `json:"chat_id"`
	Text        string               `json:"text"`
	ReplyMarkup *ReplyKeyboardMarkup `json:"reply_markup,omitempty"`
}

//ReplyKeyboardMarkup struct
type ReplyKeyboardMarkup struct {
	Keyboard [][]KeyboardButton `json:"keyboard"`
}

//KeyboardButton struct
type KeyboardButton struct {
	Text string `json:"text"`
}
