package config

import (
	"fmt"
	"time"

	"overnode/common/pkg/config/schema"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func registerSchema(v *viper.Viper, flags *pflag.FlagSet, fields []schema.Field) error {
	for _, field := range fields {
		if flags.Lookup(field.Key) != nil {
			return fmt.Errorf("configuration key %q conflicts with a built-in flag", field.Key)
		}
		v.SetDefault(field.Key, field.Default)
		if err := v.BindEnv(field.Key); err != nil {
			return fmt.Errorf("bind environment variable for %q: %w", field.Key, err)
		}
		switch field.Kind {
		case schema.KindString:
			flags.String(field.Key, field.Default.(string), field.Usage)
		case schema.KindUint16:
			flags.Uint16(field.Key, field.Default.(uint16), field.Usage)
		case schema.KindDuration:
			flags.Duration(field.Key, field.Default.(time.Duration), field.Usage)
		case schema.KindStringSlice:
			flags.StringSlice(field.Key, field.Default.([]string), field.Usage)
		case schema.KindInt:
			flags.Int(field.Key, field.Default.(int), field.Usage)
		default:
			return fmt.Errorf("register configuration field %q: unsupported kind %d", field.Key, field.Kind)
		}
	}
	return nil
}

func bindSchemaFlags(v *viper.Viper, flags *pflag.FlagSet, fields []schema.Field) error {
	for _, field := range fields {
		if err := v.BindPFlag(field.Key, flags.Lookup(field.Key)); err != nil {
			return fmt.Errorf("bind flag %q: %w", field.Key, err)
		}
	}
	return nil
}
