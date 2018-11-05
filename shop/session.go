package shop

import (
	"encoding/base64"
	"os"

	"github.com/spf13/cobra"
	"github.com/gorilla/securecookie"
)


//
var sessionCommand = &cobra.Command{
	Use:   "session",
	Short: "Thatiq's session management",
	Long:  "Thatiq's session management",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var sessionGenerateKey = &cobra.Command{
	Use: "generate",
	Short: "generate session key",
	Long:  "generate session key",
	Run: func(cmd *cobra.Command, args []string) {
		sess := securecookie.GenerateRandomKey(64)
		encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
		encoder.Write(sess)
		encoder.Close()
	},
}
