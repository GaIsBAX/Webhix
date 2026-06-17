package cli

import (
	"context"

	"github.com/GaIsBAX/Webhix/internal/cli/forward"
	"github.com/GaIsBAX/Webhix/internal/cli/notify"
	"github.com/GaIsBAX/Webhix/internal/cli/serve"
	"github.com/GaIsBAX/Webhix/internal/cli/tunnel"
	"github.com/GaIsBAX/Webhix/internal/cli/version"
	"github.com/GaIsBAX/Webhix/internal/config"
	"github.com/GaIsBAX/Webhix/internal/core"
	"github.com/spf13/cobra"
)

func NewRootCommand(
	ctx context.Context,
	cfg *config.Config,
	serveFactory serve.ServiceFactory,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "webhix",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       core.WebhixVersion,
	}
	cmd.SetVersionTemplate("webhix {{.Version}}\n")

	addGroup(cmd, serve.ServeGroup, serve.ServeTitle)

	cmd.AddCommand(serve.NewCommand(ctx, cfg, serveFactory))
	cmd.AddCommand(forward.NewCommand(ctx, cfg))
	cmd.AddCommand(notify.NewCommand(ctx, cfg))
	cmd.AddCommand(tunnel.NewCommand(ctx, cfg))
	cmd.AddCommand(version.NewCommand(ctx))

	return cmd
}

func addGroup(cmd *cobra.Command, id, title string) {
	cmd.AddGroup(&cobra.Group{
		ID:    id,
		Title: title,
	})
}
