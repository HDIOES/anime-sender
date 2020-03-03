package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

//URLTemplate is const to initialize new URL
const URLTemplate = "%s%s%s"

const (
	startType       = "startType"
	answerQueryType = "answerQueryType"
	subscribeType   = "subscribeType"
	unsubscribeType = "unsubscribeType"
	defaultType     = "defaultType"
)

const (
	sendMessageURL            = "/sendMessage"
	sendPhotoURL              = "/sendPhoto"
	answerCallbackQueryURL    = "/answerCallbackQuery"
	answerInlineQueryURL      = "/answerInlineQuery"
	editMessageReplyMarkupURL = "/editMessageReplyMarkup"
)

//TelegramService struct
type TelegramService struct {
	HTTPGateway *HTTPGateway
	Settings    *Settings
}

func (ts *TelegramService) receiveTelegramMessageFromQueue(msg *nats.Msg) {
	errors := make([]error, 0, 2)
	if telegramMessage, decodeErr := decodeMessage(msg); decodeErr != nil {
		errors = append(errors, decodeErr)
	} else {
		switch telegramMessage.Type {
		case startType:
			{
				if telegramMessage.InlineAnime == nil {
					if err := ts.sendStartMessage(telegramMessage); err != nil {
						errors = append(errors, err)
					}
				} else {
					if err := ts.animeInfoMessage(telegramMessage); err != nil {
						errors = append(errors, err)
					}
				}
			}
		case answerQueryType:
			{
				if err := ts.answerInlineQuery(telegramMessage); err != nil {
					errors = append(errors, err)
				}
			}
		case subscribeType:
			{
				answerErr, editMessageErr := ts.answerCallbackQueryWithSubscribeInfo(telegramMessage)
				if answerErr != nil {
					errors = append(errors, answerErr)
				}
				if editMessageErr != nil {
					errors = append(errors, editMessageErr)
				}
			}
		case unsubscribeType:
			{
				answerErr, editMessageErr := ts.answerCallbackQueryWithUnsubscribeInfo(telegramMessage)
				if answerErr != nil {
					errors = append(errors, answerErr)
				}
				if editMessageErr != nil {
					errors = append(errors, editMessageErr)
				}
			}
		case defaultType:
			{
				if err := ts.sendDefaultMessage(telegramMessage); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}
	for _, e := range errors {
		HandleError(e)
	}
}

func decodeMessage(msg *nats.Msg) (*TelegramCommandMessage, error) {
	message := &TelegramCommandMessage{}
	unmarshalErr := json.Unmarshal(msg.Data, message)
	if unmarshalErr != nil {
		return nil, errors.WithStack(unmarshalErr)
	}
	stringLogBuilder := strings.Builder{}
	stringLogBuilder.WriteString("NATS message body:\n")
	stringLogBuilder.Write(msg.Data)
	stringLogBuilder.WriteString("\n")
	log.Print(stringLogBuilder)
	return message, nil
}

//TelegramCommandMessage struct
type TelegramCommandMessage struct {
	Type string `json:"type"`
	//fields for notification and /start
	TelegramID  int64        `json:"telegramId"`
	Text        string       `json:"text"`
	InlineAnime *InlineAnime `json:"inlineAnime"`
	//inline query fields
	InlineQueryID string        `json:"inlineQueryId"`
	InlineAnimes  []InlineAnime `json:"inlineAnimes"`
	//fields for subscribe/unsubscribe action
	ChatID          int64  `json:"chatId"`
	MessageID       int64  `json:"messageId"`
	CallbackQueryID string `json:"callback_query_id"`
	InternalAnimeID int64  `json:"internal_anime_id"`
}

//InlineAnime struct
type InlineAnime struct {
	InternalID           int64  `json:"id"`
	AnimeName            string `json:"animeName"`
	AnimeThumbnailPicURL string `json:"animeThumbNailPicUrl"`
	UserHasSubscription  bool   `json:"userHasSubscription"`
}

func (ts *TelegramService) sendStartMessage(message *TelegramCommandMessage) error {
	sendMessage := SendMessage{
		Text:   message.Text,
		ChatID: message.TelegramID,
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, sendMessageURL), sendMessage)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) animeInfoMessage(message *TelegramCommandMessage) error {
	sendPhoto := SendPhoto{
		ChatID:      message.TelegramID,
		Caption:     message.InlineAnime.AnimeName,
		Photo:       message.InlineAnime.AnimeThumbnailPicURL,
		ReplyMarkup: &InlineKeyboardMarkup{},
	}
	sendPhoto.ReplyMarkup.Keyboard = make([][]InlineKeyboardButton, 1)
	sendPhoto.ReplyMarkup.Keyboard[0] = make([]InlineKeyboardButton, 1)
	if message.InlineAnime.UserHasSubscription {
		sendPhoto.ReplyMarkup.Keyboard[0][0] = InlineKeyboardButton{
			Text:         "Отписаться",
			CallbackData: fmt.Sprintf("unsub %d", message.InlineAnime.InternalID),
		}
	} else {
		sendPhoto.ReplyMarkup.Keyboard[0][0] = InlineKeyboardButton{
			Text:         "Подписаться",
			CallbackData: fmt.Sprintf("sub %d", message.InlineAnime.InternalID),
		}
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, sendPhotoURL), sendPhoto)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) sendDefaultMessage(message *TelegramCommandMessage) error {
	sendMessage := SendMessage{
		ChatID: message.TelegramID,
		Text:   message.Text,
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, sendMessageURL), sendMessage)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) answerInlineQuery(message *TelegramCommandMessage) error {
	answerInlineQuery := AnswerInlineQuery{
		InlineQueryID: message.InlineQueryID,
		CacheTime:     0,
	}
	answerInlineQuery.Results = make([]InlineQueryResultArticle, 0, len(message.InlineAnimes))
	for i, anime := range message.InlineAnimes {
		animeInfo := InlineQueryResultArticle{
			Type:     "article",
			ID:       strconv.Itoa(i),
			Title:    anime.AnimeName,
			ThumbURL: anime.AnimeThumbnailPicURL,
			InputTextMessageContent: InputTextMessageContent{
				MessageText: fmt.Sprintf("%s %s", anime.AnimeName, anime.AnimeThumbnailPicURL),
			},
		}
		if anime.UserHasSubscription {
			animeInfo.Description = "Подписка есть"
		}
		animeInfo.ReplyMarkup.Keyboard = make([][]InlineKeyboardButton, 1)
		animeInfo.ReplyMarkup.Keyboard[0] = make([]InlineKeyboardButton, 1)
		animeInfo.ReplyMarkup.Keyboard[0][0] = InlineKeyboardButton{
			Text: "Подробнее",
			URL:  fmt.Sprintf(ts.Settings.OngoingBotURL, anime.InternalID),
		}
		answerInlineQuery.Results = append(answerInlineQuery.Results, animeInfo)
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, answerInlineQueryURL), answerInlineQuery)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	if httpStatus != 200 {
		return errors.New("Http status not equals 200")
	}
	return nil
}

func (ts *TelegramService) answerCallbackQueryWithSubscribeInfo(message *TelegramCommandMessage) (answerErr error, editMessageErr error) {
	var answerCallbackBarrier sync.WaitGroup
	answerCallbackBarrier.Add(2)
	go func() {
		defer answerCallbackBarrier.Done()
		err := ts.answerCallbackQuery(message.CallbackQueryID)
		if err != nil {
			answerErr = err
		}
	}()
	go func() {
		defer answerCallbackBarrier.Done()
		editMessageReplyMarkup := EditMessageReplyMarkup{
			ChatID:      message.ChatID,
			MessageID:   message.MessageID,
			ReplyMarkup: InlineKeyboardMarkup{},
		}
		editMessageReplyMarkup.ReplyMarkup.Keyboard = make([][]InlineKeyboardButton, 1)
		editMessageReplyMarkup.ReplyMarkup.Keyboard[0] = make([]InlineKeyboardButton, 1)
		editMessageReplyMarkup.ReplyMarkup.Keyboard[0][0] = InlineKeyboardButton{
			Text:         "Отписаться",
			CallbackData: fmt.Sprintf("unsub %d", message.InternalAnimeID),
		}
		httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, editMessageReplyMarkupURL), editMessageReplyMarkup)
		if resErr != nil {
			editMessageErr = errors.WithStack(resErr)
		}
		if httpStatus != 200 {
			editMessageErr = errors.New("Http status not equals 200")
		}
	}()
	answerCallbackBarrier.Wait()
	return nil, nil
}

func (ts *TelegramService) answerCallbackQueryWithUnsubscribeInfo(message *TelegramCommandMessage) (answerErr error, editMessageErr error) {
	var answerCallbackBarrier sync.WaitGroup
	answerCallbackBarrier.Add(2)
	go func() {
		defer answerCallbackBarrier.Done()
		err := ts.answerCallbackQuery(message.CallbackQueryID)
		if err != nil {
			answerErr = err
		}
	}()
	go func() {
		defer answerCallbackBarrier.Done()
		editMessageReplyMarkup := EditMessageReplyMarkup{
			ChatID:      message.ChatID,
			MessageID:   message.MessageID,
			ReplyMarkup: InlineKeyboardMarkup{},
		}
		editMessageReplyMarkup.ReplyMarkup.Keyboard = make([][]InlineKeyboardButton, 1)
		editMessageReplyMarkup.ReplyMarkup.Keyboard[0] = make([]InlineKeyboardButton, 1)
		editMessageReplyMarkup.ReplyMarkup.Keyboard[0][0] = InlineKeyboardButton{
			Text:         "Подписаться",
			CallbackData: fmt.Sprintf("sub %d", message.InternalAnimeID),
		}
		httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, editMessageReplyMarkupURL), editMessageReplyMarkup)
		if resErr != nil {
			editMessageErr = errors.WithStack(resErr)
		}
		if httpStatus != 200 {
			editMessageErr = errors.New("Http status not equals 200")
		}
	}()
	answerCallbackBarrier.Wait()
	return nil, nil
}

func (ts *TelegramService) answerCallbackQuery(callbackQueryID string) error {
	answerCallbackQuery := AnswerCallbackQuery{
		CallbackQueryID: callbackQueryID,
	}
	httpStatus, resErr := ts.HTTPGateway.PostWithJSONApplication(fmt.Sprintf(URLTemplate, ts.Settings.TelegramURL, ts.Settings.TelegramToken, answerCallbackQueryURL), answerCallbackQuery)
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
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

//SendPhoto struct
type SendPhoto struct {
	ChatID      int64                 `json:"chat_id"`
	Photo       string                `json:"photo"`
	Caption     string                `json:"caption"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

//AnswerInlineQuery struct
type AnswerInlineQuery struct {
	InlineQueryID string                     `json:"inline_query_id"`
	CacheTime     int                        `json:"cache_time"`
	Results       []InlineQueryResultArticle `json:"results"`
}

//AnswerCallbackQuery struct
type AnswerCallbackQuery struct {
	CallbackQueryID string `json:"callback_query_id"`
}

//InlineQueryResultArticle struct
type InlineQueryResultArticle struct {
	Type                    string                  `json:"type"`
	ID                      string                  `json:"id"`
	Title                   string                  `json:"title"`
	Description             string                  `json:"description"`
	ThumbURL                string                  `json:"thumb_url"`
	InputTextMessageContent InputTextMessageContent `json:"input_message_content"`
	ReplyMarkup             InlineKeyboardMarkup    `json:"reply_markup"`
}

//InputTextMessageContent struct
type InputTextMessageContent struct {
	MessageText string `json:"message_text"`
}

//InlineKeyboardMarkup struct
type InlineKeyboardMarkup struct {
	Keyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

//InlineKeyboardButton struct
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

//EditMessageReplyMarkup struct
type EditMessageReplyMarkup struct {
	ChatID      int64                `json:"chat_id"`
	MessageID   int64                `json:"message_id"`
	ReplyMarkup InlineKeyboardMarkup `json:"reply_markup"`
}
