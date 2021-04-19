package gocense_test

import (
	"testing"
	"time"

	"github.com/Nerzal/gocense"
)

func Benchmark_GetSwaggo(b *testing.B) {
	// Arrange
	client := gocense.New()

	b.N = 10

	for n := 0; n < b.N; n++ {
		start := time.Now()
		client.Get("github.com/swaggo/swag")
		end := time.Now()

		diff := end.Sub(start)
		println(diff.Seconds())
	}
}
