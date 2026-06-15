package telegram

import (
	"context"
	"net/url"

	"github.com/GaIsBAX/Webhix/internal/cli/apiclient"
	"github.com/spf13/cobra"
)

type Options struct {
	BotToken string
	ChatID   string
	ProxyURL string
}

func NewCommand(ctx context.Context, client *apiclient.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "Manage Telegram notifications",
	}

	cmd.AddCommand(newSetCmd(ctx, client))
	cmd.AddCommand(newTestCmd(ctx, client))
	cmd.AddCommand(newRemoveCmd(ctx, client))

	return cmd
}

func newSetCmd(ctx context.Context, client *apiclient.Client) *cobra.Command {
	opts := Options{}

	cmd := &cobra.Command{
		Use:   "set <token>",
		Short: "Configure Telegram notifications for an endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := map[string]string{"bot_token": opts.BotToken, "chat_id": opts.ChatID}
			if opts.ProxyURL != "" {
				cfg["proxy_url"] = opts.ProxyURL
			}

			body := map[string]any{"provider": "telegram", "config": cfg}
			path := "/api/endpoints/" + url.PathEscape(args[0]) + "/notifications/telegram"
			if err := client.Put(ctx, path, body); err != nil {
				return err
			}

			cmd.Println("Telegram notifications configured.")
			return nil
		},
	}

	RegisterFlags(cmd, &opts)
	must(cmd.MarkFlagRequired(flagBotToken))
	must(cmd.MarkFlagRequired(flagChatID))

	return cmd
}

func newTestCmd(ctx context.Context, client *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "test <token>",
		Short: "Send a test Telegram message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/endpoints/" + url.PathEscape(args[0]) + "/notifications/telegram/test"
			if err := client.Post(ctx, path, nil); err != nil {
				return err
			}

			cmd.Println("Test message sent.")
			return nil
		},
	}
}

func newRemoveCmd(ctx context.Context, client *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <token>",
		Short: "Remove Telegram notifications from an endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/endpoints/" + url.PathEscape(args[0]) + "/notifications/telegram"
			if err := client.Delete(ctx, path); err != nil {
				return err
			}

			cmd.Println("Telegram notifications removed.")
			return nil
		},
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
