package fl

import (
	"log"
	"sync"
	"time"
)

func (a *API) Login(wg *sync.WaitGroup) string {

	controlCh := make(chan bool, 1)

	go waitForSuccessLogin(&controlCh)
	if <-controlCh {
		// success login
		log.Println("Successfully logged in!")
	} else {
		// return error with login
	}

	wg.Add(1)
	go a.prolongateSession(wg)

	return "http://url-for-user"
}

func waitForSuccessLogin(controlCh *chan bool) {
	defer _wg.Done()

	// chromedp.Run(... .Wait())
	*controlCh <- false
}

func waitForLoginError() {

}

func (a *API) prolongateSession(wg *sync.WaitGroup) {
	defer wg.Done()

	<-time.Tick(time.Minute * 1)
	// Some GET request to fl.ru
	// Get Set-Cookie header from response and update a.Cookie
}

func ping() {

}
