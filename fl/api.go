package fl

import (
	"sync"

	"github.com/SlyMarbo/rss"
)

type API struct {
	Cookies   string
	CSRFToken string
}

var _wg = &sync.WaitGroup{}

func (a *API) GetOffers() {

}

func (a *API) GetOffersInCategoty(category string) ([]*rss.Item, error) {

	feed, err := rss.Fetch("https://www.fl.ru/rss/all.xml?category=" + category)

	if err != nil {
		return []*rss.Item{}, err

	}

	return feed.Items, nil
}
