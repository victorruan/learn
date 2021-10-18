# 内存管理单元

`runtime.mspan`是Go语言内存管理的基本单元。

每个`runtime.mspan`都管理`npages`个大小为`8KB`的页。

```go
type mspan struct {
	startAddr uintptr  // 起始地址
	npages    uintptr  // 页数
	freeindex uintptr  // 下一个空闲对象的索引

	allocBits  *gcBits // 标记内存的占用
	gcmarkBits *gcBits // 标记内存的回收
	allocCache uint64  // 用于快速找到下一个空闲对象的辅助字段
	...
}
```



## allocCache
![img.png](./allocCache.png)

上图有几处不清晰的地方需要指出.

1. allocCache是64bit , 上图没有表达清楚
2. allocCache是从 `startAddr + freeindex` 开始的,上图的 startAddr 有误



## 状态

运行时会使用 `runtime.mSpanStateBox`存储内存管理单元的状态 `runtime.mSpanState`：

```go
type mspan struct {
	...
	state       mSpanStateBox
	...
}
```

该状态可能处于 `mSpanDead`、`mSpanInUse`、`mSpanManual` 和 `mSpanFree` 四种情况。当 `runtime.mspan`在空闲堆中，它会处于 `mSpanFree` 状态；当 `runtime.mspan `已经被分配时，它会处于`mSpanInUse`、`mSpanManual` 状态
