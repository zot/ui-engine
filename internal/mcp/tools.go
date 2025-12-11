// CRC: crc-MCPTool.md
// Spec: interfaces.md
package mcp

// Tool represents an MCP tool.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     func(args map[string]interface{}) (interface{}, error)
}

// NewTool creates a new tool definition.
func NewTool(name, description string, schema map[string]interface{}) *Tool {
	return &Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}
}

// WithHandler sets the tool handler.
func (t *Tool) WithHandler(handler func(args map[string]interface{}) (interface{}, error)) *Tool {
	t.Handler = handler
	return t
}

// Schema helper for building JSON schemas.
type Schema map[string]interface{}

// ObjectSchema creates an object schema.
func ObjectSchema(properties Schema, required []string) Schema {
	return Schema{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// StringProp creates a string property.
func StringProp(description string) Schema {
	return Schema{
		"type":        "string",
		"description": description,
	}
}

// IntProp creates an integer property.
func IntProp(description string) Schema {
	return Schema{
		"type":        "integer",
		"description": description,
	}
}

// BoolProp creates a boolean property.
func BoolProp(description string) Schema {
	return Schema{
		"type":        "boolean",
		"description": description,
	}
}

// ObjectProp creates an object property.
func ObjectProp(description string) Schema {
	return Schema{
		"type":        "object",
		"description": description,
	}
}

// CreateSessionTool creates a new session.
func CreateSessionTool() *Tool {
	return NewTool(
		"create_session",
		"Create a new UI session and return the session URL",
		ObjectSchema(Schema{}, []string{}),
	)
}

// CreatePresenterTool creates a presenter.
func CreatePresenterTool() *Tool {
	return NewTool(
		"create_presenter",
		"Create a presenter with type and properties",
		ObjectSchema(Schema{
			"type":       StringProp("Presenter type name"),
			"properties": ObjectProp("Initial properties"),
			"parentId":   IntProp("Parent variable ID (default: session root)"),
		}, []string{"type"}),
	)
}

// UpdatePresenterTool updates a presenter.
func UpdatePresenterTool() *Tool {
	return NewTool(
		"update_presenter",
		"Update presenter properties or call a method",
		ObjectSchema(Schema{
			"id":         IntProp("Presenter variable ID"),
			"properties": ObjectProp("Properties to update"),
			"call":       StringProp("Method to call"),
			"args":       ObjectProp("Method arguments"),
		}, []string{"id"}),
	)
}

// DestroyPresenterTool destroys a presenter.
func DestroyPresenterTool() *Tool {
	return NewTool(
		"destroy_presenter",
		"Remove a presenter",
		ObjectSchema(Schema{
			"id": IntProp("Presenter variable ID"),
		}, []string{"id"}),
	)
}

// CreateViewdefTool creates a viewdef.
func CreateViewdefTool() *Tool {
	return NewTool(
		"create_viewdef",
		"Create an HTML viewdef for TYPE.VIEW",
		ObjectSchema(Schema{
			"key":     StringProp("Viewdef key (TYPE.NAMESPACE)"),
			"content": StringProp("HTML template with ui-* bindings"),
		}, []string{"key", "content"}),
	)
}

// UpdateViewdefTool updates a viewdef.
func UpdateViewdefTool() *Tool {
	return NewTool(
		"update_viewdef",
		"Modify an existing viewdef",
		ObjectSchema(Schema{
			"key":     StringProp("Viewdef key (TYPE.NAMESPACE)"),
			"content": StringProp("New HTML template content"),
		}, []string{"key", "content"}),
	)
}

// LoadPresenterLogicTool loads Lua code.
func LoadPresenterLogicTool() *Tool {
	return NewTool(
		"load_presenter_logic",
		"Load Lua presenter logic code into the runtime",
		ObjectSchema(Schema{
			"name": StringProp("Logic module name"),
			"code": StringProp("Lua source code"),
		}, []string{"name", "code"}),
	)
}

// RegisterURLPathTool registers a URL path.
func RegisterURLPathTool() *Tool {
	return NewTool(
		"register_url_path",
		"Associate a URL path with a presenter",
		ObjectSchema(Schema{
			"path":        StringProp("URL path (after session ID)"),
			"presenterId": IntProp("Presenter variable ID"),
		}, []string{"path", "presenterId"}),
	)
}

// ActivateTabTool activates the browser tab.
func ActivateTabTool() *Tool {
	return NewTool(
		"activate_tab",
		"Bring the user's browser tab to focus via notification",
		ObjectSchema(Schema{
			"sessionId": StringProp("Session ID to activate"),
			"message":   StringProp("Optional notification message"),
		}, []string{"sessionId"}),
	)
}

// RegisterStandardTools adds the standard UI tools to a server.
func RegisterStandardTools(s *Server) {
	s.RegisterTool(CreateSessionTool())
	s.RegisterTool(CreatePresenterTool())
	s.RegisterTool(UpdatePresenterTool())
	s.RegisterTool(DestroyPresenterTool())
	s.RegisterTool(CreateViewdefTool())
	s.RegisterTool(UpdateViewdefTool())
	s.RegisterTool(LoadPresenterLogicTool())
	s.RegisterTool(RegisterURLPathTool())
	s.RegisterTool(ActivateTabTool())
}
