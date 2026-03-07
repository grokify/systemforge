package filter

import (
	"reflect"
	"strings"
)

// Evaluator evaluates filter expressions against SCIM resources.
type Evaluator struct {
	// AttributeResolver resolves attribute values from a resource.
	AttributeResolver func(resource any, attrPath string) (any, bool)
}

// DefaultEvaluator returns an evaluator with a reflection-based attribute resolver.
func DefaultEvaluator() *Evaluator {
	return &Evaluator{
		AttributeResolver: reflectAttributeResolver,
	}
}

// Evaluate evaluates a filter expression against a resource.
func (e *Evaluator) Evaluate(node Node, resource any) bool {
	if node == nil {
		return true
	}

	switch n := node.(type) {
	case *ComparisonNode:
		return e.evaluateComparison(n, resource)
	case *LogicalAndNode:
		return e.Evaluate(n.Left, resource) && e.Evaluate(n.Right, resource)
	case *LogicalOrNode:
		return e.Evaluate(n.Left, resource) || e.Evaluate(n.Right, resource)
	case *LogicalNotNode:
		return !e.Evaluate(n.Operand, resource)
	case *ValuePathNode:
		return e.evaluateValuePath(n, resource)
	default:
		return false
	}
}

// evaluateComparison evaluates a comparison expression.
func (e *Evaluator) evaluateComparison(node *ComparisonNode, resource any) bool {
	value, found := e.AttributeResolver(resource, node.AttributePath)

	switch node.Operator {
	case OpPresent:
		return found && !isNilOrEmpty(value)

	case OpEqual:
		if !found {
			return node.Value == nil
		}
		return compareValues(value, node.Value, false) == 0

	case OpNotEqual:
		if !found {
			return node.Value != nil
		}
		return compareValues(value, node.Value, false) != 0

	case OpContains:
		if !found {
			return false
		}
		return stringContains(value, node.Value, false)

	case OpStartsWith:
		if !found {
			return false
		}
		return stringStartsWith(value, node.Value, false)

	case OpEndsWith:
		if !found {
			return false
		}
		return stringEndsWith(value, node.Value, false)

	case OpGreaterThan:
		if !found {
			return false
		}
		return compareValues(value, node.Value, false) > 0

	case OpGreaterThanOrEqual:
		if !found {
			return false
		}
		return compareValues(value, node.Value, false) >= 0

	case OpLessThan:
		if !found {
			return false
		}
		return compareValues(value, node.Value, false) < 0

	case OpLessThanOrEqual:
		if !found {
			return false
		}
		return compareValues(value, node.Value, false) <= 0

	default:
		return false
	}
}

// evaluateValuePath evaluates a value path expression against multi-valued attributes.
func (e *Evaluator) evaluateValuePath(node *ValuePathNode, resource any) bool {
	value, found := e.AttributeResolver(resource, node.AttributePath)
	if !found {
		return false
	}

	// Value must be a slice
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		// Single value, evaluate filter against it
		return e.Evaluate(node.Filter, value)
	}

	// Evaluate filter against each element
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i).Interface()
		if e.Evaluate(node.Filter, elem) {
			return true
		}
	}

	return false
}

// reflectAttributeResolver resolves attribute values using reflection.
func reflectAttributeResolver(resource any, attrPath string) (any, bool) {
	if resource == nil {
		return nil, false
	}

	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}

	parts := strings.Split(attrPath, ".")
	return resolvePathParts(rv, parts)
}

// resolvePathParts resolves a path through a struct using reflection.
func resolvePathParts(rv reflect.Value, parts []string) (any, bool) {
	if len(parts) == 0 {
		if !rv.IsValid() {
			return nil, false
		}
		return rv.Interface(), true
	}

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, false
	}

	// Try to find the field by name (case-insensitive)
	fieldName := parts[0]
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// Check JSON tag first
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			jsonName := strings.Split(jsonTag, ",")[0]
			if strings.EqualFold(jsonName, fieldName) {
				return resolvePathParts(rv.Field(i), parts[1:])
			}
		}

		// Fall back to field name
		if strings.EqualFold(field.Name, fieldName) {
			return resolvePathParts(rv.Field(i), parts[1:])
		}
	}

	return nil, false
}

// compareValues compares two values.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareValues(a, b any, caseSensitive bool) int { //nolint:unparam // caseSensitive kept for SCIM spec compliance
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// String comparison
	aStr, aIsStr := toString(a)
	bStr, bIsStr := toString(b)
	if aIsStr && bIsStr {
		if !caseSensitive {
			aStr = strings.ToLower(aStr)
			bStr = strings.ToLower(bStr)
		}
		return strings.Compare(aStr, bStr)
	}

	// Numeric comparison
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	// Boolean comparison
	aBool, aIsBool := a.(bool)
	bBool, bIsBool := b.(bool)
	if aIsBool && bIsBool {
		if aBool == bBool {
			return 0
		}
		if aBool {
			return 1
		}
		return -1
	}

	// Fall back to string comparison
	return strings.Compare(toString2(a), toString2(b))
}

// stringContains checks if a contains b.
func stringContains(a, b any, caseSensitive bool) bool {
	aStr := toString2(a)
	bStr := toString2(b)
	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}
	return strings.Contains(aStr, bStr)
}

// stringStartsWith checks if a starts with b.
func stringStartsWith(a, b any, caseSensitive bool) bool {
	aStr := toString2(a)
	bStr := toString2(b)
	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}
	return strings.HasPrefix(aStr, bStr)
}

// stringEndsWith checks if a ends with b.
func stringEndsWith(a, b any, caseSensitive bool) bool {
	aStr := toString2(a)
	bStr := toString2(b)
	if !caseSensitive {
		aStr = strings.ToLower(aStr)
		bStr = strings.ToLower(bStr)
	}
	return strings.HasSuffix(aStr, bStr)
}

// toString converts a value to a string if it is one.
func toString(v any) (string, bool) {
	if s, ok := v.(string); ok {
		return s, true
	}
	return "", false
}

// toString2 converts any value to its string representation.
func toString2(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	return rv.String()
}

// toFloat64 converts a value to float64 if it's numeric.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// isNilOrEmpty checks if a value is nil or empty.
func isNilOrEmpty(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		return rv.IsNil()
	case reflect.Slice, reflect.Map, reflect.String:
		return rv.Len() == 0
	default:
		return false
	}
}

// FilterResources filters a slice of resources using a filter expression.
func FilterResources[T any](resources []T, filterExpr string) ([]T, error) {
	if filterExpr == "" {
		return resources, nil
	}

	node, err := ParseFilter(filterExpr)
	if err != nil {
		return nil, err
	}

	evaluator := DefaultEvaluator()
	var result []T

	for _, resource := range resources {
		if evaluator.Evaluate(node, resource) {
			result = append(result, resource)
		}
	}

	return result, nil
}
