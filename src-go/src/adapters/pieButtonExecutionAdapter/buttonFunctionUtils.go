package pieButtonExecutionAdapter

// Defines the signature for basic functions without coordinates.
type ButtonFunctionNoArgHandler func() error

// Defines the signature for functions needing coordinates.
type ButtonFunctionWithCoordinatesHandler func(x, y int) error

// Provides a common interface for executing different function types.
type ButtonFunctionExecutor interface {
	Execute(x, y int) error
}

// Wraps a FunctionHandler to satisfy the HandlerWrapper interface.
type NoArgButtonFunctionExecutor struct {
	fn ButtonFunctionNoArgHandler
}

// Wraps a FunctionHandlerWithCoords to satisfy the HandlerWrapper interface.
type CoordinatesButtonFunctionExecutor struct {
	fn ButtonFunctionWithCoordinatesHandler
}

// --- Function Handlers & Wrappers ---

// Execute calls the wrapped basic function, ignoring coordinates.
func (h NoArgButtonFunctionExecutor) Execute(x, y int) error {
	return h.fn()
}

// Execute calls the wrapped function, passing coordinates.
func (h CoordinatesButtonFunctionExecutor) Execute(x, y int) error {
	return h.fn(x, y)
}