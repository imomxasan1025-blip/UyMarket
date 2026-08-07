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

type UpdateResponse struct {
	OK     bool `json:"ok"`
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"result"`
}

func telegramRequest(token, method string, data map[string]interface{}) error {
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

func sendMessage(token string, chatID int64, text string) {
	err := telegramRequest(token, "sendMessage", map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		log.Println("Ошибка отправки:", err)
	}
}

func getUpdates(token string, offset int) ([]struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}, error) {

	url := telegramAPI + token + "/getUpdates?timeout=30&offset=" + strconv.Itoa(offset)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UpdateResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("Telegram API вернул ошибку")
	}

	return result.Result, nil
}

func handleMessage(token string, chatID int64, text string) {

	switch text {

	case "/start":

		message := `🏠 Добро пожаловать в UyMarket!

Я помогу вам найти или разместить недвижимость.

Выберите действие:

🏠 Купить квартиру
🏢 Продать квартиру
🔎 Найти квартиру
📋 Мои объявления
📞 Связаться с менеджером`

		sendMessage(token, chatID, message)

	case "🏠 Купить квартиру":

		sendMessage(token, chatID,
			"🏠 Поиск недвижимости\n\nСкоро здесь появится поиск квартир по району, цене, площади и количеству комнат.")

	case "🏢 Продать квартиру":

		sendMessage(token, chatID,
			"🏢 Продажа квартиры\n\nСкоро я помогу вам разместить объявление о продаже квартиры.")

	case "🔎 Найти квартиру":

		sendMessage(token, chatID,
			"🔎 Поиск квартиры\n\nНапишите район, бюджет или количество комнат — мы постепенно добавим автоматический поиск.")

	case "📋 Мои объявления":

		sendMessage(token, chatID,
			"📋 У вас пока нет объявлений.")

	case "📞 Связаться с менеджером":

		sendMessage(token, chatID,
			"📞 Связаться с менеджером\n\nНапишите ваш вопрос, и менеджер свяжется с вами.")

	default:

		sendMessage(token, chatID,
			"Я пока не понял ваш запрос.\n\nНажмите /start, чтобы открыть меню UyMarket.")
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

		log.Println("Telegram bot запущен")

		offset := 0

		for {

			updates, err := getUpdates(token, offset)

			if err != nil {
				log.Println("Ошибка Telegram:", err)
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

	log.Println("Web server запущен на порту", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
