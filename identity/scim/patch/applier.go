package patch

import (
	"fmt"
	"reflect"
	"strings"
)

// Applier applies PATCH operations to resources.
type Applier struct {
	// CaseSensitive controls whether attribute names are matched case-sensitively.
	CaseSensitive bool
}

// NewApplier creates a new patch applier.
func NewApplier() *Applier {
	return &Applier{
		CaseSensitive: false,
	}
}

// Apply applies a list of PATCH operations to a resource.
// The resource must be a pointer to a struct.
func (a *Applier) Apply(resource any, operations []Operation) error {
	rv := reflect.ValueOf(resource)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("resource must be a pointer to a struct")
	}

	for _, op := range operations {
		if err := op.Validate(); err != nil {
			return err
		}
		if err := a.applyOperation(rv.Elem(), op); err != nil {
			return err
		}
	}

	return nil
}

// applyOperation applies a single PATCH operation.
func (a *Applier) applyOperation(rv reflect.Value, op Operation) error {
	switch op.Op {
	case OpAdd:
		return a.applyAdd(rv, op.Path, op.Value)
	case OpReplace:
		return a.applyReplace(rv, op.Path, op.Value)
	case OpRemove:
		return a.applyRemove(rv, op.Path, op.Value)
	default:
		return fmt.Errorf("unsupported operation: %s", op.Op)
	}
}

// applyAdd applies an add operation.
func (a *Applier) applyAdd(rv reflect.Value, path string, value any) error {
	if path == "" {
		// Add to the resource itself (merge attributes)
		return a.mergeValue(rv, value)
	}

	parsedPath, err := ParsePath(path)
	if err != nil {
		return err
	}

	field, err := a.findField(rv, parsedPath.Attribute)
	if err != nil {
		return err
	}

	if parsedPath.Filter != "" {
		// Add to multi-valued attribute with filter
		return a.addToMultiValuedWithFilter(field, parsedPath, value)
	}

	if parsedPath.SubAttribute != "" {
		// Add to sub-attribute
		return a.setSubAttribute(field, parsedPath.SubAttribute, value)
	}

	// Simple add to single-valued or append to multi-valued
	if field.Kind() == reflect.Slice {
		return a.appendToSlice(field, value)
	}

	return a.setValue(field, value)
}

// applyReplace applies a replace operation.
func (a *Applier) applyReplace(rv reflect.Value, path string, value any) error {
	if path == "" {
		// Replace the entire resource
		return a.mergeValue(rv, value)
	}

	parsedPath, err := ParsePath(path)
	if err != nil {
		return err
	}

	field, err := a.findField(rv, parsedPath.Attribute)
	if err != nil {
		return err
	}

	if parsedPath.Filter != "" {
		// Replace in multi-valued attribute with filter
		return a.replaceInMultiValuedWithFilter(field, parsedPath, value)
	}

	if parsedPath.SubAttribute != "" {
		// Replace sub-attribute
		return a.setSubAttribute(field, parsedPath.SubAttribute, value)
	}

	return a.setValue(field, value)
}

// applyRemove applies a remove operation.
func (a *Applier) applyRemove(rv reflect.Value, path string, _ any) error {
	if path == "" {
		return fmt.Errorf("remove operation requires a path")
	}

	parsedPath, err := ParsePath(path)
	if err != nil {
		return err
	}

	field, err := a.findField(rv, parsedPath.Attribute)
	if err != nil {
		return err
	}

	if parsedPath.Filter != "" {
		// Remove from multi-valued attribute with filter
		return a.removeFromMultiValuedWithFilter(field, parsedPath)
	}

	if parsedPath.SubAttribute != "" {
		// Remove sub-attribute
		return a.clearSubAttribute(field, parsedPath.SubAttribute)
	}

	// Clear the entire attribute
	field.Set(reflect.Zero(field.Type()))
	return nil
}

// findField finds a field in a struct by name.
func (a *Applier) findField(rv reflect.Value, name string) (reflect.Value, error) {
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("cannot find field in non-struct")
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// Check JSON tag first
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			jsonName := strings.Split(jsonTag, ",")[0]
			if a.matchFieldName(jsonName, name) {
				return rv.Field(i), nil
			}
		}

		// Fall back to field name
		if a.matchFieldName(field.Name, name) {
			return rv.Field(i), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("field not found: %s", name)
}

// matchFieldName checks if two field names match.
func (a *Applier) matchFieldName(a1, b string) bool {
	if a.CaseSensitive {
		return a1 == b
	}
	return strings.EqualFold(a1, b)
}

// setValue sets a field value.
func (a *Applier) setValue(field reflect.Value, value any) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set field")
	}

	if value == nil {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	valueRV := reflect.ValueOf(value)

	// Handle pointer fields
	if field.Kind() == reflect.Pointer {
		if valueRV.Type().AssignableTo(field.Type().Elem()) {
			newPtr := reflect.New(field.Type().Elem())
			newPtr.Elem().Set(valueRV)
			field.Set(newPtr)
			return nil
		}
		if valueRV.Type().ConvertibleTo(field.Type().Elem()) {
			newPtr := reflect.New(field.Type().Elem())
			newPtr.Elem().Set(valueRV.Convert(field.Type().Elem()))
			field.Set(newPtr)
			return nil
		}
	}

	// Direct assignment
	if valueRV.Type().AssignableTo(field.Type()) {
		field.Set(valueRV)
		return nil
	}

	// Try conversion
	if valueRV.Type().ConvertibleTo(field.Type()) {
		field.Set(valueRV.Convert(field.Type()))
		return nil
	}

	return fmt.Errorf("cannot convert %T to %s", value, field.Type())
}

// mergeValue merges a value (map) into a struct.
func (a *Applier) mergeValue(rv reflect.Value, value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map for merge, got %T", value)
	}

	for k, v := range m {
		field, err := a.findField(rv, k)
		if err != nil {
			continue // Skip unknown fields
		}
		if err := a.setValue(field, v); err != nil {
			return err
		}
	}

	return nil
}

// appendToSlice appends value(s) to a slice field.
func (a *Applier) appendToSlice(field reflect.Value, value any) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set slice field")
	}

	valueRV := reflect.ValueOf(value)

	// If value is a slice, append all elements
	if valueRV.Kind() == reflect.Slice {
		for i := 0; i < valueRV.Len(); i++ {
			elem := valueRV.Index(i)
			field.Set(reflect.Append(field, elem))
		}
		return nil
	}

	// Single value
	if valueRV.Type().AssignableTo(field.Type().Elem()) {
		field.Set(reflect.Append(field, valueRV))
		return nil
	}

	return fmt.Errorf("cannot append %T to slice of %s", value, field.Type().Elem())
}

// setSubAttribute sets a sub-attribute value.
func (a *Applier) setSubAttribute(field reflect.Value, subAttr string, value any) error {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	if field.Kind() != reflect.Struct {
		return fmt.Errorf("cannot set sub-attribute on non-struct")
	}

	subField, err := a.findField(field, subAttr)
	if err != nil {
		return err
	}

	return a.setValue(subField, value)
}

// clearSubAttribute clears a sub-attribute.
func (a *Applier) clearSubAttribute(field reflect.Value, subAttr string) error {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil // Already nil, nothing to clear
		}
		field = field.Elem()
	}

	if field.Kind() != reflect.Struct {
		return fmt.Errorf("cannot clear sub-attribute on non-struct")
	}

	subField, err := a.findField(field, subAttr)
	if err != nil {
		return err
	}

	subField.Set(reflect.Zero(subField.Type()))
	return nil
}

// addToMultiValuedWithFilter adds to a multi-valued attribute that matches a filter.
func (a *Applier) addToMultiValuedWithFilter(field reflect.Value, path *Path, value any) error {
	if field.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice for multi-valued attribute")
	}

	selector, err := ParseFilter(path.Filter)
	if err != nil {
		return err
	}

	valueRV := reflect.ValueOf(value)

	// Find matching element and update it
	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)
		if a.matchesSelector(elem, selector) {
			if path.SubAttribute != "" {
				return a.setSubAttribute(elem.Addr(), path.SubAttribute, value)
			}
			// Merge value into element
			if elem.Kind() == reflect.Struct || (elem.Kind() == reflect.Pointer && elem.Elem().Kind() == reflect.Struct) {
				return a.mergeValue(elem, value)
			}
			return a.setValue(elem, value)
		}
	}

	// No match found, append new element
	if valueRV.Type().AssignableTo(field.Type().Elem()) {
		field.Set(reflect.Append(field, valueRV))
		return nil
	}

	return nil
}

// replaceInMultiValuedWithFilter replaces in a multi-valued attribute that matches a filter.
func (a *Applier) replaceInMultiValuedWithFilter(field reflect.Value, path *Path, value any) error {
	if field.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice for multi-valued attribute")
	}

	selector, err := ParseFilter(path.Filter)
	if err != nil {
		return err
	}

	// Find matching element and replace it
	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)
		if a.matchesSelector(elem, selector) {
			if path.SubAttribute != "" {
				return a.setSubAttribute(elem.Addr(), path.SubAttribute, value)
			}
			return a.setValue(elem, value)
		}
	}

	return fmt.Errorf("no target found for filter: %s", path.Filter)
}

// removeFromMultiValuedWithFilter removes from a multi-valued attribute that matches a filter.
func (a *Applier) removeFromMultiValuedWithFilter(field reflect.Value, path *Path) error {
	if field.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice for multi-valued attribute")
	}

	selector, err := ParseFilter(path.Filter)
	if err != nil {
		return err
	}

	// Build new slice without matching elements
	newSlice := reflect.MakeSlice(field.Type(), 0, field.Len())
	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)
		if !a.matchesSelector(elem, selector) {
			newSlice = reflect.Append(newSlice, elem)
		} else if path.SubAttribute != "" {
			// Only clear the sub-attribute
			if err := a.clearSubAttribute(elem.Addr(), path.SubAttribute); err != nil {
				return err
			}
			newSlice = reflect.Append(newSlice, elem)
		}
	}

	field.Set(newSlice)
	return nil
}

// matchesSelector checks if an element matches a selector.
func (a *Applier) matchesSelector(elem reflect.Value, selector *TargetSelector) bool {
	if selector.All {
		return true
	}

	if elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return false
		}
		elem = elem.Elem()
	}

	// Convert to map for selector matching
	m := make(map[string]any)
	if elem.Kind() == reflect.Struct {
		rt := elem.Type()
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			jsonTag := field.Tag.Get("json")
			name := field.Name
			if jsonTag != "" {
				name = strings.Split(jsonTag, ",")[0]
			}
			m[name] = elem.Field(i).Interface()
		}
	}

	return selector.Matches(m)
}
