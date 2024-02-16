// получить заказы через https://www.fl.ru/rss/all.xml?subcategory=37&category=5
// проверить  новые заказы (с момента последней проверки)
// для новых заказов выбрать шаблон отклика
// заполинть шаблон
// откликнуться на заказ

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"syscall"

	"main/bots"
	"main/chatsWatcher"
	chromeproxy "main/chrome-proxy"
	"main/offerMessagesNotifier"

	"os"

	"log"
	"os/exec"
	"os/signal"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"

	"github.com/go-telegram/bot"
)

// categories 3 10 17 19

/*
SELECTORS отклика на оффер
project_info_5275913 - инфа с описанием и кнопкой отклика
a.ui-button._responsive._primary _md - кнопка отклика с про
#vacancy-offer button._primary - кнопка отклика, когда доступна
#vacancy-offer input[data-id="file"] - загрузка резюме, если доступна
#vacancy-offer textarea - поле для отклика
#projectp5275913 - описание проекта
.py-32.text-right.unmobile.flex-shrink-0.ml-auto.mobile - бюджет и дедлайн
*/

// https://github.com/struCoder/pmgo

var CHATS map[string]string = map[string]string{
	// "713587013": "aringai09",
	"972086219": "nast-ka.666",
}

var ctx context.Context

var csrfToken string = "\"4X7UcBStnbhXpWmqujzDO38csegsw7qK50cRq76I\""
var cookies string = "" //"\"uechat_3_pages_count=4;_ga_RD9LL0K106=GS1.1.1706716444.1.1.1706716484.0.0.0;pwd=ed02ae7a7ac284a3acb76c7abf1940b8;name=aringai09;_ga=GA1.2.1617029702.1706716445;uechat_3_mode=0;uechat_3_first_time=1706716445187;_ym_d=1706716445;_ym_uid=1706716445483799674;analytic_id=1706716447023416;_ym_visorc=w;PHPSESSID=k06LScKmXkhwwyaYaBKjFL9gR00YL4AFYQqUobJB;_gat=1;_gid=GA1.2.691180979.1706716445;uechat_3_disabled=true;id=8488671;XSRF-TOKEN=4X7UcBStnbhXpWmqujzDO38csegsw7qK50cRq76I;user_device_id=0fv59x3qw9thbxeh82v8hi5dw9ucyssm;_ym_isad=2;_ga_cid=1617029702.1706716445;__ddg1_=76ExwmPsn2gTwMmAA1PL;\""

// = "PHPSESSID=yzlIAzYjpr1wYVBb64ANQ4cy1VcADjt9GOpNsPOH;"+"\"XSRF-TOKEN=XI1rnYgonhbszJQjkMdQu6Wgn10HCdyuB1OQgWkX; _gid=GA1.2.1816440299.1706715424; _ga_cid=1405734.1706715424; _gat=1; _ym_uid=1706715424607799129; _ym_d=1706715424; _ym_isad=2; uechat_3_first_time=1706715424109; _ym_visorc=w; uechat_3_disabled=true; uechat_3_mode=0; analytic_id=1706715425730366; _ga_RD9LL0K106=GS1.1.1706715423.1.1.1706715474.0.0.0; _ga=GA1.2.1405734.1706715424; uechat_3_pages_count=4\""

// Докинуть команду подписки
// Писать последнию дату синхронизации в файл для каждого подписчика

func restoreState() {
	// db.restore()
}

func main() {
	_, isProd := os.LookupEnv("PROD")
	if !isProd {
		if err := godotenv.Load(); err != nil {
			log.Fatalln("Cannot load env!")
		}

		CHATS = map[string]string{
			"713587013": "aringai09",
		}
	}

	// defer db.persist(state)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	wg := &sync.WaitGroup{}

	go func() {
		<-ctx.Done()
		fmt.Println("Main exit")
		os.Exit(1)
	}()

	wg.Add(1)
	// Сделать канал, куда писать сообщения, бот будет читать канал и их отсылать. Таким образом спрячу бота внутри пакета
	go bots.StartBots(ctx, wg)

	if <-bots.IsNotificationsBotReady {
		fmt.Println("Starting to watch new offers ", bots.NotificationsBot != nil)
		wg.Add(1)
		go offerMessagesNotifier.Start(ctx, wg)
	}

	if <-bots.IsOfferChatBotReady {
		for chatId := range CHATS {
			wg.Add(1)
			fmt.Println("Starting to watch new messages ", chatId, bots.OfferChatsBot != nil)
			go chatsWatcher.Watch(ctx, wg, chatId)
		}
	}

	wg.Wait()
}

func runLogin(c *context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if len(cookies) == 0 && <-bots.IsOfferChatBotReady {
		fmt.Println("Trying to log in")
		isOk, cancel := login(*c, bots.OfferChatsBot)
		if <-isOk {
			wg.Add(1)
			go getChatMessages(ctx, wg, bots.OfferChatsBot)
			cancel()
		} else {
			wg.Add(1)
			go runLogin(c, wg)
		}
	}
}

type User struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type OfferResponseItem struct {
	Id          int    `json:"id"`
	OrderUrl    string `json:"order_url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ProjectId   int    `json:"project_id"`
	Url         string `json:"url"`
	Author      User   `json:"author"`
}

type OfferMessageItem struct {
	Id          int    `json:"id"`
	FromId      int    `json:"from_id"`
	Text        string `json:"text"`
	Format      string `json:"format"`
	OfferId     int    `json:"offer_id"`
	ProjectId   int    `json:"project_id"`
	IsReadByMe  bool   `json:"is_frl_read"`
	IsReadByEmp bool   `json:"is_emp_read"`
}

type OfferProjectItem struct {
	Id        int    `json:"id"`
	Author    User   `json:"author"`
	Name      string `json:"name"`
	Url       string `json:"url"`
	IsTrashed bool   `json:"in_trash"`
}

type OffersResponse struct {
	LastOfferTime int  `json:"last_offer_time"`
	HasNextPage   bool `json:"has_next_page"`
	// Отклики
	Items []OfferResponseItem `json:"items"`
	// Все сообщения
	Messages []OfferMessageItem `json:"messages"`
	// Список проектов == список чатов
	Projects []OfferProjectItem `json:"projects"`
}

// Мапа с проектами и месседжами
// Нужно сматчить чаты в телеге с айдишниками авторов ?? - при /start просить логин из fl и запоминать
// Нужно хранить все в файлах

type Project struct {
	Id     int
	Name   string
	Author string
	Url    string
}

type Message struct {
	Project     Project
	Id          int
	FromId      int
	Text        string
	Format      string
	OfferId     int
	IsReadByMe  bool
	IsReadByEmp bool
}

func getChatMessages(c context.Context, wg *sync.WaitGroup, b *bot.Bot) {
	defer wg.Done()

	const CHECK_PERIOD_SEC = 5
	ticker := time.NewTicker(CHECK_PERIOD_SEC * time.Second)
	if ok := <-bots.IsOfferChatBotReady; !ok {
		fmt.Println("Not ok with starting offer chats bot")
		return
	}

	go func() {
		<-c.Done()
		fmt.Println("Context closed")
		os.Exit(1)
	}()
	for range ticker.C {
		fmt.Println("Reading messages")
		req, err := http.NewRequest("GET", "https://www.fl.ru/projects/offers/?limit=20&dialogues=1&deleted=1&sort=lastMessage&offset=0", nil)
		if err != nil {
			fmt.Println("Error getting chats", err)
		}

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

		var result OffersResponse

		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Println("Cannot unmarshal ", err)
		}
		res.Body.Close()

		notReadMessages := make(map[string][]Message)

		chatsByLogin := make(map[string]string)
		for chatId, login := range CHATS {
			chatsByLogin[login] = chatId
		}

		authorIdChatMap := make(map[int]string)
		for _, item := range result.Items {
			chatId, ok := chatsByLogin[item.Author.Username]
			if ok {
				authorIdChatMap[item.Author.Id] = chatId
			}
		}

		projectsMap := make(map[int]Project)
		fmt.Println("Projects: ", len(result.Messages))
		for _, project := range result.Projects {
			if project.IsTrashed {
				fmt.Println("Trashed project")
				continue
			}

			projectsMap[project.Id] = Project{
				Id:     project.Id,
				Author: project.Author.Username,
				Name:   project.Name,
				Url:    project.Url,
			}
		}

		for _, message := range result.Messages {
			_, ok := authorIdChatMap[message.FromId]
			var chatId string
			for _, item := range result.Items {
				if item.ProjectId != message.ProjectId {
					continue
				}
				candidate, ok := authorIdChatMap[item.Author.Id]
				if ok {
					chatId = candidate
				}
			}
			// if !ok && !message.IsReadByMe {
			if !ok && len(chatId) > 0 {
				project, ok := projectsMap[message.ProjectId]
				if !ok {
					fmt.Println("Unknown project ", message.ProjectId)
				}
				fmt.Println("New message! ", message)
				if notReadMessages[chatId] == nil {
					notReadMessages[chatId] = []Message{}
				}
				notReadMessages[chatId] = append(notReadMessages[chatId], Message{
					Id:          message.Id,
					FromId:      message.FromId,
					Text:        message.Text,
					Format:      message.Format,
					OfferId:     message.OfferId,
					IsReadByMe:  message.IsReadByMe,
					IsReadByEmp: message.IsReadByEmp,
					Project:     project,
				})
			} else {
				fmt.Println(" Read or from me ", ok, message.IsReadByMe)
			}
		}

		fmt.Println("Not read map ", len(notReadMessages))
		for chatId, messages := range notReadMessages {
			fmt.Println("Sending message to chat ", chatId)
			for _, message := range messages {
				sendNewMessages(c, b, chatId, message)
			}
		}
	}
}

func sendNewMessages(c context.Context, b *bot.Bot, chatId string, item Message) {
	orderUrl := item.Project.Url
	content := item.Text
	message := "От: " + item.Project.Author + "\n" + "По заказу " + item.Project.Name + " (" + orderUrl + ")\n" + content
	fmt.Println("Sending message: ", chatId, "   ", message)
	_, err := b.SendMessage(c, &bot.SendMessageParams{
		ChatID: chatId,
		Text:   message,
	})
	if err != nil {
		log.Println("Error sending message ", err)
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

// const LOGIN = "aringai09@gmail.com"
// const PASS = "7fJxtyFQsamsung!"
const LOGIN = "Nast-ka.666@mail.ru"
const PASS = "fyrgonSk-Doo2023"

func login(c context.Context, b *bot.Bot) (chan bool, func() error) {
	url := "https://www.fl.ru/account/login/"
	chromeproxy.PrepareProxy(":9223", ":9221", chromedp.DisableGPU)

	go func() {
		<-c.Done()

		cmd := exec.Command("pgrep chrome | xargs kill -9")
		cmd.Run()

		fmt.Println("Context closed")
		os.Exit(1)
	}()

	ip := "89.104.67.153" //getLocalIp()

	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	if err != nil {
		fmt.Println("Error launching chrome", err)
		log.Fatalln("Error launching chrome: ", err.Error())
	}
	ctx := chromeproxy.GetTarget(targetId)

	isSucceed := make(chan bool, 1)
	isCsrfToken := make(chan bool, 1)
	isCookies := make(chan bool, 1)

	go func() {
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("#no-session", chromedp.NodeVisible),
		})
		if err == nil {
			chromeproxy.CloseTarget(targetId)
			fmt.Println("No session found")
			isSucceed <- false
		}
	}()

	go func() {
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitReady("window"),
			chromedp.WaitVisible("input[name='username']", chromedp.NodeVisible),
			chromedp.WaitVisible("input[name='password']", chromedp.NodeVisible),
			chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='username']`); username.value = '"+LOGIN+"' })()", nil),
			chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='password']`); username.value = '"+PASS+"' })()", nil),
			chromedp.Evaluate("(() => { const frames = document.querySelectorAll('iframe'); if (!frames[2]) {return;} frames[2].style.position='fixed'; frames[2].style.left ='0'; })()", nil),
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
					ChatID: "713587013",
					Text:   string(msg),
				})
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: "972086219",
					Text:   string(msg),
				})
				fmt.Println(msg)
				return nil
			}),
		})

		if err != nil {
			chromeproxy.CloseTarget(targetId)
			fmt.Println("err 1", err)
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
			fmt.Println("err 2", err)

			isCsrfToken <- false
			chromeproxy.CloseTarget(targetId)

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
			chromeproxy.CloseTarget(targetId)

			fmt.Println("err 3", err)

			isCookies <- false
			log.Println("Error waiting for auth token: ", err)
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
