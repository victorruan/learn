# [类型系统还挺重要](https://www.bilibili.com/video/BV1iZ4y1T7zF?p=1)

## 内置类型
- int
- string
- slice
- map
- func
- ....

## 自定义类型
```go
type T int

type T struct{
	name string
}

type I interface{
	Name() string
}
```

## 类型元数据
给内置类型定义方法是不被允许的

数据类型虽然多，但是不管是内置类型还是自定义类型都有对应的类型描述信息

称之为它的 `类型元数据`，每种类型元数据都是全局唯一的，

这些类型元数据共同组成了go的类型系统

### 类型元数据 runtime._type 包含的信息
- 类型大小
- 类型的名称
- 对齐边界
- 是否自定义
- ...... 

#### src/runtime/type.go
```
type _type struct {
	size       uintptr
	ptrdata    uintptr // size of memory prefix holding all pointers
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	......
}
```

runtime._type实际上是每个类型元数据的header，
对于slicetype，除了会存slice本身的类型元数据
还会存slice所存数据的类型元数据的指针信息

```
type slicetype struct {
	typ  _type
	elem *_type
}
```
如果是 []string ,那么elem就指向string类型的类型元数据 string type

```
type uncommontype struct {
	pkgpath nameOff
	mcount  uint16 // number of methods
	xcount  uint16 // number of exported methods
	moff    uint32 // offset from this uncommontype to [mcount]method
}
```
如果是自定义类型,类型元数据里还会存 一个uncommontype的结构体
- pkgpath是类型所在的包路径
- mcount是方法的数量
- xcount是导出方法的数量,大写方法的数量
- moff是方法地址偏移量

### 举个自定义类型的例子
```go
type myslice []string

func (ms myslice) Len()  {
    ...
}

func (ms myslice) Cap()  {
    ...
}
```
slicetype = _type + stringtype

myslice的类型元数据 = slicetype+uncommontype

`&uncommontype+moff`就是myslice关联的方法数组 Len,Cap 