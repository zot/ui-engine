// CRC: crc-PathSyntax.md
// Spec: protocol.md
package path

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SegmentType identifies the type of path segment.
type SegmentType int

const (
	SegmentProperty    SegmentType = iota // Simple property: name
	SegmentIndex                          // Array index: 1, 2 (1-based)
	SegmentParent                         // Parent traversal: ..
	SegmentMethod                         // Method call: getName()
	SegmentStandard                       // Standard variable: @name
)

// Segment represents a single path segment.
type Segment struct {
	Type   SegmentType
	Value  string // property name, method name, or @name
	Index  int    // for SegmentIndex (1-based)
}

// Path represents a parsed variable path.
type Path struct {
	Segments     []Segment
	URLParams    url.Values // ?create=Type&prop=value parameters
	HasStandard  bool       // starts with @name
	StandardName string     // the @name without @
	Raw          string     // original path string
}

var (
	methodPattern   = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\(\)$`)
	standardPattern = regexp.MustCompile(`^@([a-zA-Z_][a-zA-Z0-9_]*)$`)
	indexPattern    = regexp.MustCompile(`^[1-9][0-9]*$`)
)

// Parse splits a path string into segments and parameters.
// Examples:
//   - "name" -> property access
//   - "father.name" -> property.property
//   - "@customers.2.name" -> standard.index.property
//   - "getName()" -> method call
//   - ".." -> parent traversal
//   - "path?create=Type" -> path with URL params
func Parse(pathStr string) (*Path, error) {
	p := &Path{
		Segments:  make([]Segment, 0),
		URLParams: make(url.Values),
		Raw:       pathStr,
	}

	if pathStr == "" {
		return p, nil
	}

	// Split off URL parameters
	pathPart := pathStr
	if idx := strings.Index(pathStr, "?"); idx != -1 {
		pathPart = pathStr[:idx]
		queryStr := pathStr[idx+1:]
		params, err := url.ParseQuery(queryStr)
		if err != nil {
			return nil, err
		}
		p.URLParams = params
	}

	if pathPart == "" {
		return p, nil
	}

	// Split on dots
	parts := strings.Split(pathPart, ".")

	for i, part := range parts {
		if part == "" {
			continue
		}

		seg := Segment{Value: part}

		// Check for standard variable (@name) - only valid at start
		if i == 0 && strings.HasPrefix(part, "@") {
			if match := standardPattern.FindStringSubmatch(part); match != nil {
				seg.Type = SegmentStandard
				seg.Value = match[1] // without @
				p.HasStandard = true
				p.StandardName = match[1]
				p.Segments = append(p.Segments, seg)
				continue
			}
		}

		// Check for parent traversal
		if part == ".." {
			seg.Type = SegmentParent
			p.Segments = append(p.Segments, seg)
			continue
		}

		// Check for method call
		if match := methodPattern.FindStringSubmatch(part); match != nil {
			seg.Type = SegmentMethod
			seg.Value = match[1]
			p.Segments = append(p.Segments, seg)
			continue
		}

		// Check for array index (1-based)
		if indexPattern.MatchString(part) {
			idx, _ := strconv.Atoi(part)
			seg.Type = SegmentIndex
			seg.Index = idx
			p.Segments = append(p.Segments, seg)
			continue
		}

		// Default: property access
		seg.Type = SegmentProperty
		p.Segments = append(p.Segments, seg)
	}

	return p, nil
}

// GetPropertyAccess returns the property name if this is a simple property segment.
func (s *Segment) GetPropertyAccess() (string, bool) {
	if s.Type == SegmentProperty {
		return s.Value, true
	}
	return "", false
}

// GetArrayIndex returns the 1-based index if this is an index segment.
func (s *Segment) GetArrayIndex() (int, bool) {
	if s.Type == SegmentIndex {
		return s.Index, true
	}
	return 0, false
}

// GetMethodCall returns the method name if this is a method call segment.
func (s *Segment) GetMethodCall() (string, bool) {
	if s.Type == SegmentMethod {
		return s.Value, true
	}
	return "", false
}

// IsParentTraversal returns true if this is a parent traversal segment.
func (s *Segment) IsParentTraversal() bool {
	return s.Type == SegmentParent
}

// GetStandardVariable returns the @name (without @) if present.
func (p *Path) GetStandardVariable() (string, bool) {
	if p.HasStandard {
		return p.StandardName, true
	}
	return "", false
}

// GetURLParams returns the parsed URL parameters.
func (p *Path) GetURLParams() url.Values {
	return p.URLParams
}

// HasURLParams returns true if the path has URL parameters.
func (p *Path) HasURLParams() bool {
	return len(p.URLParams) > 0
}

// String reconstructs the path string.
func (p *Path) String() string {
	if len(p.Segments) == 0 && len(p.URLParams) == 0 {
		return ""
	}

	var parts []string
	for _, seg := range p.Segments {
		switch seg.Type {
		case SegmentStandard:
			parts = append(parts, "@"+seg.Value)
		case SegmentParent:
			parts = append(parts, "..")
		case SegmentMethod:
			parts = append(parts, seg.Value+"()")
		case SegmentIndex:
			parts = append(parts, strconv.Itoa(seg.Index))
		case SegmentProperty:
			parts = append(parts, seg.Value)
		}
	}

	result := strings.Join(parts, ".")
	if len(p.URLParams) > 0 {
		result += "?" + p.URLParams.Encode()
	}
	return result
}

// IsEmpty returns true if the path has no segments.
func (p *Path) IsEmpty() bool {
	return len(p.Segments) == 0
}

// Len returns the number of segments.
func (p *Path) Len() int {
	return len(p.Segments)
}
