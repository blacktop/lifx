/*
Copyright © 2025 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"os"
	"time"

	"github.com/blacktop/lifx/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagBackend, "backend", "auto", "Backend to use: auto|lan|api")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "LIFX API key (overrides LIFX_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&flagLanListen, "lan-listen", "", "LAN listen IP (default 0.0.0.0)")
	rootCmd.PersistentFlags().StringVar(&flagLanBroadcast, "lan-broadcast", "", "LAN broadcast or target IP (e.g. 255.255.255.255 or 192.168.1.255)")
	rootCmd.PersistentFlags().DurationVar(&flagRefresh, "refresh", 5*time.Second, "Auto-refresh interval (0 to disable)")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug logging")

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

var (
	flagBackend      string
	flagAPIKey       string
	flagLanListen    string
	flagLanBroadcast string
	flagRefresh      time.Duration
	flagDebug        bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lifx",
	Short: "A gorgeous LIFX terminal UI",
	Long: `lifx is a gorgeous terminal UI for controlling LIFX lights.
It prefers the LIFX LAN protocol, but can use the HTTP API when LIFX_API_KEY is set.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := tui.Options{
			Backend:      flagBackend,
			APIKey:       flagAPIKey,
			LanListen:    flagLanListen,
			LanBroadcast: flagLanBroadcast,
			AutoRefresh:  flagRefresh,
			Debug:        flagDebug,
		}
		return tui.Run(cmd.Context(), opts)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
