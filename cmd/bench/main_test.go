package main

import (
	"flag"
	"testing"
)

func BenchmarkMain(b *testing.B) {
	flag.Parse()

	*maxreqs = b.N

	b.ReportAllocs()

	main()
}
