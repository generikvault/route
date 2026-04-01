// Package route builds HTTP handlers from functions with the shape
// func(context.Context, Input) (Output, error).
//
// The route is derived from the fields of Input. Options decide how each field
// is bound from the request, for example from fixed path segments, path IDs, or
// the request body.
//
// A typical setup combines an HTTP method such as Get or Post, field bindings
// such as ByType(IntPathIDs()) or ByName("Body", JSONBody()), and a response
// encoder such as JSONResponse().
//
// See ExampleNew for a minimal end-to-end setup.
package route
