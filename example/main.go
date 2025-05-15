package main

import (
	"fmt"
	"os"

	"github.com/solarlune/wrengo"
)

func main() {

	err := wrengo.InitFromDirectory("./lib")
	if err != nil {
		panic(err)
	}

	fsys := os.DirFS(".")

	config := wrengo.NewConfig().
		WithModuleLoaderFromFS(fsys).
		WithForeignMethodResolver(func(vm *wrengo.VM, module, className, signature string, isStatic bool) wrengo.GoForeignFunction {
			return func(vm *wrengo.VM, args []any) any {
				return fmt.Sprintf("This is a Go foreign method! It was called with the args: %v", args)
			}
		})

	vm := wrengo.NewVM(config)

	vm.RunFile(fsys, "./my_module.wren")

	// Call from Go
	h, err := vm.CallHandle("./my_module.wren", "testFunc", "call(_)")
	if err != nil {
		panic(err)
	}

	h.Call("Wren") // Call the testFunc with "Wren" as an argument

}
