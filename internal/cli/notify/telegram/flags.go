package telegram

import "github.com/spf13/cobra"

const (
	flagBotToken = "bot-token"
	flagChatID   = "chat"
	flagProxyURL = "proxy"
)

func RegisterFlags(cmd *cobra.Command, opt *Options) {
	cmd.Flags().StringVar(&opt.BotToken, flagBotToken, opt.BotToken, "Telegram bot token")
	cmd.Flags().StringVar(&opt.ChatID, flagChatID, opt.ChatID, "Telegram chat ID")
	cmd.Flags().StringVar(&opt.ProxyURL, flagProxyURL, opt.ProxyURL, "Proxy URL (e.g. socks5://127.0.0.1:1080)")
}
