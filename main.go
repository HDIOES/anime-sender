package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

func main() {
	container := dig.New()
	container.Provide(func() *Settings {
		if jsonFile, openErr := os.Open("settings.json"); openErr != nil {
			log.Panicln(openErr)
		} else {
			defer jsonFile.Close()
			decoder := json.NewDecoder(jsonFile)
			settings := &Settings{}
			if decodeErr := decoder.Decode(settings); decodeErr != nil {
				log.Panicln(decodeErr)
			} else {
				return settings
			}
		}
		panic("Unreachable code")
	})
	container.Provide(func(settings *Settings) (*nats.Conn, *TelegramService) {
		natsConnection, ncErr := nats.Connect(settings.NatsURL)
		if ncErr != nil {
			log.Panicln(ncErr)
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

//StackTracer struct
type StackTracer interface {
	StackTrace() errors.StackTrace
}

//HandleError func
func HandleError(handledErr error) {
	if err, ok := handledErr.(StackTracer); ok {
		for _, f := range err.StackTrace() {
			log.Printf("%+s:%d\n", f, f)
		}
	} else {
		log.Println("Unknown error: ", err)
	}
}
