# fastxml
A "fast" implementation of Golang's [xml.TokenReader](https://godoc.org/encoding/xml#TokenReader) for well-formed XML input. 

## Security
Some of fastxml's performance gains come from assuming that the input XML is well-formed. It should never be used in a security sensitive context (ex: parsing SAML data) as it can almost certainly be tricked into parsing data incorrectly or even panicing. 

## Benchmark
Testing against the [SwissProt](http://aiweb.cs.washington.edu/research/projects/xmltk/xmldata/www/repository.html) (109 MB) XML file shows a 2x performance improvement over stdlib and a 26x improvement when using just Scanner (somewhat unfair):
```
$ go test -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/bored-engineer/fastxml
BenchmarkScanner-12               	       8	 126334701 ns/op	       0 B/op	       0 allocs/op
BenchmarkEncodingXMLDecoder-12    	       1	3336588490 ns/op	715211208 B/op	23563878 allocs/op
BenchmarkXMLTokenReader-12        	       1	1526152566 ns/op	702095696 B/op	15335500 allocs/op
PASS
ok  	github.com/bored-engineer/fastxml	8.168s
```
Also note, fastxml has an unfair advantage in these benchmarks over stdlib as it only operates on a complete `[]byte` slice instead of a streaming `io.Reader`.

## Usage
```go
import (
  "log"
  
  "github.com/bored-engineer/fastxml"
)

func main() {
  tr := fastxml.NewScanner([]byte(`<!directive>some <xml key="value">data`))
  for {
    token, chardata, err := tr.Next()
    if err != nil {
      log.Fatal(err)
    }
    switch {
    case chardata:
      decoded, err := fastxml.CharData(token)
      if err != nil {
        log.Fatalf("failed to decode %q: %s", string(token), err)
      }
      log.Printf("CharData: %q", string(decoded))
    case fastxml.IsDirective(token):
      dir := fastxml.Directive(token)
      log.Printf("Directive: %q", string(dir))
    case fastxml.IsProcInst(token):
      target, inst := fastxml.ProcInst(token)
      log.Printf("ProcInst: (%q, %q)", string(target), string(inst))
    case fastxml.IsComment(token):
      comment := fastxml.Comment(token)
      log.Printf("Comment: %q", comment)
    default:
      name, attrs := fastxml.Element(token)
      space, local := fastxml.Name(name)
      log.Printf("Element: (%q, %q) %b", string(space), string(local), fastxml.IsSelfClosing(token))
      if fastxml.IsStartElement(token) {
        if err := fastxml.Attrs(attrs, func(key, val []byte) error{
          decoded, err := fastxml.DecodeEntities(val)
          if err != nil {
            log.Fatalf("failed to decode %q: %s", string(val), err)
          }
          log.Printf("%q: %q", string(key), string(decoded))
          return nil
        }); err != nil {
          log.Fatalf("failed to read attribute: %s", err)
        }
      }
    }
  }
}
```
