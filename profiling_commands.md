## Useful Commands

go test -run=. -bench=. -benchtime=5s -count 5 -benchmem -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out ./package | tee bench.txt
go test -run=^$ -bench=^Benchmark_Put$ -count 5 -benchmem -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out
go test -run=. -bench=. -benchmem -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out

go test -run=^$ -bench=^BenchmarkWrite$ -benchmem -cpuprofile=cpu.out -memprofile=mem.out -trace=trace.out

go tool pprof -http :8080 cpu.out
go tool pprof -http :8081 mem.out
go tool trace trace.out

go tool pprof pool.test mem.out
go tool pprof pool.test cpu.out
go tool pprof -nodecount=1000 -cum -edgefraction=0 -nodefraction=0 mem.out (shows all nodes)  
tagfocus=phase=get
tagfocus=phase=put
top
list .*pool.*Get
go tool pprof -top -nodecount=1000 cpu.prof
(pprof) top -cum -nodecount=1000

benchstat bench.txt
rm cpu.out mem.out trace.out \*.test
