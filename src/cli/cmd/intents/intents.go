package intents

import (
	"github.com/otterize/otternose/cli/cmd/intents/crd"
	"github.com/otterize/otternose/cli/cmd/intents/list"
	"github.com/spf13/cobra"
)

var IntentsCmd = &cobra.Command{
	Use:   "intents",
	Short: "",
	Long:  ``,
}

func init() {
	IntentsCmd.AddCommand(crd.CrdCmd)
	IntentsCmd.AddCommand(list.ListCmd)
}
