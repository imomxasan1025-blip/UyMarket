package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
		log.Println("Ошибка отправки:", err)
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

func russianWelcome(token string, chatID int64) {
	sendMessage(
		token,
		chatID,
		"🏠 UyMarket\n\nДобро пожаловать!\n\nВыберите нужное действие:",
		russianMenu(),
	)
}

func uzbekWelcome(token string, chatID int64) {
	sendMessage(
		token,
		chatID,
		"🏠 UyMarket\n\nXush kelibsiz!\n\nKerakli bo‘limni tanlang:",
		uzbekMenu(),
	)
}

func handleMessage(token string, chatID int64, text string) {

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
		russianWelcome(token, chatID)

	case "🇺🇿 O‘zbekcha":
		uzbekWelcome(token, chatID)

	case "🏠 Купить недвижимость":
		sendMessage(
			token,
			chatID,
			"🏠 Купить недвижимость\n\nСкоро здесь появится каталог квартир и домов.",
			russianMenu(),
		)

	case "➕ Продать недвижимость":
		sendMessage(
			token,
			chatID,
			"➕ Продать недвижимость\n\nСкоро бот поможет вам бесплатно создать объявление.",
			russianMenu(),
		)

	case "🔎 Найти квартиру":
		sendMessage(
			token,
			chatID,
			"🔎 Найти квартиру\n\nСкоро вы сможете выбрать район, количество комнат, площадь и бюджет.",
			russianMenu(),
		)

	case "📋 Мои объявления":
		sendMessage(
			token,
			chatID,
			"📋 Мои объявления\n\nПока у вас нет опубликованных объявлений.",
			russianMenu(),
		)

	case "👨‍💼 Менеджер":
		sendMessage(
			token,
			chatID,
			"👨‍💼 Менеджер\n\nНапишите ваш вопрос. Мы свяжемся с вами.",
			russianMenu(),
		)

	case "🏠 Uy-joy sotib olish":
		sendMessage(
			token,
			chatID,
			"🏠 Uy-joy sotib olish\n\nTez orada kvartira va uylar katalogi paydo bo‘ladi.",
			uzbekMenu(),
		)

	case "➕ Uy-joy sotish":
		sendMessage(
			token,
			chatID,
			"➕ Uy-joy sotish\n\nTez orada bot sizga bepul e'lon yaratishga yordam beradi.",
			uzbekMenu(),
		)

	case "🔎 Kvartira qidirish":
		sendMessage(
			token,
			chatID,
			"🔎 Kvartira qidirish\n\nTez orada hudud, xonalar soni, maydon va byudjet bo‘yicha qidirish mumkin bo‘ladi.",
			uzbekMenu(),
		)

	case "📋 Mening e'lonlarim":
		sendMessage(
			token,
			chatID,
			"📋 Mening e'lonlarim\n\nHozircha sizda e'lonlar yo‘q.",
			uzbekMenu(),
		)

	case "👨‍💼 Menejer":
		sendMessage(
			token,
			chatID,
			"👨‍💼 Menejer\n\nSavolingizni yozing. Biz siz bilan bog‘lanamiz.",
			uzbekMenu(),
		)

	default:
		sendMessage(
			token,
			chatID,
			"Пожалуйста, выберите пункт меню.\n\nIltimos, menyudan bo‘limni tanlang.",
			russianMenu(),
		)
	}
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

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
