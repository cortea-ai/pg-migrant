package config

import (
	"fmt"
	"strings"
)

// Vars represents a map of variable key-value pairs
type Vars map[string]string

// String returns a string representation of Vars
func (v *Vars) String() string {
	if v == nil {
		return ""
	}
	pairs := make([]string, 0, len(*v))
	for k, val := range *v {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, val))
	}
	return strings.Join(pairs, ", ")
}

// Set implements the pflag.Value interface
func (v *Vars) Set(value string) error {
	if *v == nil {
		*v = make(map[string]string)
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid variable format, expected key=value, got %s", value)
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return fmt.Errorf("empty key is not allowed")
	}
	(*v)[key] = parts[1]
	return nil
}

// Type implements the pflag.Value interface
func (v *Vars) Type() string {
	return "key=value"
}
