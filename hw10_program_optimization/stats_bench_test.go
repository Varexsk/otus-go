//go:build !bench
// +build !bench

package hw10programoptimization

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func BenchmarkGetDomainStat(b *testing.B) {
	r, err := zip.OpenReader("testdata/users.dat.zip")
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	if len(r.File) != 1 {
		b.Fatalf("unexpected archive content: %d files", len(r.File))
	}

	f, err := r.File[0].Open()
	if err != nil {
		b.Fatal(err)
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := GetDomainStat(bytes.NewReader(data), "biz"); err != nil {
			b.Fatal(err)
		}
	}
}
