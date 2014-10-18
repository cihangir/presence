package presence

import "testing"

func generateUserBatchs(batchSize int, count int) [][]string {
	users := make([][]string, count)
	batch := make([]string, batchSize)

	for i := 0; i < batchSize; i++ {
		batch[i] = <-nextId
	}

	for n := 0; n < count; n++ {
		users[n] = batch
	}

	return users
}

func benchmarkOnline(i int, b *testing.B) {
	f := func(s *Session, users ...string) {
		s.Online(users...)
	}

	benchmark(i, f, b)
}

func benchmarkOffline(i int, b *testing.B) {
	f := func(s *Session, users ...string) {
		s.Offline(users...)
	}

	benchmark(i, f, b)
}

func benchmarkStatus(i int, b *testing.B) {
	f := func(s *Session, users ...string) {
		s.Status(users...)
	}

	benchmark(i, f, b)
}

func benchmark(i int, f func(s *Session, users ...string), b *testing.B) {
	err := withConn(func(s *Session) {
		users := generateUserBatchs(i, b.N)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			f(s, users[n]...)
		}
	})
	if err != nil {
		b.Fatal(err.Error())
	}
}

func benchmarkOnlineParallel(pCount, i int, b *testing.B) {
	f := func(s *Session, users ...string) {
		s.Online(users...)
	}

	benchmarkParallel(pCount, i, f, b)
}

func benchmarkOfflineParallel(pCount, i int, b *testing.B) {
	f := func(s *Session, users ...string) {
		s.Offline(users...)
	}

	benchmarkParallel(pCount, i, f, b)
}

func benchmarkStatusParallel(pCount, i int, b *testing.B) {
	f := func(s *Session, users ...string) {
		s.Status(users...)
	}

	benchmarkParallel(pCount, i, f, b)
}

func benchmarkParallel(pCount, i int, f func(s *Session, users ...string), b *testing.B) {
	err := withConn(func(s *Session) {
		users := generateUserBatchs(i, 1)
		b.ReportAllocs()
		b.ResetTimer()
		b.SetParallelism(pCount)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				f(s, users[0]...)
			}
		})
	})
	if err != nil {
		b.Fatal(err.Error())
	}
}

func BenchmarkOnline1(b *testing.B)     { benchmarkOnline(1, b) }
func BenchmarkOnline10(b *testing.B)    { benchmarkOnline(10, b) }
func BenchmarkOnline100(b *testing.B)   { benchmarkOnline(100, b) }
func BenchmarkOnline1000(b *testing.B)  { benchmarkOnline(1000, b) }
func BenchmarkOnline10000(b *testing.B) { benchmarkOnline(10000, b) }

func BenchmarkOffline1(b *testing.B)     { benchmarkOffline(1, b) }
func BenchmarkOffline10(b *testing.B)    { benchmarkOffline(10, b) }
func BenchmarkOffline100(b *testing.B)   { benchmarkOffline(100, b) }
func BenchmarkOffline1000(b *testing.B)  { benchmarkOffline(1000, b) }
func BenchmarkOffline10000(b *testing.B) { benchmarkOffline(10000, b) }

func BenchmarkStatus1(b *testing.B)     { benchmarkStatus(1, b) }
func BenchmarkStatus10(b *testing.B)    { benchmarkStatus(10, b) }
func BenchmarkStatus100(b *testing.B)   { benchmarkStatus(100, b) }
func BenchmarkStatus1000(b *testing.B)  { benchmarkStatus(1000, b) }
func BenchmarkStatus10000(b *testing.B) { benchmarkStatus(10000, b) }

func BenchmarkOnlineParallel1(b *testing.B) { benchmarkOnlineParallel(1, 1000, b) }
func BenchmarkOnlineParallel2(b *testing.B) { benchmarkOnlineParallel(2, 1000, b) }
func BenchmarkOnlineParallel4(b *testing.B) { benchmarkOnlineParallel(4, 1000, b) }
func BenchmarkOnlineParallel8(b *testing.B) { benchmarkOnlineParallel(8, 1000, b) }

func BenchmarkOfflineParallel1(b *testing.B) { benchmarkOfflineParallel(1, 1000, b) }
func BenchmarkOfflineParallel2(b *testing.B) { benchmarkOfflineParallel(2, 1000, b) }
func BenchmarkOfflineParallel4(b *testing.B) { benchmarkOfflineParallel(4, 1000, b) }
func BenchmarkOfflineParallel8(b *testing.B) { benchmarkOfflineParallel(8, 1000, b) }

func BenchmarkStatusParallel1(b *testing.B) { benchmarkStatusParallel(1, 1000, b) }
func BenchmarkStatusParallel2(b *testing.B) { benchmarkStatusParallel(2, 1000, b) }
func BenchmarkStatusParallel4(b *testing.B) { benchmarkStatusParallel(4, 1000, b) }
func BenchmarkStatusParallel8(b *testing.B) { benchmarkStatusParallel(8, 1000, b) }

// ➜ redisence git:(master) ✗ go test -run=XXX -bench=.
// PASS
// BenchmarkOnline1         500       2730336 ns/op         384 B/op         12 allocs/op
// BenchmarkOnline10       1000       2943270 ns/op        1577 B/op         62 allocs/op
// BenchmarkOnline100       500       4949865 ns/op       13807 B/op        559 allocs/op
// BenchmarkOnline1000      100      19082094 ns/op      136164 B/op       5584 allocs/op
// BenchmarkOnline10000          10     171953025 ns/op     1508567 B/op      61651 allocs/op
// BenchmarkOffline1       1000       1329691 ns/op         325 B/op         11 allocs/op
// BenchmarkOffline10       500       3584185 ns/op        1537 B/op         61 allocs/op
// BenchmarkOffline100      500       5400142 ns/op       13727 B/op        557 allocs/op
// BenchmarkOffline1000         200      13314189 ns/op      134482 B/op       5518 allocs/op
// BenchmarkOffline10000         10     111973531 ns/op     1348632 B/op      55134 allocs/op
// BenchmarkStatus1        2000       2896557 ns/op         312 B/op         10 allocs/op
// BenchmarkStatus10        500       2903742 ns/op        1369 B/op         51 allocs/op
// BenchmarkStatus100       500       4230765 ns/op       12120 B/op        457 allocs/op
// BenchmarkStatus1000      100      10162690 ns/op      118170 B/op       4516 allocs/op
// BenchmarkStatus10000          20      76042699 ns/op     1180966 B/op      45095 allocs/op
// BenchmarkOnlineParallel1         100      18442989 ns/op      136728 B/op       5585 allocs/op
// BenchmarkOnlineParallel2         100      15698227 ns/op      138172 B/op       5651 allocs/op
// BenchmarkOnlineParallel4         100      15893257 ns/op      141469 B/op       5781 allocs/op
// BenchmarkOnlineParallel8         100      16382508 ns/op      165632 B/op       6045 allocs/op
// BenchmarkOfflineParallel1        100      12841998 ns/op      151028 B/op       5521 allocs/op
// BenchmarkOfflineParallel2        100      12875952 ns/op      150916 B/op       5521 allocs/op
// BenchmarkOfflineParallel4        100      11707740 ns/op      151239 B/op       5522 allocs/op
// BenchmarkOfflineParallel8        100      11283041 ns/op      151623 B/op       5522 allocs/op
// BenchmarkStatusParallel1         100      10007535 ns/op      133349 B/op       4516 allocs/op
// BenchmarkStatusParallel2         200       8350930 ns/op      133343 B/op       4515 allocs/op
// BenchmarkStatusParallel4         200       7381851 ns/op      133423 B/op       4516 allocs/op
// BenchmarkStatusParallel8         200       7275941 ns/op      133658 B/op       4516 allocs/op
// ok      _/Users/siesta/Documents/redisence  64.228s
// ➜ redisence git:(master) ✗
