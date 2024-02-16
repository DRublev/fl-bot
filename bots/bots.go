package bots

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/go-telegram/bot"
)

var NotificationsBot *bot.Bot
var IsNotificationsBotReady chan bool = make(chan bool, 1)
var OfferChatsBot *bot.Bot
var IsOfferChatBotReady chan bool = make(chan bool, 1)

func StartBots(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	default:
		w := &sync.WaitGroup{}

		w.Add(1)
		go startNotificationsBot(ctx, w)
		w.Add(1)
		go startChatsBot(ctx, w)

		w.Wait()
	}
}

func startNotificationsBot(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	_, isProd := os.LookupEnv("PROD")
	tokenKey := "TG_NOTIFICATIONS_BOT_TOKEN"
	if !isProd {
		tokenKey = "TG_DEV_NOTIFICATIONS_BOT_TOKEN"
	}

	token, exists := os.LookupEnv(tokenKey)
	if !exists {
		log.Default().Println("No notifications bot token provided!")
		IsNotificationsBotReady <- false
		return
	}

	log.Default().Println("Starting notifications bot")

	notificationsBot, err := StartNotificationsBot(token)
	if err != nil {
		log.Default().Println("Failed to start notifications bot: ", err)
		IsNotificationsBotReady <- false
		return
	}
	NotificationsBot = notificationsBot
	IsNotificationsBotReady <- NotificationsBot != nil

	fmt.Println("Starting notifications bot ", NotificationsBot != nil)
	NotificationsBot.Start(ctx)
}

func startChatsBot(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	_, isProd := os.LookupEnv("PROD")
	tokenKey := "TG_OFFER_CHATS_BOT_TOKEN"
	if !isProd {
		tokenKey = "TG_DEV_OFFER_CHATS_BOT_TOKEN"
	}

	token, exists := os.LookupEnv(tokenKey)
	if !exists {
		log.Default().Println("No offer chats bot token provided!")
		IsOfferChatBotReady <- false
		return
	}

	log.Default().Println("Starting offer chats bot")

	offerChatsBot, err := StartOfferChatsBot(token)
	if err != nil {
		log.Default().Println("Failed to start ffer chats bot: ", err)
		IsOfferChatBotReady <- false
		return
	}
	OfferChatsBot = offerChatsBot
	IsOfferChatBotReady <- offerChatsBot != nil

	offerChatsBot.Start(ctx)
}
