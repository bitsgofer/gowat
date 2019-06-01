## What's this?

This is a debug story.
I want to document this debugging process down, plus anything learnt from it.

The bug happened because the code didn't do what I expected it to do. This means my mental
model has divereged from reality. And it happened in such a subtle way that I wasn't able to
understand what happened at a glance.

Having the debugging story here will hopefully allow learning and refining that mental model.

## The story

I was trying to write a rate limiter that can deal with concurrent requests in Go.
To check one implementation, I wrote some tests that would involved:

- Call `IsAllow(timestamp)` N times, concurrently.
- For each call, I would expect whether it should be allowed or not.
- The test should fail when the result is unexpected.

This seems reasonable at first, so I proceeded to write the test.
However, when running it, I got a deadlock, which puzzled me.

P.S: you can jump to [the extra part](#) to see what's wrong with the spec above.
Hint: it's related to temporal properties (how the system behave over time).

Anw, while the problem was with me, tracing this down helped me learn to debug Go code better,
so it's not all bad :D.

> Much thanks to my colleagues: `jay7x` and `choonkeat` for pitching in ideas/code :)

## Reproduce

The test's code [is linked here](#). It's quite bloated as I tried to annotate the problem
and print out what's going on. For the most part, you can ignore the comments. I only keep
them there so the debugging log matches what happened.

The system I'm working on

	$> go version
	go version go1.12.4 linux/amd64

	$> go env
	GOARCH="amd64"
	GOHOSTARCH="amd64"
	GOHOSTOS="linux"
	GOOS="linux"
	GORACE=""
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

While I started running this as normal with `go test -mod=vendor -v ./...`, it occurred to
me that I could compile the test into a binary as well.
This removes the doubt of whether I'm running the same thing when I repeat the debugging steps.

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

What we see from the output:

- `makeCall` was called twice and ran to completion each time (as we saw its deferred calls).
- one msg was sent to `done` and the [waiting code](#) saw it.
- then we got a deadlock

At this point, keen readers will see the obvious thing: one of the `makeCall` function fail, call
`Fatalf` and don't send anything to the `done` channel. Meanwhile, because I want to wait for
**two** messages, a deadlock will definitely occur.

Well, that's it, really! However, it was not this clear when I was looking at it. What I expected
was that the `Fatalf` call would fail my test and terminate it there.

## Debug

To understand what I did wrong here, I used [delve](https://github.com/go-delve/delve) to trace:

	$ GO111MODULE=on dlv exec ./myTest
	Type 'help' for list of commands.
	(dlv) b TestWat
	Breakpoint 1 set at 0x4eeceb for github.com/bitsgofer/gowat/channel-in-test.TestWat() ./concurrent_rate_limiter_test.go:8
	(dlv) c
	> github.com/bitsgofer/gowat/channel-in-test.TestWat() ./concurrent_rate_limiter_test.go:8 (hits goroutine(19):1 total:1) (PC: 0x4eeceb)
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
	(dlv) b 41
	Breakpoint 2 set at 0x4ef0ad for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41
	(dlv) b 51
	Breakpoint 3 set at 0x4ef1e8 for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:51
	(dlv) b 73
	Breakpoint 4 set at 0x4ef1b6 for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:73
	(dlv) b 83
	Breakpoint 5 set at 0x4ef38c for github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83


I first set breakpoints at:

- [L41](#): to see that `makeCall` went through what's inside before moving on to its deferred calls.
- [L51](#): right before call to `Fatalf`
- [L73](#): right before sending a signal to the `done` channel
- [L83](#): right before receiving a signal from the `done` channel

Then as we continue to run till the first break point:

	(dlv) c
	> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:51 (hits goroutine(21):1 total:1) (PC: 0x4ef1e8)
	Warning: debugging optimized function
	> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (hits goroutine(20):1 total:1) (PC: 0x4ef38c)
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
	(dlv) bt
	0  0x00000000004ef38c in github.com/bitsgofer/gowat/channel-in-test.TestWat.func1
	   at ./concurrent_rate_limiter_test.go:83
	1  0x00000000004b3950 in testing.tRunner
	   at /usr/local/go/src/testing/testing.go:865
	2  0x0000000000457a41 in runtime.goexit
	   at /usr/local/go/src/runtime/asm_amd64.s:1337
	(dlv) grs
	  Goroutine 1 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
	  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 18 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 19 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
	* Goroutine 20 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef38c) (thread 5034)
	  Goroutine 21 - User: ./concurrent_rate_limiter_test.go:51 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2 (0x4ef1e8) (thread 5041)
	  Goroutine 22 - User: ./concurrent_rate_limiter_test.go:39 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2 (0x4ef120)
	[8 goroutines]
```

The stack trace (`bt`) showed that we start from `tRunner` (the test runner) and went down into
[L83](#), waiting for a signal.

Meanwhile, the interesting goroutines are:

- 20: the one we are on, called from `tRunner`
- 21: one `makeCall`. It currently stopped at [L51](#), where `Fatalf` is
- 22: another `makeCall`. This one have not started executing yet.
  Because we have setup one call to fail (happend in goroutine 21), this one should not fail.


Let's continue.

	(dlv) c
	> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(21):1 total:1) (PC: 0x4ef0ad)
	Warning: debugging optimized function
	> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2() ./concurrent_rate_limiter_test.go:73 (hits goroutine(22):1 total:1) (PC: 0x4ef1b6)
	Warning: debugging optimized function
	    68:                                                 // > Those methods, as well as the Parallel method, must be called
	    69:                                                 // > only from the goroutine running the Test function.
	    70:                                                 //
	    71:                                                 // welcome to the rabbit hole... (╯°□°)╯︵ ┻━┻
	    72:                                         }
	=>  73:                                         done <- struct{}{} // style 1
	    74:                                         // wg.Done() // style 2
	    75:                                 }
	    76:
	    77:                                 // make n concurrent calls
	    78:                                 for _, call := range callGroup {
	(dlv) grs
	  Goroutine 1 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
	  Goroutine 2 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 18 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 19 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
	  Goroutine 20 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef39e)
	  Goroutine 21 - User: ./concurrent_rate_limiter_test.go:41 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1 (0x4ef0ad) (thread 5041)
	* Goroutine 22 - User: ./concurrent_rate_limiter_test.go:73 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2 (0x4ef1b6) (thread 5034)
	[8 goroutines]

Here, we see goroutine 22 suceeding and is about to send a signal on `done`.

Let's continue again.

	(dlv) c
	calling deferred func of makeCall
	> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1() ./concurrent_rate_limiter_test.go:41 (hits goroutine(22):1 total:2) (PC: 0x4ef0ad)
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
	  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 18 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 19 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
	  Goroutine 20 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef39e)
	* Goroutine 22 - User: ./concurrent_rate_limiter_test.go:41 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1.2.1 (0x4ef0ad) (thread 5034)
	[7 goroutines]

We see one `calling deferred func of makeCall` printed out. At the same time, goroutine 21
disappeared. Here, it's reasonable to guess that the print is caused by goroutine 21's deferred
call.

Moving on.

	(dlv) c
	calling deferred func of makeCall
	received once
	> github.com/bitsgofer/gowat/channel-in-test.TestWat.func1() ./concurrent_rate_limiter_test.go:83 (hits goroutine(20):2 total:2) (PC: 0x4ef38c)
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
	  Goroutine 17 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 18 - User: /usr/local/go/src/runtime/proc.go:302 runtime.gopark (0x42d0cf)
	  Goroutine 19 - User: /usr/local/go/src/testing/testing.go:917 testing.(*T).Run (0x4b3d31)
	* Goroutine 20 - User: ./concurrent_rate_limiter_test.go:83 github.com/bitsgofer/gowat/channel-in-test.TestWat.func1 (0x4ef38c) (thread 5034)
	[6 goroutines]

We see another `calling deferred func of makeCall` and `received once`. Furthermore, one more
goroutine disappeared. This matches with what should have happened: goroutine 22
sent the singal on `done`, call its deferred function and exit. Meanwhile goroutine 20 received
a signal, print nd stop at the break point again.

Let's try to move to the next line: [L84](#)

	(dlv) n
	fatal error: all goroutines are asleep - deadlock!
	> [runtime-fatal-throw] runtime.fatalthrow() /usr/local/go/src/runtime/panic.go:663 (hits total:1) (PC: 0x42b4f0)
	Warning: debugging optimized function
	Command failed: no G executing on thread 0

Now we have our deadlock. As I mentioned earlier, it's obvious now that we saw the steps. We can't
expect a second signal ever so there's no way to continue.

## Conclusion

What I missed out was in fact right there in the doc for [testing.T](#) (emphasis mine):

> A test ends when its Test function returns or calls any of the methods FailNow, Fatal,
> Fatalf, SkipNow, Skip, or Skipf.
> Those methods, as well as the Parallel method,
> **must be called only from the goroutine running the Test function**.

While it's tempting to just say RTFM. I wouldn't have internalized o debugging till here.

`ノ┬─┬ノ ︵ ( \o°o)\`

## Extra

Now back to how to write this test properly.

As I have mentioned at the beginning, the test would not work as it did not take temporal properties
of the system into account. In English, this basically means:

- When we make N concurrent calls, there should be no expected order for which call is made first.
- When these concurrent calls reach the rate limiter, they are **queued** due to the mutex,
  and hence have an implicit order. However, this information is exclusive to the receiver side,
  not the caller (where we do our checks).
- So when I attach the expected result for `IsAllow` to my call, I did the wrong thing.
- There are better ways to specify this behavior:
  - Make the checks on server side instead: 1st call reaching the limiter passes, 2nd call fails.
  - Verify an invariant properties of all the calls: out of 2 call, 1 pass and the other fails,
    regardless of order. If we count the result, it would be fine, too.
