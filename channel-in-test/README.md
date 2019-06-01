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

Edit `concurrent_rate_limiter_test.go` and comment out [the line with `Fatalf`](https://github.com/bitsgofer/gowat/blob/master/channel-in-test/concurrent_rate_limiter_test.go#L54).
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

## Debugging

```
$ GO111MODULE=on go test -mod=vendor -c -o myTest



$ ./myTest -test.v
=== RUN   TestWat
=== RUN   TestWat/oneRPS
calling deferred func of makeCall
calling deferred func of makeCall
received once
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan receive]:
testing.(*T).Run(0xc000086100, 0x539e5e, 0x7, 0x5429f8, 0x469626)
        /usr/local/go/src/testing/testing.go:917 +0x381
testing.runTests.func1(0xc000086000)
        /usr/local/go/src/testing/testing.go:1157 +0x78
testing.tRunner(0xc000086000, 0xc000045e30)
        /usr/local/go/src/testing/testing.go:865 +0xc0
testing.runTests(0xc00000c0a0, 0x6280f0, 0x1, 0x1, 0x0)
        /usr/local/go/src/testing/testing.go:1155 +0x2a9
testing.(*M).Run(0xc000084000, 0x0)
        /usr/local/go/src/testing/testing.go:1072 +0x162
main.main()
        _testmain.go:42 +0x13e

goroutine 5 [chan receive]:
testing.(*T).Run(0xc000086300, 0x539cd4, 0x6, 0xc000020460, 0xc00000a268)
        /usr/local/go/src/testing/testing.go:917 +0x381
github.com/bitsgofer/gowat/channel-in-test.TestWat(0xc000086100)
        /home/dev/workspace/src/github.com/bitsgofer/gowat/channel-in-test/concurrent_rate_limiter_test.go:27 +0x1ec
testing.tRunner(0xc000086100, 0x5429f8)
        /usr/local/go/src/testing/testing.go:865 +0xc0
created by testing.(*T).Run
        /usr/local/go/src/testing/testing.go:916 +0x35a

goroutine 6 [chan receive]:
github.com/bitsgofer/gowat/channel-in-test.TestWat.func1(0xc000086300)
        /home/dev/workspace/src/github.com/bitsgofer/gowat/channel-in-test/concurrent_rate_limiter_test.go:83 +0xde
testing.tRunner(0xc000086300, 0xc000020460)
        /usr/local/go/src/testing/testing.go:865 +0xc0
created by testing.(*T).Run
        /usr/local/go/src/testing/testing.go:916 +0x35a



$ GO111MODULE=on dlv exec ./myTest
Type 'help' for list of commands.
(dlv) funcs TestWat
github.com/bitsgofer/gowat/channel-in-test.TestWat
github.com/bitsgofer/gowat/channel-in-test.TestWat.func1
github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.1
github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2
github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1
(dlv) b TestWat
Breakpoint 1 set at 0x4eeceb for github.com/bitsgofer/gowat/channel-in-test.TestWat() ./concurrent_rate_limiter_test.go:8
(dlv) c
> github.com/bitsgofer/gowat/channel-in-test.TestWat() ./concurrent_rate_limiter_test.go:8 (hits goroutine(18):1 total:1) (PC: 0x4eeceb)
Warning: debugging optimized function
     3: import (
     4:         "fmt"
     5:         "testing"
     6: )
     7:
=>   8: func TestWat(t *testing.T) {
     9:         type call struct {
    10:                 ts      int64
    11:                 isAllow bool
    12:         }
    13:         var testCases = map[string]struct {


(dlv) b 32
Breakpoint 2 set at 0x4ef01d for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.1() ./concurrent_rate_limiter_test.go:32
(dlv) b 41
Breakpoint 3 set at 0x4ef0ad for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41
(dlv) b 51
Breakpoint 4 set at 0x4ef1e8 for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:51
(dlv) b 83
Breakpoint 5 set at 0x4ef38c for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83
(dlv) bp
Breakpoint runtime-fatal-throw at 0x42b4f0 for runtime.fatalthrow() /usr/local/go/src/runtime/panic.go:663 (0)
Breakpoint unrecovered-panic at 0x42b560 for runtime.fatalpanic() /usr/local/go/src/runtime/panic.go:690 (0)
        print runtime.curg._panic.arg
Breakpoint 1 at 0x4eeceb for github.com/bitsgofer/gowat/channel-in-test.TestWat() ./concurrent_rate_limiter_test.go:8 (1)
Breakpoint 2 at 0x4ef01d for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.1() ./concurrent_rate_limiter_test.go:32 (0)
Breakpoint 3 at 0x4ef0ad for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (0)
Breakpoint 4 at 0x4ef1e8 for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:51 (0)
Breakpoint 5 at 0x4ef38c for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (0)


(dlv) c
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:51 (hits goroutine(20):1 total:1) (PC: 0x4ef1e8)
Warning: debugging optimized function
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (hits goroutine(19):1 total:1) (PC: 0x4ef38c)
Warning: debugging optimized function
    78:                                 for _, call := range callGroup {
    79:                                         go makeCall(call)
    80:                                 }
    81:                                 // wait for n calls to finish
    82:                                 for i := 0; i < n; i++ { // style 1
=>  83:                                         <-done
    84:                                         fmt.Println("received once")
    85:                                 }
    86:                                 // wg.Wait() // style 2
    87:                         }
    88:                 })
(dlv) grs
  Goroutine 1 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 3 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 18 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
* Goroutine 19 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef38c) (thread 4846)
  Goroutine 20 - User: ./concurrent_rate_limiter_test.go:51 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2 (0x4ef1e8) (thread 4850)
  Goroutine 21 - User: ./concurrent_rate_limiter_test.go:39 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2 (0x4ef120)
[8 goroutines]


(dlv) gr 20
Switched from 19 to 20 (thread 4850)
(dlv) l
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (hits goroutine(19):1 total:1) (PC: 0x4ef38c)
Warning: debugging optimized function
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:51 (hits goroutine(20):1 total:1) (PC: 0x4ef1e8)
Warning: debugging optimized function
    46:                                         // But it's not the focus.
    47:                                         // Guess first, will this test ever deadlock?
    48:
    49:                                         if want, got := c.isAllow, limiter.IsAllow(c.ts); want != got {
    50:                                                 // we have a deadlock with this line
=>  51:                                                 subT.Fatalf("IsAllow(%v), want= %v, got= %v", c.ts, want, got)
    52:
    53:                                                 // but if we comment out the Fatalf and use Logf/Errorf instead,
    54:                                                 // we will not have the deadlock
    55:                                                 // subT.Logf("IsAllow(%v), want= %v, got= %v", c.ts, want, got)
    56:
(dlv) gr 21
Switched from 20 to 21 (thread 4850)
(dlv) l
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (hits goroutine(19):1 total:1) (PC: 0x4ef38c)
Warning: debugging optimized function
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:39 (PC: 0x4ef120)
Warning: debugging optimized function
    34:                                 n := len(callGroup)
    35:                                 done := make(chan struct{}, n) // allow n concurrent, non-blocking send // style 1
    36:                                 // wg := &sync.WaitGroup{} // style 2
    37:                                 // wg.Add(n)
    38:
=>  39:                                 makeCall := func(c call) {
    40:                                         defer func() {
    41:                                                 fmt.Println("calling deferred func of makeCall")
    42:                                         }()
    43:                                         // the following error checking code is logically wrong
    44:                                         // (should count no of pass/fail instead).


(dlv) c
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(20):1 total:2) (PC: 0x4ef0ad)
Warning: debugging optimized function
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(21):1 total:2) (PC: 0x4ef0ad)
Warning: debugging optimized function
    36:                                 // wg := &sync.WaitGroup{} // style 2
    37:                                 // wg.Add(n)
    38:
    39:                                 makeCall := func(c call) {
    40:                                         defer func() {
=>  41:                                                 fmt.Println("calling deferred func of makeCall")
    42:                                         }()
    43:                                         // the following error checking code is logically wrong
    44:                                         // (should count no of pass/fail instead).
    45:                                         //
    46:                                         // But it's not the focus.
(dlv) grs
  Goroutine 1 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 3 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 18 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 19 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef39e)
  Goroutine 20 - User: ./concurrent_rate_limiter_test.go:41 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1 (0x4ef0ad) (thread 4850)
* Goroutine 21 - User: ./concurrent_rate_limiter_test.go:41 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1 (0x4ef0ad) (thread 4846)
[8 goroutines]
(dlv) gr 20
Switched from 21 to 20 (thread 4850)
(dlv) l
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(21):1 total:2) (PC: 0x4ef0ad)
Warning: debugging optimized function
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(20):1 total:2) (PC: 0x4ef0ad)
Warning: debugging optimized function
    36:                                 // wg := &sync.WaitGroup{} // style 2
    37:                                 // wg.Add(n)
    38:
    39:                                 makeCall := func(c call) {
    40:                                         defer func() {
=>  41:                                                 fmt.Println("calling deferred func of makeCall")
    42:                                         }()
    43:                                         // the following error checking code is logically wrong
    44:                                         // (should count no of pass/fail instead).
    45:                                         //
    46:                                         // But it's not the focus.



(dlv) grs
  Goroutine 1 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 3 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 18 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 19 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef39e)
* Goroutine 20 - User: ./concurrent_rate_limiter_test.go:41 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1 (0x4ef0ad) (thread 4850)
  Goroutine 21 - User: ./concurrent_rate_limiter_test.go:41 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1 (0x4ef0ad) (thread 4846)
[8 goroutines]
(dlv) gr 19
Switched from 20 to 19 (thread 4850)
(dlv) l
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(21):1 total:2) (PC: 0x4ef0ad)
Warning: debugging optimized function
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:421 (PC: 0x405fde)
Warning: debugging optimized function
   416: // ep may be nil, in which case received data is ignored.
   417: // If block == false and no elements are available, returns (false, false).
   418: // Otherwise, if c is closed, zeros *ep and returns (true, false).
   419: // Otherwise, fills in *ep with an element and returns (true, true).
   420: // A non-nil ep must point to the heap or the caller's stack.
=> 421: func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
   422:         // raceenabled: don't need to check ep, as it is always on the stack
   423:         // or is new memory allocated by reflect.
   424:
   425:         if debugChan {
   426:                 print("chanrecv: chan=", c, "\n")


(dlv) n
calling deferred func of makeCall
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:429 (PC: 0x40595f)
Warning: debugging optimized function
   424:
   425:         if debugChan {
   426:                 print("chanrecv: chan=", c, "\n")
   427:         }
   428:
=> 429:         if c == nil {
   430:                 if !block {
   431:                         return
   432:                 }
   433:                 gopark(nil, nil, waitReasonChanReceiveNilChan, traceEvGoStop, 2)
   434:                 throw("unreachable")


(dlv) n
calling deferred func of makeCall
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:449 (PC: 0x40597a)
Warning: debugging optimized function
   444:         // first observation. We behave as if we observed the channel at that moment
   445:         // and report that the receive cannot proceed.
   446:         //
   447:         // The order of operations is important here: reversing the operations can lead to
   448:         // incorrect behavior when racing with a close.
=> 449:         if !block && (c.dataqsiz == 0 && c.sendq.first == nil ||
   450:                 c.dataqsiz > 0 && atomic.Loaduint(&c.qcount) == 0) &&
   451:                 atomic.Load(&c.closed) == 0 {
   452:                 return
   453:         }
   454:
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:450 (PC: 0x405982)
Warning: debugging optimized function
   445:         // and report that the receive cannot proceed.
   446:         //
   447:         // The order of operations is important here: reversing the operations can lead to
   448:         // incorrect behavior when racing with a close.
   449:         if !block && (c.dataqsiz == 0 && c.sendq.first == nil ||
=> 450:                 c.dataqsiz > 0 && atomic.Loaduint(&c.qcount) == 0) &&
   451:                 atomic.Load(&c.closed) == 0 {
   452:                 return
   453:         }
   454:
   455:         var t0 int64
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:449 (PC: 0x405992)
Warning: debugging optimized function
   444:         // first observation. We behave as if we observed the channel at that moment
   445:         // and report that the receive cannot proceed.
   446:         //
   447:         // The order of operations is important here: reversing the operations can lead to
   448:         // incorrect behavior when racing with a close.
=> 449:         if !block && (c.dataqsiz == 0 && c.sendq.first == nil ||
   450:                 c.dataqsiz > 0 && atomic.Loaduint(&c.qcount) == 0) &&
   451:                 atomic.Load(&c.closed) == 0 {
   452:                 return
   453:         }
   454:



(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:456 (PC: 0x40599a)
Warning: debugging optimized function
   451:                 atomic.Load(&c.closed) == 0 {
   452:                 return
   453:         }
   454:
   455:         var t0 int64
=> 456:         if blockprofilerate > 0 {
   457:                 t0 = cputicks()
   458:         }
   459:
   460:         lock(&c.lock)
   461:
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:460 (PC: 0x4059af)
Warning: debugging optimized function
   455:         var t0 int64
   456:         if blockprofilerate > 0 {
   457:                 t0 = cputicks()
   458:         }
   459:
=> 460:         lock(&c.lock)
   461:
   462:         if c.closed != 0 && c.qcount == 0 {
   463:                 if raceenabled {
   464:                         raceacquire(c.raceaddr())
   465:                 }
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:462 (PC: 0x4059c1)
Warning: debugging optimized function
   457:                 t0 = cputicks()
   458:         }
   459:
   460:         lock(&c.lock)
   461:
=> 462:         if c.closed != 0 && c.qcount == 0 {
   463:                 if raceenabled {
   464:                         raceacquire(c.raceaddr())
   465:                 }
   466:                 unlock(&c.lock)
   467:                 if ep != nil {


(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:473 (PC: 0x4059d9)
Warning: debugging optimized function
   468:                         typedmemclr(c.elemtype, ep)
   469:                 }
   470:                 return true, false
   471:         }
   472:
=> 473:         if sg := c.sendq.dequeue(); sg != nil {
   474:                 // Found a waiting sender. If buffer is size 0, receive value
   475:                 // directly from sender. Otherwise, receive from head of queue
   476:                 // and add sender's value to the tail of the queue (both map to
   477:                 // the same buffer slot because the queue is full).
   478:                 recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:482 (PC: 0x4059f4)
Warning: debugging optimized function
   477:                 // the same buffer slot because the queue is full).
   478:                 recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
   479:                 return true, true
   480:         }
   481:
=> 482:         if c.qcount > 0 {
   483:                 // Receive directly from queue
   484:                 qp := chanbuf(c, c.recvx)
   485:                 if raceenabled {
   486:                         raceacquire(qp)
   487:                         racerelease(qp)
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:484 (PC: 0x405a07)
Warning: debugging optimized function
   479:                 return true, true
   480:         }
   481:
   482:         if c.qcount > 0 {
   483:                 // Receive directly from queue
=> 484:                 qp := chanbuf(c, c.recvx)
   485:                 if raceenabled {
   486:                         raceacquire(qp)
   487:                         racerelease(qp)
   488:                 }
   489:                 if ep != nil {

(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/stubs.go:12 (PC: 0x405a14)
Warning: debugging optimized function
     7: import "unsafe"
     8:
     9: // Should be a built-in for unsafe.Pointer?
    10: //go:nosplit
    11: func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
=>  12:         return unsafe.Pointer(uintptr(p) + x)
    13: }
    14:
    15: // getg returns the pointer to the current g.
    16: // The compiler rewrites calls to this function into instructions
    17: // that fetch the g directly (from TLS or from the dedicated register).
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:489 (PC: 0x405a18)
Warning: debugging optimized function
   484:                 qp := chanbuf(c, c.recvx)
   485:                 if raceenabled {
   486:                         raceacquire(qp)
   487:                         racerelease(qp)
   488:                 }
=> 489:                 if ep != nil {
   490:                         typedmemmove(c.elemtype, ep, qp)
   491:                 }
   492:                 typedmemclr(c.elemtype, qp)
   493:                 c.recvx++
   494:                 if c.recvx == c.dataqsiz {
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:492 (PC: 0x405a25)
Warning: debugging optimized function
   487:                         racerelease(qp)
   488:                 }
   489:                 if ep != nil {
   490:                         typedmemmove(c.elemtype, ep, qp)
   491:                 }
=> 492:                 typedmemclr(c.elemtype, qp)
   493:                 c.recvx++
   494:                 if c.recvx == c.dataqsiz {
   495:                         c.recvx = 0
   496:                 }
   497:                 c.qcount--


(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:493 (PC: 0x405a37)
Warning: debugging optimized function
   488:                 }
   489:                 if ep != nil {
   490:                         typedmemmove(c.elemtype, ep, qp)
   491:                 }
   492:                 typedmemclr(c.elemtype, qp)
=> 493:                 c.recvx++
   494:                 if c.recvx == c.dataqsiz {
   495:                         c.recvx = 0
   496:                 }
   497:                 c.qcount--
   498:                 unlock(&c.lock)
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:494 (PC: 0x405a4a)
Warning: debugging optimized function
   489:                 if ep != nil {
   490:                         typedmemmove(c.elemtype, ep, qp)
   491:                 }
   492:                 typedmemclr(c.elemtype, qp)
   493:                 c.recvx++
=> 494:                 if c.recvx == c.dataqsiz {
   495:                         c.recvx = 0
   496:                 }
   497:                 c.qcount--
   498:                 unlock(&c.lock)
   499:                 return true, true
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:497 (PC: 0x405a58)
Warning: debugging optimized function
   492:                 typedmemclr(c.elemtype, qp)
   493:                 c.recvx++
   494:                 if c.recvx == c.dataqsiz {
   495:                         c.recvx = 0
   496:                 }
=> 497:                 c.qcount--
   498:                 unlock(&c.lock)
   499:                 return true, true
   500:         }
   501:
   502:         if !block {



(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:498 (PC: 0x405a5b)
Warning: debugging optimized function
   493:                 c.recvx++
   494:                 if c.recvx == c.dataqsiz {
   495:                         c.recvx = 0
   496:                 }
   497:                 c.qcount--
=> 498:                 unlock(&c.lock)
   499:                 return true, true
   500:         }
   501:
   502:         if !block {
   503:                 unlock(&c.lock)
(dlv) n
> runtime.chanrecv() /usr/local/go/src/runtime/chan.go:499 (PC: 0x405a69)
Warning: debugging optimized function
   494:                 if c.recvx == c.dataqsiz {
   495:                         c.recvx = 0
   496:                 }
   497:                 c.qcount--
   498:                 unlock(&c.lock)
=> 499:                 return true, true
   500:         }
   501:
   502:         if !block {
   503:                 unlock(&c.lock)
   504:                 return false, false
(dlv) n
> runtime.chanrecv1() /usr/local/go/src/runtime/chan.go:407 (PC: 0x40591b)
Warning: debugging optimized function
Values returned:
        selected: (unreadable could not find loclist entry at 0x4841 for address 0x405930)
        received: (unreadable could not find loclist entry at 0x48ed for address 0x405930)

   402:
   403: // entry points for <- c from compiled code
   404: //go:nosplit
   405: func chanrecv1(c *hchan, elem unsafe.Pointer) {
   406:         chanrecv(c, elem, true)
=> 407: }
   408:
   409: //go:nosplit
   410: func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool) {
   411:         _, received = chanrecv(c, elem, true)
   412:         return


(dlv) n
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:84 (PC: 0x4ef39e)
Warning: debugging optimized function
Values returned:

    79:                                         go makeCall(call)
    80:                                 }
    81:                                 // wait for n calls to finish
    82:                                 for i := 0; i < n; i++ { // style 1
    83:                                         <-done
=>  84:                                         fmt.Println("received once")
    85:                                 }
    86:                                 // wg.Wait() // style 2
    87:                         }
    88:                 })
    89:         }
(dlv) n
received once
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:82 (PC: 0x4ef403)
Warning: debugging optimized function
    77:                                 // make n concurrent calls
    78:                                 for _, call := range callGroup {
    79:                                         go makeCall(call)
    80:                                 }
    81:                                 // wait for n calls to finish
=>  82:                                 for i := 0; i < n; i++ { // style 1
    83:                                         <-done
    84:                                         fmt.Println("received once")
    85:                                 }
    86:                                 // wg.Wait() // style 2
    87:                         }


(dlv) n
> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (hits goroutine(19):2 total:2) (PC: 0x4ef38c)
Warning: debugging optimized function
    78:                                 for _, call := range callGroup {
    79:                                         go makeCall(call)
    80:                                 }
    81:                                 // wait for n calls to finish
    82:                                 for i := 0; i < n; i++ { // style 1
=>  83:                                         <-done
    84:                                         fmt.Println("received once")
    85:                                 }
    86:                                 // wg.Wait() // style 2
    87:                         }
    88:                 })
(dlv) n
fatal error: all goroutines are asleep - deadlock!
> [runtime-fatal-throw] runtime.fatalthrow() /usr/local/go/src/runtime/panic.go:663 (hits total:1) (PC: 0x42b4f0)
Warning: debugging optimized function
Command failed: no G executing on thread 0


(dlv) grs
  Goroutine 1 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 3 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
  Goroutine 18 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
  Goroutine 19 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef39e)
[6 goroutines]
```
