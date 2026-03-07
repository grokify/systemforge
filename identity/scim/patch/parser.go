package patch

import (
	"fmt"
	"strings"
)

// Path represents a parsed PATCH path.
type Path struct {
	// Attribute is the main attribute name.
	Attribute string

	// SubAttribute is the optional sub-attribute name.
	SubAttribute string

	// Filter is the optional filter for multi-valued attributes (e.g., "type eq 'work'").
	Filter string

	// URIPrefix is the optional schema URI prefix.
	URIPrefix string
}

// String returns the string representation of the path.
func (p *Path) String() string {
	var sb strings.Builder

	if p.URIPrefix != "" {
		sb.WriteString(p.URIPrefix)
		sb.WriteString(":")
	}

	sb.WriteString(p.Attribute)

	if p.Filter != "" {
		sb.WriteString("[")
		sb.WriteString(p.Filter)
		sb.WriteString("]")
	}

	if p.SubAttribute != "" {
		sb.WriteString(".")
		sb.WriteString(p.SubAttribute)
	}

	return sb.String()
}

// IsMultiValued returns true if the path has a filter for multi-valued attributes.
func (p *Path) IsMultiValued() bool {
	return p.Filter != ""
}

// ParsePath parses a SCIM PATCH path string.
// Supported formats:
//   - "attribute"
//   - "attribute.subAttribute"
//   - "attribute[filter]"
//   - "attribute[filter].subAttribute"
//   - "urn:schema:attribute"
//   - "urn:schema:attribute.subAttribute"
func ParsePath(path string) (*Path, error) {
	if path == "" {
		return nil, nil
	}

	p := &Path{}

	// Check for URI prefix
	if strings.HasPrefix(path, "urn:") {
		// Find the last colon that separates URI from attribute
		lastColon := strings.LastIndex(path, ":")
		if lastColon > 0 {
			p.URIPrefix = path[:lastColon]
			path = path[lastColon+1:]
		}
	}

	// Check for filter
	bracketStart := strings.Index(path, "[")
	bracketEnd := strings.Index(path, "]")

	if bracketStart > 0 && bracketEnd > bracketStart {
		p.Attribute = path[:bracketStart]
		p.Filter = path[bracketStart+1 : bracketEnd]

		// Check for sub-attribute after filter
		remainder := path[bracketEnd+1:]
		if strings.HasPrefix(remainder, ".") {
			p.SubAttribute = remainder[1:]
		}
	} else {
		// No filter, check for sub-attribute
		dotIndex := strings.Index(path, ".")
		if dotIndex > 0 {
			p.Attribute = path[:dotIndex]
			p.SubAttribute = path[dotIndex+1:]
		} else {
			p.Attribute = path
		}
	}

	return p, nil
}

// TargetSelector represents a selector for multi-valued attribute targets.
type TargetSelector struct {
	// All indicates all values should be affected.
	All bool

	// Index is a specific index in the array (0-based, -1 if not specified).
	Index int

	// FilterAttr is the attribute to match.
	FilterAttr string

	// FilterOp is the filter operator.
	FilterOp string

	// FilterValue is the value to match.
	FilterValue any
}

// ParseFilter parses a filter expression from a path.
// Supported formats:
//   - "type eq 'work'"
//   - "value eq 'xxx'"
//   - "primary eq true"
func ParseFilter(filter string) (*TargetSelector, error) {
	if filter == "" {
		return &TargetSelector{All: true}, nil
	}

	filter = strings.TrimSpace(filter)

	// Split into parts: attribute, operator, value
	parts := strings.Fields(filter)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid filter: %s", filter)
	}

	selector := &TargetSelector{
		Index:      -1,
		FilterAttr: parts[0],
		FilterOp:   strings.ToLower(parts[1]),
	}

	// Parse the value
	valueStr := strings.Join(parts[2:], " ")
	valueStr = strings.TrimSpace(valueStr)

	// Handle quoted strings
	if strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'") {
		selector.FilterValue = valueStr[1 : len(valueStr)-1]
	} else if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {
		selector.FilterValue = valueStr[1 : len(valueStr)-1]
	} else if valueStr == "true" {
		selector.FilterValue = true
	} else if valueStr == "false" {
		selector.FilterValue = false
	} else if valueStr == "null" {
		selector.FilterValue = nil
	} else {
		selector.FilterValue = valueStr
	}

	return selector, nil
}

// Matches returns true if the selector matches the given value.
func (s *TargetSelector) Matches(value map[string]any) bool {
	if s.All {
		return true
	}

	if s.FilterAttr == "" {
		return false
	}

	attrValue, ok := value[s.FilterAttr]
	if !ok {
		// Try case-insensitive lookup
		for k, v := range value {
			if strings.EqualFold(k, s.FilterAttr) {
				attrValue = v
				ok = true
				break
			}
		}
	}

	if !ok {
		return false
	}

	switch s.FilterOp {
	case "eq":
		return compareValues(attrValue, s.FilterValue)
	case "ne":
		return !compareValues(attrValue, s.FilterValue)
	case "co":
		aStr, aOk := attrValue.(string)
		bStr, bOk := s.FilterValue.(string)
		if aOk && bOk {
			return strings.Contains(strings.ToLower(aStr), strings.ToLower(bStr))
		}
		return false
	case "sw":
		aStr, aOk := attrValue.(string)
		bStr, bOk := s.FilterValue.(string)
		if aOk && bOk {
			return strings.HasPrefix(strings.ToLower(aStr), strings.ToLower(bStr))
		}
		return false
	case "ew":
		aStr, aOk := attrValue.(string)
		bStr, bOk := s.FilterValue.(string)
		if aOk && bOk {
			return strings.HasSuffix(strings.ToLower(aStr), strings.ToLower(bStr))
		}
		return false
	default:
		return false
	}
}

// compareValues compares two values for equality.
func compareValues(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// String comparison (case-insensitive)
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	if aOk && bOk {
		return strings.EqualFold(aStr, bStr)
	}

	// Boolean comparison
	aBool, aOk := a.(bool)
	bBool, bOk := b.(bool)
	if aOk && bOk {
		return aBool == bBool
	}

	// Numeric comparison
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if aOk && bOk {
		return aNum == bNum
	}

	// Fall back to direct comparison
	return a == b
}

// toFloat converts a value to float64 if possible.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
