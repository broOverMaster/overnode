package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

// resolvePathFields преобразует все строковые поля с суффиксом _path в абсолютные пути.
func resolvePathFields(configuration any, configFile string) error {
	baseDirectory, err := pathBaseDirectory(configFile)
	if err != nil {
		return err
	}
	return resolvePathValue(reflect.ValueOf(configuration), baseDirectory, "")
}

// pathBaseDirectory возвращает каталог конфигурационного файла или текущий рабочий каталог.
func pathBaseDirectory(configFile string) (string, error) {
	if configFile == "" {
		workingDirectory, err := filepath.Abs(".")
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
		return workingDirectory, nil
	}

	absoluteConfigFile, err := filepath.Abs(configFile)
	if err != nil {
		return "", fmt.Errorf("resolve configuration file path: %w", err)
	}
	return filepath.Dir(absoluteConfigFile), nil
}

func resolvePathValue(value reflect.Value, baseDirectory, parentKey string) error {
	return resolvePathSeen(value, baseDirectory, parentKey, make(map[uintptr]bool), 0)
}

func resolvePathSeen(value reflect.Value, baseDirectory, parentKey string, seen map[uintptr]bool, depth int) error {
	if depth > 128 {
		return fmt.Errorf("configuration paths exceed maximum nesting depth at %s", parentKey)
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if seen[value.Pointer()] {
			return nil
		}
		seen[value.Pointer()] = true
		return resolvePathSeen(value.Elem(), baseDirectory, parentKey, seen, depth+1)
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return nil
		}
		copyValue := reflect.New(value.Elem().Type()).Elem()
		copyValue.Set(value.Elem())
		if err := resolvePathSeen(copyValue, baseDirectory, parentKey, seen, depth+1); err != nil {
			return err
		}
		if value.CanSet() {
			value.Set(copyValue)
		}
		return nil
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := resolvePathSeen(value.Index(index), baseDirectory, fmt.Sprintf("%s[%d]", parentKey, index), seen, depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		iterator := value.MapRange()
		for iterator.Next() {
			copyValue := reflect.New(iterator.Value().Type()).Elem()
			copyValue.Set(iterator.Value())
			if err := resolvePathSeen(copyValue, baseDirectory, parentKey, seen, depth+1); err != nil {
				return err
			}
			value.SetMapIndex(iterator.Key(), copyValue)
		}
		return nil
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		definition := valueType.Field(index)
		if !definition.IsExported() {
			continue
		}
		key := strings.Split(definition.Tag.Get("mapstructure"), ",")[0]
		if key == "-" {
			continue
		}
		fullKey := key
		if parentKey != "" && key != "" {
			fullKey = parentKey + "." + key
		}

		if field.Kind() == reflect.String && strings.HasSuffix(key, "_path") {
			if !field.CanSet() || field.String() == "" {
				continue
			}
			path := field.String()
			if !filepath.IsAbs(path) {
				path = filepath.Join(baseDirectory, path)
			}
			absolute, err := filepath.Abs(filepath.Clean(path))
			if err != nil {
				return fmt.Errorf("resolve %s: %w", fullKey, err)
			}
			field.SetString(absolute)
			continue
		}

		if err := resolvePathSeen(field, baseDirectory, fullKey, seen, depth+1); err != nil {
			return err
		}
	}
	return nil
}
