// CRC: crc-PathNavigator.md
// Spec: protocol.md, libraries.md
package uiclient

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// PathNavigator handles path resolution for variable bindings.
type PathNavigator struct {
	pathCache       map[string][]Segment
	standardVars    map[string]interface{}
}

// Segment represents a path segment.
type Segment struct {
	Type  string // "property", "index", "method", "parent"
	Value string
	Index int
}

var (
	methodRegex = regexp.MustCompile(`^(\w+)\(\)$`)
	indexRegex  = regexp.MustCompile(`^[1-9]\d*$`)
)

// NewPathNavigator creates a new path navigator.
func NewPathNavigator() *PathNavigator {
	return &PathNavigator{
		pathCache:    make(map[string][]Segment),
		standardVars: make(map[string]interface{}),
	}
}

// RegisterStandardVariable registers a @name variable.
func (n *PathNavigator) RegisterStandardVariable(name string, value interface{}) {
	n.standardVars[name] = value
}

// ParsePath splits a path string into segments.
func (n *PathNavigator) ParsePath(path string) []Segment {
	if cached, ok := n.pathCache[path]; ok {
		return cached
	}

	parts := strings.Split(path, ".")
	segments := make([]Segment, 0, len(parts))

	for i, part := range parts {
		if part == "" {
			continue
		}

		seg := Segment{Value: part}

		// Check for @name at start
		if i == 0 && strings.HasPrefix(part, "@") {
			seg.Type = "standard"
			seg.Value = strings.TrimPrefix(part, "@")
			segments = append(segments, seg)
			continue
		}

		// Check for parent traversal
		if part == ".." {
			seg.Type = "parent"
			segments = append(segments, seg)
			continue
		}

		// Check for method call
		if match := methodRegex.FindStringSubmatch(part); match != nil {
			seg.Type = "method"
			seg.Value = match[1]
			segments = append(segments, seg)
			continue
		}

		// Check for array index
		if indexRegex.MatchString(part) {
			idx, _ := strconv.Atoi(part)
			seg.Type = "index"
			seg.Index = idx
			segments = append(segments, seg)
			continue
		}

		// Default: property access
		seg.Type = "property"
		segments = append(segments, seg)
	}

	n.pathCache[path] = segments
	return segments
}

// Resolve navigates a path to get a value.
func (n *PathNavigator) Resolve(root interface{}, path string) (interface{}, error) {
	segments := n.ParsePath(path)
	if len(segments) == 0 {
		return root, nil
	}

	current := root

	for _, seg := range segments {
		if current == nil {
			return nil, fmt.Errorf("cannot navigate nil value at %s", seg.Value)
		}

		var err error
		current, err = n.navigateSegment(current, seg)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

// navigateSegment handles a single path segment.
func (n *PathNavigator) navigateSegment(current interface{}, seg Segment) (interface{}, error) {
	switch seg.Type {
	case "standard":
		if val, ok := n.standardVars[seg.Value]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("standard variable @%s not found", seg.Value)

	case "property":
		return n.getProperty(current, seg.Value)

	case "index":
		return n.getArrayIndex(current, seg.Index)

	case "method":
		return n.callMethod(current, seg.Value)

	case "parent":
		return nil, fmt.Errorf("parent traversal requires context")

	default:
		return nil, fmt.Errorf("unknown segment type: %s", seg.Type)
	}
}

// getProperty gets a property from an object.
func (n *PathNavigator) getProperty(obj interface{}, name string) (interface{}, error) {
	// Handle map
	if m, ok := obj.(map[string]interface{}); ok {
		if val, ok := m[name]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("property %s not found", name)
	}

	// Handle struct via reflection
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		field := v.FieldByName(name)
		if field.IsValid() && field.CanInterface() {
			return field.Interface(), nil
		}
		// Try method
		method := reflect.ValueOf(obj).MethodByName(name)
		if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() >= 1 {
			results := method.Call(nil)
			return results[0].Interface(), nil
		}
	}

	return nil, fmt.Errorf("cannot get property %s from %T", name, obj)
}

// getArrayIndex gets an element at 1-based index.
func (n *PathNavigator) getArrayIndex(obj interface{}, index int) (interface{}, error) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		// Convert 1-based to 0-based
		idx := index - 1
		if idx >= 0 && idx < v.Len() {
			return v.Index(idx).Interface(), nil
		}
		return nil, fmt.Errorf("index %d out of bounds", index)
	}
	return nil, fmt.Errorf("cannot index %T", obj)
}

// callMethod calls a method on an object.
func (n *PathNavigator) callMethod(obj interface{}, name string) (interface{}, error) {
	method := reflect.ValueOf(obj).MethodByName(name)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", name)
	}

	if method.Type().NumIn() != 0 {
		return nil, fmt.Errorf("method %s requires arguments", name)
	}

	if method.Type().NumOut() == 0 {
		method.Call(nil)
		return nil, nil
	}

	results := method.Call(nil)
	return results[0].Interface(), nil
}

// ResolveForWrite navigates and returns parent + key for setting.
func (n *PathNavigator) ResolveForWrite(root interface{}, path string) (parent interface{}, key string, index int, err error) {
	segments := n.ParsePath(path)
	if len(segments) == 0 {
		return nil, "", 0, fmt.Errorf("empty path")
	}

	if len(segments) == 1 {
		seg := segments[0]
		if seg.Type == "property" {
			return root, seg.Value, 0, nil
		}
		if seg.Type == "index" {
			return root, "", seg.Index, nil
		}
		return nil, "", 0, fmt.Errorf("cannot write to %s", seg.Type)
	}

	// Navigate all but last segment
	parentPath := make([]Segment, len(segments)-1)
	copy(parentPath, segments[:len(segments)-1])

	current := root
	for _, seg := range parentPath {
		current, err = n.navigateSegment(current, seg)
		if err != nil {
			return nil, "", 0, err
		}
	}

	lastSeg := segments[len(segments)-1]
	if lastSeg.Type == "property" {
		return current, lastSeg.Value, 0, nil
	}
	if lastSeg.Type == "index" {
		return current, "", lastSeg.Index, nil
	}

	return nil, "", 0, fmt.Errorf("cannot write to %s", lastSeg.Type)
}
