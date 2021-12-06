# Overview
This package provides the boilerplate code to be able to add pprof debugging endpoints
using echo framework.

## Usage

First install the kit package
```go
go get github.com/sanservices/kit
```

Then import the library
```go
import github.com/sanservices/kit/pprofdebug
```

After that you can register the endpoint
```go
func (h Handler) RegisterRoutes(e *echo.Group) {
	pprofdebug.WrapGroup("", e)
}
```