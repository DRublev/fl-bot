package fl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"main/bots"
	chromeproxy "main/chrome-proxy"
	"os"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/go-telegram/bot"
)

func (a *API) Login(ctx context.Context, chatId string) error {

	controlCh := make(chan error, 1)

	a.waitForSuccessLogin(ctx, &controlCh, chatId)

	err := <-controlCh
	if err != nil {
		fmt.Println("Login failed for ", chatId, err)
		return err
	}

	go a.prolongateSession()

	return nil
}

func (a *API) waitForSuccessLogin(c context.Context, controlCh *chan error, chatId string) {
	url := "https://www.fl.ru/account/login/"
	chromeproxy.PrepareProxy(":9223", ":9221", chromedp.DisableGPU)

	targetId, err := chromeproxy.NewTab(url, chromedp.WithLogf(log.Printf))
	if err != nil {
		fmt.Println("Error launching chrome", err)
		log.Fatalln("Error launching chrome: ", err.Error())
	}
	ctx := chromeproxy.GetTarget(targetId)

	go a.checkIfNoSession(ctx, controlCh, targetId)
	go a.enterLoginInfo(ctx, chatId)
	go a.sendLoginLink(ctx, controlCh, targetId, bots.OfferChatsBot, chatId)
	go a.getCsrfToken(ctx, controlCh, targetId)
	go a.getCookies(ctx, controlCh, targetId)

	fmt.Println("Login success!")
}

func (a *API) checkIfNoSession(ctx context.Context, controlCh *chan error, targetId target.ID) {
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitReady("window"),
		chromedp.WaitVisible("#no-session", chromedp.NodeVisible),
	})
	if err == nil {
		chromeproxy.CloseTarget(targetId)
		fmt.Println("No session found")
		*controlCh <- errors.New("no session was found for " + targetId.String())
	}
}

func (a *API) enterLoginInfo(ctx context.Context, chatId string) {
	user, exists := users[chatId]
	if !exists {
		return
	}
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitReady("window"),
		chromedp.WaitVisible("input[name='username']", chromedp.NodeVisible),
		chromedp.WaitVisible("input[name='password']", chromedp.NodeVisible),
		chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='username']`); username.value = '"+user.email+"' })()", nil),
		chromedp.Evaluate("(() => { const username = document.querySelector(`input[name='password']`); username.value = '"+user.pass+"' })()", nil),
		chromedp.Evaluate("(() => { const frames = document.querySelectorAll('iframe'); if (!frames[2]) {return;} frames[2].style.position='fixed'; frames[2].style.left ='0'; })()", nil),
		chromedp.ActionFunc(func(c context.Context) error {
			fmt.Println("Login info entered for ", user.email)
			return nil
		}),
	})
	if err != nil {
		fmt.Println("Error entering login info!", err)
	}
}

func (a *API) sendLoginLink(ctx context.Context, controlCh *chan error, targetId target.ID, b *bot.Bot, chatId string) {
	ip := "89.104.67.153"
	if _, isProd := os.LookupEnv("PROD"); !isProd {
		ip = getLocalIp()
	}
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitReady("window"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			msg := "Войди по ссылке: http://" + ip + ":9221/?id=" + string(targetId)
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatId,
				Text:   string(msg),
			})
			if err != nil {
				chromeproxy.CloseTarget(targetId)
				*controlCh <- err
			}

			fmt.Println(msg)
			return nil
		}),
	})

	if err != nil {
		chromeproxy.CloseTarget(targetId)
		fmt.Println("Error logging in:", err)
		*controlCh <- errors.New("error sending login link for " + chatId + err.Error())
	}
}

func (a *API) getCsrfToken(ctx context.Context, controlCh *chan error, targetId target.ID) {
	var result []byte
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitReady("window"),
		chromedp.WaitVisible("#navbarRightDropdown"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			<-time.Tick(time.Second * 2)
			return nil
		}),
		chromedp.Evaluate("document.querySelector(`meta[name='csrf-token']`).content", &result),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("csrf token", string(result))
			if len(result) > 0 {
				a.CSRFToken = string(result)
			}
			return nil
		}),
	})

	if err != nil {
		fmt.Println("Error waiting for csrf token", err)
	}
}

func (a *API) getCookies(ctx context.Context, controlCh *chan error, targetId target.ID) {
	var result []byte
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitReady("window"),
		chromedp.WaitVisible("#navbarRightDropdown"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			<-time.Tick(time.Second * 2)
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
			cookies := ""
			for i, cookie := range c {
				fmt.Printf("chrome cookie %d: %+v \n", i, cookie.Name)
				cookies += cookie.Name + "=" + cookie.Value + ";"
			}
			a.Cookies = cookies
			fmt.Println(a.Cookies)
			chromeproxy.CloseTarget(targetId)
			*controlCh <- nil

			return nil
		}),
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		chromeproxy.CloseTarget(targetId)

		fmt.Println("Error waiting for cookies ", err)
		*controlCh <- err
	}
}

func (a *API) prolongateSession() {
	<-time.Tick(time.Minute * 1)
	// Some GET request to fl.ru
	// Get Set-Cookie header from response and update a.Cookie
}

func ping() {

}
