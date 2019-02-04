package configuration

import (
	"fmt"
	"os"
	"path/filepath"
)

func Resolve(args []string) (*Configuration, error) {
	var (
		configurationPath string
		err error
	)

	defaultPath, err := filepath.Abs("./thatiq-config.yml")
	fmt.Printf("absolute path in %s\n", defaultPath)
	if err != nil {
		defaultPath = ""
	}
	if len(args) > 0 {
		configurationPath = args[0]
	} else if os.Getenv("THATIQ_CONFIGURATION_PATH") != "" {
		configurationPath = os.Getenv("THATIQ_CONFIGURATION_PATH")
	} else if len(defaultPath) > 0 {
		if  _, err = os.Stat(defaultPath); err == nil {
			configurationPath = defaultPath
		}
	}

	if configurationPath == "" {
		return nil, fmt.Errorf("configuration path unspecified and default path not found")
	}

	fp, err := os.Open(configurationPath)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	config, err := Parse(fp)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", configurationPath, err)
	}

	return config, nil
}
