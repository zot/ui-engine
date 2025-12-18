# ObjectReference

**Source Spec:** protocol.md

## Responsibilities

### Knows
- id: Object ID (positive = backend-managed, negative = UI server-managed)
- isBackendManaged: True if ID > 0
- isServerManaged: True if ID < 0

### Does
- toJson: Return {obj: ID} JSON representation
- fromJson: Parse {obj: ID} from JSON
- isObjectReference: Check if value is object reference form
- resolve: Get actual object data from variable store
- getManager: Determine if backend or server manages object

## Collaborators

- Variable: Contains object references as values
- VariableStore: Resolves references to objects
- ProtocolHandler: Handles get/getObjects for resolution
- PathNavigator: Traverses into object properties

## Sequences

- seq-create-variable.md: Creating variables with object references
- seq-path-resolve.md: Navigating through object references
