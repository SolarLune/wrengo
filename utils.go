package wrengo

import (
	"log"
	"unsafe"
)

func BytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	if *p == 0 {
		return ""
	}

	// Find NUL terminator.
	n := 0
	for ptr := unsafe.Pointer(p); *(*byte)(ptr) != 0; n++ {
		ptr = unsafe.Pointer(uintptr(ptr) + 1)
	}

	return string(unsafe.Slice(p, n))
}

// Quick note about the way the slot system works in Wren:
//
// In general, when a method is called, slot 0 contains the receiver of the method,
// while the other slots contains the arguments. This is how data is transferred from C (go) to Wren and back.
// When a Wren function calls a method and gets a response, the response is set to slot 0; in other words, if a Go function
// returns a value, putting it into slot 0 binds it.
func goValueToSlot(vmHandle uintptr, slot int, arg any) bool {

	switch a := arg.(type) {
	case bool:
		setSlotBool(vmHandle, slot, a)
		return true
	case int:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case int32:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case int64:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case float32:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case float64:
		setSlotDouble(vmHandle, slot, a)
		return true
	case uint:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case uint8:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case uint16:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case uint32:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case uint64:
		setSlotDouble(vmHandle, slot, float64(a))
		return true
	case string:
		setSlotString(vmHandle, slot, a)
		return true
	case []byte:
		setSlotBytes(vmHandle, slot, uintptr(unsafe.Pointer(&a)), len(a))
		return true
	case nil:
		setSlotNull(vmHandle, slot)
		return true
	case []any:
		setSlotNewList(vmHandle, slot)
		for _, i := range a {
			if !goValueToSlot(vmHandle, slot+1, i) {
				return false
			}
			insertSlotListElement(vmHandle, slot, -1, slot+1)
		}
		return true
	case map[any]any:
		setSlotNewMap(vmHandle, slot)
		for k, v := range a {

			// Put the key in the current slot
			if !goValueToSlot(vmHandle, slot+1, k) {
				return false
			}

			// The value in the next slot
			if !goValueToSlot(vmHandle, slot+2, v) {
				return false
			}
			setSlotMapValue(vmHandle, slot, slot+1, slot+2) // Put it in the map in the current slot
		}
		return true
	}

	return false

}

func slotValueToGo(vm uintptr, slot int) any {

	t := SlotType(getSlotType(vm, slot))

	switch t {
	case SlotTypeBool:
		// args = append(args, getSlotBool(vm, slot))
		return getSlotBool(vm, slot)
	case SlotTypeNumber:
		return getSlotDouble(vm, slot)
	case SlotTypeNull:
		return nil
	case SlotTypeString:
		return getSlotString(vm, slot)
	case SlotTypeList:
		listCount := getSlotListCount(vm, slot)
		list := []any{}
		for listIndex := range listCount {
			// Put the element in the next slot
			getSlotListElement(vm, slot, listIndex, slot+1)
			// And parse it from there; this makes sure we're not overwriting slots (which would make it crash)
			list = append(list, slotValueToGo(vm, slot+1))
		}
		return list
	case SlotTypeMap:
		mapCount := getSlotMapCount(vm, slot)
		mapping := map[any]any{}
		for n := range mapCount {
			getSlotMapKey(vm, slot, n, slot+1)
			key := slotValueToGo(vm, slot+1)

			getSlotMapValue(vm, slot, slot+1, slot+2)
			value := slotValueToGo(vm, slot+2)
			mapping[key] = value
		}
		return mapping
	default:
		log.Println("Cannot parse argument of type %v", t)
	}

	return nil

}
