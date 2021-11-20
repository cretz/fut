package fut_test

import (
	"context"
	"fmt"

	"github.com/cretz/fut"
)

func ExampleWait() {
	f1 := fut.New(func(context.Context) (string, error) {
		return "Hello, World!", nil
	})
	out, _ := fut.Wait(context.TODO(), f1)
	fmt.Println(out)
	// Output: Hello, World!
}

func ExampleWait_err() {
	f1 := fut.Err[struct{}](fmt.Errorf("Hello, Err!"))
	_, err := fut.Wait(context.TODO(), f1)
	fmt.Println(err)
	// Output: Hello, Err!
}

func ExampleMustWait() {
	f1 := fut.New(func(context.Context) (string, error) {
		return "Hello, World!", nil
	})
	out := fut.MustWait(context.TODO(), f1)
	fmt.Println(out)
	// Output: Hello, World!
}

func ExampleMustWait_err() {
	defer func() { fmt.Println(recover()) }()
	f1 := fut.Err[struct{}](fmt.Errorf("Hello, Err!"))
	_ = fut.MustWait(context.TODO(), f1)
	fmt.Println("Does not get here")
	// Output: Hello, Err!
}

func ExampleFixed() {
	f1 := fut.Fixed("Hello, World!")
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, World!
}

func ExampleErr() {
	f1 := fut.Err[struct{}](fmt.Errorf("Hello, Err!"))
	_, err := fut.Wait(context.TODO(), f1)
	fmt.Println(err)
	// Output: Hello, Err!
}

func ExampleStart() {
	f1 := fut.Start(context.TODO(), func(context.Context) (string, error) {
		return "Hello, World!", nil
	})
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, World!
}

func ExampleThen() {
	f1 := fut.Fixed("Hello")
	f1 = fut.Then(f1, func(_ context.Context, in string) (string, error) {
		return in + ", World!", nil
	})
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, World!
}

func ExampleThenFut() {
	f1 := fut.Fixed("Hello")
	f1 = fut.ThenFut(f1, func(_ context.Context, in string) (fut.Fut[string], error) {
		return fut.Fixed(in + ", World!"), nil
	})
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, World!
}

func ExampleThenErr() {
	f1 := fut.Err[string](fmt.Errorf("Hello"))
	f1 = fut.ThenErr(f1, func(_ context.Context, in error) (string, error) {
		return in.Error() + ", Err!", nil
	})
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, Err!
}

func ExampleThenErrFut() {
	f1 := fut.Err[string](fmt.Errorf("Hello"))
	f2 := fut.ThenErrFut(f1, func(_ context.Context, in error) (fut.Fut[string], error) {
		return fut.Fixed(in.Error() + ", Err!"), nil
	})
	fmt.Println(fut.MustWait(context.TODO(), f2))
	// Output: Hello, Err!
}

func ExampleDefer() {
	f1 := fut.Fixed(", World!")
	f1 = fut.Defer(f1, func(context.Context) error {
		fmt.Print("Hello")
		return nil
	})
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, World!
}

func ExampleDeferOK() {
	f1 := fut.Fixed("Hello")
	f1 = fut.DeferOK(f1, func(_ context.Context, in string) error {
		fmt.Println(in + ", World!")
		return nil
	})
	fut.MustWait(context.TODO(), f1)
	// Output: Hello, World!
}

func ExampleDeferErr() {
	f1 := fut.Err[struct{}](fmt.Errorf("Hello"))
	f1 = fut.DeferErr(f1, func(_ context.Context, in error) error {
		fmt.Println(in.Error() + ", World!")
		return nil
	})
	fut.Wait(context.TODO(), f1)
	// Output: Hello, World!
}

func ExampleWithContext() {
	f1 := fut.New(func(ctx context.Context) (string, error) {
		return "Hello" + ctx.Value("mykey").(string), nil
	})
	f1 = fut.WithContext(f1, context.WithValue(context.TODO(), "mykey", ", World!"))
	fmt.Println(fut.MustWait(context.TODO(), f1))
	// Output: Hello, World!
}
