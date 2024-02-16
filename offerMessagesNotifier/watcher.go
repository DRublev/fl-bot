package offerMessagesNotifier

import (
	"context"
	"fmt"
	"log"
	"main/db"
	"main/fl"
	"os"
	"sync"
	"time"

	"main/bots"

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

func Start(ctx context.Context, w *sync.WaitGroup) {
	defer w.Done()
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
			go watchCategories(ctx, wg, chatId, categories)
		}
		wg.Wait()
	}

}

var baseCheckHistoryStoragePath []string = []string{"checks"}

type Check []time.Time
type CheckChannelItem struct {
	Category string
	Time     time.Time
}

func watchCategories(ctx context.Context, wg *sync.WaitGroup, chatID string, categories []string) {
	defer wg.Done()
	fmt.Println("Watching categories for", chatID, categories)
	chatStoragePath := append(baseCheckHistoryStoragePath, chatID)
	checks := make(chan CheckChannelItem)
	defer close(checks)

	go func() {
		<-ctx.Done()
		fmt.Println("Context closed")
		os.Exit(1)
	}()

	w := &sync.WaitGroup{}

	w.Add(1)
	go func() {
		defer w.Done()
		select {
		case <-ctx.Done():
			return
		case check, ok := <-checks:
			if !ok {
				fmt.Println("Check channel closed")
				return
			}
			dbPath := append(chatStoragePath, check.Category+".txt")
			dbInstance.Append(dbPath, []byte(check.Time.String()))
		}
	}()

	for _, category := range categories {
		log.Default().Println("Start watching category", category, chatID)
		notViewedItems := make(chan rss.Item)
		defer close(notViewedItems)

		w.Add(1)
		go watch(ctx, w, chatID, category, &checks, &notViewedItems)

		w.Add(1)
		go sendUpdates(ctx, w, chatID, &notViewedItems)
	}
	w.Wait()

}

func watch(ctx context.Context, wg *sync.WaitGroup, chatID string, category string, checks *chan CheckChannelItem, notViewedItems *chan rss.Item) {
	defer wg.Done()

	lastCheck := getLastCheck(&chatID, &category)

	ticker := time.NewTicker(CHECK_PERIOD_SEC * time.Second)
	defer ticker.Stop()

	go func() {
		<-ctx.Done()
		fmt.Println("Context closed")
		os.Exit(1)
	}()
	for range ticker.C {
		fmt.Println("Tick ", category)
		getNewItemsForCategory(&category, &lastCheck, notViewedItems)
		go func() {
			select {
			case *checks <- CheckChannelItem{Time: lastCheck, Category: category}:
				fmt.Println("Check ", lastCheck, category)
			}
		}()
	}
}

func getNewItemsForCategory(category *string, lastCheckDate *time.Time, notViewedItems *chan rss.Item) {
	items, err := flApi.GetOffersInCategoty(*category)
	if err != nil {
		fmt.Println("Error getting items ", err)
		log.Default().Println("Error getting items for category ", *category, "\n", err)
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

func getLastCheck(chatID *string, category *string) time.Time {
	// get from db or return now
	return time.Now().Add(-15 * time.Minute)
}

func sendUpdates(ctx context.Context, wg *sync.WaitGroup, chatID string, items *chan rss.Item) {
	defer wg.Done()
	fmt.Println("Sending updates")
	select {
	case <-ctx.Done():
		fmt.Println("Contxt closed 2")
		return
	case item, ok := <-*items:
		if ok {
			message := "[" + item.Date.Local().Format("15:04:05 02.01.2006") + "] " + item.Title + "\n" + item.Content + "\n" + item.Link + "\n"
			_, err := bots.NotificationsBot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   message,
			})
			if err != nil {
				fmt.Println("Error sending update message: ", err)
			}
		} else {
			fmt.Println("Cannot read from channel")
			log.Default().Panicln("Cannot read from channel")
			return
		}
	}
}

func Subscribe(chatId string, username string) {
	// add new chat to file
	// watch for messages for username
}
