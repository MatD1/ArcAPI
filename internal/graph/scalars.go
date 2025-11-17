package graph

import (
	"encoding/json"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalTime converts time.Time to GraphQL Time scalar
func MarshalTime(t time.Time) graphql.Marshaler {
	return graphql.MarshalTime(t)
}

// UnmarshalTime converts GraphQL Time scalar to time.Time
func UnmarshalTime(v interface{}) (time.Time, error) {
	return graphql.UnmarshalTime(v)
}

// MarshalJSON converts map[string]interface{} to GraphQL JSON scalar
// Note: This is a simplified implementation. After code generation, gqlgen may require
// a different signature. Check the generated code and adjust if needed.
func MarshalJSON(v map[string]interface{}) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	data, err := json.Marshal(v)
	if err != nil {
		return graphql.Null
	}
	// Marshal as JSON string - gqlgen will serialize this properly
	return graphql.MarshalString(string(data))
}

// UnmarshalJSON converts GraphQL JSON scalar to map[string]interface{}
func UnmarshalJSON(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return nil, nil
	}
	
	// If it's already a map, return it
	if m, ok := v.(map[string]interface{}); ok {
		return m, nil
	}
	
	// If it's a byte slice, unmarshal it
	if b, ok := v.([]byte); ok {
		var result map[string]interface{}
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	
	// Otherwise, marshal and unmarshal to convert
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

