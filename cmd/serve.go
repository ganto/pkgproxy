package cmd

import (
	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	"github.com/spf13/cobra"
)

var (
	listenAddress string
	listenPort    uint16
)

const (
	defaultAddress = "localhost"
	defaultPort    = 8080
)

func newServeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "serve",
		Args:  cobra.ArbitraryArgs,
		Short: "Start reverse proxy",
		Run: func(cmd *cobra.Command, args []string) {
			enableDebug, _ := cmd.Flags().GetBool("debug")
			path, _ := cmd.Flags().GetString("path")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetUint16("port")
			pkgproxy.StartServer(enableDebug, path, host, port)
		},
		TraverseChildren: true,
	}
	c.PersistentFlags().StringVar(&listenAddress, "host", defaultAddress, "listen address of the pkgproxy.")
	c.PersistentFlags().Uint16Var(&listenPort, "port", defaultPort, "listen port of the pkgproxy.")

	return c
}
