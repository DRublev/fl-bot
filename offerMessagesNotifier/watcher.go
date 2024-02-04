package offerMessagesNotifier

import (
	"context"
	"fmt"
	"log"
	"main/db"
	"main/fl"
	"sync"
	"time"

	"fl.ru/bots"
	"github.com/SlyMarbo/rss"
	"github.com/go-telegram/bot"
)

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

var STORAGE_NAME []string = []string{"chatsCategories"}

type ChatCategoriesToWatchState map[string][]string

const CHECK_PERIOD_SEC = 5

var flApi fl.API = fl.API{
	Cookies:   "",
	CSRFToken: "",
}

var dbInstance = db.DB{}

var testWatchCategories = map[string][]string{
	"713587013": {"3", "10", "17", "19"},
	"972086219": {"3", "10", "17", "19"},
}

func Start(ctx context.Context) {
	select {
	case <-ctx.Done():
		fmt.Println("Context closed!")
		return
	default:
		// chatCategoriesRaw, err := dbInstance.Get(STORAGE_NAME)
		// if err != nil {
		// 	log.Fatalln("Failed to restore from db: ", err)
		// 	return
		// }
		// chatCategories, err := bytesToJSON[ChatCategoriesToWatchState](chatCategoriesRaw)
		// if err != nil {
		// 	log.Fatalln("Failed to restore from db: ", err)
		// 	return
		// }
		chatCategories := testWatchCategories
		wg := &sync.WaitGroup{}
		for chatId, categories := range chatCategories {
			wg.Add(1)
			go watchCategories(wg, ctx, chatId, categories)
		}
		wg.Wait()
	}

}

var BASE_CHECK_HISTORY_STORAGE_PATH []string = []string{"checks"}

type Check []time.Time
type CheckChannelItem struct {
	Category string
	Time     time.Time
}

func watchCategories(wg *sync.WaitGroup, ctx context.Context, chatId string, categories []string) {
	defer wg.Done()
	fmt.Println("Watching categories for", chatId, categories)
	chatStoragePath := append(BASE_CHECK_HISTORY_STORAGE_PATH, chatId)
	checks := make(chan CheckChannelItem)
	defer close(checks)

	w := &sync.WaitGroup{}

	w.Add(1)
	go func() {
		for {
			check, ok := <-checks
			if !ok {
				break
			}
			dbPath := append(chatStoragePath, check.Category)
			dbInstance.Append(dbPath, []byte(check.Time.String()))
		}
	}()

	for _, category := range categories {
		log.Default().Println("Start watching category %v for %v", category, chatId)
		notViewedItems := make(chan rss.Item)

		w.Add(1)
		go watch(w, chatId, category, &checks, &notViewedItems)

		w.Add(1)
		go sendUpdates(w, ctx, chatId, &notViewedItems)
	}
	w.Wait()
}

func watch(wg *sync.WaitGroup, chatId string, category string, checks *chan CheckChannelItem, notViewedItems *chan rss.Item) {
	defer wg.Done()

	lastCheck := getLastCheck(&chatId, &category)

	ticker := time.NewTicker(time.Duration(CHECK_PERIOD_SEC) * time.Second)

	for range ticker.C {
		fmt.Println("Tick ", category)
		getNewItemsForCategory(&category, &lastCheck, notViewedItems)
		*checks <- CheckChannelItem{Time: lastCheck, Category: category}
	}
}

func getNewItemsForCategory(category *string, lastCheckDate *time.Time, notViewedItems *chan rss.Item) {
	items, err := flApi.GetOffersInCategoty(*category)

	if err != nil {
		log.Default().Println("Error getting items for category ", *category, "\n", err, "\n\n")
		getNewItemsForCategory(category, lastCheckDate, notViewedItems)
	}

	for _, item := range items {
		if item.Date.After(*lastCheckDate) {
			fmt.Println("Not viewed item: ", item.Date, item.Title)
			*notViewedItems <- *item
			*lastCheckDate = item.Date
		}
	}
}

func getLastCheck(chatId *string, category *string) time.Time {
	// get from db or return now
	return time.Now().Add(time.Duration(-5) * time.Second)
}

func sendUpdates(wg *sync.WaitGroup, ctx context.Context, chatId string, items *chan rss.Item) {
	defer wg.Done()

	select {
	// case <-ctx.Done():
	// 	fmt.Println("Ctx done 127")
	// 	return
	case item, ok := <-*items:
		if ok {
			message := "[" + item.Date.Local().Format("15:04:05 02.01.2006") + "] " + item.Title + "\n" + item.Content + "\n" + item.Link + "\n"
			_, err := bots.NotificationsBot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: "713587013",
				Text:   message,
			})
			if err != nil {
				fmt.Println("Error sending update message: ", err)
			}
		} else {
			log.Default().Panicln("Cannot read from channel")
		}
	}
}

func Subscribe(chatId string, username string) {
	// add new chat to file
	// watch for messages for username
}
