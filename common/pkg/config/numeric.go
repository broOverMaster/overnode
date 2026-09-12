package config

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// strictUint16Hook rejects overflow before mapstructure narrows external numbers.
// CLI pflag checks its own input; file and environment need the same guarantee.
func strictUint16Hook(from, to reflect.Type, data any) (any, error) {
	if from == nil || to == nil || to.Kind() != reflect.Uint16 {
		return data, nil
	}
	value := reflect.ValueOf(data)
	switch value.Kind() {
	case reflect.String:
		parsed, err := strconv.ParseUint(value.String(), 10, 16)
		if err != nil {
			return nil, fmt.Errorf("value must be an integer between 0 and 65535")
		}
		return uint16(parsed), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n := value.Int(); n >= 0 && n <= 65535 {
			return uint16(n), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n := value.Uint(); n <= 65535 {
			return uint16(n), nil
		}
	case reflect.Float32, reflect.Float64:
		n := value.Float()
		if !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0 && n <= 65535 && n == math.Trunc(n) {
			return uint16(n), nil
		}
	}
	return nil, fmt.Errorf("value must be an integer between 0 and 65535")
}
