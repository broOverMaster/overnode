package schema

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

// Combine объединяет схемы компонентов и проверяет однозначность всех полей.
func Combine(groups ...[]Field) ([]Field, error) {
	var result []Field
	keys := make(map[string]struct{})
	for _, group := range groups {
		for _, field := range group {
			if err := validate(field); err != nil {
				return nil, err
			}
			if _, exists := keys[field.Key]; exists {
				return nil, fmt.Errorf("duplicate configuration key %q", field.Key)
			}
			keys[field.Key] = struct{}{}
			for key := range keys {
				if strings.HasPrefix(key, field.Key+".") || strings.HasPrefix(field.Key, key+".") ||
					(key != field.Key && strings.ReplaceAll(key, ".", "__") == strings.ReplaceAll(field.Key, ".", "__")) {
					return nil, fmt.Errorf("ambiguous configuration keys %q and %q", key, field.Key)
				}
			}
			result = append(result, field)
		}
	}
	return result, nil
}

func validate(field Field) error {
	if strings.TrimSpace(field.Key) == "" {
		return fmt.Errorf("configuration key must not be empty")
	}
	if !keyPattern.MatchString(field.Key) {
		return fmt.Errorf("configuration key %q must use lowercase dot-separated identifiers", field.Key)
	}
	if strings.TrimSpace(field.Usage) == "" {
		return fmt.Errorf("configuration field %q must have usage text", field.Key)
	}
	switch field.Kind {
	case KindString:
		if _, ok := field.Default.(string); !ok {
			return invalidDefaultType(field, "string")
		}
	case KindUint16:
		if _, ok := field.Default.(uint16); !ok {
			return invalidDefaultType(field, "uint16")
		}
	case KindDuration:
		if _, ok := field.Default.(time.Duration); !ok {
			return invalidDefaultType(field, "time.Duration")
		}
	case KindStringSlice:
		if _, ok := field.Default.([]string); !ok {
			return invalidDefaultType(field, "[]string")
		}
	case KindInt:
		if _, ok := field.Default.(int); !ok {
			return invalidDefaultType(field, "int")
		}
	default:
		return fmt.Errorf("configuration field %q has unsupported kind %d", field.Key, field.Kind)
	}
	return nil
}

func invalidDefaultType(field Field, expected string) error {
	return fmt.Errorf("configuration field %q default must have type %s", field.Key, expected)
}
