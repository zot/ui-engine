// Package variable implements the Variable Protocol System.
// CRC: crc-Variable.md, crc-VariableStore.md, crc-WatchManager.md
// Spec: protocol.md
package variable

import (
	"encoding/json"
	"strings"
	"sync"
)

// Variable represents a single variable in the variable tree.
// Supports dual value architecture: monitored value for change detection,
// stored value for frontend (which can be a wrapper instance).
type Variable struct {
	ID              int64             `json:"id"`
	ParentID        int64             `json:"parentId,omitempty"`
	Value           interface{}       `json:"value,omitempty"` // Raw value from path resolution
	MonitoredValue  interface{}       `json:"-"`               // For change detection (shallow copy for arrays)
	StoredValue     interface{}       `json:"-"`               // Can be the wrapper instance itself
	WrapperInstance interface{}       `json:"-"`               // Internal wrapper object (if wrapper property set)
	Properties      map[string]string `json:"properties,omitempty"`
	Unbound         bool              `json:"unbound,omitempty"`
	mu              sync.RWMutex
}

// NewVariable creates a new Variable with the given ID.
func NewVariable(id int64) *Variable {
	return &Variable{
		ID:         id,
		Properties: make(map[string]string),
	}
}

// GetID returns the variable's unique identifier.
// Implements WrapperVariable interface.
func (v *Variable) GetID() int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.ID
}

// GetValue returns the current value.
func (v *Variable) GetValue() interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Value
}

// SetValue updates the value.
func (v *Variable) SetValue(value interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.Value = value
}

// GetProperty returns a property value (empty string if unset).
func (v *Variable) GetProperty(name string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Properties[name]
}

// SetProperty sets a property value. Empty string removes the property.
func (v *Variable) SetProperty(name, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if value == "" {
		delete(v.Properties, name)
	} else {
		v.Properties[name] = value
	}
}

// GetProperties returns a copy of all properties.
func (v *Variable) GetProperties() map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	props := make(map[string]string, len(v.Properties))
	for k, val := range v.Properties {
		props[k] = val
	}
	return props
}

// SetProperties sets multiple properties at once, handling priority suffixes.
func (v *Variable) SetProperties(props map[string]string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Process by priority: high, med, low, then no suffix
	priorities := []string{":high", ":med", ":low", ""}
	for _, suffix := range priorities {
		for name, value := range props {
			baseName, propSuffix := parsePropertyName(name)
			if propSuffix == suffix {
				if value == "" {
					delete(v.Properties, baseName)
				} else {
					v.Properties[baseName] = value
				}
			}
		}
	}
}

// parsePropertyName splits a property name into base name and priority suffix.
func parsePropertyName(name string) (baseName, suffix string) {
	for _, s := range []string{":high", ":med", ":low"} {
		if strings.HasSuffix(name, s) {
			return strings.TrimSuffix(name, s), s
		}
	}
	return name, ""
}

// IsStandardVariable checks if this is registered with @NAME pattern.
func (v *Variable) IsStandardVariable() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	// Standard variables are identified by the store, not the variable itself
	return false
}

// IsObjectReference checks if the value is {obj: ID} form.
func (v *Variable) IsObjectReference() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	// This check is now more complex as Value is interface{}
	// For now, we assume this is handled elsewhere.
	return false
}

// IsUnbound checks if storage is in UI server (not external backend).
func (v *Variable) IsUnbound() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Unbound
}

// HasWrapper checks if a wrapper instance is stored internally.
func (v *Variable) HasWrapper() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.WrapperInstance != nil
}

// GetWrapperTypeName returns the wrapper type name from properties (empty string if unset).
func (v *Variable) GetWrapperTypeName() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.Properties["wrapper"]
}

// SetWrapperInstance stores the wrapper instance internally.
// Called during variable creation when wrapper property is set.
func (v *Variable) SetWrapperInstance(wrapper interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.WrapperInstance = wrapper
}

// GetWrapperInstance returns the internal wrapper instance (nil if none).
func (v *Variable) GetWrapperInstance() interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.WrapperInstance
}

// GetStoredValue returns the stored value (sent to frontend).
// This is the wrapper instance if it exists, or the raw value otherwise.
func (v *Variable) GetStoredValue() interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.WrapperInstance != nil {
		return v.WrapperInstance
	}
	if v.StoredValue != nil {
		return v.StoredValue
	}
	return v.Value
}

// SetStoredValue sets the stored value.
func (v *Variable) SetStoredValue(value interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.StoredValue = value
}

// ObjectReference represents a reference to another object.
type ObjectReference struct {
	Obj int64 `json:"obj"`
}

// IsObjectReference checks if a JSON value is an object reference.
func IsObjectReference(value json.RawMessage) bool {
	if len(value) == 0 {
		return false
	}
	var ref ObjectReference
	if err := json.Unmarshal(value, &ref); err != nil {
		return false
	}
	return ref.Obj != 0
}

// GetObjectReferenceID extracts the object ID from an object reference.
func GetObjectReferenceID(value json.RawMessage) (int64, bool) {
	var ref ObjectReference
	if err := json.Unmarshal(value, &ref); err != nil {
		return 0, false
	}
	if ref.Obj == 0 {
		return 0, false
	}
	return ref.Obj, true
}
