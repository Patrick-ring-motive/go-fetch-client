package main
import "sync"
type Promise[T any] struct {
	Channel chan T
	Error   error
	Done    bool
	Result  T
  await *sync.Once
  cancel *sync.Once
  lock *sync.Cond
}

func NewPromise[T any](fn func() T) Promise[T] {
	ch, err := GoChannel(fn)
	return Promise[T]{
    Channel: ch,
    Error: err,
    await: &sync.Once{},
    cancel: &sync.Once{},
    lock: sync.NewCond(&sync.Mutex{}),
  }
}

func (p *Promise[T]) Await() *Promise[T] {

  p.lock.L.Lock()
  if p.Done {
     p.lock.L.Unlock()
      return p
  }
  p.await.Do(func(){
    defer Close(p.Channel)
    p.Result, p.Error = Receive(p.Channel)
    p.Done = true
    p.lock.Broadcast()
  })
  for !p.Done {
      p.lock.Wait()
  }
  p.lock.L.Unlock()
	return p
}

func (p *Promise[T]) Cancel() error {
	if p.Done {
		return p.Error
	}
  p.cancel.Do(func(){
	  p.Done = true
    p.Error = Close(p.Channel)
  })
  return p.Error
}

type Option[T any] struct {
	Result T
	Error  error
}

func NewOption[T any](fn func() (T, error)) func() Option[T] {
	return func() Option[T] {
		t, err := fn()
		return Option[T]{t, err}
	}
}

type Thunk[T any] struct {
	Func   func() T
	Error  error
	Done   bool
	Result T
  await *sync.Once
  lock *sync.Cond
}

func NewThunk[T any](fn func() T) *Thunk[T] {
	return &Thunk[T]{
    Func: fn,
    await: &sync.Once{},
    lock: sync.NewCond(&sync.Mutex{}),
  }
}

func (t *Thunk[T]) Await() *Thunk[T] {
  t.lock.L.Lock()
  if t.Done {
      t.lock.L.Unlock()
      return t
  }
  t.await.Do(func(){
	  t.Result = t.Func()
	  t.Done = true
    t.lock.Broadcast()
  })
  for !t.Done {
      t.lock.Wait()
  }
  t.lock.L.Unlock()
	return t
}

func (t *Thunk[T]) Cancel() error {
	t.Done = true
	return t.Error
}
