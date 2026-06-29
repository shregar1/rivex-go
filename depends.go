package rivex

// Dependency is the interface for injectable dependencies. Implementations
// extract or compute values from the request context and return them for use
// in handlers.
type Dependency interface {
	// Resolve extracts or computes a value from the request context.
	Resolve(ctx *Context) (interface{}, error)
}

// DependencyFunc is a function type that implements the Dependency interface.
// This allows using plain functions as dependencies without defining a struct.
type DependencyFunc func(ctx *Context) (interface{}, error)

// Resolve implements the Dependency interface for DependencyFunc.
func (f DependencyFunc) Resolve(ctx *Context) (interface{}, error) {
	return f(ctx)
}

// Depends wraps a dependency (either a Dependency interface implementation or
// a DependencyFunc) for use in handler chains. The resolved value can be stored
// in the context for downstream handlers.
//
// Example:
//
//	authDep := rivex.Depends(func(ctx *rivex.Context) (interface{}, error) {
//	    token := ctx.Header("Authorization")
//	    if token == "" {
//	        return nil, rivex.NewHTTPException(401, "Missing token")
//	    }
//	    return token, nil
//	})
func Depends(dep interface{}) DependencyFunc {
	switch d := dep.(type) {
	case DependencyFunc:
		return d
	case func(ctx *Context) (interface{}, error):
		return DependencyFunc(d)
	case Dependency:
		return d.Resolve
	default:
		// For any other type, wrap it as a constant value dependency
		return func(ctx *Context) (interface{}, error) {
			return d, nil
		}
	}
}

// ResolveDependency is a helper that resolves a dependency and stores the
// result in the context under the given key. If the dependency returns an
// error, the error is returned immediately (typically an HTTPException).
func ResolveDependency(ctx *Context, key string, dep DependencyFunc) error {
	value, err := dep(ctx)
	if err != nil {
		return err
	}
	ctx.Set(key, value)
	return nil
}

// DependencyMiddleware creates a middleware that resolves a dependency and
// stores the result in the context under the given key before calling the
// next handler.
func DependencyMiddleware(key string, dep DependencyFunc) MiddlewareFunc {
	return func(ctx *Context, next NextFunc) error {
		err := ResolveDependency(ctx, key, dep)
		if err != nil {
			return err
		}
		return next(ctx)
	}
}
