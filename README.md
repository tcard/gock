# gock [![Build Status](https://secure.travis-ci.org/tcard/gock.svg?branch=master)](http://travis-ci.org/tcard/gock) [![GoDoc](https://godoc.org/github.com/tcard/gock?status.svg)](https://godoc.org/github.com/tcard/gock)

Package gock (a portmanteau of the `go` statement and "block") provides [structured concurrency](https://vorpus.org/blog/notes-on-structured-concurrency-or-go-statement-considered-harmful/) utilities for Go.

```go
things := make(chan Thing)

err := gock.Wait(func() error {
	defer close(things)
	return Produce(ctx, things)
}, func() error {
	return Consume(things)
})

// Both Produce and Consume are done here, and err is their combined errors.
```
