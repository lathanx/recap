package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/lathanx/recap/internal/server"
	"github.com/lathanx/recap/web"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the recap web UI",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dataDir := filepath.Join(home, ".recap")
		recapBin, _ := os.Executable()
		handler := server.New(dataDir, web.StaticFiles(), recapBin)
		fmt.Printf("Listening on http://localhost:%s\n", port)
		if err := http.ListenAndServe(":"+port, handler); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	webCmd.Flags().String("port", "8484", "port to listen on")
	rootCmd.AddCommand(webCmd)
}
