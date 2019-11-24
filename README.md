# fastxml
A "fast" implementation of Golang's [xml.TokenReader](https://godoc.org/encoding/xml#TokenReader) for well-formed XML input. 

## Security
Some of fastxml's performance gains come from assuming that the input XML is well-formed. Generally speaking it should return a relevant error when handling invalid XML, but it should never be used in a security sensitive context (ex: parsing SAML data). 

## Usage
```go
import (
  "log"
  
  "github.com/bored-engineer/fastxml"
)

func main() {
  tr := fastxml.NewTokenReader(`<xml></xml>`)
  for {
    token, err := tr.Token()
    if err != nil {
      log.Fatal(err)
    } else if token == nil {
      break
    }
    
    log.Printf("%#v", token)
  }
}
```

## Tests
TODO (:fine)

## Benchmark
Testing against a completely arbitrary XML file I had locally:
```
$ go test -benchmem -bench .
goos: darwin
goarch: amd64
pkg: github.com/bored-engineer/fastxml
BenchmarkFastXMLTokenReader-12    	      34	  33719859 ns/op	22905024 B/op	  644238 allocs/op
BenchmarkStdlibTokenReader-12     	       7	 146010512 ns/op	27672720 B/op	  719542 allocs/op
```
