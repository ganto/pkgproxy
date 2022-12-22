package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cachePath   string
	enableDebug bool
)

const (
	defaultPath = "cache"
)

// NewRootCommand creates a new root cli command instance
func NewRootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "pkgproxy",
		Args:  cobra.MinimumNArgs(1),
		Short: "pkgproxy is a caching reverse proxy for Linux package repositories ",
		Long: `
Complete documentation is available at https://github.com/ganto/pkgproxy`,
	}
	c.PersistentFlags().StringVar(&cachePath, "path", defaultPath, "cache base path")
	c.PersistentFlags().BoolVar(&enableDebug, "debug", false, "enable debugging")
	c.AddCommand(newServeCommand())

	return c
}

// Execute starts the command
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
