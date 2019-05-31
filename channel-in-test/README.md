# What's this?

This is an example where something unexpected happened.

What makes it interesting is the behavior doesn't fit with the mental model
of how channel, function and sub tests works in Go.

## The story

I was trying to write a rate limiter that can deal with concurrent requests in Go.
To check one implementation, I wrote some tests that involved:

- Making N concurrent call `IsAllow(timestamp)` (same timestamp)
- Check the result and call `Fatalf` it it doesn't match
- Note that this test logic is wrong (since there's no guaranteed ordering of goroutines).
- However, what caught me offguard was getting a deadlock where I thought it was impossible.
- Digging further, the problem seems to be with the use of `Fatalf` vs `Logf` or `Errorf`.
  I only get deadlock when using `Fatalf`.

## Try out

The system I'm working on

```
$> go version
go version go1.12.4 linux/amd64

$> go env
GOARCH="amd64"
GOBIN=""
GOCACHE="/home/dev/.cache/go-build"
GOEXE=""
GOFLAGS=""
GOHOSTARCH="amd64"
GOHOSTOS="linux"
GOOS="linux"
GOPATH="/home/dev/workspace"
GOPROXY=""
GORACE=""
GOROOT="/usr/local/go"
GOTMPDIR=""
GOTOOLDIR="/usr/local/go/pkg/tool/linux_amd64"
GCCGO="gccgo"
CC="gcc"
CXX="g++"
CGO_ENABLED="1"
GOMOD=""
CGO_CFLAGS="-g -O2"
CGO_CPPFLAGS=""
CGO_CXXFLAGS="-g -O2"
CGO_FFLAGS="-g -O2"
CGO_LDFLAGS="-g -O2"
PKG_CONFIG="pkg-config"
GOGCCFLAGS="-fPIC -m64 -pthread -fmessage-length=0 -fdebug-prefix-map=/tmp/go-build109693962=/tmp/go-build -gno-record-gcc-switches"

$> cat /proc/cpuinfo  | grep processor | wc -l
2
```

Follow along:

Run the test as it is, you might see a deadlock

```
$> GO111MODULE=on go test -mod=vendor -v -count=1
=== RUN   TestWat
=== RUN   TestWat/oneRPS
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan receive]:
testing.(*T).Run(0xc0000aa100, 0x53a1a4, 0x7, 0x542ce8, 0x469626)
        /usr/local/go/src/testing/testing.go:917 +0x381
testing.runTests.func1(0xc0000aa000)
        /usr/local/go/src/testing/testing.go:1157 +0x78
testing.tRunner(0xc0000aa000, 0xc00009de30)
        /usr/local/go/src/testing/testing.go:865 +0xc0
testing.runTests(0xc000088020, 0x6280f0, 0x1, 0x1, 0x0)
        /usr/local/go/src/testing/testing.go:1155 +0x2a9
testing.(*M).Run(0xc0000a8000, 0x0)
        /usr/local/go/src/testing/testing.go:1072 +0x162
main.main()
        _testmain.go:42 +0x13e

goroutine 18 [chan receive]:
testing.(*T).Run(0xc0000aa300, 0x53a014, 0x6, 0xc000066440, 0xc0000b20b0)
        /usr/local/go/src/testing/testing.go:917 +0x381
github.com/bitsgofer/gowat/channel-in-test.TestWat(0xc0000aa100)
        /home/dev/workspace/src/github.com/bitsgofer/gowat/channel-in-test/concurrent_rate_limiter_test.go:38 +0x4cd
testing.tRunner(0xc0000aa100, 0x542ce8)
        /usr/local/go/src/testing/testing.go:865 +0xc0
created by testing.(*T).Run
        /usr/local/go/src/testing/testing.go:916 +0x35a

goroutine 19 [chan receive]:
github.com/bitsgofer/gowat/channel-in-test.TestWat.func1(0xc0000aa300)
        /home/dev/workspace/src/github.com/bitsgofer/gowat/channel-in-test/concurrent_rate_limiter_test.go:85 +0xdb
testing.tRunner(0xc0000aa300, 0xc000066440)
        /usr/local/go/src/testing/testing.go:865 +0xc0
created by testing.(*T).Run
        /usr/local/go/src/testing/testing.go:916 +0x35a
exit status 2
FAIL    github.com/bitsgofer/gowat/channel-in-test      0.021s
```

Edit `concurrent_rate_limiter_test.go` and comment out the line with `Fatalf`.
Replace it with `Logf` or `Errorf` instead.

No deadlock now.

```
$> GO111MODULE=on go test -mod=vendor -v -count=1
=== RUN   TestWat
=== RUN   TestWat/spread
=== RUN   TestWat/oneRPS
--- PASS: TestWat (0.00s)
    --- PASS: TestWat/spread (0.00s)
        concurrent_rate_limiter_test.go:58: IsAllow(1), want= false, got= true
        concurrent_rate_limiter_test.go:58: IsAllow(1), want= true, got= false
    --- PASS: TestWat/oneRPS (0.00s)
        concurrent_rate_limiter_test.go:58: IsAllow(0), want= false, got= true
        concurrent_rate_limiter_test.go:58: IsAllow(0), want= true, got= false
        concurrent_rate_limiter_test.go:58: IsAllow(3), want= false, got= true
        concurrent_rate_limiter_test.go:58: IsAllow(3), want= true, got= false
PASS
ok      github.com/bitsgofer/gowat/channel-in-test      0.003s
```

Then try the same steps with `go test -race`. You might see the test pass even for `Fatalf`,
or you could go into an infinite loop.
