package bots

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func StartOfferChatsBot(token string) (*bot.Bot, error) {
	if len(token) == 0 {
		return nil, errors.New("must provide a token")
	}
	options := []bot.Option{
		bot.WithDefaultHandler(defaultMessageHandler),
	}

	b, err := bot.New(token, options...)

	if err != nil {
		fmt.Println("Error starting bot ", err)
		return nil, err
	}

	fmt.Println("StartOfferChatsBot defined ", b)
	return b, nil
}

func defaultMessageHandler(c context.Context, b *bot.Bot, update *models.Update) {
	fmt.Println(update.Message.Chat.ID)
	b.SendMessage(c, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
