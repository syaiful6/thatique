package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/gorilla/securecookie"
	"github.com/spf13/cobra"
)

var secretKey64Len bool

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
	Long:  `generate secret key, 32 or 64 bytes length. To generate 64 bytes key
use flag -L.
`,
	Run: func(cmd *cobra.Command, args []string) {
		var len = 32
		if secretKey64Len {
			len = 64
		}
		sess := securecookie.GenerateRandomKey(len)
		fmt.Println("base64:" + base64.StdEncoding.EncodeToString(sess))
	},
}
