// Пакет schema определяет независимое от загрузчика описание конфигурации.
package schema

import (
	"slices"
	"time"
)

// Kind определяет внешний тип конфигурационного параметра.
type Kind uint8

const (
	// KindString обозначает строковый параметр.
	KindString Kind = iota + 1
	// KindUint16 обозначает беззнаковый 16-битный параметр.
	KindUint16
	// KindDuration обозначает продолжительность Go.
	KindDuration
	// KindStringSlice обозначает список строк.
	KindStringSlice
	// KindInt обозначает целочисленный параметр.
	KindInt
)

// Field описывает внешний ключ, его тип, значение по умолчанию и CLI-справку.
type Field struct {
	Key     string
	Kind    Kind
	Default any
	Usage   string
}

// String создаёт описание строкового параметра.
func String(key, defaultValue, usage string) Field {
	return Field{Key: key, Kind: KindString, Default: defaultValue, Usage: usage}
}

// Uint16 создаёт описание 16-битного беззнакового параметра.
func Uint16(key string, defaultValue uint16, usage string) Field {
	return Field{Key: key, Kind: KindUint16, Default: defaultValue, Usage: usage}
}

// Duration создаёт описание параметра продолжительности.
func Duration(key string, defaultValue time.Duration, usage string) Field {
	return Field{Key: key, Kind: KindDuration, Default: defaultValue, Usage: usage}
}

// StringSlice создаёт описание параметра со списком строк.
func StringSlice(key string, defaultValue []string, usage string) Field {
	return Field{Key: key, Kind: KindStringSlice, Default: slices.Clone(defaultValue), Usage: usage}
}

// Int создаёт описание целочисленного параметра.
func Int(key string, defaultValue int, usage string) Field {
	return Field{Key: key, Kind: KindInt, Default: defaultValue, Usage: usage}
}
