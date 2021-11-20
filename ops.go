package fut

import (
	"context"
	"sync"
)

func New[Out any, C CallOut[Out]](c C) Fut[Out] {
	// TODO(cretz): Consider not memoizing by default?
	// TODO(cretz): Make chaining cheaper?
	var once sync.Once
	doneCh := make(chan struct{})
	var out Out
	var err error
	return func(ctx context.Context) (<-chan struct{}, *Out, *error) {
		go once.Do(func() {
			defer close(doneCh)
			// If context closed, don't even run
			if ctxErr := ctx.Err(); ctxErr != nil {
				err = ctxErr
				return
			}
			// Run async
			var localOut Out
			var localErr error
			localDone := make(chan interface{}, 1)
			go func() {
				defer close(localDone)
				// Handle panics
				defer func() {
					if v := recover(); v != nil {
						if panicHandler, _ := ctx.Value(PanicHandlerKey).(PanicHandler); panicHandler != nil {
							localErr = panicHandler(v)
							if localErr == nil {
								localErr = &ErrPanic{Value: v}
							}
						} else {
							panic(v)
						}
					}
				}()

				localOut, localErr = InvokeCallOut[Out](ctx, c)
			}()
			// Wait for context or async done
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-localDone:
				if localErr != nil {
					err = localErr
				} else {
					out = localOut
				}
			}
		})
		return doneCh, &out, &err
	}
}

func Wait[Out any](ctx context.Context, f Fut[Out]) (out Out, err error) {
	if ctx.Err() != nil {
		err = ctx.Err()
		return
	}
	done, outPtr, errPtr := f(ctx)
	select {
	case <-ctx.Done():
		return out, ctx.Err()
	case <-done:
		return *outPtr, *errPtr
	}
}

func MustWait[Out any](ctx context.Context, f Fut[Out]) Out {
	out, err := Wait(ctx, f)
	if err != nil {
		panic(err)
	}
	return out
}

var alwaysDone = make(chan struct{})

func init() { close(alwaysDone) }

func Fixed[Out any](v Out) Fut[Out] {
	return func(context.Context) (<-chan struct{}, *Out, *error) {
		return alwaysDone, &v, nil
	}
}

func Err[Out any](err error) Fut[Out] {
	return func(context.Context) (<-chan struct{}, *Out, *error) {
		return alwaysDone, nil, &err
	}
}

func Start[Out any, C CallOut[Out]](ctx context.Context, c C) Fut[Out] {
	f := New[Out](c)
	go f(ctx)
	return f
}

func Then[In, Out any, C CallInOut[In, Out]](f Fut[In], c C) Fut[Out] {
	return New(func(ctx context.Context) (out Out, err error) {
		in, err := Wait(ctx, f)
		if err != nil {
			return out, err
		}
		// TODO(cretz): If this is just "return InvokeCallInOut" an ICE occurs
		out, err = InvokeCallInOut[In, Out](ctx, c, in)
		return
	})
}

func ThenFut[In, Out any, C CallInOut[In, Fut[Out]]](f Fut[In], c C) Fut[Out] {
	return New(func(ctx context.Context) (out Out, err error) {
		in, err := Wait(ctx, f)
		if err != nil {
			return out, err
		}
		futOut, err := InvokeCallInOut[In, Fut[Out]](ctx, c, in)
		if err != nil {
			return out, err
		}
		// TODO(cretz): If this is just "return Wait" an ICE occurs
		out, err = Wait(ctx, futOut)
		return
	})
}

func ThenErr[In any, C CallInOut[error, In]](f Fut[In], c C) Fut[In] {
	return New(func(ctx context.Context) (in In, err error) {
		in, err = Wait(ctx, f)
		if err != nil {
			in, err = InvokeCallInOut[error, In](ctx, c, err)
		}
		return
	})
}

func ThenErrFut[In any, C CallInOut[error, Fut[In]]](f Fut[In], c C) Fut[In] {
	return New(func(ctx context.Context) (in In, err error) {
		in, err = Wait(ctx, f)
		if err != nil {
			if inFut, errFut := InvokeCallInOut[error, Fut[In]](ctx, c, err); errFut != nil {
				err = errFut
			} else {
				in, err = Wait(ctx, inFut)
			}
		}
		return
	})
}

// Does run on panic, error is ignored
func Defer[In any, C Call](f Fut[In], c C) Fut[In] {
	return New(func(ctx context.Context) (in In, err error) {
		defer c(ctx)
		// TODO(cretz): If this is just "return Wait" an ICE occurs
		in, err = Wait(ctx, f)
		return
	})
}

// Does not run on panic, error is ignored
func DeferOK[In any, C CallIn[In]](f Fut[In], c C) Fut[In] {
	return New(func(ctx context.Context) (in In, err error) {
		success := false
		defer func() {
			if success {
				InvokeCallIn[In](ctx, c, in)
			}
		}()
		in, err = Wait(ctx, f)
		success = err == nil
		return
	})
}

// Does not run on panic, error is ignored
func DeferErr[In any, C CallIn[error]](f Fut[In], c C) Fut[In] {
	return New(func(ctx context.Context) (in In, err error) {
		defer func() {
			if err != nil {
				InvokeCallIn[error](ctx, c, err)
			}
		}()
		in, err = Wait(ctx, f)
		return
	})
}

func WithContext[In any](f Fut[In], ctx context.Context) Fut[In] {
	return New(func(context.Context) (in In, err error) {
		in, err = Wait(ctx, f)
		return
	})
}

// Not implemented yet
//
// TODO(cretz): Wait for https://github.com/golang/go/issues/36503 or do
// something like https://github.com/monogon-dev/monogon/blob/main/metropolis/pkg/combinectx/combinectx.go
func WithMergedContext[In any](v Fut[In], ctx context.Context) Fut[In] {
	panic("Not implemented")
}

// This starts execution
func Recv[Out any](ctx context.Context, f Fut[Out]) (<-chan Out, <-chan error) {
	outCh := make(chan Out, 1)
	errCh := make(chan error, 1)
	go func() {
		if out, err := Wait(ctx, f); err != nil {
			errCh <- err
		} else {
			outCh <- out
		}
	}()
	return outCh, errCh
}

// Same as Recv but only the success side (never sent to on failure)
func OKRecv[Out any](ctx context.Context, f Fut[Out]) <-chan Out {
	outCh, _ := Recv(ctx, f)
	return outCh
}

// Same as Recv but only the error side (never sent to on failure)
func ErrRecv[Out any](ctx context.Context, f Fut[Out]) <-chan error {
	_, errCh := Recv(ctx, f)
	return errCh
}

// Does not block, but needs context since it may start execution
func Poll[Out any](ctx context.Context, f Fut[Out]) (*Out, error) {
	done, outPtr, errPtr := f(ctx)
	select {
	case <-done:
		if errPtr != nil {
			return nil, *errPtr
		}
		return outPtr, nil
	default:
		return nil, nil
	}
}

func As[In, Out any](f Fut[In]) Fut[Out] {
	// TODO(cretz): This does not compile
	// return New(func(ctx context.Context) (out Out, err error) {
	// 	in, err := Wait(ctx, f)
	// 	if err != nil {
	// 		return out, err
	// 	}
	// 	out = in.(Out)
	// 	return out, nil
	// })
	panic("TODO")
}

func AsMaybe[In, Out any](f Fut[In]) Fut[*Out] {
	// TODO(cretz): Type assert
	panic("TODO")
}

// Fails on first error, executes all async
func All[Out any](f ...Fut[Out]) Fut[[]Out] {
	return New(func(ctx context.Context) ([]Out, error) {
		outs := make([]Out, len(f))
		// Nil error means it succeeded
		errCh := make(chan error, len(f))
		for i, fOut := range f {
			go func(i int, fOut Fut[Out]) {
				out, err := Wait(ctx, fOut)
				if err == nil {
					outs[i] = out
				}
				errCh <- err
			}(i, fOut)
		}
		// Wait for all nil errors or return first error. We don't check context
		// done here, we count on execution stopping for that reason
		for range f {
			if err := <-errCh; err != nil {
				return nil, err
			}
		}
		return outs, nil
	})
}

// Slices 1:1 with input, zero values in slice results when not applicable
func WaitAll[Out any](ctx context.Context, f ...Fut[Out]) ([]Out, []error) {
	panic("TODO")
}

func First[Out any](ctx context.Context, f ...Fut[Out]) (Out, error) {
	panic("TODO")
}

func Completable[Out any](ctx context.Context, outCh chan<- Out, errCh chan<- error) Fut[Out] {
	panic("TODO")
}

// Thoughts:
// * Can Fut[struct{}] be embedded in custom to have a state-machine-like setup?
// * Show how Fut can help make otherwise confusing code appear more imperative
// * Show how this can easily be used to implement worker pools
