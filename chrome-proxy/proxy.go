package chromeproxy

//https://github.com/jarylc/go-chromedpproxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	chromedpundetected "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

var mutex = sync.RWMutex{}

var loaded = make(chan bool, 1)
var mainContext context.Context
var mainCancel chan bool
var totalTargets = 0

// PrepareProxy abstracts chromedp.NewExecAllocator to the use case of this package
// it accepts listen addresses for both Chrome remote debugging and frontend as configuration
// it is also a variadic function that accepts extra chromedp.ExecAllocatorOption to be passed to the chromedp.NewExecAllocator
func PrepareProxy(chromeListenAddr string, frontendListenAddr string, customOpts ...chromedp.ExecAllocatorOption) {
	// ensure only exactly one context is prepared
	mutex.Lock()
	if mainContext != nil {
		mutex.Unlock()
		return
	}

	// split up chromeListenAddr, default host to 127.0.0.1 if not specified
	chromeListenAddrSplit := strings.Split(chromeListenAddr, ":")
	if chromeListenAddrSplit[0] == "" {
		// chromeListenAddrSplit[0] = "89.104.67.153"
		chromeListenAddrSplit[0] = "127.0.0.1"
	}

	// insert remote-debugging flags and any additional options
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("remote-debugging-address", chromeListenAddrSplit[0]),
		chromedp.Flag("remote-debugging-port", chromeListenAddrSplit[1]),
	}
	if len(customOpts) > 0 {
		opts = append(opts, customOpts...)
	}

	// create context and keep alive
	go func() {
		var conf chromedpundetected.Config
		if _, isProd := os.LookupEnv("PROD"); isProd {
			conf = chromedpundetected.NewConfig(
				chromedpundetected.WithChromeFlags(opts...),
				chromedpundetected.WithHeadless(),
			)
		} else {
			conf = chromedpundetected.NewConfig(
				chromedpundetected.WithChromeFlags(opts...),
			)
		}
		conf.Ctx = context.Background()
		ctx, cancel, err := chromedpundetected.New(conf)

		if err != nil {
			fmt.Println("error in PrepareProxy", err)
		}

		defer cancel()
		mainContext = ctx
		fmt.Println("Creating mainContext ", ctx)
		loaded <- true
		mutex.Unlock()
		mainCancel = make(chan bool, 1)
		defer close(mainCancel)
		startFrontEnd(frontendListenAddr, chromeListenAddrSplit[1], mainCancel)
	}()
}

// NewTab abstracts creating a new tab in the root context
// it returns a target ID or error
func NewTab(url string, customOpts ...chromedp.ContextOption) (target.ID, error) {
	// if context is not prepared, create with default values
	if mainContext == nil {
		fmt.Println("main context is nil")
		PrepareProxy(":9222", ":9221")
		<-loaded
	}
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Println("Call NewContext", mainContext)

	mainContext, _ = chromedp.NewContext(mainContext, customOpts...)

	err := chromedp.Run(mainContext, chromedp.Tasks{
		chromedp.Navigate(url),
	})
	if err != nil {
		return "", err
	}

	err = chromedp.Run(mainContext, chromedp.Tasks{
		chromedp.Navigate(url),
	})
	if err != nil {
		return "", err
	}
	totalTargets++

	// return target ID
	chromeContext := chromedp.FromContext(mainContext)
	return chromeContext.Target.TargetID, nil
}

// GetTarget returns a context from a target ID
func GetTarget(id target.ID) context.Context {
	mutex.RLock()
	defer mutex.RUnlock()

	// return context from target ID
	ctx, _ := chromedp.NewContext(mainContext, chromedp.WithTargetID(id))
	return ctx
}

// CloseTarget closes a target by closing the page
// if the last page has been closed, clean up everything
// it returns an error if any
func CloseTarget(id target.ID) error {
	mutex.Lock()
	defer mutex.Unlock()

	if mainContext == nil {
		return errors.New("context not prepared or already closed")
	}

	ctx, cancel := chromedp.NewContext(mainContext, chromedp.WithTargetID(id))
	_ = chromedp.Run(ctx, page.Close())
	_ = target.CloseTarget(id).Do(mainContext)
	cancel()
	targets, err := chromedp.Targets(mainContext)
	if err != nil || len(targets) <= 0 {
		loaded = make(chan bool, 1)
		mainContext = nil
		mainCancel <- true
		totalTargets = 0
	}
	return nil
}
