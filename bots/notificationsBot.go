package bots

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func StartNotificationsBot() (*bot.Bot, error) {
	var token string = "6721949149:AAG7WYIY6PmJCmpJY5eA3Il12tQQNw1jjfE"

	options := []bot.Option{
		bot.WithDefaultHandler(handleBotMessage),
	}

	// token, _ := os.LookupEnv("TG_TOKEN")
	// if !exists || len([]rune(token)) == 0 {
	// 	log.Fatalln("No TG token provided!")
	// }

	b, err := bot.New(token, options...)
	if err != nil {
		fmt.Println("Error initing bot ", err)
		return nil, err
	}

	log.Println("Tg bot inited successfully")
	return b, nil
}

func handleBotMessage(c context.Context, b *bot.Bot, update *models.Update) {
	fmt.Println(update.Message.Chat.ID)
	b.SendMessage(c, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
