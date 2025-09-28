package configs

import (
	"bytes"
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed petagpt.config.yaml
var defaultConfig []byte

var CfgFile string

func InitConfig(cmd *cobra.Command) error {
	viper.SetEnvPrefix("PETAGPT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "*", "-", "*"))
	viper.AutomaticEnv() // read in environment variables that match

	home, err := os.UserConfigDir()
	cobra.CheckErr(err)

	defaultConfigDir := filepath.Join(home, "/.petagpt")
	defaultConfigPath := filepath.Join(defaultConfigDir, "config.yaml")

	viper.SetDefault("data_dir", defaultConfigDir)

	viper.SetConfigType("yaml")
	if len(defaultConfig) > 0 {
		err := viper.ReadConfig(bytes.NewReader(defaultConfig))
		if err != nil {
			return err
		}
	}

	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath(defaultConfigPath)
		viper.SetConfigName("config")
	}

	if err := viper.MergeInConfig(); err != nil {
		var configFileNotFoundErr viper.ConfigFileNotFoundError

		if !errors.As(err, &configFileNotFoundErr) {
			return err
		}

		err := os.MkdirAll(defaultConfigDir, 0750)
		if err != nil {
			return err
		}

		configFile, err := os.Create(defaultConfigPath)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Join(err, configFile.Close())
		}()

		err = viper.WriteConfigAs(defaultConfigPath)
		if err != nil {
			return err
		}
	}

	err = viper.BindPFlags(cmd.Flags())
	if err != nil {
		return err
	}

	return nil
}
