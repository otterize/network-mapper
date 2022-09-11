package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/otterize/intents-operator/src/operator/api/v1alpha1"
	"github.com/otterize/network-mapper/cli/pkg/consts"
	"github.com/otterize/network-mapper/cli/pkg/intentsprinter"
	"github.com/otterize/network-mapper/cli/pkg/mapperclient"
	"github.com/otterize/network-mapper/cli/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"
)

const OutputFileKey = "file"
const OutputFormatKey = "format"
const OutputFormatYAML = "yaml"
const OutputFormatJSON = "json"
const NamespacesKey = "namespaces"
const NamespacesShorthand = "n"

var ExportCmd = &cobra.Command{
	Use:   "export",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mapperclient.WithClient(func(c *mapperclient.Client) error {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			namespacesFilter := viper.GetStringSlice(NamespacesKey)
			intentsFromMapper, err := c.ServiceIntents(ctxTimeout, namespacesFilter)
			if err != nil {
				return err
			}

			outputList := make([]v1alpha1.ClientIntents, 0)

			for _, serviceIntents := range intentsFromMapper {
				intentList := make([]v1alpha1.Intent, 0)

				for _, serviceIntent := range serviceIntents.Intents {
					intent := v1alpha1.Intent{
						Type: v1alpha1.IntentTypeHTTP,
						Name: serviceIntent.Name,
					}
					if len(serviceIntent.Namespace) != 0 {
						intent.Namespace = serviceIntent.Namespace
					}
					intentList = append(intentList, intent)
				}

				intentsOutput := v1alpha1.ClientIntents{
					TypeMeta: v1.TypeMeta{
						Kind:       consts.IntentsKind,
						APIVersion: consts.IntentsAPIVersion,
					},
					ObjectMeta: v1.ObjectMeta{
						Name:      serviceIntents.Client.Name,
						Namespace: serviceIntents.Client.Namespace,
					},
					Spec: &v1alpha1.IntentsSpec{Service: v1alpha1.Service{Name: serviceIntents.Client.Name}},
				}

				if len(intentList) != 0 {
					intentsOutput.Spec.Calls = intentList
				}

				outputList = append(outputList, intentsOutput)
			}

			formatted, err := getFormattedIntents(outputList)
			if err != nil {
				return err
			}

			if viper.GetString(OutputFileKey) != "" {
				f, err := os.Create(viper.GetString(OutputFileKey))
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = f.WriteString(formatted)
				if err != nil {
					return err
				}
				output.PrintStderr("Successfully wrote intents into %s\n", viper.GetString(OutputFileKey))
			} else {
				output.PrintStdout(formatted)
			}
			return nil
		})
	},
}

func getFormattedIntents(intentList []v1alpha1.ClientIntents) (string, error) {
	switch outputFormatVal := viper.GetString(OutputFormatKey); {
	case outputFormatVal == OutputFormatJSON:
		formatted, err := json.MarshalIndent(intentList, "", "  ")
		if err != nil {
			return "", err
		}

		return string(formatted), nil
	case outputFormatVal == OutputFormatYAML:
		buf := bytes.Buffer{}

		printer := intentsprinter.IntentsPrinter{}
		for _, intentYAML := range intentList {
			err := printer.PrintObj(&intentYAML, &buf)
			if err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	default:
		return "", fmt.Errorf("unexpected output format %s, use one of (%s, %s)", outputFormatVal, OutputFormatJSON, OutputFormatYAML)
	}
}

func init() {
	ExportCmd.Flags().String(OutputFileKey, "", "file path to write the output into")
	ExportCmd.Flags().String(OutputFormatKey, OutputFormatYAML, "format to output the intents - yaml/json")
	ExportCmd.Flags().StringSliceP(NamespacesKey, NamespacesShorthand, nil, "filter for specific namespaces")
}
