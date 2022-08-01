package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	MapperServiceNameKey     = "mapper-service-name"
	MapperServiceNameDefault = "otternose-mapper"
	MapperNamespaceKey       = "mapper-namespace"
	MapperNamespaceDefault   = "otternose"
	MapperServicePortKey     = "mapper-service-port"
	MapperServicePortDefault = 9090
)

func BindPFlags(cmd *cobra.Command, _ []string) {
	cobra.CheckErr(viper.BindPFlags(cmd.Flags()))
	cobra.CheckErr(viper.BindPFlags(cmd.PersistentFlags()))
}
