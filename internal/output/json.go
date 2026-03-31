package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

// PrintJSON writes compact JSON to w.
func PrintJSON(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

// PrintPrettyJSON writes indented JSON to w.
func PrintPrettyJSON(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

// FilterJQ applies a jq expression to data and returns the result.
func FilterJQ(data interface{}, expr string) (interface{}, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression: %w", err)
	}

	// gojq requires the input to be an interface{} that is either
	// a map[string]interface{}, []interface{}, or a primitive. We
	// round-trip through JSON to normalise Go structs.
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshalling data for jq: %w", err)
	}
	var normalised interface{}
	if err := json.Unmarshal(raw, &normalised); err != nil {
		return nil, fmt.Errorf("unmarshalling data for jq: %w", err)
	}

	iter := query.Run(normalised)

	var results []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("jq evaluation error: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 0 {
		return []interface{}{}, nil
	}
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}
