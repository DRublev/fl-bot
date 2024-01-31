// получить заказы через https://www.fl.ru/rss/all.xml?subcategory=37&category=5
// проверить  новые заказы (с момента последней проверки)
// для новых заказов выбрать шаблон отклика
// заполинть шаблон
// откликнуться на заказ

// go parse rss https://github.com/mmcdole/gofeed
// go http - https://pkg.go.dev/net/http

// Проект по откликам на fl https://github.com/valentinkh1/fl.ru.am/blob/master/src/common/js/background.js

// recaptcha solver - https://github.com/JacobLinCool/recaptcha-solver
// https://pkg.go.dev/github.com/metabypass/captcha-solver-go#section-readme
// https://capmonster.cloud/ru/l/recaptcha?h=recaptcha?utm_source=yandex.search&utm_medium=cpc&utm_campaign=76898902&utm_content=5130188909.13474568196&utm_term=Recaptcha%20solving.43263016990&dv=desktop&pos=premium1&place=none&match=syn&feed=feed&added=&etext=2202.iEoGqpKp1BmGv-vHxH68KrwvCRfiDZXTiIZZRNqBu0JqY3R0YWRvZHZ2b3NtaHpu.1c2844cbcc9ed9a8a8a2eb33d76d08474fe38c68&yclid=1804780213823340543
// ??? Можно пднять как отдельный сервис в соседнем контейнере, скармливать ему процесс хрома

// Для авторизации можно слать ссылку в телегу на открытый хромиум со страницей авторизацц
// Получать токен и сохранять
// Потом этот токен юзать во всей проге и обновлять с помощью https://github.com/valentinkh1/fl.ru.am/blob/7bec76f076f073ac8cfa0203d7fa361839c8628a/src/common/js/background.js#L27

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"fl.ru/bots"
	"fl.ru/chromeproxy"

	"os"

	"log"
	"os/signal"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/go-telegram/bot"
)

// categories 3 10 17 19

/*
SELECTORS
project_info_5275913 - инфа с описанием и кнопкой отклика
a.ui-button._responsive._primary _md - кнопка отклика с про
#vacancy-offer button._primary - кнопка отклика, когда доступна
#vacancy-offer input[data-id="file"] - загрузка резюме, если доступна
#vacancy-offer textarea - поле для отклика
#projectp5275913 - описание проекта
.py-32.text-right.unmobile.flex-shrink-0.ml-auto.mobile - бюджет и дедлайн
*/

var CHATS []string
var WATCH_CATEGORIES []string

var ctx context.Context

var csrfToken string = "\"4X7UcBStnbhXpWmqujzDO38csegsw7qK50cRq76I\""
var cookies string = "\"uechat_3_pages_count=4;_ga_RD9LL0K106=GS1.1.1706716444.1.1.1706716484.0.0.0;pwd=ed02ae7a7ac284a3acb76c7abf1940b8;name=aringai09;_ga=GA1.2.1617029702.1706716445;uechat_3_mode=0;uechat_3_first_time=1706716445187;_ym_d=1706716445;_ym_uid=1706716445483799674;analytic_id=1706716447023416;_ym_visorc=w;PHPSESSID=k06LScKmXkhwwyaYaBKjFL9gR00YL4AFYQqUobJB;_gat=1;_gid=GA1.2.691180979.1706716445;uechat_3_disabled=true;id=8488671;XSRF-TOKEN=4X7UcBStnbhXpWmqujzDO38csegsw7qK50cRq76I;user_device_id=0fv59x3qw9thbxeh82v8hi5dw9ucyssm;_ym_isad=2;_ga_cid=1617029702.1706716445;__ddg1_=76ExwmPsn2gTwMmAA1PL;\""

var notificationsBot *bot.Bot
var isNotificationsBotReady chan bool = make(chan bool, 1)
var offerChatsBot *bot.Bot
var isOfferChatBotReady chan bool = make(chan bool, 1)

// = "PHPSESSID=yzlIAzYjpr1wYVBb64ANQ4cy1VcADjt9GOpNsPOH;"+"\"XSRF-TOKEN=XI1rnYgonhbszJQjkMdQu6Wgn10HCdyuB1OQgWkX; _gid=GA1.2.1816440299.1706715424; _ga_cid=1405734.1706715424; _gat=1; _ym_uid=1706715424607799129; _ym_d=1706715424; _ym_isad=2; uechat_3_first_time=1706715424109; _ym_visorc=w; uechat_3_disabled=true; uechat_3_mode=0; analytic_id=1706715425730366; _ga_RD9LL0K106=GS1.1.1706715423.1.1.1706715474.0.0.0; _ga=GA1.2.1405734.1706715424; uechat_3_pages_count=4\""

// Докинуть команду подписки
// Писать последнию дату синхронизации в файл для каждого подписчика

func main() {
	// CHATS = []string{"972086219", "713587013"}
	CHATS = []string{"713587013"}
	// WATCH_CATEGORIES = []string{"3", "10", "17", "19"}
	WATCH_CATEGORIES = []string{}
	// WATCH_CATEGORIES = []string{"1", "2", "4", "5", "6", "7", "8", "9", "11", "3", "10", "17", "19"}

	now := time.Now()
	initialCheckDate := now.Add(time.Duration(-30) * time.Second)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	wg := &sync.WaitGroup{}

	go func() {
		notificationsBot, err := bots.StartNotificationsBot()
		if err != nil {
			isNotificationsBotReady <- false
			log.Panicln("Error staring notifications bot ", err)
		}
		isNotificationsBotReady <- notificationsBot != nil
		notificationsBot.Start(ctx)
	}()

	fmt.Println("Starting offerChatsBot")
	offerChatsBot, err := bots.StartOfferChatsBot()
	if err != nil {
		isOfferChatBotReady <- false
		log.Panicln("Error staring notifications bot ", err)
	}
	fmt.Println("offerChatsBot defined", offerChatsBot)

	isOfferChatBotReady <- offerChatsBot != nil
	go func() {
		offerChatsBot.Start(ctx)
		fmt.Println("offerChatsBot started")
	}()

	if <-isNotificationsBotReady {
		for _, category := range WATCH_CATEGORIES {
			wg.Add(1)
			go watchCategory(wg, ctx, category, initialCheckDate)
		}
	}

	isSucceed := make(chan bool, 1)
	if len(cookies) == 0 {
		isOk, cancelChrome := login(notificationsBot)
		isSucceed <- <-isOk
		defer cancelChrome()
	} else {
		isSucceed <- true
	}

	if <-isSucceed {
		wg.Add(1)
		go getChatMessages(&ctx, wg, offerChatsBot)
	}

	wg.Wait()

	input := bufio.NewScanner(os.Stdin)
	var kw string
	input.Scan()
	kw = input.Text()
	if kw == "e" {
		return
	}
}

type OfferResponseItem struct {
	Id          int    `json:id`
	OrderUrl    string `json:order_url`
	Title       string `json:title`
	Description string `json:description`
	ProjectId   int    `json:project_id`
}

type OffersResponse struct {
	LastOfferTime int  `json:"last_offer_time"`
	HasNextPage   bool `json:"has_next_page"`
	// Отклики
	Items [](map[string]any) `json:"items"`
	// Все сообщения
	Messages [](map[string]any) `json:"messages"`
	// Список проектов == список чатов
	Projects [](map[string]any) `json:"projects"`
}

// Мапа с проектами и месседжами
// Нужно сматчить чаты в телеге с айдишниками авторов ?? - при /start просить логин из fl и запоминать
// Нужно хранить все в файлах

func getChatMessages(c *context.Context, wg *sync.WaitGroup, b *bot.Bot) {
	defer wg.Done()

	req, err := http.NewRequest("GET", "https://www.fl.ru/projects/offers/?limit=20&dialogues=1&deleted=1&sort=lastMessage&offset=0", nil)
	if err != nil {
		fmt.Println("Error getting chats", err)
	}
	fmt.Println("offerChatsBot", b)

	fmt.Println("Getting chats with params \n" + strings.Trim(csrfToken, "\"") + "\n" + strings.Trim(cookies, "\""))
	req.Header.Set("x-csrf-token", strings.Trim(csrfToken, "\""))
	req.Header.Set("x-xsrf-token", strings.Trim(csrfToken, "\""))
	req.Header.Set("Cookie", strings.Trim(cookies, "\""))
	req.Header.Set("referer", "https://www.fl.ru/messages/")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Println("Error getting chats res", err)
	}

	fmt.Println("response", res.Status)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error getting chats res", err)
	}
	defer res.Body.Close()

	var result OffersResponse

	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Cannot unmarshal ", err)
	}

	// fmt.Println("Response: ", result.Items[0])
	if <-isOfferChatBotReady {
		for _, item := range result.Items {
			sendNewMessages(c, b, item)
		}
	}

}

func sendNewMessages(c *context.Context, b *bot.Bot, item map[string]any) {
	for _, chatId := range CHATS {
		orderUrl := item["order_url"].(string)
		content := item["description"].(string)
		message := "От: \n" + "По заказу " + orderUrl + "\n" + content
		fmt.Println(b, chatId, "   ", message)
		_, err := b.SendMessage(*c, &bot.SendMessageParams{
			ChatID: chatId,
			Text:   message,
		})
		if err != nil {
			log.Println("Error sending message ", err)
		}
	}
}

func readMessage(projectId int, offerId int) {
	// POST https://www.fl.ru/projects/5272136/offers/73529385/read/
	req, err := http.NewRequest("POST", "https://www.fl.ru/projects/"+string(projectId)+"/offers/"+string(offerId)+"/read/", nil)
	if err != nil {
		log.Println("Error creating read message req", projectId, " ", offerId, " \n", err)
		return
	}

	req.Header.Set("x-csrf-token", strings.Trim(csrfToken, "\""))
	req.Header.Set("x-xsrf-token", strings.Trim(csrfToken, "\""))
	req.Header.Set("Cookie", strings.Trim(cookies, "\""))
	req.Header.Set("referer", "https://www.fl.ru/messages/")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Failed mark message as read ", projectId, " ", offerId, " \n", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Println("Not ok reading", projectId, " ", offerId, ", code ", res.StatusCode)

	}
}

func watchCategory(wg *sync.WaitGroup, ctx context.Context, category string, initialCheckDate time.Time) {
	defer wg.Done()

	fmt.Println("Watching category ", category)
	lastCheckDate := initialCheckDate

	for range time.Tick(time.Second * 5) {
		select {
		case <-ctx.Done():
			return
		default:
			if ctx != nil {
				// Тут можно юзать каналы
				rescentItems := getItemsForCategory(&category, &lastCheckDate)
				fmt.Println("len items ", len(rescentItems))
				sendUpdates(&rescentItems)
			} else {
				fmt.Println("Timer tick, but ctx is nil")
			}
		}
	}

}

func sendUpdates(items *[]rss.Item) {
	for _, item := range *items {
		message := formatUpdateMessage(&item)

		for _, chatId := range CHATS {
			_, err := notificationsBot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatId,
				Text:   message,
			})
			if err != nil {
				fmt.Println("Error sending update message: ", err)
			}
		}
	}
}

func formatUpdateMessage(item *rss.Item) string {
	message := "[" + item.Date.Local().Format("15:04:05 02.01.2006") + "] " + item.Title + "\n" + item.Content + "\n" + item.Link

	message += "\n"

	return message
}

func getItemsForCategory(category *string, lastCheckDate *time.Time) []rss.Item {
	if time.Now().Local().Hour() > 22 || time.Now().Local().Hour() < 8 {
		*lastCheckDate = time.Now().Add(time.Duration(-30) * time.Second)

		return []rss.Item{}
	}
	feed, err := rss.Fetch("https://www.fl.ru/rss/all.xml?category=" + *category)
	if err != nil {
		log.Default().Println("Error getting items for category ", *category, "\n", err, "\n\n")
	}
	mostRescent := getMostRescentItems(feed.Items, *lastCheckDate)
	fmt.Println("items ", lastCheckDate.Local().String(), " ", feed.Items[0].Date.Local().String(), " ", feed.Items[0].Title, " ", len(feed.Items), " ", len(mostRescent))
	if len(mostRescent) > 0 {
		*lastCheckDate = mostRescent[0].Date
	}

	return mostRescent
}

func getMostRescentItems(items []*rss.Item, lastCheck time.Time) []rss.Item {
	filtered := []rss.Item{}
	for _, item := range items {
		if item.Date.After(lastCheck) {
			fmt.Print(item.Date, " | ", item.Title, " | ", item.Link, "\n")
			filtered = append(filtered, *item)
		}
	}
	return filtered
}

// name="websocket-token" - meta token

/*
var r = e.content
              , o = t(document.querySelector('meta[name="websocket-url"]').content, {
                extraHeaders: {
                    Authorization: "Bearer ".concat(r)
                }
            });
*/

// token - window.csrf_token

// script := "function(){const check = () => {alert('check is started');const content = document.innerHTML;if (content.includes('_TOKEN_KEY')) {alert('TOKEN FOUND');}}window.onload(check);check()}()";
// var initScript playwright.Script
// initScript.Content = &script
// err = page.AddInitScript(initScript)
// log.Fatalln(err)

const LOGIN = "aringai09@gmail.com" // 'Nast-ka.666@mail.ru
const PASS = "7fJxtyFQsamsung!"     //fyrgonSk-Doo2023

func login(b *bot.Bot) (chan bool, func() error) {
	url := "https://www.fl.ru/account/login/"
	chromeproxy.PrepareProxy(":9223", ":9221", chromedp.DisableGPU)

	ip := getLocalIp()

	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	if err != nil {
		log.Fatalln("Error launching chrome: ", err.Error())
	}
	ctx := chromeproxy.GetTarget(targetId)

	isSucceed := make(chan bool, 1)
	isCsrfToken := make(chan bool, 1)
	isCookies := make(chan bool, 1)

	go func() {
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("input[name='username']", chromedp.NodeVisible),
			chromedp.WaitVisible("input[name='password']", chromedp.NodeVisible),
			chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='username']`); username.value = '"+LOGIN+"' })()", nil),
			chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='password']`); username.value = '"+PASS+"' })()", nil),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("Login info entered")
				return nil
			}),
		})
		if err != nil {
			log.Fatalln("Error running chromedp task", err)
			isSucceed <- false
		}
	}()

	go func() {
		var result []byte
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("Captcha clicked ", string(result), len(result))
				msg := "Login here: http://" + ip + ":9221/?id=" + string(targetId)
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: CHATS[0],
					Text:   string(msg),
				})
				fmt.Println(msg)
				return nil
			}),
		})

		if err != nil {
			chromeproxy.CloseTarget(targetId)
			isSucceed <- false
			log.Println("Error logging in: ", err)
		}
	}()

	go func() {
		var result []byte
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("#navbarRightDropdown"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				<-time.Tick((time.Second * 2))
				return nil
			}),
			chromedp.Evaluate("document.querySelector(`meta[name='csrf-token']`).content", &result),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("csrf token", string(result))
				if len(result) > 0 {
					csrfToken = string(result)
					isCsrfToken <- true
				} else {
					isCsrfToken <- false
				}
				return nil
			}),
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			isCsrfToken <- false
			log.Println("Error waiting for auth token: ", err)
		}
	}()

	go func() {
		var result []byte
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("#navbarRightDropdown"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				<-time.Tick((time.Second * 2))
				return nil
			}),
			chromedp.Evaluate("document.cookie", &result),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("Getting cookies")
				c, err := network.GetCookies().Do(ctx)
				if err != nil {
					fmt.Println("Error cookies: ", err)
					return err
				}
				cookies = ""
				for i, cookie := range c {
					log.Printf("chrome cookie %d: %+v", i, cookie.Name)
					cookies += cookie.Name + "=" + cookie.Value + ";"
				}
				isCookies <- true

				return nil
			}),
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			isCookies <- false
			log.Println("Error waiting for auth token: ", err)
		}
	}()

	go func() {
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitVisible(`.recaptcha-checkbox-checked`),
			chromedp.Click("#submit-button", chromedp.NodeNotVisible),
			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Print("Captcha solved!")
				return nil
			}),
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			isSucceed <- false
			log.Println("Error waiting capcha solved: ", err)
		}
	}()

	isSucceed <- <-isCsrfToken && <-isCookies

	return isSucceed, func() error {
		err := chromeproxy.CloseTarget(targetId)
		return err
	}
}

// Run every X seconds
// select {
// case <-time.Tick(time.Second * 10):
// wg.Add(1)
// go getToken()
// }

// Write to file
// testProjUrl := "https://www.fl.ru/projects/5275913/razrabotat-poster-dlya-dokumentalnogo-tsikla-monologov-tolko-ip-ili-samozanyatyiy.html"
// res, err := http.Get(testProjUrl)
// if err != nil {
// 	log.Fatal("cannot get page", err)
// }
// defer res.Body.Close()
// body, err := io.ReadAll(res.Body)
// ioutil.WriteFile("pageContent.html", body, 0644)
