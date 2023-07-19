package main

import (
	"log"
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