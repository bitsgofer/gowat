package ratelimiter

import (
	"testing"
)

func TestWat(t *testing.T) {
	type call struct {
		ts      int64
		isAllow bool
	}
	var testCases = map[string]struct {
		windowLength, maxRPS int64
		callGroups           [][]call // groups of call that happen concurrently (same timestamp)
	}{
		"oneRPS": {
			windowLength: 1,
			maxRPS:       1,
			callGroups: [][]call{
				[]call{{0, true}, {0, false}}, // only one can pass
				[]call{{1, false}},            // same window with the one above
				[]call{{3, true}, {3, false}}, // new window, only one can pass
				[]call{{5, true}},             // new window
				[]call{{7, true}},             // new window
			},
		},
		"spread": {
			windowLength: 3,
			maxRPS:       2,
			callGroups: [][]call{ // will allow for at most 8 calls in a window (bcuz int division)
				[]call{{0, true}, {0, true}, {0, true}, {0, true}},             // all can pass
				[]call{{1, true}, {1, true}, {1, true}, {1, true}, {1, false}}, // one should fail
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(subT *testing.T) {
			limiter := New(tc.windowLength, tc.maxRPS)

			for _, callGroup := range tc.callGroups {
				n := len(callGroup)
				done := make(chan struct{}, n) // allow n concurrent, non-blocking send

				makeCall := func(c call) {
					// the following error checking code is logically wrong
					// (should count no of pass/fail instead).
					//
					// But it's not the focus.
					// Guess first, will this test ever deadlock?

					if want, got := c.isAllow, limiter.IsAllow(c.ts); want != got {
						// we have a deadlock with this line
						subT.Fatalf("IsAllow(%v), want= %v, got= %v", c.ts, want, got)

						// but if we comment out the Fatalf and use Logf/Errorf instead,
						// we will not have the deadlock
						// subT.Logf("IsAllow(%v), want= %v, got= %v", c.ts, want, got)

						// running with -race (at least for me) either remove the deadlock
						// or put the test/race detector into infinite loop!

						// the explanation may lie somewhere in testing.T's implementation
						// or how the actual test is compiled and generated
						// or maybe this doc on tesing.T:
						//
						// https://golang.org/pkg/testing/#T
						//
						// > A test ends when its Test function returns or calls any of the
						// > methods FailNow, Fatal, Fatalf, SkipNow, Skip, or Skipf.
						// > Those methods, as well as the Parallel method, must be called
						// > only from the goroutine running the Test function.
						//
						// welcome to the rabbit hole... (╯°□°)╯︵ ┻━┻
					}
					done <- struct{}{}
				}

				// make n concurrent calls
				for _, call := range callGroup {
					go makeCall(call)
				}
				// wait for n calls to finish
				for i := 0; i < n; i++ {
					<-done
				}
			}
		})
	}
}
