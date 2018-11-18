package shop

import (
	"encoding/base64"
	"fmt"

	"github.com/gorilla/securecookie"
	"github.com/spf13/cobra"
)

//
var secretKeyCommand = &cobra.Command{
	Use:   "secretkey",
	Short: "Secret key management",
	Long:  "Secret key management, generate",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var secretKeyGenerate = &cobra.Command{
	Use:   "generate",
	Short: "generate secret key",
	Long:  "generate secret key",
	Run: func(cmd *cobra.Command, args []string) {
		sess := securecookie.GenerateRandomKey(32)
		fmt.Println("base64:" + base64.StdEncoding.EncodeToString(sess))
	},
}
