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
	"path/filepath"
	"time"
)

const OutputLocationKey = "output"
const OutputLocationShorthand = "o"
const OutputTypeKey = "output-type"
const OutputTypeDefault = OutputTypeSingleFile
const OutputTypeSingleFile = "single-file"
const OutputTypeDirectory = "dir"
const OutputFormatKey = "format"
const OutputFormatDefault = OutputFormatYAML
const OutputFormatYAML = "yaml"
const OutputFormatJSON = "json"
const NamespacesKey = "namespaces"
const NamespacesShorthand = "n"

func writeIntentsFile(filePath string, intents []v1alpha1.ClientIntents) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	formatted, err := getFormattedIntents(intents)
	if err != nil {
		return err
	}
	_, err = f.WriteString(formatted)
	if err != nil {
		return err
	}
	return nil
}

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

			if viper.GetString(OutputLocationKey) != "" {
				switch outputTypeVal := viper.GetString(OutputTypeKey); {
				case outputTypeVal == OutputTypeSingleFile:
					err := writeIntentsFile(viper.GetString(OutputLocationKey), outputList)
					if err != nil {
						return err
					}
					output.PrintStderr("Successfully wrote intents into %s", viper.GetString(OutputLocationKey))
				case outputTypeVal == OutputTypeDirectory:
					err := os.MkdirAll(viper.GetString(OutputLocationKey), 0700)
					if err != nil {
						return fmt.Errorf("could not create dir %s: %w", viper.GetString(OutputLocationKey), err)
					}

					for _, intent := range outputList {
						filePath := fmt.Sprintf("%s.%s.yaml", intent.Name, intent.Namespace)
						if err != nil {
							return err
						}
						filePath = filepath.Join(viper.GetString(OutputLocationKey), filePath)
						err := writeIntentsFile(filePath, []v1alpha1.ClientIntents{intent})
						if err != nil {
							return err
						}
					}
					output.PrintStderr("Successfully wrote intents into %s", viper.GetString(OutputLocationKey))
				default:
					return fmt.Errorf("unexpected output type %s, use one of (%s, %s)", outputTypeVal, OutputTypeSingleFile, OutputTypeDirectory)
				}

			} else {
				formatted, err := getFormattedIntents(outputList)
				if err != nil {
					return err
				}
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
	ExportCmd.Flags().StringP(OutputLocationKey, OutputLocationShorthand, "", "file or dir path to write the output into")
	ExportCmd.Flags().String(OutputTypeKey, OutputTypeDefault, fmt.Sprintf("whether to write output to file or dir: %s/%s", OutputTypeSingleFile, OutputTypeDirectory))
	ExportCmd.Flags().String(OutputFormatKey, OutputFormatDefault, fmt.Sprintf("format to output the intents - %s/%s", OutputFormatYAML, OutputFormatJSON))
	ExportCmd.Flags().StringSliceP(NamespacesKey, NamespacesShorthand, nil, "filter for specific namespaces")
}
