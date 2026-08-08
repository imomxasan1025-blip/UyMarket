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
	Price    string
}

var users = make(map[int64]*UserState)
var usersMutex sync.Mutex

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
		return fmt.Errorf("Telegram HTTP error: %s", resp.Status)
	}

	return nil
}

func sendMessage(token string, chatID int64, text string, keyboard [][]string) {
	request := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	if keyboard != nil {
		rows := make([][]map[string]string, len(keyboard))

		for i, row := range keyboard {
			for _, button := range row {
				rows[i] = append(rows[i], map[string]string{
					"text": button,
				})
			}
		}

		request["reply_markup"] = map[string]interface{}{
			"keyboard":        rows,
			"resize_keyboard": true,
		}
	}

	if err := telegramRequest(token, "sendMessage", request); err != nil {
		log.Println("Ошибка Telegram:", err)
	}
}

func getUpdates(token string, offset int) ([]Update, error) {
	url := telegramAPI + token +
		"/getUpdates?timeout=30&offset=" +
		strconv.Itoa(offset)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var result UpdatesResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("Telegram API error")
	}

	return result.Result, nil
}

func russianMenu() [][]string {
	return [][]string{
		{"🏠 Купить недвижимость", "➕ Продать недвижимость"},
		{"🔎 Найти квартиру", "📋 Мои объявления"},
		{"👨‍💼 Менеджер", "🇺🇿 O‘zbekcha"},
	}
}

func uzbekMenu() [][]string {
	return [][]string{
		{"🏠 Uy-joy sotib olish", "➕ Uy-joy sotish"},
		{"🔎 Kvartira qidirish", "📋 Mening e'lonlarim"},
		{"👨‍💼 Menejer", "🇷🇺 Русский"},
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

func startSale(token string, chatID int64) {
	usersMutex.Lock()

	users[chatID] = &UserState{
		Step:     1,
		Language: "ru",
	}

	usersMutex.Unlock()

	sendMessage(
		token,
		chatID,
		"➕ Продажа недвижимости\n\nВыберите тип недвижимости:",
		propertyMenu(),
	)
}

func processSale(token string, chatID int64, text string) {
	usersMutex.Lock()

	user, exists := users[chatID]

	if !exists {
		usersMutex.Unlock()
		return
	}

	switch user.Step {

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
			"📍 В каком городе находится объект?\n\nНапишите город.",
			nil,
		)

	case 2:
		user.City = text
		user.Step = 3

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"📍 В каком районе находится объект?\n\nНапишите район.",
			nil,
		)

	case 3:
		user.District = text
		user.Step = 4

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"🛏 Сколько комнат?\n\nНапример: 2",
			nil,
		)

	case 4:
		user.Rooms = text
		user.Step = 5

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"📐 Какая площадь?\n\nНапример: 65 м²",
			nil,
		)

	case 5:
		user.Area = text
		user.Step = 6

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"🏢 Какой этаж и сколько этажей в доме?\n\nНапример: 7/16",
			nil,
		)

	case 6:
		user.Floor = text
		user.Step = 7

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			"💰 Какая цена?\n\nНапример: 85000 USD",
			nil,
		)

	case 7:
		user.Price = text
		user.Step = 8

		summary := fmt.Sprintf(
			"📋 Проверьте объявление:\n\n"+
				"🏷 Тип: %s\n"+
				"📍 Город: %s\n"+
				"📍 Район: %s\n"+
				"🛏 Комнаты: %s\n"+
				"📐 Площадь: %s\n"+
				"🏢 Этаж: %s\n"+
				"💰 Цена: %s\n\n"+
				"Всё правильно?",
			user.Type,
			user.City,
			user.District,
			user.Rooms,
			user.Area,
			user.Floor,
			user.Price,
		)

		usersMutex.Unlock()

		sendMessage(
			token,
			chatID,
			summary,
			[][]string{
				{"✅ Опубликовать"},
				{"❌ Отменить"},
			},
		)

	case 8:
		if text == "❌ Отменить" {
			delete(users, chatID)
			usersMutex.Unlock()

			sendMessage(
				token,
				chatID,
				"❌ Объявление отменено.",
				russianMenu(),
			)
			return
		}

		if text == "✅ Опубликовать" {
			data := map[string]string{
				"type":     user.Type,
				"city":     user.City,
				"district": user.District,
				"rooms":    user.Rooms,
				"area":     user.Area,
				"floor":    user.Floor,
				"price":    user.Price,
			}

			usersMutex.Unlock()

			err := sendToGoogleSheets(data)

			if err != nil {
				log.Println("Ошибка сохранения объявления:", err)

				sendMessage(
					token,
					chatID,
					"❌ Не удалось сохранить объявление.\n\n"+
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
				"✅ Объявление успешно опубликовано!\n\n"+
					"Оно сохранено в базе UyMarket.",
				russianMenu(),
			)
			return
		}

		usersMutex.Unlock()

	default:
		usersMutex.Unlock()
	}
}

func handleMessage(token string, chatID int64, text string) {
	usersMutex.Lock()
	_, inSale := users[chatID]
	usersMutex.Unlock()

	if inSale {
		processSale(token, chatID, text)
		return
	}

	switch text {

	case "/start":
		sendMessage(
			token,
			chatID,
			"🏠 UyMarket\n\nВыберите язык / Tilni tanlang:",
			[][]string{
				{"🇷🇺 Русский", "🇺🇿 O‘zbekcha"},
			},
		)

	case "🇷🇺 Русский":
		sendMessage(
			token,
			chatID,
			"🏠 UyMarket\n\nДобро пожаловать!\n\nВыберите действие:",
			russianMenu(),
		)

	case "🇺🇿 O‘zbekcha":
		sendMessage(
			token,
			chatID,
			"🏠 UyMarket\n\nXush kelibsiz!\n\nKerakli bo‘limni tanlang:",
			uzbekMenu(),
		)

	case "➕ Продать недвижимость":
		startSale(token, chatID)

	case "🏠 Купить недвижимость":
		sendMessage(
			token,
			chatID,
			"🏠 Каталог недвижимости\n\n"+
				"Скоро здесь появятся реальные объявления.",
			russianMenu(),
		)

	case "🔎 Найти квартиру":
		sendMessage(
			token,
			chatID,
			"🔎 Поиск квартиры\n\n"+
				"Скоро здесь появится поиск по району, цене, площади и комнатам.",
			russianMenu(),
		)

	case "📋 Мои объявления":
		sendMessage(
			token,
			chatID,
			"📋 Мои объявления\n\n"+
				"Пока объявлений нет.",
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
		startSale(token, chatID)

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

func sendToGoogleSheets(data map[string]string) error {
	url := os.Getenv("GOOGLE_SHEETS_URL")

	if url == "" {
		return fmt.Errorf("GOOGLE_SHEETS_URL не найден")
	}

	body, err := json.Marshal(data)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"Google Apps Script вернул статус %d",
			resp.StatusCode,
		)
	}

	return nil
}

func main() {
	token := os.Getenv("BOT_TOKEN")

	if token == "" {
		log.Fatal("BOT_TOKEN не найден")
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "10000"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "UyMarket bot is running!")
	})

	go func() {
		log.Println("UyMarket Telegram bot started")

		offset := 0

		for {
			updates, err := getUpdates(token, offset)

			if err != nil {
				log.Println("Telegram error:", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, update := range updates {
				offset = update.UpdateID + 1

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

	log.Println("Web server started on port", port)

	log.Fatal(
		http.ListenAndServe(":"+port, nil),
	)
}
