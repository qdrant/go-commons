package main

import (
	"errors"
	"fmt"
	"log/slog"

	errhelper "github.com/qdrant/go-commons/pkg/errors"
)

func main() {
	// call some function that produces error
	err := someCodeProducesError()
	if err != nil {
		// log the error with context
		slog.Error(fmt.Sprintf("something went wrong: %s", err.Error()), errhelper.GetMetadata(err)...)
	}
}

func someCodeProducesError() error {
	// root wrapper that include function name as context
	errMetadata := errhelper.Metadata{"function", "someCodeProducesError"}

	// another block of code that can produce error
	err := doSomething()
	if err != nil {
		// wrap the error with additional context
		// the resulting error will include the context from root wrapper and this wrapper
		return errhelper.WithMetadata(err, errMetadata.Extend("key1", "value1")...)
	}

	// this will only wrap error with root wrapper context
	return errhelper.WithMetadata(errors.New("foo"), errMetadata...)
}

func doSomething() error {
	// simulate an error
	return errhelper.WithMetadata(
		errors.New("bar"),
		"function", "doSomething",
		"key2", "value2",
	)
}
