package lua

// CRC: crc-Module.md

// Module tracks resources registered by a single Lua module file.
// This enables clean unloading by tracking what each module registered.
type Module struct {
	// Name is the module's tracking key (baseDir-relative file path)
	Name string
	// Directory is the directory containing this module
	Directory string
	// Prototypes tracks prototype names registered by this module
	Prototypes []string
	// PresenterTypes tracks presenter type names registered by this module
	PresenterTypes []string
	// Wrappers tracks wrapper names registered by this module
	Wrappers []string
}

// NewModule creates a new Module with the given tracking key and directory.
func NewModule(name, directory string) *Module {
	return &Module{
		Name:      name,
		Directory: directory,
	}
}

// AddPrototype tracks a prototype registered by this module.
func (m *Module) AddPrototype(name string) {
	m.Prototypes = append(m.Prototypes, name)
}

// AddPresenterType tracks a presenter type registered by this module.
func (m *Module) AddPresenterType(name string) {
	m.PresenterTypes = append(m.PresenterTypes, name)
}

// AddWrapper tracks a wrapper registered by this module.
func (m *Module) AddWrapper(name string) {
	m.Wrappers = append(m.Wrappers, name)
}
