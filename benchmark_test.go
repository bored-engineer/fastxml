package fastxml

import (
	"compress/gzip"
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

var benchCache struct {
	sync.Once
	b   []byte
	err error
}

func benchData(b *testing.B) []byte {
	benchCache.Do(func() {
		f, err := os.Open("./benchmark_SwissProt.xml.gz")
		if err != nil {
			benchCache.err = err
			return
		}
		defer f.Close()
		gr, err := gzip.NewReader(f)
		if err != nil {
			benchCache.err = err
			return
		}
		defer gr.Close()
		b, err := ioutil.ReadAll(gr)
		if err != nil {
			benchCache.err = err
			return
		}
		benchCache.b = b
	})
	if benchCache.err != nil {
		b.Fatalf("failed to load benchmark data: %v", benchCache.err)
	}
	return benchCache.b
}
