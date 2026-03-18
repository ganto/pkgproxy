// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Version is injected at build time via -ldflags.
	Version = "unknown"
	// GitCommit is injected at build time via -ldflags.
	GitCommit = "unknown"
	// BuildDate is injected at build time via -ldflags.
	// Falls back to current time when not set.
	BuildDate string
)

func buildDate() string {
	if BuildDate != "" {
		return BuildDate
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("Version:    %s\n", Version)
			cmd.Printf("GitCommit:  %s\n", GitCommit)
			cmd.Printf("GoVersion:  %s\n", runtime.Version())
			cmd.Printf("BuildDate:  %s\n", buildDate())
		},
	}
}
