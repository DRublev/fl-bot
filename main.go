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
	"fmt"

	// "io/ioutil"
	"strings"

	"os"

	"github.com/SlyMarbo/rss"
	"github.com/playwright-community/playwright-go"

	"log"
	"time"
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

func main() {
	// now := time.Now()
	// lastCheckDate := now.Add(-time.Hour * 3)

	// feed, err := rss.Fetch("https://www.fl.ru/rss/all.xml?category=3")
	// if err != nil {
	// 	fmt.Print("err", err)
	// 	return
	// }

	// fmt.Print(lastCheckDate.Local(), "\n")
	// filteredItems := getMostRescentItems(feed.Items, &lastCheckDate)
	// fmt.Print(len(filteredItems))

	go getToken()

	input := bufio.NewScanner(os.Stdin)
	var kw string
	// for kw != "e" {
	input.Scan()
	kw = input.Text()
	// }
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

func getMostRescentItems(items []*rss.Item, lastCheck *time.Time) (filtered []*rss.Item) {
	for _, item := range items {
		if item.Date.After(*lastCheck) {
			fmt.Print(item.Date, " | ", item.Title, " | ", item.Link, "\n")
			filtered = append(filtered, item)
		}
	}
	return
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

func getToken() {
	headless := false
	var options playwright.BrowserTypeLaunchOptions
	options.Headless = &headless

	browser := launchBrowser(options)
	// defer browser.Close()

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

	page.On("framenavigated", func() {
		fmt.Println("page framenavigated", page.URL())
		// if strings.Contains(page.URL(), "fl.ru") {
		// 	_, err := page.InnerHTML("body")
		// 	if err != nil {
		// 		log.Fatal("cannot get page content", err)
		// 	}
		// 		if strings.Contains(content, "_TOKEN_KEY") {
		// 			ioutil.WriteFile("./tokenPageContent.html", []byte(content), 0644)
		// 			log.Fatalln("Founded TOKEN!!!")
		// 		}
		// }
	})

	page.On("domcontentloaded", func() {
		fmt.Println("page domcontentloaded", page.URL())
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

	page.On("requestfinished", func(data playwright.Request) {
		if strings.Contains("fl", data.URL()) {
			fmt.Println("page requestfinished", data.URL())
		}
	})

	// page.OnDOMContentLoaded(func(p playwright.Page) {
	// 	fmt.Println("page domcontentloaded", page.URL())
	// 	if strings.Contains(p.URL(), "fl.ru") {
	// 		content, err := p.InnerHTML("body")
	// 		if err != nil {
	// 			log.Fatal("cannot get p content", err)
	// 		}
	// 		title, _ := p.Title()
	// 		ioutil.WriteFile("./"+title+".html", []byte(content), 0644)
	// 	}
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

func launchBrowser(browserOptions playwright.BrowserTypeLaunchOptions) playwright.Browser {
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

// Write to file
// testProjUrl := "https://www.fl.ru/projects/5275913/razrabotat-poster-dlya-dokumentalnogo-tsikla-monologov-tolko-ip-ili-samozanyatyiy.html"
// res, err := http.Get(testProjUrl)
// if err != nil {
// 	log.Fatal("cannot get page", err)
// }
// defer res.Body.Close()
// body, err := io.ReadAll(res.Body)
// ioutil.WriteFile("pageContent.html", body, 0644)
