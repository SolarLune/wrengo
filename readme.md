# Wren-Go Bindings

These are PureGo bindings for the Wren scripting language to be used with Go.

[pkg.dev](https://pkg.go.dev/github.com/solarlune/wrengo)

# What is Wren?

["Wren is a small, fast, class-based concurrent scripting language."](https://wren.io/)

# Other bindings exist, right?

Yes, but they:

- May not use Purego. This hurts cross-compilation.
- May be outdated.
- May not be efficient.
- May not be as Go-like as these bindings.

These bindings, while incomplete and imperfect, still fulfill the above restrictions.

The following platforms _should_ work with these bindings:

- Windows x86 / x86_64
- Linux x86_64
- Mac x86_64 / arm_64

Though only Linux has been tested.

# How do I use the bindings?

1. `go get github.com/solarlune/wren-go`
2. Copy the `lib` directory to your project directory.
3.

```go

func main() {

	err := wrengo.InitFromDirectory("./lib")
	if err != nil {
		panic(err)
	}

	// Create a config.
	config := wrengo.NewConfig()

	// Create a VM with the config.
	vm := wrengo.NewVM(config)

	// Run Wren!
	vm.Run("main", `System.print("Hi from Wren!")`)

}

```
