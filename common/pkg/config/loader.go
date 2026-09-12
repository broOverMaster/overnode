// Пакет config загружает конфигурацию из поддерживаемых внешних источников.
package config

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"overnode/common/pkg/config/schema"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ErrHelp сообщает вызывающему коду, что справка напечатана и запуск следует завершить успешно.
var ErrHelp = errors.New("help requested")

// Options задаёт схему и текст справки загружаемой конфигурации.
type Options struct {
	Command     string
	Description string
	Fields      []schema.Field
}

// Load объединяет внешние источники и строго декодирует результат в destination.
func Load(args []string, output io.Writer, destination any, options Options) error {
	if output == nil {
		output = io.Discard
	}
	if err := validateDestination(destination); err != nil {
		return err
	}
	if strings.TrimSpace(options.Command) == "" {
		return fmt.Errorf("configuration command name must not be empty")
	}
	fields, err := schema.Combine(options.Fields)
	if err != nil {
		return fmt.Errorf("build configuration schema: %w", err)
	}

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	flags := pflag.NewFlagSet(options.Command, pflag.ContinueOnError)
	flags.SetOutput(output)
	configFile := flags.String("config", "", "path to a TOML, YAML or JSON configuration file")
	help := flags.BoolP("help", "h", false, "show command help")
	if err := registerSchema(v, flags, fields); err != nil {
		return err
	}
	flags.Usage = func() {
		fmt.Fprintf(output, "Usage: %s [flags]\n", options.Command)
		if options.Description != "" {
			fmt.Fprintln(output)
			fmt.Fprintln(output, options.Description)
		}
		fmt.Fprintln(output)
		fmt.Fprintln(output, "Flags:")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse command line: %w", err)
	}
	if *help {
		flags.Usage()
		return ErrHelp
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if err := bindSchemaFlags(v, flags, fields); err != nil {
		return err
	}

	if *configFile != "" {
		v.SetConfigFile(*configFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("read configuration file %q: %w", *configFile, err)
		}
	}
	if err := v.UnmarshalExact(destination, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		strictUint16Hook,
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	if err := resolvePathFields(destination, v.ConfigFileUsed()); err != nil {
		return fmt.Errorf("resolve configuration paths: %w", err)
	}
	return nil
}

func validateDestination(destination any) error {
	if destination == nil {
		return fmt.Errorf("configuration destination must be a non-nil pointer")
	}
	value := reflect.ValueOf(destination)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("configuration destination must be a non-nil pointer to a struct")
	}
	return nil
}
