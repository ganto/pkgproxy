package cmd

import (
	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	"github.com/spf13/cobra"
)

var (
	proxyHost string
	proxyPort uint16
)

const (
	defaultHost = "localhost"
	defaultPort = 8080
)

func newServeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "serve",
		Args:  cobra.ArbitraryArgs,
		Short: "Start reverse proxy",
		Run: func(cmd *cobra.Command, args []string) {
			host, _ := cmd.Flags().GetString("proxyHost")
			port, _ := cmd.Flags().GetUint16("port")
			pkgproxy.StartServer(host, port)
		},
	}
	c.PersistentFlags().StringVar(&proxyHost, "proxyHost", defaultHost, "host of the reverse proxy.")
	c.PersistentFlags().Uint16Var(&proxyPort, "port", defaultPort, "local port of the reverse proxy.")

	return c
}
