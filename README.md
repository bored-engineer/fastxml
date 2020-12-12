# fastxml
A "fast" implementation of Golang's [xml.TokenReader](https://godoc.org/encoding/xml#TokenReader) for well-formed XML input. 

## Security
Some of fastxml's performance gains come from assuming that the input XML is well-formed. Generally speaking it should return a relevant error when handling invalid XML, but it should never be used in a security sensitive context (ex: parsing SAML data). 

## Benchmark
Testing against the [SwissProt](http://aiweb.cs.washington.edu/research/projects/xmltk/xmldata/www/repository.html) (109 MB) XML file shows a 3x performance improvement over stdlib:
```
$ go test -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/bored-engineer/fastxml
BenchmarkStdlibReader-12     	       1	4715479832 ns/op	984328568 B/op	23578836 allocs/op
BenchmarkFastXMLReader-12    	       1	1526542832 ns/op	1211033528 B/op	13541058 allocs/op
PASS
ok  	github.com/bored-engineer/fastxml	6.328s
```
Also note, fastxml has an unfair advantage in these benchmarks over stdlib as it only operates on a complete `[]byte` slice instead of a streaming `io.Reader` and decoded xml entities on-demand.

## Usage
```go
import (
  "log"
  
  "github.com/bored-engineer/fastxml"
)

func main() {
  tr := fastxml.NewDecoder([]byte(`<xml></xml>`))
  for {
    token, err := tr.RawToken()
    if err != nil {
      log.Fatal(err)
    } else if token == nil {
      break
    }
    
    log.Printf("%#v", token)
  }
}
```
