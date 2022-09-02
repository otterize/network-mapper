package convert

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/otterize/intents-operator/src/operator/api/v1alpha1"
	"github.com/otterize/network-mapper/cli/pkg/intentsprinter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

const regularFile = 0
const DirPathKey = "path"
const DirPathShorthand = "f"

func NewIntentsResourceFromIntentsSpec(spec v1alpha1.IntentsSpec) *v1alpha1.ClientIntents {
	return &v1alpha1.ClientIntents{
		TypeMeta: v1.TypeMeta{
			Kind:       "ClientIntents",
			APIVersion: "k8s.otterize.com/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: spec.Service.Name,
		},
		Spec: spec.DeepCopy(),
	}
}

var ConvertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Converts Otterize intents to Kubernetes ClientIntents resources.",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := intentsprinter.IntentsPrinter{}
		allowedExts := sets.NewString(".yaml", ".yml")
		entries, err := os.ReadDir(viper.GetString(DirPathKey))
		if err != nil {
			return fmt.Errorf("failed to read dir %s: %w", viper.GetString(DirPathKey), err)
		}

		for _, entry := range entries {
			err := func() error {
				if !allowedExts.Has(filepath.Ext(entry.Name())) || entry.Type() != regularFile {
					return nil
				}
				if err != nil {
					return err
				}

				path := filepath.Join(viper.GetString(DirPathKey), entry.Name())

				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				yamlReader := k8syaml.NewYAMLReader(bufio.NewReader(file))
				for {
					// Read document
					doc, err := yamlReader.Read()
					if err != nil {
						if errors.Is(err, io.EOF) {
							break
						}

						return fmt.Errorf("unable to parse YAML file %s: %w", path, err)
					}

					var intentsSpec v1alpha1.IntentsSpec
					err = yaml.UnmarshalStrict(doc, &intentsSpec)
					if err != nil {
						return fmt.Errorf("unable to parse YAML file %s: %w", path, err)
					}

					resource := NewIntentsResourceFromIntentsSpec(intentsSpec)
					err = printer.PrintObj(resource, os.Stdout)
					if err != nil {
						return err
					}
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	ConvertCmd.Flags().StringP(DirPathKey, DirPathShorthand, ".", "directory containing intents")
}
