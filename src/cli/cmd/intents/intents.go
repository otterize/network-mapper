package intents

import (
	"github.com/otterize/otternose/cli/cmd/intents/list"
	"github.com/otterize/otternose/cli/cmd/intents/serialize"
	"github.com/spf13/cobra"
)

var IntentsCmd = &cobra.Command{
	Use:   "intents",
	Short: "",
	Long:  ``,
}

func init() {
	IntentsCmd.AddCommand(serialize.SerializeCmd)
	IntentsCmd.AddCommand(list.ListCmd)
}
