package common

import (
	"encoding/json"
	"fmt"
)

// UserAttributes is a map of user attributes. It supports both string and []string values for attributes.
// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
type UserAttributes map[string][]string

// UnmarshalJSON implements json.Unmarshaler.
func (a *UserAttributes) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if *a == nil {
		*a = make(UserAttributes, len(raw))
	}

	result := *a

	for k, v := range raw {
		switch value := v.(type) {
		case string:
			result[k] = []string{value}
		case []interface{}:
			var strSlice []string

			for _, item := range value {
				if str, ok := item.(string); ok {
					strSlice = append(strSlice, str)
				} else {
					return fmt.Errorf("attribute '%s' contains a non-string value in the list", k)
				}
			}

			result[k] = strSlice
		default:
			return fmt.Errorf("unsupported type for attribute '%s': %T", k, v)
		}
	}

	return nil
}
