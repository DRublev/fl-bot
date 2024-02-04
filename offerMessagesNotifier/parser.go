package offerMessagesNotifier

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
