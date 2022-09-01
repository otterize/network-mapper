package list

import (
	"context"
	"fmt"
	"github.com/otterize/network-mapper/cli/pkg/mapperclient"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mapperclient.WithClient(func(c *mapperclient.Client) error {
			servicesIntents, err := c.ServiceIntents(context.Background())
			if err != nil {
				return err
			}
			for _, service := range servicesIntents {
				println(service.Name, "calls:")
				for _, intent := range service.Intents {
					fullName := lo.Ternary(intent.Namespace != "", fmt.Sprintf("%s.%s", intent.Name, intent.Namespace), intent.Name)
					fmt.Printf("  - %s\n", fullName)
				}
				println()
			}
			return nil
		})
	},
}
