package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	Major  = "1"
	Minor  = "0"
	Fix    = "0"
	Verbal = "Initial"
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:  "kii",
	Long: "Kii - Signed Webhook Challenge Service",
}

// Run enters into the cobra command to start the service.
func Run() error {
	// Check if the CONFIG_ENV environment variable is set
	configEnv := os.Getenv("CONFIG_ENV")
	if configEnv == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Warning: CONFIG_ENV is not set. Using 'local' as default.")
	}
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}

	return nil
}

var versionCmd = &cobra.Command{ //nolint:gochecknoglobals
	Use:   "version",
	Short: "Describes version.",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Version: %s.%s.%s %s\n", Major, Minor, Fix, Verbal)
	},
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(versionCmd)
}
