// CRC: crc-MCPResource.md
// Spec: interfaces.md
package mcp

// Resource represents an MCP resource.
type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Handler     func() (interface{}, error)
}

// NewResource creates a new resource definition.
func NewResource(uri, name, description, mimeType string) *Resource {
	return &Resource{
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
	}
}

// WithHandler sets the resource handler.
func (r *Resource) WithHandler(handler func() (interface{}, error)) *Resource {
	r.Handler = handler
	return r
}

// PresenterTypesResource lists available presenter types.
func PresenterTypesResource() *Resource {
	return NewResource(
		"ui://presenter-types",
		"Presenter Types",
		"List of available presenter types and their properties",
		"application/json",
	)
}

// ViewdefsResource lists available viewdefs.
func ViewdefsResource() *Resource {
	return NewResource(
		"ui://viewdefs",
		"Viewdefs",
		"List of available TYPE.VIEW viewdefs and their bindings",
		"application/json",
	)
}

// SessionStateResource returns current session state.
func SessionStateResource(sessionID string) *Resource {
	return NewResource(
		"ui://session/"+sessionID+"/state",
		"Session State",
		"Current session variable tree",
		"application/json",
	)
}

// PendingMessagesResource returns queued user messages.
func PendingMessagesResource(sessionID string) *Resource {
	return NewResource(
		"ui://session/"+sessionID+"/pending",
		"Pending Messages",
		"Queued user messages and requests",
		"application/json",
	)
}

// PresenterStateResource returns specific presenter data.
func PresenterStateResource(sessionID string, presenterID int64) *Resource {
	return NewResource(
		"ui://session/"+sessionID+"/presenter/"+string(rune(presenterID)),
		"Presenter State",
		"Specific presenter data",
		"application/json",
	)
}

// RegisterStandardResources adds the standard UI resources to a server.
func RegisterStandardResources(s *Server) {
	s.RegisterResource(PresenterTypesResource())
	s.RegisterResource(ViewdefsResource())
}
