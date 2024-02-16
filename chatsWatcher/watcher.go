package chatsWatcher

import (
	"context"
	"fmt"
	"main/bots"
	"main/db"
	"main/fl"
	"sync"
	"time"

	"github.com/go-telegram/bot"
)

const CHECK_PERIOD_SEC = 5

var baseCookieStoragePath []string = []string{"cookies"}
var dbInstance = db.DB{}

func Watch(ctx context.Context, wg *sync.WaitGroup, chatId string) {
	defer wg.Done()

	//"_ga_RD9LL0K106=GS1.1.1708098593.1.1.1708098638.0.0.0;uechat_3_pages_count=4;pwd=fd23188772abad1ad239b87636922a65;name=aringai09;_ga=GA1.2.383238720.1708098593;uechat_3_mode=0;uechat_3_first_time=1708098593606;_ym_d=1708098594;_ym_uid=1708098594636194184;analytic_id=1708098595295194;_ym_visorc=w;PHPSESSID=qFfkw4QQhwilVBCNW5T4U9ji1L1KVxT5mNPrOjYG;_gat=1;_gid=GA1.2.1052214669.1708098594;uechat_3_disabled=true;id=8488671;XSRF-TOKEN=nhRYffdxN7iT230L8yAKOJkmYZKMRQLt5M5wIlHh;user_device_id=xrpmaxn7rk9kyzh9b6bhjgqej22a677l;_ym_isad=2;_ga_cid=383238720.1708098593;__ddg1_=1ZcG5DGL4LAvCQmOjoUK;"
	rememberedCookie := getCookie(chatId)

	flApi := fl.API{
		Cookies: rememberedCookie,
	}

	ticker := time.NewTicker(CHECK_PERIOD_SEC * time.Second)
	for range ticker.C {
		notRead, err := flApi.GetChats(ctx, chatId)
		if err != nil {
			fmt.Println("Cannot get messages for ", chatId, " ", err)
		} else {
			if len(rememberedCookie) == 0 {
				rememberedCookie = "saved"
				wg.Add(1)
				go saveCookie(wg, chatId, flApi.Cookies)
			}
			fmt.Println("Not read messages for ", chatId, ": ", len(notRead))
			for _, item := range notRead {
				sendNewMessages(ctx, chatId, item)
			}
		}
	}
}

func saveCookie(wg *sync.WaitGroup, chatId string, cookie string) {
	defer wg.Done()

	dbPath := append(baseCookieStoragePath, chatId)

	dbInstance.Append(dbPath, []byte(cookie+"\n"))

}

func getCookie(chatId string) string {
	dbPath := append(baseCookieStoragePath, chatId)

	cookiesBytes, err := dbInstance.Get(dbPath)
	if err != nil {
		fmt.Println("Cannot get cookies from db for ", dbPath, err)
		return ""
	}

	fmt.Println("Found cookies for ", chatId)

	return string(cookiesBytes)
}

func sendNewMessages(ctx context.Context, chatId string, item fl.Message) {
	orderUrl := item.Project.Url
	content := item.Text
	message := "От: " + item.Project.Author + "\n" + "По заказу " + item.Project.Name + " (" + orderUrl + ")\n" + content
	bots.OfferChatsBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatId,
		Text:   message,
	})
}
