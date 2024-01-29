// получить заказы через https://www.fl.ru/rss/all.xml?subcategory=37&category=5
// проверить  новые заказы (с момента последней проверки)
// для новых заказов выбрать шаблон отклика
// заполинть шаблон
// откликнуться на заказ

// go parse rss https://github.com/mmcdole/gofeed
// go playwright - https://pkg.go.dev/github.com/mxschmitt/playwright-go#Page
// go http - https://pkg.go.dev/net/http

// Installing Playwright deps (for docker)
// https://github.com/playwright-community/playwright-go#installation

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
	"chromeproxy"
	"context"
	"fmt"
	"sync"

	"strings"

	"os"

	"github.com/SlyMarbo/rss"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/playwright-community/playwright-go"

	"log"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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

func main() {
	CHATS = []string{"972086219", "713587013"}
	// CHATS = []string{"713587013"}
	// WATCH_CATEGORIES = []string{"3", "10", "17", "19"}
	WATCH_CATEGORIES = []string{}
	// WATCH_CATEGORIES = []string{"1", "2", "4", "5", "6", "7", "8", "9", "11", "3", "10", "17", "19"}

	now := time.Now()
	initialCheckDate := now.Add(time.Duration(-30) * time.Second)

	// feed, err := rss.Fetch("https://www.fl.ru/rss/all.xml?category=3")
	// if err != nil {
	// 	fmt.Print("err", err)
	// 	return
	// }

	// fmt.Print(lastCheckDate.Local(), "\n")
	// filteredItems := getMostRescentItems(feed.Items, &lastCheckDate)
	// fmt.Print(len(filteredItems))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handleBotMessage),
	}

	b, err := bot.New(TOKEN, opts...)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}

	// select {
	// case <-time.Tick(time.Second * 10):
	// wg.Add(1)
	// go getToken()
	// }
	cancelChrome := login()
	defer cancelChrome()

	for _, category := range WATCH_CATEGORIES {
		wg.Add(1)
		go watchCategory(wg, b, ctx, category, initialCheckDate)
	}

	b.Start(ctx)

	wg.Wait()

	input := bufio.NewScanner(os.Stdin)
	var kw string
	input.Scan()
	kw = input.Text()
	if kw == "e" {
		return
	}

	// browser := launchBrowser()

	// page, err := browser.NewPage()
	// if err != nil {
	// 	log.Fatal("No able to create page", err)
	// }

	// if _, err = page.Goto("https://www.fl.ru/account/login/"); err != nil {
	// 	log.Fatal("Login page open failed", err)
	// }

	// defer browser.Close()
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

func handleBotMessage(c context.Context, b *bot.Bot, update *models.Update) {
	fmt.Println(update.Message.Chat.ID)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}

func sendUpdates(ctx *context.Context, b *bot.Bot, items *[]rss.Item) {
	// for _, item := range *items {
	// 	message := formatUpdateMessage(&item)

	// 	for _, chatId := range CHATS {
	// 		_, err := b.SendMessage(*ctx, &bot.SendMessageParams{
	// 			ChatID: chatId,
	// 			Text:   message,
	// 		})
	// 		if err != nil {
	// 			fmt.Println("Error sending update message: ", err)
	// 		}
	// 	}
	// }
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

// 178.34.162.1
func getToken() {
	headless := false
	var options playwright.BrowserTypeLaunchOptions
	options.Headless = &headless
	// options.Args = []string{"--proxy-server=http://178.34.162.1:80"}

	// options.Proxy = &playwright.Proxy{
	// 	// Server: "localhost:80",
	// 	// Server: "http://178.34.162.1:80",
	// }

	browser := launchPlaywright(options)
	defer browser.Close()
	page, err := browser.NewPage()

	if err != nil {
		log.Fatal("No able to create page", err)
	}

	page.On("load", func() {
		fmt.Println("page load", page.URL())
		// if strings.Contains(page.URL(), "fl.ru") {
		// 	content, err := page.InnerHTML("body")
		// 	if err != nil {
		// 		log.Fatal("cannot get page content", err)
		// 	}
		// 	if strings.Contains(content, "_TOKEN_KEY") {
		// 		ioutil.WriteFile("./tokenPageContent.html", []byte(content), 0644)
		// 		log.Fatalln("Founded TOKEN!!!")
		// 	}
		// }
	})

	page.On("frameattached", func() {
		fmt.Println("page frameattached", page.URL())
	})

	page.On("framenavigated", func() {
		fmt.Println("page framenavigated", page.URL())
	})

	page.On("domcontentloaded", func() {
		fmt.Println("page domcontentloaded", page.URL())
	})

	page.On("requestfinished", func(data playwright.Request) {
		if strings.Contains("fl", data.URL()) {
			fmt.Println("page requestfinished", data.URL())
		}
	})

	// page.OnDOMContentLoaded(func(p playwright.Page) {
	// 	fmt.Println("page domcontentloaded", page.URL())
	// })

	_, err = page.Goto("https://www.fl.ru/account/login/")
	if err != nil {
		log.Fatal("Login page open failed", err)
	}
	var waitTimeout float64 = 5 * 1000 * 60
	var waitOptions playwright.PageWaitForFunctionOptions
	waitOptions.Timeout = &waitTimeout
	handle, err := page.WaitForFunction("() => window.csrf_token && !document.querySelector(`[data-id='qa-head-sign-in']`)", nil, waitOptions)
	// handle, err := page.WaitForFunction("() => window.csrf_token", nil)

	if err != nil {
		log.Fatalln("Handle js evaluate fatal", err)
	}
	fmt.Println("handle res: ")
	fmt.Println(handle)

	tokenData, err := page.Evaluate("(() => {return window.csrf_token})()")
	if err != nil {
		log.Fatalln("Cannot get ttoken", err)
	}
	fmt.Println("TOKEN DATA:")
	fmt.Println(tokenData)

	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	entered := input.Text()
	if entered == "token" {
		tokenData, err := page.Evaluate("(() => {return window.csrf_token})()")
		if err != nil {
			log.Fatalln("Cannot get ttoken", err)
		}
		fmt.Println("TOKEN DATA:")
		fmt.Println(tokenData)
	} else {
		fmt.Println("Not token: ", entered)
	}
	if entered == "e" {
		return
	} else {
		for entered != "e" {
			input.Scan()
			entered = input.Text()
			if entered == "e" {
				return
			}
		}
	}

	// script := "function(){const check = () => {alert('check is started');const content = document.innerHTML;if (content.includes('_TOKEN_KEY')) {alert('TOKEN FOUND');}}window.onload(check);check()}()";
	// var initScript playwright.Script
	// initScript.Content = &script
	// err = page.AddInitScript(initScript)
	// log.Fatalln(err)
}

func launchPlaywright(browserOptions playwright.BrowserTypeLaunchOptions) playwright.Browser {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatal("Error running playwright", err)
	}

	browser, err := pw.Chromium.Launch(browserOptions)

	if err != nil {
		log.Fatal("Error running browser", err)
	}
	// browser.On('disconnected', closeAppFunction())
	return browser
}

func getProxyCtx(url string) (target.ID, error) {
	chromeproxy.PrepareProxy(":9222", ":9221", chromedp.DisableGPU)
	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	return targetId, err
}

func launchInChrome(url string, tasks chromedp.Tasks) (target.ID, error) {
	chromeproxy.PrepareProxy(":9223", ":9221", chromedp.DisableGPU)

	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	if err != nil {
		log.Fatalln("Error launching chrome: ", err.Error())
	}

	context := chromeproxy.GetTarget(targetId)

	execErr := chromedp.Run(context, tasks)

	return targetId, execErr

}

func login() func() error {
	url := "https://www.fl.ru/account/login/"
	// url := "https://www.google.com/recaptcha/api2/demo"
	chromeproxy.PrepareProxy(":9223", ":9221", chromedp.DisableGPU)

	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	if err != nil {
		log.Fatalln("Error launching chrome: ", err.Error())
	}
	ctx := chromeproxy.GetTarget(targetId)

	err = chromedp.Run(ctx, chromedp.Tasks{
		// chromedp.Evaluate("() => (window.csrf_token && !document.querySelector(`[data-id='qa-head-sign-in']`)) || '1234123'", nil),
		// chromedp.Evaluate("() => (window.csrf_token && !document.querySelector(`[data-id='qa-head-sign-in']`)) || '1234123'", nil),
		// chromedp.Evaluate("() => (window.csrf_token && !document.querySelector(`[data-id='qa-head-sign-in']`)) || '1234123'", nil),
		// chromedp.WaitVisible(`#recaptcha-element`, chromedp.NodeVisible),
		// chromedp.WaitVisible(`.recaptcha-checkbox-border`, chromedp.NodeVisible),
		// chromedp.Evaluate("setTimeout(() => document.querySelector('.recaptcha-checkbox-border').click(), 1000)", nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Before click")
			return nil
		}),
		// chromedp.Click(`.recaptcha-checkbox-border`, chromedp.NodeVisible),
		chromedp.WaitVisible(`#rc-imageselect`, chromedp.NodeVisible),

		// chromedp.ActionFunc(func(ctx context.Context) error {
		// 	var res []byte
		// 	chromedp.Evaluate("setTimeout(() => document.querySelector('#recaptcha-element iframe .recaptcha-checkbox-border').click(), 3000)", res)
		// 	fmt.Println("res clicking", len(res), res)
		// 	return nil
		// }),
		// chromedp.Click(`.recaptcha-checkbox-border`),
		// chromedp.ActionFunc(func(ctx context.Context) error {
		// 	fmt.Println("Click .recaptcha-checkbox-border")
		// 	return nil
		// }),
		// chromedp.WaitVisible(`#rc-imageselect`, chromedp.NodeVisible),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Wait for #rc-imageselect")
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Printf("Login here: http://127.0.0.1:9221/?id=%s", targetId)
			return nil
		}),
	})

	if err != nil {
		chromeproxy.CloseTarget(targetId)
		log.Fatalln("Error logging in: ", err)
	}

	return func() error {
		err := chromeproxy.CloseTarget(targetId)
		return err
	}
}

// Write to file
// testProjUrl := "https://www.fl.ru/projects/5275913/razrabotat-poster-dlya-dokumentalnogo-tsikla-monologov-tolko-ip-ili-samozanyatyiy.html"
// res, err := http.Get(testProjUrl)
// if err != nil {
// 	log.Fatal("cannot get page", err)
// }
// defer res.Body.Close()
// body, err := io.ReadAll(res.Body)
// ioutil.WriteFile("pageContent.html", body, 0644)
