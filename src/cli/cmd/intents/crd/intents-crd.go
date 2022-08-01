package crd

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/cli/pkg/mapperclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

const OutputFileKey = "file"

var CrdCmd = &cobra.Command{
	Use:   "crd",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mapperclient.WithClient(func(c *mapperclient.Client) error {
			res, err := c.FormattedCRDs(context.Background())
			if err != nil {
				return err
			}
			if viper.GetString(OutputFileKey) != "" {
				f, err := os.Create(viper.GetString(OutputFileKey))
				if err != nil {
					return err
				}
				_, err = f.WriteString(res)
				if err != nil {
					return err
				}
				fmt.Printf("Successfully wrote intents into %s\n", viper.GetString(OutputFileKey))
			} else {
				println(res)
			}
			return nil
		})
	},
}

func init() {
	CrdCmd.PersistentFlags().String(OutputFileKey, "", "file path to write the CRDs into")
}
