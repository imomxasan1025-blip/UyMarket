```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const telegramAPI = "https://api.telegram.org/bot"

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type UpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type UserState struct {
	Step     int
	Language string

	Type     string
	City     string
	District string
	Rooms    string
	Area     string
	Floor    string

	// Это цена, которую указал продавец.
	SellerPrice string

	Description string
}

var users = make(map[int64]*UserState)
var usersMutex sync.Mutex

// ============================================================
// TELEGRAM REQUEST
// ============================================================

func telegramRequest(token, method string, data interface{}) error {

	body, err := json.Marshal(data)

	if err != nil {
		return err
	}

	resp, err := http.Post(
		telegramAPI+token+"/"+method,
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"Telegram HTTP error: %s",
			resp.Status,
		)
	}

	return nil
}

// ============================================================
// SEND MESSAGE
// ============================================================

func sendMessage(
	token string,
	chatID int64,
	text string,
	keyboard [][]string,
) {

	request := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	if keyboard != nil {

		rows := make(
			[][]map[string]string,
			len(keyboard),
		)

		for i, row := range keyboard {

			for _, button := range row {

				rows[i] = append(
					rows[i],
					map[string]string{
						"text": button,
					},
				)
			}
		}

		request["reply_markup"] =
			map[string]interface{}{
				"keyboard":        rows,
				"resize_keyboard": true,
			}
	}

	if err := telegramRequest(
		token,
		"sendMessage",
		request,
	); err != nil {

		log.Println(
			"Ошибка Telegram:",
			err,
		)
	}
}

// ============================================================
// GET UPDATES
// ============================================================

func getUpdates(
	token string,
	offset int,
) ([]Update, error) {

	url := telegramAPI +
		token +
		"/getUpdates?timeout=30&offset=" +
		strconv.Itoa(offset)

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var result UpdatesResponse

	if err := json.NewDecoder(
		resp.Body,
	).Decode(&result); err != nil {

		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf(
			"Telegram API error",
		)
	}

	return result.Result, nil
}

// ============================================================
// MENUS
// ============================================================

func russianMenu() [][]string {

	return [][]string{

		{
			"🏠 Купить недвижимость",
			"➕ Продать недвижимость",
		},

		{
			"🔎 Найти квартиру",
			"📋 Мои объявления",
		},

		{
			"👨‍💼 Менеджер",
			"🇺🇿 O‘zbekcha",
		},
	}
}

func uzbekMenu() [][]string {

	return [][]string{

		{
			"🏠 Uy-joy sotib olish",
			"➕ Uy-joy sotish",
		},

		{
			"🔎 Kvartira qidirish",
			"📋 Mening e'lonlarim",
		},

		{
			"👨‍💼 Menejer",
			"🇷🇺 Русский",
		},
	}
}

func propertyMenu() [][]string {

	return [][]string{

		{"🏢 Квартира"},
		{"🏠 Дом"},
		{"🏪 Коммерческая недвижимость"},
		{"🌳 Земельный участок"},
		{"⬅️ Назад"},
	}
}

// ============================================================
// START SALE
// ============================================================

func startSale(
	token string,
	chatID int64,
) {

	usersMutex.Lock()

	users[chatID] = &UserState{
		Step:     1,
		Language: "ru",
	}

	usersMutex.Unlock()

	sendMessage(
		token,
		chatID,
		"➕ Продажа недвижимости\n\n"+
			"Выберите тип недвижимости:",
		propertyMenu(),
	)
}

// ============================================================
// PROCESS SALE
// ============================================================

func processSale(
	token string,
	chatID int64,
	text string,
) {

	usersMutex.Lock()

	user, exists := users[chatID]

	if !exists {

		usersMutex.Unlock()
		return
	}

	switch user.Step {

	// --------------------------------------------------------
	// ШАГ 1 — ТИП
	// --------------------------------------------------------

	case 1:

		if text == "⬅️ Назад" {

			delete(users, chatID)

			usersMutex.Unlock()

			sendMessage(
				token,
				chatID,
				"Главное меню:",
				russianMenu(),
			)

			return
		}

		if text != "🏢 Квартира" &&
			text != "🏠 Дом" &&
			text != "🏪 Коммерческая недвижимость" &&
			text != "🌳 Земельный участок" {

			usersMutex.Unlock()

			sendMessage(
				token,
				chatID,
				"Пожалуйста, выберите тип недвижимости кнопкой.",
				propertyMenu(),
			)

			return
		}

		user.Type = text
		user.Step = 2

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"📍 В каком городе находится объект?\n\n"+
				"Напишите город.",
			nil,
		)

	// --------------------------------------------------------
	// ШАГ 2 — ГОРОД
	// --------------------------------------------------------

	case 2:

		user.City = text
		user.Step = 3

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"📍 В каком районе находится объект?\n\n"+
				"Напишите район.",
			nil,
		)

	// --------------------------------------------------------
	// ШАГ 3 — РАЙОН
	// --------------------------------------------------------

	case 3:

		user.District = text
		user.Step = 4

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"🛏 Сколько комнат?\n\n"+
				"Например: 2",
			nil,
		)

	// --------------------------------------------------------
	// ШАГ 4 — КОМНАТЫ
	// --------------------------------------------------------

	case 4:

		user.Rooms = text
		user.Step = 5

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"📐 Какая площадь?\n\n"+
				"Например: 65 м²",
			nil,
		)

	// --------------------------------------------------------
	// ШАГ 5 — ПЛОЩАДЬ
	// --------------------------------------------------------

	case 5:

		user.Area = text
		user.Step = 6

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"🏢 Какой этаж и сколько этажей в доме?\n\n"+
				"Например: 7/16",
			nil,
		)

	// --------------------------------------------------------
	// ШАГ 6 — ЭТАЖ
	// --------------------------------------------------------

	case 6:

		user.Floor = text
		user.Step = 7

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"💰 За какую сумму вы хотите продать квартиру?\n\n"+
				"Например: 50000 USD\n\n"+
				"🔒 Эта цена будет доступна только менеджеру.",
			nil,
		)

	// --------------------------------------------------------
	// ШАГ 7 — ЦЕНА ПРОДАВЦА
	// --------------------------------------------------------

	case 7:

		user.SellerPrice = text
		user.Step = 8

		summary := fmt.Sprintf(

			"📋 Проверьте заявку:\n\n"+

				"🏷 Тип: %s\n"+
				"📍 Город: %s\n"+
				"📍 Район: %s\n"+
				"🛏 Комнаты: %s\n"+
				"📐 Площадь: %s\n"+
				"🏢 Этаж: %s\n"+
				"💰 Ваша цена: %s\n\n"+

				"🔒 Цена будет видна только менеджеру.\n\n"+

				"Всё правильно?",

			user.Type,
			user.City,
			user.District,
			user.Rooms,
			user.Area,
			user.Floor,
			user.SellerPrice,
		)

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			summary,
			[][]string{
				{"✅ Отправить заявку"},
				{"❌ Отменить"},
			},
		)

	// --------------------------------------------------------
	// ШАГ 8 — ПОДТВЕРЖДЕНИЕ
	// --------------------------------------------------------

	case 8:

		if text == "❌ Отменить" {

			delete(users, chatID)

			usersMutex.Unlock()

			sendMessage(
				token,
				chatID,
				"❌ Заявка отменена.",
				russianMenu(),
			)

			return
		}

		if text == "✅ Отправить заявку" {

			data := map[string]string{

				"source": "uymarket",

				"id": "UY-" +
					strconv.FormatInt(
						time.Now().UnixNano(),
						10,
					),

				"type": user.Type,

				"city": user.City,

				"district": user.District,

				"rooms": user.Rooms,

				"area": user.Area,

				"floor": user.Floor,

				// Только цена продавца.
				// Цена покупателя здесь НЕ передается.

				"seller_price":
					user.SellerPrice,

				"buyer_price": "",

				"photo": "",

				"description": "",

				"status":
					"Новая заявка",
			}

			usersMutex.Unlock()

			err := sendToGoogleSheets(data)

			if err != nil {

				log.Println(
					"Ошибка сохранения объявления:",
					err,
				)

				sendMessage(
					token,
					chatID,
					"❌ Не удалось сохранить заявку.\n\n"+
						"Попробуйте ещё раз.",
					russianMenu(),
				)

				return
			}

			usersMutex.Lock()

			delete(users, chatID)

			usersMutex.Unlock()

			sendMessage(
				token,
				chatID,

				"✅ Заявка успешно отправлена!\n\n"+

					"Спасибо. Менеджер свяжется с вами "+
					"для уточнения деталей.\n\n"+

					"🔒 Ваша цена является конфиденциальной "+
					"и не показывается покупателям.",

				russianMenu(),
			)

			return
		}

		usersMutex.Unlock()

	default:

		usersMutex.Unlock()
	}
}

// ============================================================
// HANDLE MESSAGE
// ============================================================

func handleMessage(
	token string,
	chatID int64,
	text string,
) {

	usersMutex.Lock()

	_, inSale :=
		users[chatID]

	usersMutex.Unlock()

	if inSale {

		processSale(
			token,
			chatID,
			text,
		)

		return
	}

	switch text {

	case "/start":

		sendMessage(
			token,
			chatID,

			"🏠 UyMarket\n\n"+
				"Выберите язык / Tilni tanlang:",

			[][]string{
				{
					"🇷🇺 Русский",
					"🇺🇿 O‘zbekcha",
				},
			},
		)

	case "🇷🇺 Русский":

		sendMessage(
			token,
			chatID,

			"🏠 UyMarket\n\n"+
				"Добро пожаловать!\n\n"+
				"Выберите действие:",

			russianMenu(),
		)

	case "🇺🇿 O‘zbekcha":

		sendMessage(
			token,
			chatID,

			"🏠 UyMarket\n\n"+
				"Xush kelibsiz!\n\n"+
				"Kerakli bo‘limni tanlang:",

			uzbekMenu(),
		)

	case "➕ Продать недвижимость":

		startSale(
			token,
			chatID,
		)

	case "🏠 Купить недвижимость":

		sendMessage(
			token,
			chatID,

			"🏠 Каталог недвижимости\n\n"+
				"Сейчас готовим каталог квартир.",

			russianMenu(),
		)

	case "🔎 Найти квартиру":

		sendMessage(
			token,
			chatID,

			"🔎 Поиск квартиры\n\n"+
				"Сейчас готовим поиск по району, "+
				"цене, площади и комнатам.",

			russianMenu(),
		)

	case "📋 Мои объявления":

		sendMessage(
			token,
			chatID,

			"📋 Мои объявления\n\n"+
				"Раздел находится в разработке.",

			russianMenu(),
		)

	case "👨‍💼 Менеджер":

		sendMessage(
			token,
			chatID,

			"👨‍💼 Менеджер\n\n"+
				"Напишите ваш вопрос.",

			russianMenu(),
		)

	case "🏠 Uy-joy sotib olish":

		sendMessage(
			token,
			chatID,

			"🏠 Uy-joy sotib olish\n\n"+
				"Tez orada e'lonlar paydo bo‘ladi.",

			uzbekMenu(),
		)

	case "➕ Uy-joy sotish":

		startSale(
			token,
			chatID,
		)

	case "🔎 Kvartira qidirish":

		sendMessage(
			token,
			chatID,

			"🔎 Kvartira qidirish\n\n"+
				"Tez orada qidiruv ishlaydi.",

			uzbekMenu(),
		)

	case "📋 Mening e'lonlarim":

		sendMessage(
			token,
			chatID,

			"📋 Mening e'lonlarim\n\n"+
				"Hozircha e'lonlar yo‘q.",

			uzbekMenu(),
		)

	case "👨‍💼 Menejer":

		sendMessage(
			token,
			chatID,

			"👨‍💼 Menejer\n\n"+
				"Savolingizni yozing.",

			uzbekMenu(),
		)

	default:

		sendMessage(
			token,
			chatID,

			"Пожалуйста, выберите пункт меню.",

			russianMenu(),
		)
	}
}

// ============================================================
// GOOGLE SHEETS
// ============================================================

func sendToGoogleSheets(
	data map[string]string,
) error {

	url :=
		os.Getenv("GOOGLE_SHEETS_URL")

	if url == "" {

		return fmt.Errorf(
			"GOOGLE_SHEETS_URL не найден",
		)
	}

	body, err :=
		json.Marshal(data)

	if err != nil {
		return err
	}

	resp, err := http.Post(

		url,

		"application/json",

		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		return fmt.Errorf(
			"Google Apps Script вернул статус %d",
			resp.StatusCode,
		)
	}

	return nil
}

// ============================================================
// MAIN
// ============================================================

func main() {

	token :=
		os.Getenv("BOT_TOKEN")

	if token == "" {

		log.Fatal(
			"BOT_TOKEN не найден",
		)
	}

	port :=
		os.Getenv("PORT")

	if port == "" {
		port = "10000"
	}

	http.HandleFunc(
		"/",
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			fmt.Fprintln(
				w,
				"UyMarket bot is running!",
			)
		},
	)

	go func() {

		log.Println(
			"UyMarket Telegram bot started",
		)

		offset := 0

		for {

			updates, err :=
				getUpdates(
					token,
					offset,
				)

			if err != nil {

				log.Println(
					"Telegram error:",
					err,
				)

				time.Sleep(
					5 * time.Second,
				)

				continue
			}

			for _, update :=
				range updates {

				offset =
					update.UpdateID + 1

				if update.Message != nil {

					handleMessage(
						token,
						update.Message.Chat.ID,
						update.Message.Text,
					)
				}
			}
		}

	}()

	log.Println(
		"Web server started on port",
		port,
	)

	log.Fatal(
		http.ListenAndServe(
			":"+port,
			nil,
		),
	)
}
```
