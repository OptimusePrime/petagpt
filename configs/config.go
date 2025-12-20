package configs

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed petagpt.config.yaml
var defaultConfig []byte

//go:embed spacy_worker.py
var spacyWorkerScript []byte

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
		viper.AddConfigPath(defaultConfigDir)
		viper.SetConfigName("config")
	}

	if err := viper.MergeInConfig(); err != nil {
		var configFileNotFoundErr viper.ConfigFileNotFoundError

		if !errors.As(err, &configFileNotFoundErr) || err == nil {
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

		err = os.MkdirAll(filepath.Join(defaultConfigDir, "bin"), 0750)
		if err != nil {
			return err
		}

		spacyWorkerFile, err := os.Create(filepath.Join(defaultConfigDir, "bin/spacy_worker.py"))
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Join(err, spacyWorkerFile.Close())
		}()

		_, err = spacyWorkerFile.Write(spacyWorkerScript)
		if err != nil {
			return err
		}

		err = setupPythonEnv(context.Background())
		if err != nil {
			return err
		}
	}

	// err = viper.BindPFlags(cmd.Flags())
	// if err != nil {
	// 	return err
	// }

	return nil
}

func GetPythonPath() string {
	venvDir := filepath.Join(viper.GetString("data_dir"), "bin/venv")
	pythonPath := filepath.Join(venvDir, "bin/python3")

	return pythonPath
}

func setupPythonEnv(ctx context.Context) error {
	venvDir := filepath.Join(viper.GetString("data_dir"), "bin/venv")
	pythonPath := filepath.Join(venvDir, "bin/python3")

	_, err := os.Stat(venvDir)
	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	createVenvCmd := exec.CommandContext(ctx, "python3", "-m", "venv", venvDir)
	err = createVenvCmd.Run()
	if err != nil {
		return err
	}

	installSpacyPreCmd := exec.CommandContext(ctx, pythonPath, "-m", "pip", "install", "setuptools", "wheel")
	err = installSpacyPreCmd.Run()
	if err != nil {
		return err
	}

	installSpacyCmd := exec.CommandContext(ctx, pythonPath, "-m", "pip", "install", "spacy")
	err = installSpacyCmd.Run()
	if err != nil {
		return err
	}

	downloadSpacyModelCmd := exec.CommandContext(ctx, pythonPath, "-m", "spacy", "download", "hr_core_news_lg")
	err = downloadSpacyModelCmd.Run()
	if err != nil {
		return err
	}

	return nil
}
