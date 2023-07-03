package main

import (
	// "encoding/json"
	"log"
	// "net/http"

	// "github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const botToken string = "6153044382:AAG3hbeuY-zo1DFReKOQAIAofzStO3NXBUs"
const chatID  int64 = -921018861 // testBot group

var (
	bot *tgbotapi.BotAPI
)

func init() {
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
}

// func telegramBot(c *gin.Context) {
// 	type MessageTGRequest struct {
// 		Text 	string `json:"Text"`
// 		Path	string `json:"FilePath"`	
// 	}

//     r := c.Request
// 	var req MessageTGRequest
// 	err := json.NewDecoder(r.Body).Decode(&req)
// 	if err != nil {
// 		c.JSON(http.StatusOK, Response{Code:ErrorCodeMember})
// 		return
// 	}

// 	// use package
// 	err = SendWithMessage(req.Text)
// 	if err != nil {
// 		c.JSON(http.StatusOK, Response{Code:ErrorCodeTGBot, Msg:err.Error()})
// 		return 
// 	}
// 	if len(req.Path) > 0 {
// 		err = SendWithDocument(req.Path)
// 		if err != nil {
// 			c.JSON(http.StatusOK, Response{Code:ErrorCodeTGBot, Msg:err.Error()})
// 			return 
// 		}
// 	}

// 	c.JSON(http.StatusOK, Response{Code:ErrorCodeOK, Msg:"Send successful"})
// }

func SendWithMessage(message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := bot.Send(msg)

	return err
}

func SendWithDocument(path string) error {
	log.Println("SendWithDocument")
	msg := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(path)) 
	_, err := bot.Send(msg)

	return err
}