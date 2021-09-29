package environ

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// GetEnv function to read an environment or return a default value
func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}

	return defaultVal
}

//GetEnvAsInt function to read an environment variable into integer or return a default value
func GetEnvAsInt(name string, defaultVal int) int {
	valueStr := GetEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

//GetEnvAsInt64 function to read an environment variable into int64 or return a default value
func GetEnvAsInt64(name string, defaultVal int64) int64 {
	valueStr := GetEnv(name, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}

	return defaultVal
}

//GetEnvAsFloat32 function to read an environment variable into float32 or return a default value
func GetEnvAsFloat32(name string, defaultVal float32) float32 {
	valueStr := GetEnv(name, "")
	if value, err := strconv.ParseFloat(valueStr, 32); err == nil {
		return float32(value)
	}

	return defaultVal
}

//GetEnvAsFloat64 function to read an environment variable into float64 or return a default value
func GetEnvAsFloat64(name string, defaultVal float64) float64 {
	valueStr := GetEnv(name, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}

	return defaultVal
}

//GetEnvAsBool function to read an environment variable into a bool or return default value
func GetEnvAsBool(name string, defaultVal bool) bool {
	valStr := GetEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultVal
}

//GetEnvAsSlice to read an environment variable into a string slice or return default value
func GetEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valStr := GetEnv(name, "")

	if valStr == "" {
		return defaultVal
	}

	val := strings.Split(valStr, sep)

	return val
}

// SetEnvInt function set integer environment variable
func SetEnvInt(key string, value int) {
	os.Setenv(key, fmt.Sprintf("%d", value))
}

// SetEnvInt32 function set int32 environment variable
func SetEnvInt32(key string, value int32) {
	os.Setenv(key, fmt.Sprintf("%d", value))
}

// SetEnvInt64 function set int64 environment variable
func SetEnvInt64(key string, value int64) {
	os.Setenv(key, fmt.Sprintf("%d", value))
}

// SetEnvFloat32 function set float32 environment variable
func SetEnvFloat32(key string, value float32) {
	os.Setenv(key, fmt.Sprintf("%f", value))
}

// SetEnvFloat64 function set float64 environment variable
func SetEnvFloat64(key string, value float64) {
	os.Setenv(key, fmt.Sprintf("%f", value))
}

// SetEnvBool function set boolean environment variable
func SetEnvBool(key string, value bool) {
	os.Setenv(key, fmt.Sprintf("%t", value))
}

// SetEnvAsSlice function set a string slice environment variable
func SetEnvAsSlice(key string, value []string, sep string) {
	if value == nil {
		os.Setenv(key, "")
		return
	}
	var setenv string
	for _, v := range value {
		setenv += fmt.Sprintf("%s%s", v, sep)
	}
	os.Setenv(key, strings.TrimRight(setenv, ","))
}

// SetEnv ...
func SetEnv(key string, value interface{}) {
	switch value.(type) {
	case int:
		os.Setenv(key, fmt.Sprintf("%d", value.(int)))
	case int32:
		os.Setenv(key, fmt.Sprintf("%d", value.(int32)))
	case int64:
		os.Setenv(key, fmt.Sprintf("%d", value.(int64)))
	case float32:
		os.Setenv(key, fmt.Sprintf("%f", value.(float32)))
	case float64:
		os.Setenv(key, fmt.Sprintf("%f", value.(float64)))
	case bool:
		os.Setenv(key, fmt.Sprintf("%t", value.(bool)))
	}
}
