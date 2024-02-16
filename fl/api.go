package fl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/SlyMarbo/rss"
)

type API struct {
	Cookies   string
	CSRFToken string
}

type AuthInfo struct {
	login string
	pass  string
	email string
}

var users = map[string]AuthInfo{
	"713587013": {
		login: "aringai09",
		pass:  "7fJxtyFQsamsung!",
		email: "aringai09@gmail.com",
	},
	"972086219": {
		login: "nast-ka.666",
		pass:  "fyrgonSk-Doo2023",
		email: "Nast-ka.666@mail.ru",
	},
}

func (a *API) GetOffers() {

}

func (a *API) GetOffersInCategoty(category string) ([]*rss.Item, error) {

	feed, err := rss.Fetch("https://www.fl.ru/rss/all.xml?category=" + category)

	if err != nil {
		return []*rss.Item{}, err

	}

	return feed.Items, nil
}

func (a *API) GetChats(ctx context.Context, chatId string) ([]Message, error) {
	authInfo, exists := users[chatId]
	if !exists {
		return []Message{}, errors.New("no such user found")
	}
	if len(a.Cookies) == 0 {
		fmt.Println("Waiting for login...")
		err := a.Login(ctx, chatId)

		if err != nil {
			return []Message{}, err
		}
	}

	req, err := http.NewRequest("GET", "https://www.fl.ru/projects/offers/?limit=20&dialogues=1&deleted=1&sort=lastMessage&offset=0", nil)
	if err != nil {
		fmt.Println("Error getting chats", err)
		return []Message{}, err
	}

	if len(a.CSRFToken) > 0 {
		req.Header.Set("x-csrf-token", strings.Trim(a.CSRFToken, "\""))
		req.Header.Set("x-xsrf-token", strings.Trim(a.CSRFToken, "\""))
	}
	req.Header.Set("Cookie", strings.Trim(a.Cookies, "\""))
	req.Header.Set("referer", "https://www.fl.ru/messages/")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Println("Error getting chats res", err)
		return []Message{}, err
	}

	if res.StatusCode == 401 {
		a.Login(ctx, chatId)
		return []Message{}, errors.New("not logged in for " + chatId)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error getting chats res", err)
		return []Message{}, err
	}

	var result OffersResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Cannot unmarshal ", err)
		return []Message{}, err
	}
	res.Body.Close()

	notReadMessages := []Message{}

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

	authorIdChatMap := make(map[int]string)
	for _, item := range result.Items {
		if item.Author.Username == authInfo.login {
			authorIdChatMap[item.Author.Id] = chatId
		}
	}

	for _, message := range result.Messages {
		_, ok := authorIdChatMap[message.FromId]
		var cId string
		for _, item := range result.Items {
			if item.ProjectId != message.ProjectId {
				continue
			}
			candidate, ok := authorIdChatMap[item.Author.Id]
			if ok {
				cId = candidate
			}
		}

		if !ok && len(cId) > 0 && !message.IsReadByMe {
			project, ok := projectsMap[message.ProjectId]
			if !ok {
				fmt.Println("Unknown project ", message.ProjectId)
			} else {
				fmt.Println("New message! ", chatId)
				notReadMessages = append(notReadMessages, Message{
					Id:          message.Id,
					FromId:      message.FromId,
					Text:        message.Text,
					Format:      message.Format,
					OfferId:     message.OfferId,
					IsReadByMe:  message.IsReadByMe,
					IsReadByEmp: message.IsReadByEmp,
					Project:     project,
				})
			}
		} else {
			fmt.Println(" Read or from me ", ok, message.IsReadByMe)
		}
	}

	return notReadMessages, nil
}
