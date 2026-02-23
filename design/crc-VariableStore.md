# VariableStore

**Source Spec:** protocol.md, data-models.md
**Requirements:** R43, R48, R49, R84, R85, R86, R88

## Responsibilities

### Knows
- variables: Map of variable ID to Variable
- standardVariables: Map of @NAME to variable ID
- nextVarId: Counter for generating unique IDs (frontend starts at 2, server at -1)

### Does
- create: Create variable with sender-provided ID, optional parent, value, properties (synchronous). When a widget is provided, sets `elementId` in properties to `widget.elementId`
- createVarId: Vend next variable ID (frontend: 2+, server: -1 and below)
- get: Retrieve variable by ID
- getByName: Retrieve standard variable by @NAME
- update: Update variable value and/or properties
- destroy: Remove variable and all children recursively
- registerStandardVariable: Associate @NAME with variable ID
- getChildren: Find all variables with given parentId
- resolveObjectReference: Get object data for {obj: ID} references

## Collaborators

- Variable: Individual variable instances
- ProtocolHandler: Receives protocol messages
- Config: Logging delegate (variable operations and errors)

**Note:** Watch functionality is internal to VariableStore via watch()/watchErrors() methods.

## Sequences

- seq-create-variable.md: Variable creation flow
- seq-destroy-variable.md: Recursive destruction
