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
