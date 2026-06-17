package notify

import "github.com/spf13/cobra"

const (
	flagServer    = "server"
	flagAuthToken = "auth-token"
)

func RegisterFlags(cmd *cobra.Command, opt *Options) {
	cmd.PersistentFlags().StringVar(&opt.Server, flagServer, opt.Server, "Webhix server URL")
	cmd.PersistentFlags().StringVar(&opt.AuthToken, flagAuthToken, opt.AuthToken, "auth token (env: WEBHIX_SECRET_KEY)")
}
