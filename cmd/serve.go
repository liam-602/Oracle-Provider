package cmd

import (
	"log"

	"github.com/Interlocked-Labs/oracle-provider/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPath string

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start http server with configured api",
	Long:  `Starts a http server and serves the configured api`,
	Run: func(cmd *cobra.Command, args []string) {
		server, err := api.NewServer(configPath)
		if err != nil {
			log.Fatal(err)
		}
		server.Start()
	},
}

func init() {
	serveCmd.Flags().StringVar(&configPath, "config", "", "Path to the configuration file")
	RootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.
	viper.SetDefault("log_level", "debug")
}
