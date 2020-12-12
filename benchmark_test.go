package fastxml

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

// Give stdlib a fighting chance reading directly from memory not disk
func loadBenchmarkData() ([]byte, error) {
	f, err := os.Open("./benchmark_SwissProt.xml.gz")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	b, err := ioutil.ReadAll(gr)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func BenchmarkStdlibReader(b *testing.B) {
	data, err := loadBenchmarkData()
	if err != nil {
		b.Fatalf("failed to load data: %v", err)
	}
	for n := 0; n < b.N; n++ {
		d := xml.NewDecoder(bytes.NewReader(data))
		for {
			_, err := d.RawToken()
			if err == io.EOF {
				break
			} else if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}

func BenchmarkFastXMLReader(b *testing.B) {
	data, err := loadBenchmarkData()
	if err != nil {
		b.Fatalf("failed to load data: %v", err)
	}
	for n := 0; n < b.N; n++ {
		d := NewDecoder(data)
		for {
			_, err := d.RawToken()
			if err == io.EOF {
				break
			} else if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	}
}
