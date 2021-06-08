package jsonpatch

// Represents a JSON patch
type JSONPatchOperation struct {
	// The Operation to be applied.
	// Can be one of the following:
	// add, remove, replace, copy, move, test.
	Op string `json:"op"`
	// The JSON Pointer to the value on which to operate.
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// A JSONPatch is a collection of JSONPatchOperations
type JSONPatch []JSONPatchOperation
