// Package viewdef provides viewdef management.
// CRC: crc-Viewdef.md
// Spec: viewdefs.md
package viewdef

import (
	"fmt"
	"strings"
)

// Viewdef represents a view definition template.
type Viewdef struct {
	Type      string // Type name (e.g., "Contact")
	Namespace string // Namespace (e.g., "DEFAULT", "COMPACT", "OPTION")
	Content   string // HTML template content
}

// NewViewdef creates a new viewdef.
func NewViewdef(typeName, namespace, content string) *Viewdef {
	if namespace == "" {
		namespace = "DEFAULT"
	}
	return &Viewdef{
		Type:      typeName,
		Namespace: namespace,
		Content:   content,
	}
}

// Key returns the TYPE.NAMESPACE identifier string.
func (v *Viewdef) Key() string {
	return v.Type + "." + v.Namespace
}

// ParseKey parses a TYPE.NAMESPACE key into type and namespace.
func ParseKey(key string) (typeName, namespace string, err error) {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid viewdef key: %s (expected TYPE.NAMESPACE)", key)
	}
	return parts[0], parts[1], nil
}

// GetContent returns the HTML template content.
func (v *Viewdef) GetContent() string {
	return v.Content
}

// GetType returns the type name.
func (v *Viewdef) GetType() string {
	return v.Type
}

// GetNamespace returns the namespace.
func (v *Viewdef) GetNamespace() string {
	return v.Namespace
}
