package wrengo

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
)

type SlotType int

const (
	SlotTypeBool SlotType = iota
	SlotTypeNumber
	SlotTypeForeign
	SlotTypeList
	SlotTypeMap
	SlotTypeNull
	SlotTypeString
	SlotTypeUnknown // Unrepresentable by C
)

// InterpretResultSuccess InterpretResult = iota
var ErrCompileTime = errors.New("error compiling wren script")
var ErrVMFreed = errors.New("error: virtual machine already freed")

var initConfig func(uintptr)
var newVM func(*Config) uintptr
var interpret func(vm uintptr, moduleName, text string) int
var freeVM func(vm uintptr)

var ensureSlots func(vm uintptr, slotCount int)
var getSlotCount func(vm uintptr)

var setSlotBool func(vm uintptr, slot int, value bool)
var setSlotDouble func(vm uintptr, slot int, value float64)
var setSlotNull func(vm uintptr, slot int)
var setSlotBytes func(vm uintptr, slot int, bytes uintptr, length int)
var setSlotString func(vm uintptr, slot int, text string)

var getSlotBool func(vm uintptr, slot int) bool
var getSlotDouble func(vm uintptr, slot int) float64
var getSlotString func(vm uintptr, slot int) string
var getSlotBytes func(vm uintptr, slot int, length int) *byte

var setSlotNewList func(vm uintptr, slot int) int
var getSlotListCount func(vm uintptr, slot int) int
var setSlotListElement func(vm uintptr, listSlot, index, elementSlot int)
var insertSlotListElement func(vm uintptr, listSlot, index, elementSlot int)
var getSlotListElement func(vm uintptr, listSlot, index, elementSlot int)

var setSlotNewMap func(vm uintptr, slot int) int
var getSlotMapCount func(vm uintptr, mapSlot int) int
var setSlotMapValue func(vm uintptr, mapSlot, keySlot, valueSlot int)
var removeSlotMapValue func(vm uintptr, mapSlot, keySlot, removedValueSlot int)
var getSlotMapContainsKey func(vm uintptr, slot int) bool
var getSlotMapKey func(vm uintptr, mapSlot, keyIndex, targetSlot int)
var getSlotMapValue func(vm uintptr, mapSlot, keySlot, valueSlot int)

var getSlotType func(vm uintptr, slot int) int

var getSlotHandle func(vm uintptr, slot int) uintptr
var makeCallHandle func(vm uintptr, signature string) uintptr
var call func(vm uintptr, handle uintptr) int
var releaseCallHandle func(vm uintptr, handle uintptr)

var getVariable func(vm uintptr, module, name string, slot int)
var hasVariable func(vm uintptr, module, name string) bool
var hasModule func(vm uintptr, module string) bool

var getVersionNumber func() int

var initialized bool

// Returns the version number of the scripting engine.
func VersionNumber() (int, int, int) {
	versionNumber := getVersionNumber()
	major := int(versionNumber / 1000000)
	minor := int(versionNumber / 1000)
	patch := versionNumber - (major * 100000) - (minor * 1000)
	return major, minor, patch
}

// Init initializes the Wren-Go bindings using the passed shared library filepath.
// The path is, by default, relative to the executable, in the current working directory.
// The function automatically loads libraries using the OS and architecture hierarchy from the original
// DLL / library download.
func Init(libraryPath string) error {

	lib, err := loadLibrary(libraryPath)
	if err != nil {
		return err
	}

	purego.RegisterLibFunc(&getVersionNumber, lib, "wrenGetVersionNumber")
	purego.RegisterLibFunc(&initConfig, lib, "wrenInitConfiguration")
	purego.RegisterLibFunc(&newVM, lib, "wrenNewVM")
	purego.RegisterLibFunc(&interpret, lib, "wrenInterpret")
	purego.RegisterLibFunc(&freeVM, lib, "wrenFreeVM")

	purego.RegisterLibFunc(&makeCallHandle, lib, "wrenMakeCallHandle")
	purego.RegisterLibFunc(&call, lib, "wrenCall")
	purego.RegisterLibFunc(&releaseCallHandle, lib, "wrenReleaseHandle")

	purego.RegisterLibFunc(&getVariable, lib, "wrenGetVariable")
	purego.RegisterLibFunc(&hasVariable, lib, "wrenHasVariable")
	purego.RegisterLibFunc(&hasModule, lib, "wrenHasModule")

	purego.RegisterLibFunc(&ensureSlots, lib, "wrenEnsureSlots")
	purego.RegisterLibFunc(&getSlotCount, lib, "wrenGetSlotCount")

	purego.RegisterLibFunc(&setSlotBool, lib, "wrenSetSlotBool")
	purego.RegisterLibFunc(&setSlotDouble, lib, "wrenSetSlotDouble")
	purego.RegisterLibFunc(&setSlotNull, lib, "wrenSetSlotNull")
	purego.RegisterLibFunc(&setSlotBytes, lib, "wrenSetSlotBytes")
	purego.RegisterLibFunc(&setSlotString, lib, "wrenSetSlotString")

	purego.RegisterLibFunc(&setSlotNewList, lib, "wrenSetSlotNewList")
	purego.RegisterLibFunc(&setSlotListElement, lib, "wrenSetListElement")
	purego.RegisterLibFunc(&insertSlotListElement, lib, "wrenInsertInList")
	purego.RegisterLibFunc(&getSlotListCount, lib, "wrenGetListCount")
	purego.RegisterLibFunc(&getSlotListElement, lib, "wrenGetListElement")

	purego.RegisterLibFunc(&setSlotNewMap, lib, "wrenSetSlotNewMap")
	purego.RegisterLibFunc(&setSlotMapValue, lib, "wrenSetMapValue")
	purego.RegisterLibFunc(&removeSlotMapValue, lib, "wrenRemoveMapValue")
	purego.RegisterLibFunc(&getSlotMapCount, lib, "wrenGetMapCount")
	purego.RegisterLibFunc(&getSlotMapContainsKey, lib, "wrenGetMapContainsKey")
	purego.RegisterLibFunc(&getSlotMapKey, lib, "wrenGetMapKey")
	purego.RegisterLibFunc(&getSlotMapValue, lib, "wrenGetMapValue")

	purego.RegisterLibFunc(&getSlotBool, lib, "wrenGetSlotBool")
	purego.RegisterLibFunc(&getSlotDouble, lib, "wrenGetSlotDouble")
	purego.RegisterLibFunc(&getSlotBytes, lib, "wrenGetSlotBytes")
	purego.RegisterLibFunc(&getSlotString, lib, "wrenGetSlotString")
	purego.RegisterLibFunc(&getSlotType, lib, "wrenGetSlotType")

	return nil

}

// InitFromDirectory initializes the scripting bindings using the libraries in the given
// directory, automatically loading the correct one for the current platform if it's properly named.
func InitFromDirectory(libraryBaseDirectoryPath string) error {

	if initialized {
		return nil
	}

	osName := ""

	switch runtime.GOOS {
	case "darwin":
		osName = "wren_macos"
	case "linux":
		osName = "wren_linux"
	case "windows":
		osName = "wren_win"
	}

	archFolder := ""

	switch runtime.GOARCH {
	case "386":
		archFolder = "_x86"
	case "amd64":
		archFolder = "_x86_64"
	case "arm":
		archFolder = "_arm"
	case "arm64":
		archFolder = "_arm64"
	}

	filename := ""

	switch runtime.GOOS {
	case "linux":
		filename = ".so"
	case "darwin":
		filename = ".dylib"
	case "windows":
		filename = ".dll"

	}

	dllPath := filepath.Join(filepath.Clean(libraryBaseDirectoryPath), strings.Join([]string{osName, archFolder, filename}, ""))

	return Init(dllPath)
}

// GoForeignFunction represents a Go function that is called from Wren.
// args represents the arguments that were supplied from Wren through the method or function call, and
// the function should return a convertible value.
type GoForeignFunction func(vm *VM, args []any) any

type Config struct {
	reallocateFn        uintptr
	resolveModuleFn     uintptr
	loadModuleFn        uintptr
	bindForeignMethodFn uintptr
	bindForeignClassFn  uintptr
	writeFn             uintptr
	errorFn             uintptr
	initialHeapSize     int
	minHeapSize         int
	heapGrowthPercent   int
	userData            uintptr
	SlotNum             int // Number of slots for variables; by default, set to 100
}

// WithModuleLoaderFromFS sets the Wren VM to use a file system to load and import Wren modules.
//
// This can only be set once per Config.
func (cfg Config) WithModuleLoaderFromFS(filesys fs.FS) Config {

	if cfg.loadModuleFn != 0 {
		return cfg
	}

	if filesys != nil {
		cfg.loadModuleFn = purego.NewCallback(func(vm uintptr, modName *byte) *byte {
			modNameString := BytePtrToString(modName)

			if filepath.Ext(filepath.Base(modNameString)) != ".wren" {
				modNameString += ".wren"
			}

			res, err := fs.ReadFile(filesys, modNameString)

			if err != nil {
				// We don't need to output an error; Wren will say it couldn't load the module.
				// println("WrenVM: Module Loading:", err.Error())
				return nil
			}

			return &res[0]

		})

	}

	return cfg

}

// WithForeignMethodResolver sets the configuration to use a foreign method resolver.
// This allows you to call Go from Wren by simply returning a GoForeignFunction - a function
// written in Go that does something and that can return a value, depending on the module, classname,
// signature, and staticness provided.
//
// This can only be set once per Config.
func (cfg Config) WithForeignMethodResolver(resolver func(vm *VM, module, className, signature string, isStatic bool) GoForeignFunction) Config {

	if cfg.bindForeignMethodFn != 0 {
		return cfg
	}

	cfg.bindForeignMethodFn = purego.NewCallback(func(vm uintptr, module, className *byte, isStatic bool, signature *byte) uintptr {

		if resolver != nil {

			sigString := BytePtrToString(signature)

			out := resolver(vmstoVMs[vm], BytePtrToString(module), BytePtrToString(className), sigString, isStatic)

			if out != nil {

				return purego.NewCallback(func(vm uintptr) {

					argCount := strings.Count(sigString, "_")

					args := []any{}

					for i := range argCount {
						args = append(args, slotValueToGo(vm, i+1))
					}

					res := out(vmstoVMs[vm], args)
					if res != nil {
						goValueToSlot(vm, 0, res)
					}
					return
				})
			}

			return uintptr(0)
		}
		// fmt.Println("bind function?")
		test := func(vm uintptr) {
			fmt.Println("this is a function???")
		}
		return purego.NewCallback(test)
		// return uintptr(unsafe.Pointer(&test))

	})

	return cfg

}

// NewConfig creates a new configuration for creating a VM.
func NewConfig() Config {

	wrenConfig := Config{}

	initConfig(uintptr(unsafe.Pointer(&wrenConfig)))

	wrenConfig.writeFn = purego.NewCallback(func(vm uintptr, text *byte) {
		print(BytePtrToString(text))
	})

	wrenConfig.errorFn = purego.NewCallback(func(vm uintptr, errorType int, module *byte, line int, msg *byte) {
		switch errorType {
		case 0: // WREN_ERROR_COMPILE
			fmt.Printf("[%s:%d] Compile %s\n", BytePtrToString(module), line, BytePtrToString(msg))
		case 1: // WREN_ERROR_RUNTIME
			fmt.Printf("[Runtime Error] %s\n", BytePtrToString(msg))
			// fmt.Printf("[%s:%d] Runtime %s\n", BytePtrToString(module), line, BytePtrToString(msg))
		case 2: // WREN_ERROR_STACK_TRACE
			fmt.Printf("[%s:%d] in %s\n", BytePtrToString(module), line, BytePtrToString(msg))
		}
	})

	return wrenConfig
}

type VM struct {
	config Config
	handle uintptr
	freed  bool
}

var vmstoVMs = map[uintptr]*VM{}

// NewVM creates a new VM using the provided configuration.
func NewVM(config Config) *VM {
	vm := &VM{
		config: config,
		handle: uintptr(newVM(&config)),
	}
	vmstoVMs[vm.handle] = vm
	return vm
}

// Run compiles and evaluates the source text src and binds it to the given module name.
// Once run, module cannot be run again, as the VM's state is persistent.
func (vm *VM) Run(moduleName, src string) error {
	if vm.freed {
		return ErrVMFreed
	}
	res := interpret(vm.handle, moduleName, src)

	slotNum := vm.config.SlotNum
	if slotNum <= 0 {
		slotNum = 10000 // Default value; code assumes 100 slots
	}
	ensureSlots(vm.handle, slotNum)

	switch res {
	case 0:
		return nil
	case 1:
		return ErrCompileTime
	default:
		return fmt.Errorf("%s: runtime error running script", moduleName)
	}
}

// RunFile evaluates a file, found at the provided filepath in the file system.
// Once run, the file cannot be run again, as the VM's state is persistent.
func (vm *VM) RunFile(fsys fs.FS, fpath string) error {

	if vm.freed {
		return ErrVMFreed
	}

	src, err := fs.ReadFile(fsys, filepath.Clean(fpath))
	if err != nil {
		return err
	}

	return vm.Run(fpath, string(src))

}

func (vm *VM) Free() error {
	if vm.freed {
		return ErrVMFreed
	}
	vm.freed = true
	freeVM(vm.handle)
	return nil
}

// CallHandle creates a call handle to the function or method with the name and
// function signature defined of the object designated in the module provided.
// The module is the filepath of the script file for VM.InterpretFilesystem() calls,
// or the explicit module name for VM.Interpret() calls.
//
// This can be used in two ways.
//
// 1. Functions. For these, object should be the name of the function and signature ".call()".
//
// 2. Methods. For these, object should be the name of the class instance or class object
// and the signature the method to call. The method is designated by arity, with underscores being spaces for arguments.
// (So for a function named "Walk" that takes an argument on a Dog class, object = "Dog" and signature = "Walk(_)").
// You can also use it on getters, setters, and static functions.
func (vm *VM) CallHandle(module, object, signature string) (*CallHandle, error) {

	signature = strings.ReplaceAll(signature, " ", "")

	if !hasVariable(vm.handle, module, object) {
		return nil, fmt.Errorf("error getting a handle for '%s' in '%s'; does the module and object exist?", object, module)
	}

	handle := &CallHandle{
		vm:       vm,
		handle:   makeCallHandle(vm.handle, signature),
		argCount: strings.Count(signature, "_"),
		callName: signature,
		object:   object,
		module:   module,
	}

	return handle, nil
}

// var i = 0

// Variable looks up the object name in the specified module; if it exists, it attempts to parse it to a Go object.
func (vm *VM) Variable(module, objectName string) any {
	if vm.HasVariable(module, objectName) {
		getVariable(vm.handle, module, objectName, 99)
		return slotValueToGo(vm.handle, 99)
	}
	return nil
}

// HasVariable returns true if the VM has a variable of the given name in the module specified.
func (vm *VM) HasVariable(module, objectName string) bool {
	return hasVariable(vm.handle, module, objectName)
}

// HasModule returns true if the VM contains a module of the given name.
func (vm *VM) HasModule(moduleName string) bool {
	return hasModule(vm.handle, moduleName)
}

// func (vm *WrenVM) EnsureSlotCount(slotCount int) {
// 	ensureSlots(vm.handle, slotCount)
// }

// func (vm *VM) Slot(slotNum int) *SlotResult {
// 	return &SlotResult{vm.handle, slotNum}
// }

// func (vm *WrenVM) SetSlotInt(slot int, value int) {
// 	setSlotDouble(vm.handle, slot, float64(value))
// }

// func (vm *WrenVM) SetSlotFloat64(slot int, value float64) {
// 	setSlotDouble(vm.handle, slot, float64(value))
// }

// func (vm *WrenVM) SetSlotString(slot int, value string) {
// 	setSlotString(vm.handle, slot, value)
// }

// func (vm *WrenVM) SetSlotBool(slot int, value bool) {
// 	setSlotBool(vm.handle, slot, value)
// }

// func (vm *WrenVM) SetSlotNull(slot int) {
// 	setSlotNull(vm.handle, slot)
// }

// func (vm *WrenVM) SetSlotNewMap(slot int) {
// 	setSlotNewMap(vm.handle, slot)
// }

// func (vm *WrenVM) SetSlotNewList(slot int) {
// 	setSlotNewList(vm.handle, slot)
// }

// func (s *SlotResult) Set(value any) {
// 	switch v := value.(type) {
// 	case int:
// 		setSlotDouble(s.vm, s.slot, float64(v))
// 	case int32:
// 		setSlotDouble(s.vm, s.slot, float64(v))
// 	case int64:
// 		setSlotDouble(s.vm, s.slot, float64(v))
// 	case float32:
// 		setSlotDouble(s.vm, s.slot, float64(v))
// 	case float64:
// 		setSlotDouble(s.vm, s.slot, v)
// 	case string:
// 		setSlotString(s.vm, s.slot, v)
// 	case []byte:
// 		setSlotBytes(s.vm, s.slot, uintptr(unsafe.Pointer(&v)), len(v))
// 	case bool:
// 		setSlotBool(s.vm, s.slot, v)
// 	}
// }

// type Result struct {
// 	vm   uintptr
// 	slot int
// }

// func (s *Result) AsBool() bool {
// 	return getSlotBool(s.vm, s.slot)
// }

// func (s *Result) IsBool() bool {
// 	return s.Type() == SlotTypeBool
// }

// func (s *Result) AsInt() int {
// 	return int(getSlotDouble(s.vm, s.slot))
// }
// func (s *Result) AsFloat64() float64 {
// 	return getSlotDouble(s.vm, s.slot)
// }
// func (s *Result) IsNumber() bool {
// 	return s.Type() == SlotTypeNumber
// }

// func (s *Result) AsString() string {
// 	return getSlotString(s.vm, s.slot)
// }
// func (s *Result) IsString() bool {
// 	return s.Type() == SlotTypeString
// }

// func (s *Result) AsBytes() []byte {
// 	return []byte(getSlotString(s.vm, s.slot))
// }

// func (s *Result) Type() SlotType {
// 	return SlotType(getSlotType(s.vm, s.slot))
// }

type CallHandle struct {
	vm     *VM
	handle uintptr

	argCount int
	callName string
	module   string
	object   string
}

// Call attempts to call the specified function on the object in the given module from the
// creation of the CallHandle. If it can't due to an error (wrong signature, runtime errors, etc.),
// the fiber running the VM will fail, meaning further attempts to run code on this VM will fail until
// the VM is restarted by re-evaluating / re-compiling code, and the function will return an error.
//
// args is any arguments to supply to the function call. can be any primitive numeric or boolean type, a string, a byte slice, nil, a []any or a map[string]any.
// The following Go variable types are usable as arguments:
//
// - bool
// - float32 / float64 / int / int32 / rune / int64 / uint / uint8 / uint16 / uint32 / uint64 - all transformed to Double
// - string
// - []byte
// - nil
// - []any (elements must be convertible, of course)
// - map[any]any (elements must be convertible, of course)
//
// The function will return any values returned from the function in Wren, converted to Go types, and
// an error if the function couldn't be called.
func (w *CallHandle) Call(args ...any) (any, error) {

	if len(args) < w.argCount {
		return nil, fmt.Errorf("error calling function; it requires %d arguments and Call() was provided with %d", w.argCount, len(args))
	}

	slot := 1
	for i, arg := range args {
		if !goValueToSlot(w.vm.handle, slot, arg) {
			return nil, fmt.Errorf("error converting arguments; argument #%d, ( %v ) cannot be converted", i, arg)
		}
		slot++
	}

	// Put the object (receiver) from the module in the first slot
	getVariable(w.vm.handle, w.module, w.object, 0)

	res := call(w.vm.handle, w.handle)

	switch res {
	case 0:
		return slotValueToGo(w.vm.handle, 0), nil
		// return &Result{vm: w.vm.handle, slot: 0}, nil
		// No compilation; it's already compiled, that's what the Handle represents
	default:
		return nil, fmt.Errorf("%s:%s:%s: runtime error running script", w.module, w.object, w.callName)
	}
}

func (w *CallHandle) Release() {
	releaseCallHandle(w.vm.handle, w.handle)
}

// func (vm *WrenVM) EnsureSlotCount(slotCount int) {
// 	ensureSlots(vm.handle, slotCount)
// }

// func (vm *WrenVM) Slot(slotIndex int) {

// }
