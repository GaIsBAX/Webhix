package notify

import (
	"context"
	"net/url"

	"github.com/GaIsBAX/Webhix/internal/cli/apiclient"
	"github.com/GaIsBAX/Webhix/internal/cli/notify/telegram"
	"github.com/GaIsBAX/Webhix/internal/config"
	"github.com/spf13/cobra"
)

type notificationChannel struct {
	Provider string            `json:"provider"`
	Config   map[string]string `json:"config"`
	Redacted []string          `json:"redacted"`
}

func NewCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	opts := DefaultOptions()
	if cfg.SecretKey != "" {
		opts.AuthToken = cfg.SecretKey
	}

	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Manage endpoint notifications",
	}

	RegisterFlags(cmd, &opts)

	client := apiclient.New(&opts.Server, &opts.AuthToken)

	cmd.AddCommand(newListCmd(ctx, client))
	cmd.AddCommand(telegram.NewCommand(ctx, client))

	return cmd
}

func newListCmd(ctx context.Context, client *apiclient.Client) *cobra.Command {
	return &cobra.Command{
		Use:   "list <token>",
		Short: "List all configured notification channels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var channels []notificationChannel
			if err := client.Get(ctx, "/api/endpoints/"+url.PathEscape(args[0])+"/notifications", &channels); err != nil {
				return err
			}

			if len(channels) == 0 {
				cmd.Println("No notifications configured.")
				return nil
			}

			for _, ch := range channels {
				cmd.Printf("Provider: %s\n", ch.Provider)
				for k, v := range ch.Config {
					cmd.Printf("  %s: %s\n", k, v)
				}
				for _, k := range ch.Redacted {
					cmd.Printf("  %s: [set]\n", k)
				}
			}
			return nil
		},
	}
}
