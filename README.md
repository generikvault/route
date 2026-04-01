# route

Build HTTP handlers from functions shaped like:

```go
func(context.Context, Input) (Output, error)
```

The route is derived from the fields of `Input`. Options decide where each
field comes from, such as a fixed path segment, a path ID, or the request
body.

## Example

```go
handler, err := route.New(
    route.Join(
        route.PathByNameOfFixedTyped(strings.ToLower),
        route.JSONResponse(),
        route.Get(func(ctx context.Context, in struct {
            Users route.Fixed
        }) (string, error) {
            return "hello", nil
        }),
    ),
)
```

With a request to `/users`, the `Users route.Fixed` field contributes the fixed
path segment and the handler response is encoded as JSON.

## Common field bindings

| Input field       | Option                        | Request source     |
| ----------------- | ----------------------------- | ------------------ |
| `Foo route.Fixed` | `PathByNameOfFixedTyped(...)` | fixed path segment |
| `ID int`          | `ByType(IntPathIDs())`        | path segment       |
| `ID string`       | `ByType(StringPathIDs())`     | path segment       |
| `Body T`          | `ByName("Body", JSONBody())`  | JSON request body  |

## Typical setup

- Choose a method with `Get`, `Post`, `Put`, or `Delete`.
- Bind input fields with `ByType` and `ByName`.
- Encode output with `JSONResponse` or a custom `ResponseEncoder`.
- Add cross-cutting behavior with `Middleware` and `HandleError`.

## Getter Subpackage

The Subpackage `getter` maps URL query values into struct fields.
Default query names are kebab-case field names and can be overridden with a `getter` struct tag.

See the package example and tests for additional combinations.
