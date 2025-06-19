package main

import (
	"fmt"
)

func Send[T any, C chan T](ch C, value T) error {
	if ch == nil {
		return fmt.Errorf("Trying to send on nil channel: %v", value)
	}
	var err error
	(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic and recover on channel send: %v", r)
			}
		}()
		ch <- value
	})()
	return err
}

func Receive[T any, C chan T](ch C) (T, error) {
	var result T
	var ok bool
	if ch == nil {
		return result, fmt.Errorf("Trying to receive on nil channel: %v", ch)
	}
	var err error
	(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic and recover on channel receive: %v", r)
			}
		}()
		result, ok = <-ch
		if !ok {
			err = fmt.Errorf("result from receive channel not ok: %v", result)
		}
	})()
	return result, err
}

func Close[T any, C chan T](ch C) error {
	if ch == nil {
		return fmt.Errorf("Trying to close nil channel: %v", ch)
	}
	var err error
	(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic and recover on channel close: %v", r)
			}
		}()
		close(ch)
	})()
	return err
}

func Go(fn func()) error {
	var err error
	(func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic and recover on goroutine: %v", r)
			}
		}()
		go (func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic and recover on goroutine: %v", r)
				}
			}()
			fn()
		})()
	})()
	return err
}

func GoChannel[T any, C chan T](fn func() T) (C, error) {
	ch := make(chan T,1)
	err := Go(func() {
		Send(ch, fn())
	})
	return ch, err
}
