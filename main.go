package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

const (
	telegramToken          = "TELEGRAM_TOKEN"
	telegramURL            = "TELEGRAM_URL"
	pathToPublicKey        = "PATH_TO_PUBLIC_KEY"
	applicationPortEnvName = "PORT"
	natsURLEnvName         = "NATS_URL"
	natsSubjectEnvName     = "NATS_SUBJECT"
	webhook                = "WEBHOOK_URL"
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
				setSettingsFromEnv(settings)
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
			HTTPGateway: &HTTPGateway{
				Client: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
				},
			},
			Settings: settings,
		}
		return natsConnection, service
	})
	container.Invoke(func(settings *Settings, telegramService *TelegramService, natsConnection *nats.Conn) {
		natsConnection.Subscribe(settings.NatsSubject, telegramService.receiveNotification)
		srv := &http.Server{Addr: ":" + strconv.Itoa(settings.ApplicationPort)}
		log.Fatal(srv.ListenAndServe())
	})
}

//Settings mapping object for settings.json
type Settings struct {
	NatsURL         string `json:"natsUrl"`
	NatsSubject     string `json:"natsSubject"`
	TelegramToken   string `json:"telegramToken"`
	TelegramURL     string `json:"telegramUrl"`
	ApplicationPort int    `json:"port"`
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

func setSettingsFromEnv(settings *Settings) {
	if value := os.Getenv(telegramToken); value != "" {
		settings.TelegramToken = value
	}
	if value := os.Getenv(telegramURL); value != "" {
		settings.TelegramURL = value
	}
	if value := os.Getenv(pathToPublicKey); value != "" {
		settings.PathToPublicKey = value
	}
	if value := os.Getenv(webhook); value != "" {
		settings.WebhookURL = value
	}
	if value := os.Getenv(applicationPortEnvName); value != "" {
		if intValue, err := strconv.Atoi(value); err != nil {
			log.Panicln(err)
		} else {
			settings.ApplicationPort = intValue
		}
	}
	if value := os.Getenv(natsURLEnvName); value != "" {
		settings.NatsURL = value
	}
	if value := os.Getenv(natsSubjectEnvName); value != "" {
		settings.NatsSubject = value
	}
}
