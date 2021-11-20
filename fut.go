package fut

import "context"

// Do _not_ use the last two return variables until channel closed. Pointers are
// guaranteed never nil (though they may point to a nil value)
type Fut[Out any] func(context.Context) (<-chan struct{}, *Out, *error)

type PanicHandler func(interface{}) error

// Only for panic handler that doesn't return an error
type ErrPanic struct{ Value interface{} }

func (*ErrPanic) Error() string { return "Panic" }

type panicHandlerKey struct{}

var PanicHandlerKey interface{} = panicHandlerKey{}

type Call interface {
	~func(context.Context) error
	// TODO(cretz): Support when inference works better
	// 	~func(context.Context) |
	// 	~func() error |
	// 	~func()
}

func InvokeCall[C Call](ctx context.Context, c C) error {
	switch c := (interface{})(c).(type) {
	case func(context.Context) error:
		return c(ctx)
	// TODO(cretz): Support other signatures when inference improves
	default:
		panic("bad function")
	}
}

type CallIn[In any] interface {
	~func(context.Context, In) error
	// TODO(cretz): Support when inference works better
	// 	~func(context.Context, In) |
	// 	~func(In) error |
	// 	~func(In)
}

func InvokeCallIn[In any, C CallIn[In]](ctx context.Context, c C, in In) error {
	switch c := (interface{})(c).(type) {
	case func(context.Context, In) error:
		return c(ctx, in)
	// TODO(cretz): Support other signatures when inference improves
	default:
		panic("bad function")
	}
}

type CallOut[Out any] interface {
	~func(context.Context) (Out, error)
	// TODO(cretz): Support when inference works better
	// 	~func(context.Context) Out |
	// 	~func() (Out, error) |
	// 	~func() Out
}

func InvokeCallOut[Out any, C CallOut[Out]](ctx context.Context, c C) (Out, error) {
	switch c := (interface{})(c).(type) {
	case func(context.Context) (Out, error):
		return c(ctx)
	// TODO(cretz): Support other signatures when inference improves
	default:
		panic("bad function")
	}
}

type CallInOut[In, Out any] interface {
	~func(context.Context, In) (Out, error)
	// TODO(cretz): Support when inference works better
	// 	~func(context.Context, In) Out |
	// 	~func(In) (Out, error) |
	// 	~func(In) Out
}

func InvokeCallInOut[In, Out any, C CallInOut[In, Out]](ctx context.Context, c C, in In) (Out, error) {
	switch c := (interface{})(c).(type) {
	case func(context.Context, In) (Out, error):
		return c(ctx, in)
	// TODO(cretz): Support other signatures when inference improves
	default:
		panic("bad function")
	}
}
