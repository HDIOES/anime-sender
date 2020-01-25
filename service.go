package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
)

//TelegramService struct
type TelegramService struct {
	Client   *http.Client
	Settings *Settings
}

func (ts *TelegramService) receiveNotification(msg *nats.Msg) {
	if notification, notificationErr := decodeNotification(msg); notificationErr != nil {
		fmt.Println(notificationErr)
	} else {
		switch notification.Type {
		case "startCommand":
			{
				if err := ts.sendStartMessage(notification); err != nil {
					fmt.Println(err)
				}
			}
		case "animesCommand":
			{
				if err := ts.sendAnimesMessage(notification); err != nil {
					fmt.Println(err)
				}
			}
		case "subscriptionsCommand":
			{
				if err := ts.sendSubscriptionsMessage(notification); err != nil {
					fmt.Println(err)
				}
			}
		case "defaultCommand":
			{
				if err := ts.sendDefaultMessage(notification); err != nil {
					fmt.Println(err)
				}
			}
		case "setWebhookNotification":
			{
				if err := ts.sendSetWebhookMessage(notification); err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}

func decodeNotification(msg *nats.Msg) (*Notification, error) {
	notification := &Notification{}
	unmarshalErr := json.Unmarshal(msg.Data, notification)
	if unmarshalErr != nil {
		return nil, unmarshalErr
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
	data := url.Values{}
	data.Set("text", notification.Text)
	data.Set("chat_id", strconv.FormatInt(notification.TelegramID, 10))
	_, resErr := ts.Client.Post(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if resErr != nil {
		return resErr
	}
	return nil
}

func (ts *TelegramService) sendAnimesMessage(notification *Notification) error {
	sendMessage := SendMessage{
		ChatID:      notification.TelegramID,
		Text:        notification.Text,
		ReplyMarkup: InlineKeyboardMarkup{},
	}
	count := len(notification.Animes)
	sendMessage.ReplyMarkup.InlineKeyboard = make([][]InlineKeyboardButton, count)
	for i := 0; i < count; i++ {
		sendMessage.ReplyMarkup.InlineKeyboard[i] = make([]InlineKeyboardButton, 1)
		sendMessage.ReplyMarkup.InlineKeyboard[i][0] = InlineKeyboardButton{
			Text: notification.Animes[i],
		}
	}
	data, err := json.Marshal(sendMessage)
	if err != nil {
		return err
	}
	_, resErr := ts.Client.Post(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", "application/json", bytes.NewReader(data))
	if resErr != nil {
		return resErr
	}
	return nil
}

func (ts *TelegramService) sendSubscriptionsMessage(notification *Notification) error {
	sendMessage := SendMessage{
		ChatID:      notification.TelegramID,
		Text:        notification.Text,
		ReplyMarkup: InlineKeyboardMarkup{},
	}
	count := len(notification.Animes)
	sendMessage.ReplyMarkup.InlineKeyboard = make([][]InlineKeyboardButton, count)
	for i := 0; i < count; i++ {
		sendMessage.ReplyMarkup.InlineKeyboard[i] = make([]InlineKeyboardButton, 1)
		sendMessage.ReplyMarkup.InlineKeyboard[i][0] = InlineKeyboardButton{
			Text: notification.Animes[i],
		}
	}
	data, err := json.Marshal(sendMessage)
	if err != nil {
		return err
	}
	_, resErr := ts.Client.Post(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", "application/json", bytes.NewReader(data))
	if resErr != nil {
		return resErr
	}
	return nil
}

func (ts *TelegramService) sendDefaultMessage(notification *Notification) error {
	data := url.Values{}
	data.Set("text", notification.Text)
	data.Set("chat_id", strconv.FormatInt(notification.TelegramID, 10))
	_, resErr := ts.Client.Post(ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/sendMessage", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if resErr != nil {
		return resErr
	}
	return nil
}

func (ts *TelegramService) sendSetWebhookMessage(notification *Notification) error {
	file, err := os.Open(ts.Settings.PathToPublicKey)
	if err != nil {
		return err
	}
	defer file.Close()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	//write certificate
	part, err := writer.CreateFormFile("certificate", filepath.Base(file.Name()))
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(part, file)
	if copyErr != nil {
		return copyErr
	}
	//write url
	writeFieldErr := writer.WriteField("url", ts.Settings.WebhookURL)
	if writeFieldErr != nil {
		return writeFieldErr
	}
	writeErr := writer.Close()
	if writeErr != nil {
		return writeErr
	}
	request, reqErr := http.NewRequest("POST", ts.Settings.TelegramURL+ts.Settings.TelegramToken+"/setWebhook", body)
	if reqErr != nil {
		return reqErr
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	res, resErr := ts.Client.Do(request)
	if resErr != nil {
		return resErr
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

//SendMessage struct
type SendMessage struct {
	ChatID      int64                `json:"chat_id"`
	Text        string               `json:"text"`
	ReplyMarkup InlineKeyboardMarkup `json:"reply_markup"`
}

//InlineKeyboardMarkup struct
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

//InlineKeyboardButton struct
type InlineKeyboardButton struct {
	Text string `json:"text"`
}
