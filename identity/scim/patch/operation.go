// Package patch provides SCIM PATCH operation handling.
package patch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OperationType represents the type of a PATCH operation.
type OperationType string

const (
	OpAdd     OperationType = "add"
	OpRemove  OperationType = "remove"
	OpReplace OperationType = "replace"
)

// Operation represents a single SCIM PATCH operation.
type Operation struct {
	Op    OperationType `json:"op"`
	Path  string        `json:"path,omitempty"`
	Value any           `json:"value,omitempty"`
}

// Request represents a SCIM PATCH request.
type Request struct {
	Schemas    []string    `json:"schemas"`
	Operations []Operation `json:"Operations"`
}

// Validate validates a PATCH operation.
func (op *Operation) Validate() error {
	switch op.Op {
	case OpAdd:
		if op.Value == nil {
			return fmt.Errorf("add operation requires a value")
		}
	case OpReplace:
		if op.Value == nil {
			return fmt.Errorf("replace operation requires a value")
		}
	case OpRemove:
		// Remove can have a value for multi-valued attributes
	default:
		return fmt.Errorf("invalid operation: %s", op.Op)
	}
	return nil
}

// ParseOperationType parses an operation type string.
func ParseOperationType(s string) (OperationType, error) {
	switch strings.ToLower(s) {
	case "add":
		return OpAdd, nil
	case "remove":
		return OpRemove, nil
	case "replace":
		return OpReplace, nil
	default:
		return "", fmt.Errorf("invalid operation type: %s", s)
	}
}

// UnmarshalJSON implements custom JSON unmarshaling for Operation.
func (op *Operation) UnmarshalJSON(data []byte) error {
	type operationAlias Operation
	aux := &struct {
		Op string `json:"op"`
		*operationAlias
	}{
		operationAlias: (*operationAlias)(op),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	opType, err := ParseOperationType(aux.Op)
	if err != nil {
		return err
	}
	op.Op = opType

	return nil
}

// MarshalJSON implements custom JSON marshaling for Operation.
func (op Operation) MarshalJSON() ([]byte, error) {
	type operationAlias Operation
	return json.Marshal(&struct {
		Op string `json:"op"`
		operationAlias
	}{
		Op:             string(op.Op),
		operationAlias: operationAlias(op),
	})
}
