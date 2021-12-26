//Package config prepares an instance of config file
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

//CFG is config instance
var CFG General

// Read reads the given file to export configuration
func Read(e Env) *General {

	viper.AddConfigPath(".") // optionally look for config in the working directory
	viper.AutomaticEnv()
	root, _ := os.Getwd()
	viper.SetConfigFile(root + "/config/" + e.GetFile()) // Path to look for the config file in
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := viper.MergeInConfig()
	if err != nil {
		fmt.Println("Error in reading config")
		panic(err)
	}
	err = viper.Unmarshal(&CFG)
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return &CFG
}
