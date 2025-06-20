package main
import "sync"

func ptr[T any](p T) *T {
  return &p
}

func deref[T any](p *T) T {
  var d T
  if p!=nil{
    d = *p
  }
  return d
}


type WaitCond struct{
  cond *sync.Cond
  done func() bool
}

func NewWaitCond(done func() bool)WaitCond{
  return WaitCond{
    cond:sync.NewCond(&sync.Mutex{}),
    done:done,
  }
}

func (w *WaitCond) Broadcast(){
  defer func(){recover()}()
  w.cond.Broadcast()
}

func(w *WaitCond)Lock(){
    defer func(){recover()}()
    if w.done(){
      w.Broadcast()
    }
    w.cond.L.Lock()
}

func(w *WaitCond)Unlock(){
    defer func(){recover()}()
    if w.done(){
      w.Broadcast()
    }
    w.cond.L.Unlock()
}


func(w *WaitCond)Wait(){
  defer func(){recover()}()
  for !w.done(){
    w.cond.Wait()
  }
  w.cond.Broadcast()
}


type Promise[T any] struct {
	Channel chan T
	Error   error
	Done    *bool
	Result  T
  await *sync.Once
  cancel *sync.Once
  lock *WaitCond
}

func NewPromise[T any](fn func() T) Promise[T] {
	ch, err := GoChannel(fn)
	p := Promise[T]{
    Channel: ch,
    Error: err,
    Done:ptr(false),
    await: &sync.Once{},
    cancel: &sync.Once{},
  }
  p.lock = ptr(NewWaitCond(func()bool{return p.done()}))
  return p
}

func(p *Promise[T])done()bool{
  return deref(p.Done)
}

func (p *Promise[T]) Await() *Promise[T] {

  p.lock.Lock()
  if p.done() {
    p.lock.Unlock()
    return p
  }
  p.await.Do(func(){
    defer Close(p.Channel)
    p.Result, p.Error = Receive(p.Channel)
    *p.Done = true
    p.lock.Broadcast()
  })
  p.lock.Wait()
  p.lock.Unlock()
	return p
}

func (p *Promise[T]) Cancel() error {
	if p.done() {
    p.lock.Broadcast()
		return p.Error
	}
  p.cancel.Do(func(){
	  *p.Done = true
    p.Error = Close(p.Channel)
  })
  p.lock.Broadcast()
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
	Done   *bool
	Result T
  await *sync.Once
  lock *WaitCond
}

func NewThunk[T any](fn func() T) *Thunk[T] {
	t := &Thunk[T]{
    Func: fn,
    await: &sync.Once{},
    Done:ptr(false),
  }
  t.lock = ptr(NewWaitCond(func()bool{return t.done()}))
  return t
}

func(t *Thunk[T])done()bool{
  return deref(t.Done)
}

func (t *Thunk[T]) Await() *Thunk[T] {
  t.lock.Lock()
  if t.done() {
    t.lock.Unlock()
    return t
  }
  t.await.Do(func(){
	  t.Result = t.Func()
	  *t.Done = true
    t.lock.Broadcast()
  })
  t.lock.Wait()
  t.lock.Unlock()
	return t
}

func (t *Thunk[T]) Cancel() error {
	*t.Done = true
	return t.Error
}
