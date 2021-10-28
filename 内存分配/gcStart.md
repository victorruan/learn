## gcStart的三种触发方式

Go中的gcStart是Gc的入口函数，有3种情况可以触发该函数的执行。

1. 用户程序主动调用`runtime.GC`函数
2. 周期触发的方式 如果程序一直没执行GC，则每2分钟由sysmon强制执行一次
3. 内存占用达到阈值会执行GC

用户主动调用的情况比较少。 下面我们来分析下2和3。


### 周期触发的方式

#### src/runtime/proc.go
```go
// forcegcperiod is the maximum time in nanoseconds between garbage
// collections. If we go this long without a garbage collection, one
// is forced to run.
//
// This is a variable for testing purposes. It normally doesn't change.
var forcegcperiod int64 = 2 * 60 * 1e9
```
上面的全局变量定义了周期触发的时间阈值，单位是ns。设计成变量而不是常量，是为了方便测试。
目前这个时间是2分钟。也就是说，如果程序一直没执行GC，则每2分钟由sysmon强制执行一次。

#### src/runtime/mgc.go
```go
// test reports whether the trigger condition is satisfied, meaning
// that the exit condition for the _GCoff phase has been met. The exit
// condition should be tested when allocating.
func (t gcTrigger) test() bool {
	...
	switch t.kind {
	...
	case gcTriggerTime:
		if atomic.Loadint32(&gcController.gcPercent) < 0 {
			return false
		}
		lastgc := int64(atomic.Load64(&memstats.last_gc_nanotime))
		return lastgc != 0 && t.now-lastgc > forcegcperiod
	...
	}
    ...
}
```
上面的代码可以看看`gcTriggerTime`分支，如果程序GOGC变量没有被设置成`off`,
程序会检查当前时间与上一次gc的时间的差值是否已经大于2分钟。如果超过2分钟，就会允许执行GC了。

### 内存占用阈值触发GC

#### src/runtime/mgc.go
```go
func (t gcTrigger) test() bool {
	...
	switch t.kind {
	case gcTriggerHeap:
		// Non-atomic access to gcController.heapLive for performance. If
		// we are going to trigger on this, this thread just
		// atomically wrote gcController.heapLive anyway and we'll see our
		// own write.
		return gcController.heapLive >= gcController.trigger
	...
}
```
上诉代码可以看出 当全局变量 gcController.heapLive >= gcController.trigger时，就会触发GC