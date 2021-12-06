# Overview
This package provides boilerplate to get a TLS configuration and a `http.Client`.

## Usage

First install the kit package
```go
go get github.com/sanservices/kit
```

Then import the library
```go
import github.com/sanservices/kit/tls
```

After that you can use the functions
```go
func main(){
    config, err := tls.GetTLSConf(tls.TLS{})
    if err != nil{
        // handle error
    }

    client := tls.GetHTTPSClient(config)
}
```