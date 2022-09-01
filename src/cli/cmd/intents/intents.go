package intents

import (
	"github.com/otterize/otternose/cli/cmd/intents/export"
	"github.com/otterize/otternose/cli/cmd/intents/list"
	"github.com/spf13/cobra"
)

var IntentsCmd = &cobra.Command{
	Use:   "intents",
	Short: "",
	Long:  ``,
}

func init() {
	IntentsCmd.AddCommand(export.ExportCmd)
	IntentsCmd.AddCommand(list.ListCmd)
}
