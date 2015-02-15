package marshal

import(
  "reflect"
  "bytes"
  "testing"
  "encoding/binary"
)

// Data Model
type bar struct {
  Id  string
  Pool int64
  Prop map[string]uint32
}

type Foo struct {
  Uri       [0xff]uint8
  DataFlag  [3]uint8
  Version   []uint8
  Ssid      uint16
  Uid       uint32
  SessionId uint32
  Serial    uint32
  Tick      uint32
  Bar       bar
  OK        bool
}

type Pod struct {
  A [0xff]uint8
  B [3]uint8
  C [5]uint32
  D uint64
  F uint64
  G [10]uint64
}

var s_pod = Pod {B:[...]uint8{1,2,3}, C:[...]uint32{1,2,3,4,5}, D:6, F:7, G:[...]uint64{0,1,2,3,4,5,6,7,8,9}}

func createPodObject() *Pod{
  return &s_pod
}

func benchmarkPod() {
  pod := createPodObject()
  result := new (bytes.Buffer)
  binary.Write(result, binary.LittleEndian, *pod)
  var readBack Pod
  binary.Read(result, binary.LittleEndian, &readBack)
}

func BenchmarkBinary(b *testing.B){
  for i := 0; i < b.N; i++ {
    for j := 0; j < 2; j++ {
      for k := 0; k < 5; k++ {
        benchmarkPod()
      }
    }
  }
}

func TestMarshal(t *testing.T){
  orders := []binary.ByteOrder{binary.LittleEndian, binary.BigEndian}
  lengths := []LengthType{BlobLength8, BlobLength16, BlobLength32, BlobLength64, CompactLength}
  for _, o := range orders {
    for _,l := range lengths {
      testCombination(t, o, l)
    }
  }
}

func createTestObject() *Foo {
  return &s_foo;
}

var s_foo = Foo{
  DataFlag:[...]byte{1,2,3},
  Version:[]byte{1,2,3,4,5},
  Ssid:5,
  Uid:6,
  SessionId:7,
  Serial:8,
  Tick: 9,
  Bar:bar{
    "abc",
    2,
    map[string]uint32 {
      "abc":1,
      "def":2,
      "ghi":3,
    },
  },
  OK:true,
}

func BenchmarkMarshal(b *testing.B){
  orders := []binary.ByteOrder{binary.LittleEndian, binary.BigEndian}
  lengths := []LengthType{BlobLength8, BlobLength16, BlobLength32, BlobLength64, CompactLength}
  for i := 0; i < b.N; i++ {
    for _, o := range orders {
      for _,l := range lengths {
        benchmarkCombination(o, l)
      }
    }
  }
}


func benchmarkCombination(order binary.ByteOrder, length LengthType) {
  result := new (bytes.Buffer)
  proto := createTestObject()
  Marshal(proto, result, order, length)
  var readBack Foo
  Unmarshal(&readBack, bytes.NewReader(result.Bytes()), order, length)
}

func testCombination(t *testing.T, order binary.ByteOrder, length LengthType){
  t.Logf("order:%s, length:%s", reflect.TypeOf(order).Name(), reflect.TypeOf(length).Name())
  // iterate through the attributes of a Data Model instance
  result := new (bytes.Buffer)
  proto := createTestObject()

  //binary.Write(result, binary.LittleEndian, &proto)
  if testing.Verbose() {
    DeepPrint(proto, t)
  }

  e := Marshal(proto, result, order, length)
  if e != nil {
    t.Error("error: %v\n", e)
  }

  //buf := result.Bytes()
  t.Logf("result len: %d\n", result.Len())
  //for i, _ := range buf {
    //t.Logf("0x%x,", buf[i:i+1])
  //}
  t.Logf("%x\n", result)

  var readBack Foo
  err := Unmarshal(&readBack, bytes.NewReader(result.Bytes()), order, length)
  t.Logf("err:%v\n", err)
  if testing.Verbose() {
    DeepPrint(readBack, t)
  }
  if reflect.DeepEqual(*proto, readBack) {
    t.Logf("proto and readBack are equal!\n", )
  } else {
    t.Logf("proto and readBack are NOT equal!\n", )
    t.Fail()
  }
}

func DeepPrint(m interface{}, t *testing.T) {
  t.Logf("{\n")
  deepPrint(m, t, 1)
  t.Logf("}\n")
}

func indentPrintf(indent int, t *testing.T, format string, v ...interface{}) {
  for i := 0; i < indent; i++ {
    t.Logf("  ")
  }
  t.Logf(format, v...)
}

func deepPrint(m interface{}, t *testing.T, indent int) {
  v := reflect.ValueOf(m)
  // if a pointer to a struct is passed, get the type of the dereferenced object
  if v.Kind() == reflect.Ptr{
    v = v.Elem()
  }

  if v.Kind() != reflect.Struct {
    t.Logf("%v type can't have attributes inspected\n", v.Kind())
  }
  ty := v.Type()


  // loop through the struct's fields and print them
  for i := 0; i < v.NumField(); i++ {
    p := v.Field(i)
    if p.Kind() == reflect.Ptr {
      p = p.Elem()
    }


    if p.Kind() == reflect.Struct {
      indentPrintf(indent, t, "%s : {\n", ty.Field(i).Name)
      deepPrint(p.Interface(), t, indent + 1)
      indentPrintf(indent, t, "}\n")
    } else {
      indentPrintf(indent, t, "%s : %v\n", ty.Field(i).Name, p.Interface())
    }
  }
}
