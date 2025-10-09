package telegrambot

import (
	"bnb_screener/screener"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func InitBot(tokenAPI string) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(tokenAPI)
	if err != nil {
		log.Printf("Error while connected to Bot: %v", err)
		return nil, err
	}
	log.Println("Successful connect to Bot!")
	return bot, err
}

type BaseToken struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
}

type Website struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Social struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Info struct {
	ImageURL  string    `json:"imageUrl"`
	Header    string    `json:"header"`
	OpenGraph string    `json:"openGraph"`
	Websites  []Website `json:"websites"`
	Socials   []Social  `json:"socials"`
}

type Pair struct {
	BaseToken BaseToken `json:"baseToken"`
	Info      Info      `json:"info"`
	MarketCap float64   `json:"marketCap"`
}

func SendInitMsg(ctx context.Context, tgBot *tgbotapi.BotAPI, flaunch *screener.ScreenerConfig, channelID string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case token, ok := <-flaunch.Tokenchan:
				if !ok {
					return
				}
				url := fmt.Sprintf("https://api.dexscreener.com/tokens/v1/bsc/%s", token.TInfo.Address.Hex())
				resp, err := http.Get(url)
				if err != nil {
					fmt.Printf("Error request: %v\n", err)
					continue
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("Error ReadAll: %v\n", err)
					continue
				}
				var pairs []Pair
				err = json.Unmarshal(body, &pairs)
				if err != nil {
					fmt.Printf("Error read parsing JSON-Data: %v\n", err)
					continue
				}
				if len(pairs) > 0 {
					pair := pairs[0]
					text := fmt.Sprintf(
						"🚀 *New BNB Token Event found!*\n\n"+
							"🪙 *TOKEN INFO*\n"+
							"├ MarketCap: %v\n"+
							"├ Name: *%s*\n"+
							"├ Symbol: *%s*\n"+
							"└ Address: `%s`\n\n"+
							"👤 *BUYER INFO*\n"+
							"└ Address: `%s`\n\n"+
							"📦 *BLOCKCHAIN INFO*\n"+
							"├ Block: `%d`\n"+
							"└ TxHash: `%s`",
						pair.MarketCap,
						pair.BaseToken.Name,
						pair.BaseToken.Symbol,
						token.TInfo.Address.Hex(),
						token.RInfo.Address.Hex(),
						token.BlInfo.Block,
						token.BlInfo.TxHash,
					)
					var keyboard tgbotapi.InlineKeyboardMarkup
					if len(pair.Info.Websites) > 0 && len(pair.Info.Socials) > 0 {
						keyboard = tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonURL("GMGN-Token",
									fmt.Sprintf("https://gmgn.ai/bsc/token/%s", token.TInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("Explorer-Token",
									fmt.Sprintf("https://bscscan.com/token/%s", token.TInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("Website-Token",
									pair.Info.Websites[0].URL),
								tgbotapi.NewInlineKeyboardButtonURL("Twitter-Token",
									pair.Info.Socials[0].URL),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonURL("0xPPL-Buyer",
									fmt.Sprintf("https://0xppl.com/%s", token.RInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("GMGN-Buyer",
									fmt.Sprintf("https://gmgn.ai/bsc/address/%s", token.RInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("Explorer-Buyer",
									fmt.Sprintf("https://bscscan.com/address/%s", token.TInfo.Address.Hex())),
							),
						)
					} else {
						keyboard = tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonURL("GMGN-Token",
									fmt.Sprintf("https://gmgn.ai/bsc/token/%s", token.TInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("Explorer-Token",
									fmt.Sprintf("https://bscscan.com/token/%s", token.TInfo.Address.Hex())),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonURL("0xPPL-Buyer",
									fmt.Sprintf("https://0xppl.com/%s", token.RInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("GMGN-Buyer",
									fmt.Sprintf("https://gmgn.ai/bsc/address/%s", token.RInfo.Address.Hex())),
								tgbotapi.NewInlineKeyboardButtonURL("Explorer-Buyer",
									fmt.Sprintf("https://bscscan.com/address/%s", token.TInfo.Address.Hex())),
							),
						)
					}
					msg := tgbotapi.NewMessageToChannel(channelID, text)
					msg.ParseMode = "Markdown"
					msg.ReplyMarkup = keyboard
					if _, err := tgBot.Send(msg); err != nil {
						log.Printf("Telegram send error: %v", err)
						continue
					}
					log.Println("Successful send event to Bot!")
				}

			}
		}
	}()
}
