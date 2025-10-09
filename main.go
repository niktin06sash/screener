package main

import (
	"bnb_screener/screener"
	telegrambot "bnb_screener/telegramBot"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, err := screener.NewScreenerConfig(ctx)
	if err != nil {
		return
	}
	token := os.Getenv("BOT_TOKEN")
	bot, err := telegrambot.InitBot(token)
	if err != nil {
		return
	}
	err = config.ScreenerReader(ctx)
	if err != nil {
		return
	}
	chatID := os.Getenv("CHAT_ID")
	telegrambot.SendInitMsg(ctx, bot, config, chatID)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-quit:
		log.Printf("Server shutting down with signal: %v", sig)
		cancel()
	}
}
