package logging_test

import (
	"bytes"
	"fmt"
	"strings"

	"overnode/common/pkg/config"
	"overnode/common/pkg/logging"
)

func ExampleNew() {
	var settings struct {
		Logging logging.Config `mapstructure:"logging"`
	}
	err := config.Load([]string{"--logging.level=info", "--logging.format=json", "--logging.file_path="}, nil, &settings,
		config.Options{Command: "example", Fields: logging.Schema()})
	if err != nil {
		panic(err)
	}
	var output bytes.Buffer
	logger, closeLogger, err := logging.New(settings.Logging, &output)
	if err != nil {
		panic(err)
	}
	logger.Info("application started")
	if err := closeLogger(); err != nil {
		panic(err)
	}
	fmt.Println(strings.Contains(output.String(), `"msg":"application started"`))
	// Output: true
}
