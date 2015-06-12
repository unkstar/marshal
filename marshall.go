//Package marshal provide binary encoding method similar to encoding/binary
//except it support varibale length string and array
//array and string length format is defined by LengthType and ByteOrder
//This package depends on encoding/binary.ByteOrder
package marshal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
)

//LengthTypeInstance let you define a new length format,
//you can put instance-wise buffer in each instance to speed up and avoid GCs
type LengthTypeInstance interface {
	Length(io.Reader, binary.ByteOrder, reflect.Kind) int
	PutLength(io.Writer, binary.ByteOrder, reflect.Kind, int)
}

//Function to create LengthTypeInstance, see BlobLength64 for detail
type LengthType func() LengthTypeInstance

//BlobLength64 array and string length is present with 64 bit word
func BlobLength64() LengthTypeInstance {
	return &blobLength64{}
}

type blobLength64 struct {
	b [8]byte
}

func (d *blobLength64) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	var bs []byte
	bs = d.b[:8]
	order.PutUint64(bs, uint64(v))
	_, err := w.Write(bs)
	if err != nil {
		panic(err)
	}
}

func (d *blobLength64) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	var bs []byte
	bs = d.b[:8]
	if _, err := io.ReadFull(r, bs); err != nil {
		panic(err)
	}
	return int(order.Uint64(bs))
}

type bound64 struct {
	length blobLength64
	bound  int
}

func Bound64(bound int) LengthType {
	return func() LengthTypeInstance {
		return &bound64{
			bound: bound,
		}
	}
}

func (d *bound64) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	l := d.length.Length(r, order, k)
	if l > d.bound {
		panic(fmt.Errorf("bound length overflow: %d > %d", l, d.bound))
	}
	return l
}

func (d *bound64) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	if v > d.bound {
		panic(fmt.Errorf("bound length overflow: %d > %d", v, d.bound))
	}
	d.length.PutLength(w, order, k, v)
}

//BlobLength32 array and string length is present with 32 bit word
func BlobLength32() LengthTypeInstance {
	return &blobLength32{}
}

type blobLength32 struct {
	b [4]byte
}

func (d *blobLength32) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	var bs []byte
	bs = d.b[:4]
	order.PutUint32(bs, uint32(v))
	_, err := w.Write(bs)
	if err != nil {
		panic(err)
	}
}

func (d *blobLength32) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	var bs []byte
	bs = d.b[:4]
	if _, err := io.ReadFull(r, bs); err != nil {
		panic(err)
	}
	return int(order.Uint32(bs))
}

type bound32 struct {
	length blobLength32
	bound  int
}

func Bound32(bound int) LengthType {
	return func() LengthTypeInstance {
		return &bound32{
			bound: bound,
		}
	}
}

func (d *bound32) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	l := d.length.Length(r, order, k)
	if l > d.bound {
		panic(fmt.Errorf("bound length overflow: %d > %d", l, d.bound))
	}
	return l
}

func (d *bound32) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	if v > d.bound {
		panic(fmt.Errorf("bound length overflow: %d > %d", v, d.bound))
	}
	d.length.PutLength(w, order, k, v)
}

//BlobLength16 array and string length is present with 16 bit word
func BlobLength16() LengthTypeInstance {
	return &blobLength16{}
}

type blobLength16 struct {
	b [2]byte
}

func (d *blobLength16) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	var bs []byte
	bs = d.b[:2]
	order.PutUint16(bs, uint16(v))
	_, err := w.Write(bs)
	if err != nil {
		panic(err)
	}
}

func (d *blobLength16) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	var bs []byte
	bs = d.b[:2]
	if _, err := io.ReadFull(r, bs); err != nil {
		panic(err)
	}
	return int(order.Uint16(bs))
}

//BlobLength8 array and string length is present with 8 bit word
func BlobLength8() LengthTypeInstance {
	return &blobLength8{}
}

type blobLength8 struct {
	b [1]byte
}

func (d *blobLength8) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	var bs []byte
	bs = d.b[:1]
	bs[0] = uint8(v)
	_, err := w.Write(bs)
	if err != nil {
		panic(err)
	}
}

func (d *blobLength8) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	var bs []byte
	bs = d.b[:1]
	if _, err := io.ReadFull(r, bs); err != nil {
		panic(err)
	}
	return int(bs[0])
}

//CompactLength provide compact length in Tight-VNC encoding
func CompactLength() LengthTypeInstance {
	return &compactLength{}
}

type compactLength struct {
	b [3]byte
}

func (d *compactLength) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	var bs []byte
	d.b[0] = byte(v & 0x7f)
	if v > 0x7f {
		d.b[0] |= 0x80
		d.b[1] = byte(v>>7) & 0x7f
		if v > 0x3fff {
			d.b[1] |= 0x80
			d.b[2] = byte(v >> 14)
			if v > 0x3fffff {
				panic(errors.New("compactLen overflow, value=" + string(v)))
			} else {
				bs = d.b[:]
			}
		} else {
			bs = d.b[:2]
		}
	} else {
		bs = d.b[:1]
	}
	_, err := w.Write(bs)
	if err != nil {
		panic(err)
	}
}

func (d *compactLength) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	bs := d.b[:1]
	var v int
	if _, err := io.ReadFull(r, bs); err != nil {
		panic(err)
	}
	v = int(d.b[0]) & 0x7f
	if d.b[0]&0x80 != 0 {
		if _, err := io.ReadFull(r, bs); err != nil {
			panic(err)
		}
		v |= (int(d.b[0]) & 0x7f) << 7
		if d.b[0]&0x80 != 0 {
			if _, err := io.ReadFull(r, bs); err != nil {
				panic(err)
			}
			v |= int(d.b[0]) << 14
		}
	}
	return v
}

//errrrr...
type YYBlobTypeInstance struct {
	length blobLength32
}

func YYBlobType() LengthTypeInstance {
	return &YYBlobTypeInstance{}
}

func (d *YYBlobTypeInstance) Length(r io.Reader, order binary.ByteOrder, k reflect.Kind) int {
	if k != reflect.String {
		return d.length.Length(r, order, k)
	} else {
		var bs []byte
		bs = d.length.b[:2]
		if _, err := io.ReadFull(r, bs); err != nil {
			panic(err)
		}
		return int(order.Uint16(bs))
	}
}

func (d *YYBlobTypeInstance) PutLength(w io.Writer, order binary.ByteOrder, k reflect.Kind, v int) {
	if k != reflect.String {
		d.length.PutLength(w, order, k, v)
	} else {
		var bs []byte
		bs = d.length.b[:2]
		order.PutUint16(bs, uint16(v))
		_, err := w.Write(bs)
		if err != nil {
			panic(err)
		}
	}
}

type marshaler struct {
	buf   [8]byte
	w     io.Writer
	order binary.ByteOrder
}

func (m *marshaler) flush(sz int) {
	bs := m.buf[0:sz]
	_, err := m.w.Write(bs)
	if err != nil {
		panic(err)
	}
}

func (m *marshaler) uint8(x uint8) {
	m.buf[0] = x
	m.flush(1)
}

func (m *marshaler) uint16(x uint16) {
	m.order.PutUint16(m.buf[0:2], x)
	m.flush(2)
}

func (m *marshaler) uint32(x uint32) {
	m.order.PutUint32(m.buf[0:4], x)
	m.flush(4)
}

func (m *marshaler) uint64(x uint64) {
	m.order.PutUint64(m.buf[0:8], x)
	m.flush(8)
}

func (m *marshaler) int8(x int8) { m.uint8(uint8(x)) }

func (m *marshaler) int16(x int16) { m.uint16(uint16(x)) }

func (m *marshaler) int32(x int32) { m.uint32(uint32(x)) }

func (m *marshaler) int64(x int64) { m.uint64(uint64(x)) }

//Marshal put binary presentation of v into w. Bytes written to w are encoded using specified byte order and length type
func Marshal(v interface{}, w io.Writer, order binary.ByteOrder, length LengthType) (err error) {
	defer func() {
		if e := recover(); e != nil {
			switch v := e.(type) {
			case error:
				err = v
			case string:
				err = errors.New("marshal error:" + v)
			default:
				panic(e) //repanic
			}
		}
	}()
	m := &marshaler{w: w, order: order}
	m.marshal(reflect.ValueOf(v), length())
	return nil
}

func (m *marshaler) marshal(v reflect.Value, length LengthTypeInstance) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	kind := v.Kind()
	switch kind {
	case reflect.String:
		l := v.Len()
		length.PutLength(m.w, m.order, kind, l)
		if l != 0 {
			if _, e := m.w.Write([]byte(v.String())); nil != e {
				panic(e)
			}
		}
	case reflect.Struct:
		// loop through the struct's fields and set the map
		for i := 0; i < v.NumField(); i++ {
			m.marshal(v.Field(i), length)
		}
	case reflect.Map:
		l := v.Len()
		length.PutLength(m.w, m.order, kind, l)
		keys := v.MapKeys()
		for i := 0; i < l; i++ {
			m.marshal(keys[i], length)
			m.marshal(v.MapIndex(keys[i]), length)
		}
	case reflect.Array, reflect.Slice:
		l := v.Len()
		if v.Kind() == reflect.Slice {
			length.PutLength(m.w, m.order, kind, l)
		}
		kind := v.Type().Elem().Kind()
		if kind == reflect.Uint8 || kind == reflect.Int8 {
			//fast path for []byte
			if _, e := m.w.Write(v.Slice(0, l).Bytes()); nil != e {
				panic(e)
			}
		} else {
			for i := 0; i < l; i++ {
				m.marshal(v.Index(i), length)
			}
		}
	case reflect.Bool:
		if v.Bool() {
			m.uint8(1)
		} else {
			m.uint8(0)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v.Type().Kind() {
		case reflect.Int8:
			m.int8(int8(v.Int()))
		case reflect.Int16:
			m.int16(int16(v.Int()))
		case reflect.Int32:
			m.int32(int32(v.Int()))
		case reflect.Int64:
			m.int64(v.Int())
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch v.Type().Kind() {
		case reflect.Uint8:
			m.uint8(uint8(v.Uint()))
		case reflect.Uint16:
			m.uint16(uint16(v.Uint()))
		case reflect.Uint32:
			m.uint32(uint32(v.Uint()))
		case reflect.Uint64:
			m.uint64(v.Uint())
		}

	case reflect.Float32, reflect.Float64:
		switch v.Type().Kind() {
		case reflect.Float32:
			m.uint32(math.Float32bits(float32(v.Float())))
		case reflect.Float64:
			m.uint64(math.Float64bits(v.Float()))
		}

	case reflect.Complex64, reflect.Complex128:
		switch v.Type().Kind() {
		case reflect.Complex64:
			x := v.Complex()
			m.uint32(math.Float32bits(float32(real(x))))
			m.uint32(math.Float32bits(float32(imag(x))))
		case reflect.Complex128:
			x := v.Complex()
			m.uint64(math.Float64bits(real(x)))
			m.uint64(math.Float64bits(imag(x)))

		default:
			panic(errors.New("unsupport type" + v.Type().Name()))
		}
	}
}

//Unmarshal read binary presentation of data from r into m. Bytes read from r must be encoded using specified byte order and length type.
//When reading into struct, all non-blank field must be exported
func Unmarshal(m interface{}, r io.Reader, order binary.ByteOrder, length LengthType) (err error) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Ptr {
		return errors.New("unmarshal: invalid type " + v.Type().String())
	}
	defer func() {
		if e := recover(); e != nil {
			switch v := e.(type) {
			case error:
				err = v
			case string:
				err = errors.New("unmarshal error:" + v)
			default:
				panic(e) //repanic
			}
		}
	}()
	u := &unmarshaler{r: r}
	u.unmarshal(v.Elem(), order, length())
	return nil
}

type unmarshaler struct {
	buf [8]byte
	r   io.Reader
}

func (u *unmarshaler) fetch(b int) (bs []byte) {
	bs = u.buf[:b]
	if _, e := io.ReadFull(u.r, bs); e != nil {
		panic(e)
	}
	return
}

func (u *unmarshaler) unmarshal(v reflect.Value, order binary.ByteOrder, length LengthTypeInstance) {
	kind := v.Kind()
	switch kind {
	case reflect.String:
		l := length.Length(u.r, order, kind)
		if l != 0 {
			bs := make([]byte, l)
			if _, e := io.ReadFull(u.r, bs); e != nil {
				panic(e)
			}
			v.SetString(string(bs))
		}
	case reflect.Struct:
		// loop through the struct's fields and set the map
		for i := 0; i < v.NumField(); i++ {
			u.unmarshal(v.Field(i), order, length)
		}
	case reflect.Map:
		l := length.Length(u.r, order, kind)
		if l != 0 {
			v.Set(reflect.MakeMap(v.Type()))
			keyType := v.Type().Key()
			elemType := v.Type().Elem()
			for i := 0; i < l; i++ {
				key := reflect.New(keyType)
				u.unmarshal(key.Elem(), order, length)
				elem := reflect.New(elemType)
				u.unmarshal(elem.Elem(), order, length)
				v.SetMapIndex(key.Elem(), elem.Elem())
			}
		}
	case reflect.Array, reflect.Slice:
		var l int
		if reflect.Slice == v.Kind() {
			l = length.Length(u.r, order, kind)
		} else {
			l = v.Len()
		}
		if l != 0 {
			if v.Kind() == reflect.Slice {
				v.Set(reflect.MakeSlice(v.Type(), l, l))
			}
			kind := v.Type().Elem().Kind()
			if kind == reflect.Uint8 || kind == reflect.Int8 {
				//fast path for []byte
				buf := v.Slice(0, l).Bytes()
				u.r.Read(buf)
			} else {
				for i := 0; i < l; i++ {
					u.unmarshal(v.Index(i), order, length)
				}
			}
		}
	case reflect.Bool:
		v.SetBool(u.fetch(1)[0] != 0)
	case reflect.Int8:
		v.SetInt(int64(u.fetch(1)[0]))
	case reflect.Int16:
		v.SetInt(int64(order.Uint16(u.fetch(2))))
	case reflect.Int32:
		v.SetInt(int64(order.Uint32(u.fetch(4))))
	case reflect.Int64:
		v.SetInt(int64(order.Uint64(u.fetch(8))))
	case reflect.Uint8:
		v.SetUint(uint64(u.fetch(1)[0]))
	case reflect.Uint16:
		v.SetUint(uint64(order.Uint16(u.fetch(2))))
	case reflect.Uint32:
		v.SetUint(uint64(order.Uint32(u.fetch(4))))
	case reflect.Uint64:
		v.SetUint(order.Uint64(u.fetch(8)))

	case reflect.Float32:
		v.SetFloat(float64(math.Float32frombits(order.Uint32(u.fetch(4)))))
	case reflect.Float64:
		v.SetFloat(math.Float64frombits(order.Uint64(u.fetch(8))))

	case reflect.Complex64:
		v.SetComplex(complex(
			float64(math.Float32frombits(order.Uint32(u.fetch(4)))),
			float64(math.Float32frombits(order.Uint32(u.fetch(4)))),
		))
	case reflect.Complex128:
		v.SetComplex(complex(
			math.Float64frombits(order.Uint64(u.fetch(8))),
			math.Float64frombits(order.Uint64(u.fetch(8))),
		))
	default:
		panic(errors.New("unsupport type" + v.Type().Name()))
	}
}
