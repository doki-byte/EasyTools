package testsuite

import (
	"fmt"
	"math"
	"time"

	"EasyTools/app/connect/ssh/toml"
)

// addTag adds JSON tags to a data structure as expected by toml-test.
func addTag(key string, tomlData any) any {
	// Switch on the data type.
	switch orig := tomlData.(type) {
	default:
		//return map[string]interface{}{}
		panic(fmt.Sprintf("Unknown type: %T", tomlData))

	// A table: we don't need to add any tags, just recurse for every table
	// entry.
	case map[string]any:
		typed := make(map[string]any, len(orig))
		for k, v := range orig {
			typed[k] = addTag(k, v)
		}
		return typed

	// An array: we don't need to add any tags, just recurse for every table
	// entry.
	case []map[string]any:
		typed := make([]map[string]any, len(orig))
		for i, v := range orig {
			typed[i] = addTag("", v).(map[string]any)
		}
		return typed
	case []any:
		typed := make([]any, len(orig))
		for i, v := range orig {
			typed[i] = addTag("", v)
		}
		return typed

	// Datetime: tag as datetime.
	case toml.LocalTime:
		return tag("time-local", orig.String())
	case toml.LocalDate:
		return tag("date-local", orig.String())
	case toml.LocalDateTime:
		return tag("datetime-local", orig.String())
	case time.Time:
		return tag("datetime", orig.Format("2006-01-02T15:04:05.999999999Z07:00"))

	// Tag primitive values: bool, string, int, and float64.
	case bool:
		return tag("bool", fmt.Sprintf("%v", orig))
	case string:
		return tag("string", orig)
	case int64:
		return tag("integer", fmt.Sprintf("%d", orig))
	case float64:
		// Special case for nan since NaN == NaN is false.
		if math.IsNaN(orig) {
			return tag("float", "nan")
		}
		return tag("float", fmt.Sprintf("%v", orig))
	}
}

func tag(typeName string, data any) map[string]any {
	return map[string]any{
		"type":  typeName,
		"value": data,
	}
}
