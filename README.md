# Overview
This is a compilation of different tools that we use on multiple services written in Go.

It uses an extremely flexible approach, allowing any kind of database, cloud provider, log, and so on to be created.

## Installation

```sh
go get github.com/sanservices/kit
```

## Download

clone the repository:

```sh
git clone git@github.com:sanservices/kit.git
```


## Contribute

**Use issues for everything**

- For a small change, just send a PR.
- For bigger changes open an issue for discussion before sending a PR.
- PR should have:
  - Test case
  - Documentation
  - Example (If it makes sense)
- You can also contribute by:
  - Reporting issues
  - Suggesting new features or enhancements
  - Improve/fix documentation

## Test

Use the following command to run the tests:

```sh
make test
```

You can also run the linter (`make staticcheck`), find code smells (`make vet`) and print the test coverage (`make coverage`).

All four commands can be run with `make scan-cli`.

Their result can also be saved in a `report` folder in the current working directory, using `make scan`.

Also, you can see the make command with more details by doing `make help` on the shell.

## Organization

#### bundle:
It is a directory in the project that groups related resources together in one place. It's a package where all the services are initialized and used in other repositories.

#### config:
It reads the given file to export configuration

```go
import (
	"github.com/sanservices/kit/bundle"
	"github.com/sanservices/kit/config"
)
func main(){
	//read configuration and initialize services
	cfg := config.Read(config.Env(configEnv))
	//make a new instance of bundle
	b := new(bundle.All)
	//initialize all the services considering the config (*.json) file
	b.Initialize(cfg)
    //example of calling mongodb function
    b.Services.MongoDB.FindAllByFilter(...)
  }
```