package dtypes // import "github.com/go-hep/dtypes"

import (
	"encoding/gob"
	"errors"
	"fmt"
	"hash/adler32"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
)

// ID represents a Type as an integer that can be stored on disk or passed on the wire.
type ID uint32

func (id ID) Type() Type {
	if id == 0 {
		return nil
	}
	return idToType[id]
}

const (
	idInvalid ID = iota
	idBool
	idInt
	idInt8
	idInt16
	idInt32
	idInt64
	idUint
	idUint8
	idUint16
	idUint32
	idUint64
	idUintptr
	idFloat32
	idFloat64
	idComplex64
	idComplex128
	idInterface
	idString
	idBytes

	idCommonType
	idArrayType
	idMapType
	idSliceType
	idStructType

	idBuiltins ID = 64
)

// A Kind represents the specific kind of type that a Type represents.
// The zero Kind is not a valid kind.
type Kind uint

const (
	Invalid Kind = iota
	Builtin
	Array
	Map
	Slice
	Struct
)

func (k Kind) String() string {
	switch k {
	case Invalid:
		return "Invalid"
	case Builtin:
		return "Builtin"
	case Array:
		return "Array"
	case Map:
		return "Map"
	case Slice:
		return "Slice"
	case Struct:
		return "Struct"
	}
	panic(fmt.Errorf("dtypes: invalid Kind value (%d)", uint(k)))
}

// Descr describes a type in terms of builtins
type Descr struct {
	Name string
	ID   ID
}

// Type describes a serializable data type description
type Type interface {
	ID() ID
	Name() string
	Kind() Kind
	Descr() []Descr

	setID(id ID)
}

// TypeFromID returns the dtypes.Type corresponding to ID.
// TypeFromID returns nil if no such ID is known to the runtime.
func TypeFromID(id ID) Type {
	return id.Type()
}

// New returns the Type of the value v.
func New(v interface{}) Type {
	rt, err := reflectTypeOf(v)
	if err != nil {
		panic(err)
	}
	if rt == nil {
		return tInterface.Type()
	}

	typesLock.RLock()
	dt, ok := types[rt]
	typesLock.RUnlock()
	if ok {
		return dt
	}

	typesLock.Lock()
	defer typesLock.Unlock()
	if dt = types[rt]; dt != nil && dt.ID() != 0 {
		// lost the race. not an issue per se.
		return dt
	}

	dt, err = newType(rt)
	if err != nil {
		panic(err)
	}
	return dt
}

func nameFromType(rt reflect.Type) string {
	if rt == nil {
		return "interface"
	}
	// Default to printed representation for unnamed types
	name := rt.String()

	// But for named types (or pointers to them), qualify with import path.
	// Dereference one pointer looking for a named type.
	star := ""
	if rt.Name() == "" {
		pt := rt
		if pt.Kind() == reflect.Ptr {
			star = "*"
			rt = pt.Elem()
		}
	}

	if rt.Name() != "" {
		switch rt.PkgPath() {
		case "":
			name = star + rt.Name()
		default:
			name = star + rt.PkgPath() + "." + rt.Name()
		}
	}

	return name
}

func setTypeID(typ Type) {
	idToType[typ.ID()] = typ
}

type CommonType struct {
	Xname string
	Xid   ID
}

func newCommonType(name string, id ID, rt reflect.Type) CommonType {
	return CommonType{name, id}
}

func (t *CommonType) ID() ID {
	return t.Xid
}

func (t *CommonType) setID(id ID) {
	t.Xid = id
}

func (*CommonType) Kind() Kind {
	return Builtin
}

func (t *CommonType) Name() string {
	return t.Xname
}

func (t *CommonType) Descr() []Descr {
	return []Descr{{Name: "", ID: t.Xid}}
}

// Array type
type arrayType struct {
	CommonType
	Elem ID
	Len  int
}

func newArrayType(name string, id ID, rt reflect.Type) *arrayType {
	return &arrayType{newCommonType(name, id, rt), 0, 0}
}

func (a *arrayType) init(elem Type, len int) {
	// Set our type id before evaluating the element's, in case it's our own.
	setTypeID(a)
	a.Elem = elem.ID()
	a.Len = len
}

func (*arrayType) Kind() Kind {
	return Array
}

// Map type
type mapType struct {
	CommonType
	Key  ID
	Elem ID
}

func newMapType(name string, id ID, rt reflect.Type) *mapType {
	return &mapType{newCommonType(name, id, rt), 0, 0}
}

func (*mapType) Kind() Kind {
	return Map
}

func (m *mapType) init(k, v Type) {
	// Set our type id before evaluating the element's, in case it's our own.
	setTypeID(m)
	m.Key = k.ID()
	m.Elem = v.ID()
}

// Slice type
type sliceType struct {
	CommonType
	Elem ID
}

func newSliceType(name string, id ID, rt reflect.Type) *sliceType {
	return &sliceType{newCommonType(name, id, rt), 0}
}

func (*sliceType) Kind() Kind {
	return Slice
}

func (s *sliceType) init(elem Type) {
	// Set our type id before evaluating the element's, in case it's our own.
	setTypeID(s)
	s.Elem = elem.ID()
}

// Struct type
type structType struct {
	CommonType
	Fields []Descr
}

func newStructType(name string, id ID, rt reflect.Type) *structType {
	return &structType{newCommonType(name, id, rt), nil}
}

func (*structType) Kind() Kind {
	return Struct
}

func (st *structType) Descr() []Descr {
	return st.Fields
}

func (s *structType) init() {
	// Set our type id before evaluating the element's, in case it's our own.
	setTypeID(s)
}

// newType allocates a Type for the reflection type rt.
func newType(rt reflect.Type) (Type, error) {
	var err error
	var type0, type1 Type
	defer func() {
		if err != nil {
			delete(types, rt)
		}
	}()

	name := nameFromType(rt)
	id := ID(adler32.Checksum([]byte(name)))

	// Install the top-level type before the subtypes (e.g. struct before
	// fields) so recursive types can be constructed safely.
	switch rt.Kind() {
	// All basic types are easy: they are predefined.
	case reflect.Bool:
		return tBool.Type(), nil

	case reflect.Int:
		return tInt.Type(), nil

	case reflect.Int8:
		return tInt8.Type(), nil

	case reflect.Int16:
		return tInt16.Type(), nil

	case reflect.Int32:
		return tInt32.Type(), nil

	case reflect.Int64:
		return tInt64.Type(), nil

	case reflect.Uint:
		return tUint.Type(), nil

	case reflect.Uint8:
		return tUint8.Type(), nil

	case reflect.Uint16:
		return tUint16.Type(), nil

	case reflect.Uint32:
		return tUint32.Type(), nil

	case reflect.Uint64:
		return tUint64.Type(), nil

	case reflect.Uintptr:
		return tUintptr.Type(), nil

	case reflect.Float32:
		return tFloat32.Type(), nil

	case reflect.Float64:
		return tFloat64.Type(), nil

	case reflect.Complex64:
		return tComplex64.Type(), nil

	case reflect.Complex128:
		return tComplex128.Type(), nil

	case reflect.String:
		return tString.Type(), nil

	case reflect.Interface:
		return tInterface.Type(), nil

	case reflect.Array:
		at := newArrayType(name, id, rt)
		types[rt] = at
		type0, err = newType(rt.Elem())
		if err != nil {
			return nil, err
		}
		at.init(type0, rt.Len())
		return at, err

	case reflect.Map:
		mt := newMapType(name, id, rt)
		types[rt] = mt
		type0, err = newType(rt.Key())
		if err != nil {
			return nil, err
		}

		type1, err = newType(rt.Elem())
		if err != nil {
			return nil, err
		}

		mt.init(type0, type1)
		return mt, nil

	case reflect.Slice:
		// []byte == []uint8 is a special case
		if rt.Elem().Kind() == reflect.Uint8 {
			return tBytes.Type(), nil
		}
		st := newSliceType(name, id, rt)
		types[rt] = st
		type0, err = newType(rt.Elem())
		if err != nil {
			return nil, err
		}
		st.init(type0)
		return st, nil

	case reflect.Struct:
		st := newStructType(name, id, rt)
		types[rt] = st
		setTypeID(st)
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !isStorable(&f) {
				continue
			}
			ft, err := newType(f.Type)
			if err != nil {
				return nil, err
			}
			// Some mutually recursive types can cause us to be here while
			// still defining the element. Fix the element type id here.
			// We could do this more neatly by setting the id at the start of
			// building every type, but that would break binary compatibility.
			if ft.ID() == 0 {
				setTypeID(ft)
			}
			st.Fields = append(st.Fields, Descr{f.Name, ft.ID()})
		}
		return st, nil

	default:
		return nil, errors.New("dtypes: newType can't handle type: " + name)
	}

}

// isExported reports whether this is an exported - upper case - name.
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// isStorable reports whether this struct field can be stored.
// It will be stored only if it is exported and not a chan or func field
// or pointer to chan or func.
func isStorable(field *reflect.StructField) bool {
	if !isExported(field.Name) {
		return false
	}
	// If the field is a chan or func or pointer thereto, don't send it.
	// That is, treat it like an unexported field.
	typ := field.Type
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Chan || typ.Kind() == reflect.Func {
		return false
	}
	return true
}

var (
	// Protected by an RWMutex because we read it a lot and write
	// it only when we see a new type, typically when compiling.
	typesLock sync.RWMutex
	types     = make(map[reflect.Type]Type)

	idToType = make(map[ID]Type)
)

// reflectTypeOf
func reflectTypeOf(value interface{}) (reflect.Type, error) {
	rt := reflect.TypeOf(value)
	if rt == nil {
		return rt, nil
	}
	// A type that is just a cycle of pointers (such as type T *T) cannot
	// be represented in gobs, which need some concrete data.  We use a
	// cycle detection algorithm from Knuth, Vol 2, Section 3.1, Ex 6,
	// pp 539-540.  As we step through indirections, run another type at
	// half speed. If they meet up, there's a cycle.
	slowpoke := rt // walks half as fast as rt
	indir := 0
	for {
		pt := rt
		if pt.Kind() != reflect.Ptr {
			break
		}
		rt = pt.Elem()
		if rt == slowpoke { // ut.base lapped slowpoke
			// recursive pointer type.
			return nil, errors.New(
				"dtypes: can't represent recursive pointer type " + rt.String(),
			)
		}
		if indir%2 == 0 {
			slowpoke = slowpoke.Elem()
		}
		indir++
	}

	return rt, nil
}

// Register records a type, identified by a value for that type, under its
// internal type name. That name will identify the concrete type of a value
// sent or received as an interface variable. Only types that will be
// transferred as implementations of interface values need to be
// registered. Expecting to be used only during initialization, it panics
// if the mapping between types and names is not a bijection.
func Register(value interface{}) {

	rt, err := reflectTypeOf(value)
	if err != nil {
		panic(err)
	}

	dt := New(reflect.New(rt).Elem().Interface())

	typesLock.Lock()
	defer typesLock.Unlock()
	if dt == nil {
		panic(fmt.Errorf("dtypes: Register(%#T) FAILED", value))
	}

	// TODO(sbinet) check for incompatible duplicates.
	// The name must refer to the same user type, and vice versa.
	types[rt] = dt

	gob.Register(value)
}

// Create and check predefined types
// The string for tBytes is "bytes" not "[]byte" to signify its specialness.

var (
	// Primordial types, needed during initialization.
	// Always passed as pointers so the interface{} type
	// goes through without losing its interfaceness.
	tBool       = bootstrapType("bool", (*bool)(nil), idBool)
	tInt        = bootstrapType("int", (*int)(nil), idInt)
	tInt8       = bootstrapType("int8", (*int8)(nil), idInt8)
	tInt16      = bootstrapType("int16", (*int16)(nil), idInt16)
	tInt32      = bootstrapType("int32", (*int32)(nil), idInt32)
	tInt64      = bootstrapType("int64", (*int64)(nil), idInt64)
	tUint       = bootstrapType("uint", (*uint)(nil), idUint)
	tUint8      = bootstrapType("uint8", (*uint8)(nil), idUint8)
	tUint16     = bootstrapType("uint16", (*uint16)(nil), idUint16)
	tUint32     = bootstrapType("uint32", (*uint32)(nil), idUint32)
	tUint64     = bootstrapType("uint64", (*uint64)(nil), idUint64)
	tUintptr    = bootstrapType("uintptr", (*uintptr)(nil), idUintptr)
	tFloat32    = bootstrapType("float32", (*float32)(nil), idFloat32)
	tFloat64    = bootstrapType("float64", (*float64)(nil), idFloat64)
	tBytes      = bootstrapType("bytes", (*[]byte)(nil), idBytes)
	tString     = bootstrapType("string", (*string)(nil), idString)
	tComplex64  = bootstrapType("complex64", (*complex64)(nil), idComplex64)
	tComplex128 = bootstrapType("complex128", (*complex128)(nil), idComplex128)
	tInterface  = bootstrapType("interface", (*interface{})(nil), idInterface)

	tCommonType = bootstrapType("CommonType", (*CommonType)(nil), idCommonType)
	tArrayType  = bootstrapType("arrayType", (*arrayType)(nil), idArrayType)
	tMapType    = bootstrapType("mapType", (*mapType)(nil), idMapType)
	tSliceType  = bootstrapType("sliceType", (*sliceType)(nil), idSliceType)
	tStructType = bootstrapType("structType", (*structType)(nil), idStructType)
)

// used for building the basic types; called only from init().
// the incoming interface always refers to a pointer.
func bootstrapType(name string, e interface{}, expect ID) ID {
	rt := reflect.TypeOf(e).Elem()
	_, present := types[rt]
	if present {
		panic("bootstrap type already present: " + name + ", " + rt.String())
	}

	dt := newCommonType(name, expect, rt)
	setTypeID(&dt)
	types[rt] = &dt

	gob.Register(reflect.ValueOf(e).Elem())
	return expect
}

func registerBasics() {
	Register(int(0))
	Register(int8(0))
	Register(int16(0))
	Register(int32(0))
	Register(int64(0))
	Register(uint(0))
	Register(uint8(0))
	Register(uint16(0))
	Register(uint32(0))
	Register(uint64(0))
	Register(float32(0))
	Register(float64(0))
	Register(complex64(0i))
	Register(complex128(0i))
	Register(uintptr(0))
	Register(false)
	Register("")
	Register([]byte(nil))
	Register([]int(nil))
	Register([]int8(nil))
	Register([]int16(nil))
	Register([]int32(nil))
	Register([]int64(nil))
	Register([]uint(nil))
	Register([]uint8(nil))
	Register([]uint16(nil))
	Register([]uint32(nil))
	Register([]uint64(nil))
	Register([]float32(nil))
	Register([]float64(nil))
	Register([]complex64(nil))
	Register([]complex128(nil))
	Register([]uintptr(nil))
	Register([]bool(nil))
	Register([]string(nil))
}

func init() {
	registerBasics()
}
