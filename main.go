package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/nats-io/nats.go"
	"go.uber.org/dig"
)

func main() {
	container := dig.New()
	container.Provide(func() *Settings {
		if jsonFile, openErr := os.Open("settings.json"); openErr != nil {
			panic(openErr)
		} else {
			defer jsonFile.Close()
			decoder := json.NewDecoder(jsonFile)
			settings := &Settings{}
			if decodeErr := decoder.Decode(settings); decodeErr != nil {
				panic(decodeErr)
			} else {
				return settings
			}
		}
	})
	container.Provide(func(settings *Settings) (*nats.Conn, *TelegramService) {
		natsConnection, ncErr := nats.Connect(settings.NatsURL)
		if ncErr != nil {
			panic(ncErr)
		}
		service := &TelegramService{
			Client:   &http.Client{},
			Settings: settings,
		}
		return natsConnection, service
	})
	container.Invoke(func(settings *Settings, telegramService *TelegramService, natsConnection *nats.Conn) {
		natsConnection.Subscribe(settings.NatsSubject, telegramService.receiveNotification)
		srv := &http.Server{Addr: ":8001"}
		log.Fatal(srv.ListenAndServe())
	})
}

//Settings mapping object for settings.json
type Settings struct {
	NatsURL         string `json:"natsUrl"`
	NatsSubject     string `json:"natsSubject"`
	TelegramToken   string `json:"telegramToken"`
	TelegramURL     string `json:"telegramUrl"`
	PathToPublicKey string `json:"pathToPublicKey"`
	WebhookURL      string `json:"webhook"`
}
