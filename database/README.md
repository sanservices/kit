# Overview

This package provides boilerplate code for creating database connections. It uses `sqlx` interface since it provides better functionalty than the standard `sql` package.

## Usage

First install the kit package
```go
go get github.com/sanservices/kit
```

Then import the library
```go
import github.com/sanservices/kit/database
```

### Available databse supported
- Mysql
- Sqlite
- Redis