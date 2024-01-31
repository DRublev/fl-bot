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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"fl.ru/chromeproxy"

	"os"

	"github.com/SlyMarbo/rss"
	"github.com/chromedp/chromedp"

	"log"
	"os/signal"
	"time"

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

const TOKEN string = "6721949149:AAG7WYIY6PmJCmpJY5eA3Il12tQQNw1jjfE"

var CHATS []string
var WATCH_CATEGORIES []string

var ctx context.Context

var csrfToken string = "o36az3gzMApvH2fARzszWHBg6da7PPSXtiGmvyM2"
var cookies string = "__ddg1_=RUHPwLZ1Zbwp2AP3LBHX; PHPSESSID=ngBgFPnF9cZZCrCE2eKMcR9LIg7qxG17VJUIAVvm; XSRF-TOKEN=o36az3gzMApvH2fARzszWHBg6da7PPSXtiGmvyM2"

func main() {
	// CHATS = []string{"972086219", "713587013"}
	CHATS = []string{"713587013"}
	// WATCH_CATEGORIES = []string{"3", "10", "17", "19"}
	WATCH_CATEGORIES = []string{}
	// WATCH_CATEGORIES = []string{"1", "2", "4", "5", "6", "7", "8", "9", "11", "3", "10", "17", "19"}

	// now := time.Now()
	// initialCheckDate := now.Add(time.Duration(-30) * time.Second)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	startBot(nil)

	wg := &sync.WaitGroup{}

	token, _, cancelChrome := login(b, ctx)
	defer cancelChrome()

	go getChatMessages(token)

	// for _, category := range WATCH_CATEGORIES {
	// 	wg.Add(1)
	// 	go watchCategory(wg, b, ctx, category, initialCheckDate)
	// }

	wg.Wait()

	input := bufio.NewScanner(os.Stdin)
	var kw string
	input.Scan()
	kw = input.Text()
	if kw == "e" {
		return
	}
}

func getChatMessages(token string) {
	req, err := http.NewRequest("GET", "https://www.fl.ru/projects/offers/?limit=20&dialogues=1&deleted=1&sort=lastMessage&offset=0", nil)
	if err != nil {
		fmt.Println("Error getting chats", err)
	}

	req.Header.Set("x-csrf-token", csrfToken)
	req.Header.Set("x-xsrf-token", csrfToken)
	req.Header.Set("Cookie", cookies)

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

	fmt.Println("Response: ", string(body))
}

func watchCategory(wg *sync.WaitGroup, b *bot.Bot, ctx context.Context, category string, initialCheckDate time.Time) {
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
				sendUpdates(&ctx, b, &rescentItems)
			} else {
				fmt.Println("Timer tick, but ctx is nil")
			}
		}
	}

}

func sendUpdates(ctx *context.Context, b *bot.Bot, items *[]rss.Item) {
	for _, item := range *items {
		message := formatUpdateMessage(&item)

		for _, chatId := range CHATS {
			_, err := b.SendMessage(*ctx, &bot.SendMessageParams{
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

func login(b *bot.Bot, botCtx context.Context) (string, chan bool, func() error) {
	url := "https://www.fl.ru/account/login/"
	chromeproxy.PrepareProxy(":9223", ":9221", chromedp.DisableGPU)

	ip := getLocalIp()

	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	if err != nil {
		log.Fatalln("Error launching chrome: ", err.Error())
	}
	ctx := chromeproxy.GetTarget(targetId)

	isSucceed := make(chan bool, 1)

	token := "jdd5q2MM27OsfXv8PQhz32FTNUrMXvgrizUYJl4S"
	if len(token) > 0 {
		isSucceed <- true
		return token, isSucceed, func() error {
			return nil
		}
	}

	go func() {
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("input[name='username']", chromedp.NodeVisible),
			chromedp.WaitVisible("input[name='password']", chromedp.NodeVisible),

			chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='username']`); username.value = 'Nast-ka.666@mail.ru' })()", nil),
			chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='password']`); username.value = 'fyrgonSk-Doo2023' })()", nil),
			// chromedp.Evaluate("() => (window.csrf_token && !document.querySelector(`[data-id='qa-head-sign-in']`)) || '1234123'", nil),
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
				// msg := "Login here: http://" + ip + ":9221/?id=" + string(targetId)
				// b.SendMessage(botCtx, &bot.SendMessageParams{
				// 	ChatID: CHATS[0],
				// 	Text:   string(msg),
				// })
				fmt.Println("Login here: http://" + ip + ":9221/?id=" + string(targetId))
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
				select {
				case <-time.Tick((time.Second * 2)):
					return nil
				}
			}),
			chromedp.Evaluate("window.csrf_token", &result),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("result token", string(result))
				if len(result) > 0 {
					isSucceed <- true
				}
				return nil
			}),
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			isSucceed <- false
			log.Println("Error waiting for auth token: ", err)
		}
	}()

	go func() {
		var result []byte
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("#navbarRightDropdown"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				select {
				case <-time.Tick((time.Second * 2)):
					return nil
				}
			}),
			chromedp.Evaluate("document.querySelector(`meta[name='csrf-token']`).content", &result),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("csrf token", string(result))
				if len(result) > 0 {
					csrfToken = string(result)
				}
				return nil
			}),
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			isSucceed <- false
			log.Println("Error waiting for auth token: ", err)
		}
	}()

	go func() {
		var result []byte
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("#navbarRightDropdown"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				select {
				case <-time.Tick((time.Second * 2)):
					return nil
				}
			}),
			chromedp.Evaluate("document.cookie", &result),
			chromedp.ActionFunc(func(ctx context.Context) error {
				fmt.Println("cookies", string(result))
				if len(result) > 0 {
					cookies = string(result)
				}
				return nil
			}),
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			isSucceed <- false
			log.Println("Error waiting for auth token: ", err)
		}
	}()

	go func() {
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitVisible(`.recaptcha-checkbox-checked`),
			chromedp.Click("#submit-button", chromedp.NodeNotVisible),
			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Print("Captcha solved!")
				isSucceed <- true
				return nil
			}),
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			isSucceed <- false
			log.Println("Error waiting capcha solved: ", err)
		}
	}()

	return token, isSucceed, func() error {
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
