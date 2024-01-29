package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var token string

var b *bot.Bot

func startBot(opts []bot.Option) {
	if opts == nil {
		opts = []bot.Option{}
	}

	options := append([]bot.Option{
		bot.WithDefaultHandler(handleBotMessage),
	}, opts...)

	token, exists := os.LookupEnv("TG_TOKEN")

	if !exists || len([]rune(token)) == 0 {
		log.Fatalln("No TG token provided!")
	}

	bt, err := bot.New(token, options...)
	if err != nil {
		panic("Error while starting tg bot! \n" + err.Error())
	}

	b = bt
	bt.Start(ctx)

	log.Println("Tg bot inited successfully")
}

func handleBotMessage(c context.Context, b *bot.Bot, update *models.Update) {
	fmt.Println(update.Message.Chat.ID)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
