package list

import (
	"context"
	"github.com/otterize/network-mapper/cli/pkg/mapperclient"
	"github.com/otterize/network-mapper/cli/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const NamespacesKey = "namespaces"
const NamespacesShorthand = "n"

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mapperclient.WithClient(func(c *mapperclient.Client) error {
			namespacesFilter := viper.GetStringSlice(NamespacesKey)
			servicesIntents, err := c.ServiceIntents(context.Background(), namespacesFilter)
			if err != nil {
				return err
			}
			for _, service := range servicesIntents {
				output.PrintStdout("%s in namespace %s calls:", service.Client.Name, service.Client.Namespace)
				for _, intent := range service.Intents {
					if len(intent.Namespace) != 0 {
						output.PrintStdout("  - %s in namespace %s", intent.Name, intent.Namespace)
						continue
					}

					output.PrintStdout("  - %s in same namespace", intent.Name)
				}

			}
			return nil
		})
	},
}

func init() {
	ListCmd.Flags().StringSliceP(NamespacesKey, NamespacesShorthand, nil, "filter for specific namespaces")
}
