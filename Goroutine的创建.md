[原文](https://zhuanlan.zhihu.com/p/105388126)

Go代码中，利用关键字`go`启动协程。编译器发现go func(...)，将调用newproc
```go
package main
go func(){...}

/*
可以使用go tool compile -S ./main.go得到汇编代码
CALL	runtime.newproc(SB)
*/
```

## func newproc(fn *funcval)

1. 创建一个g来运行fn
2. 将g放入g等待队列中,等待被调度
3. 编译器会把go语句转化为调用newproc

```go
func newproc(fn *funcval) {
    // 【获取当前调用方正在运行的G】
    gp := getg()
    // 【获取当前调用方 PC/IP 寄存器值】
    pc := getcallerpc()
    // 【用 g0 栈创建 G 对象】
    systemstack(func () {
        newg := newproc1(fn, gp, pc)

        _p_ := getg().m.p.ptr()
        // newg 放入待运行队列
        runqput(_p_, newg, true)

        if mainStarted {
            // wackp 核心思想就是 寻找资源 执行newg
            wakep()
        }
    })
}
```

### fn *funcval
其中 newproc 函数有1个参数fn 是一个可变参数类型

```go
type funcval struct {
    fn uintptr
    // variable-size, fn-specific data here
}
```

如果我们有go程序

```go

func add(x, y int) int {
    z := x + y
    return z
}
func main() {
    x := 0x100
    y := 0x200
    go add(x, y)
}
```

那么对于 newproc 中参数 fn 结构体，扩展出来是这样的：

```
type funcval struct {
    fn uintptr
    x int
    y int
}
```
所以用”fn+ptrsize“跳过第一个函数指针参数，就可以获得参数 x 的地址

### getg()、getcallerpc()

#### getg()返回当前`G`的指针；函数如下：
```go
// getg returns the pointer to the current g.
// The compiler rewrites calls to this function into instructions
// 编译器会把这个 getg 指令翻译成从专用寄存器取
// that fetch the g directly (from TLS or from the dedicated register).
func getg() *g
```

直接从寄存器中读取就行。参考如下汇编代码：
Go1.17 R14寄存器存的就是g地址
```
TEXT runtime.acquirem(SB) /usr/local/go/src/runtime/runtime1.go
      0x104a3e0   MOVQ 0x30(R14), CX ;; CX = &g     
      0x104a3e4   INCL 0x108(CX)  ;; g.m.locks++         
      0x104a3ea   MOVQ 0x30(R14), AX  ;; AX= &m    
      0x104a3ee   RET ;;return &m
```
0x30(R14) 代表的是g.m ,这里可以观察下 g结构体
```go
type g struct {
	stack       stack   // offset 0x0
	stackguard0 uintptr // offset 0x10
	stackguard1 uintptr // offset 0x18

	_panic    *_panic // offset 0x20
	_defer    *_defer // offset 0x28
	m         *m      // offset 0x30
	...
```

#### getcallerpc()函数和getcallersp()函数是一对。
前者返回程序计数寄存器指针；后者返回栈顶指针。
但是要注意：getcallersp的结果在返回时是正确的，
但是它可能会因为随后对函数的调用导致栈扩容而失效。
一般规则是应该立即使用getcallersp的结果且只能传递给nosplit函数。

## func newproc1(fn *funcval, callergp *g, callerpc uintptr) *g
- 本函数创建一个_Grunnable的g
- g执行从fn开始
- callerpc是调用go func 的语句的地方
- caller有义务将新创建的g 加入运行时调度

### 源码1
```
// Create a new g in state _Grunnable, starting at fn. callerpc is the
// address of the go statement that created this. The caller is responsible
// for adding the new g to the scheduler.
func newproc1(fn *funcval, callergp *g, callerpc uintptr) *g {
    _g_ := getg()

    //【1】fn函数指针不能为空；为空时，就设置此时与g相关联m的throwing变量值；顺便抛出异常。
    if fn == nil {
        _g_.m.throwing = -1 // do not dump full stacks
        throw("go of nil func value")
    }
    // 【2】禁用抢占，因为在接下来的执行中,会使用到p,在此期间,不允许 p和m分离
    acquirem() // disable preemption because it can be holding p in a local var

    // 【3】获取p,然后从p.gfree中取一个g
    _p_ := _g_.m.p.ptr()
    // 【4】gfget获取p中的free g或者从全局gfree 里取一个g
    newg := gfget(_p_)

    if newg == nil {
        // 【5】malg 如果没有可用的g,就申请一个新g
        newg = malg(_StackMin)
        // 将g的状态改成_Gdead
        casgstatus(newg, _Gidle, _Gdead)
        // 将新g加入 allg，_Gdead状态保证了gc 不会去关注新g的栈空间
        allgadd(newg) // publishes with a g->status of Gdead so GC scanner doesn't look at uninitialized stack.
    }

```

#### gfget
gfget核心思想其实就是复用g,从gfree链表里取
如果本地队列为空，就从全局队列里取

首先要明白本地list结构体和全局list结构体的声明：
```
// p本地
// 用法：
// p.gFree.glist.pop() | .push()
// p.gFree.n-- | .n++


gFree struct {
    gList
    n int32
}

// 全局
// Global cache of dead G's.
gFree struct {
    lock    mutex
    stack   gList // Gs with stacks
    noStack gList // Gs without stacks
    n       int32
}
```
```
// Get from gfree list.
// If local list is empty, grab a batch from global list.
func gfget(_p_ *p) *g {
retry:
    // _p_.gFree.empty() 如果本地队列为空
    // !sched.gFree.stack.empty() 并且全局有栈队列不为空 
    // || !sched.gFree.noStack.empty() 或 全局无栈链表不为空
    // 则进入该分支
    if _p_.gFree.empty() && (!sched.gFree.stack.empty() || !sched.gFree.noStack.empty()) {
        // 全局gFree加锁
        lock(&sched.gFree.lock)
        // Move a batch of free Gs to the P.
        // 将最多32个free g加入P的本地gfree链表
        for _p_.gFree.n < 32 {
            // Prefer Gs with stacks.
            // 优先有栈g
            gp := sched.gFree.stack.pop()
            if gp == nil {
                // 优先有栈g，实在没有就使用无栈g
                gp = sched.gFree.noStack.pop()
                if gp == nil {
                    break
                }
            }
            // 每次将全局g加入p.gfree ，就将全局g的数量减一
            sched.gFree.n--
            _p_.gFree.push(gp)
            _p_.gFree.n++
        }
        // 全局gFree解锁
        unlock(&sched.gFree.lock)
        goto retry
    }

    /* 【本地链表非空，就出栈；判断g是否为nil，是nil直接返回 表面本地和全局都无free g】 */
    gp := _p_.gFree.pop()
    if gp == nil {
        return nil
    }
    _p_.gFree.n--
    if gp.stack.lo == 0 {
        // 如果是空栈g 就分配一个栈空间
        // Stack was deallocated in gfput. Allocate a new one.
        systemstack(func() {
            gp.stack = stackalloc(_FixedStack)
        })
        // 设置g分裂的保护线
        gp.stackguard0 = gp.stack.lo + _StackGuard
    } 
    return gp
}

```

### 源码2
```

    totalSize := uintptr(4*goarch.PtrSize + sys.MinFrameSize) // extra space in case of reads slightly beyond frame
    totalSize = alignUp(totalSize, sys.StackAlign)
    sp := newg.stack.hi - totalSize
    spArg := sp
    if usesLR {
        // caller's LR
        *(*uintptr)(unsafe.Pointer(sp)) = 0
        prepGoExitFrame(sp)
        spArg += sys.MinFrameSize
    }
    // 上面几行代码就是为了确定sp的位置
    // 清空 g.sched 目的是 初始化 gobuf(g切换用于保护现场的结构)
    memclrNoHeapPointers(unsafe.Pointer(&newg.sched), unsafe.Sizeof(newg.sched))

    newg.sched.sp = sp
    newg.stktopsp = sp
    newg.sched.pc = abi.FuncPCABI0(goexit) + sys.PCQuantum // +PCQuantum so that previous instruction is in same function
    newg.sched.g = guintptr(unsafe.Pointer(newg))
    gostartcallfn(&newg.sched, fn)

    // g.gopc代表返回地址是调用方执行go func 的地方
    newg.gopc = callerpc
    // saveAncestors 此函数用于保存自己的”祖先“；此函数还会设置”轨迹trace“，我们可以用”go tool trace“来跟踪go程序中的线程。
    // 参考 https://lessisbetter.site/2019/03/26/golang-scheduler-2-macro-view/
    newg.ancestors = saveAncestors(callergp)
    // g开始执行地方是 fn.fn
    newg.startpc = fn.fn
    if _g_.m.curg != nil {
        // 这是一个判断。除了g0，每个G的创建都由其他的G调用”go func()“执行；这个调用的G就是curg。
        // 如果创建这个G存在一个这样的curg，那么他们的标签设置为一样的；此标签也可以用于分析器的跟踪。
        newg.labels = _g_.m.curg.labels
    }
    if isSystemGoroutine(newg, false) {
        atomic.Xadd(&sched.ngsys, +1)
    }
    // Track initial transition?
    newg.trackingSeq = uint8(fastrand())
    if newg.trackingSeq%gTrackingPeriod == 0 {
        newg.tracking = true
    }
    // 将newg状态 设置成 _Grunnable
    casgstatus(newg, _Gdead, _Grunnable)

    if _p_.goidcache == _p_.goidcacheend {
        // 如果本地p已经没有可分配的goid了就尝试获取 _GoidCacheBatch=16 个

        // Sched.goidgen is the last allocated id,
        // this batch must be [sched.goidgen+1, sched.goidgen+GoidCacheBatch].
        // At startup sched.goidgen=0, so main goroutine receives goid=1.
        _p_.goidcache = atomic.Xadd64(&sched.goidgen, _GoidCacheBatch)
        _p_.goidcache -= _GoidCacheBatch - 1
        _p_.goidcacheend = _p_.goidcache + _GoidCacheBatch
    }
    // 给newg 一个唯一id
    newg.goid = int64(_p_.goidcache)
    _p_.goidcache++

    releasem(_g_.m)

    return newg

```

## func runqput(_p_ *p, gp *g, next bool)
runqput尝试将g加入p.runnext
并将 old runnext加入 p的本地队列中
如果本地队列是满了，就把g和一半本地队列 加入全局队列 参考runqputslow
```
// runqput tries to put g on the local runnable queue.
// If next is false, runqput adds g to the tail of the runnable queue.
// If next is true, runqput puts g in the _p_.runnext slot.
// If the run queue is full, runnext puts g on the global queue.
// Executed only by the owner P.
func runqput(_p_ *p, gp *g, next bool) {
    if randomizeScheduler && next && fastrandn(2) == 0 {
        next = false
    }

    if next {
    // runqput尝试将g加入p.runnext
    retryNext:
        oldnext := _p_.runnext
        if !_p_.runnext.cas(oldnext, guintptr(unsafe.Pointer(gp))) {
            goto retryNext
        }
        if oldnext == 0 {
            return
        }
        // Kick the old runnext out to the regular run queue.
        gp = oldnext.ptr()
    }

retry:
    // 并将 old runnext加入 p的本地队列中
    h := atomic.LoadAcq(&_p_.runqhead) // load-acquire, synchronize with consumers
    t := _p_.runqtail
    if t-h < uint32(len(_p_.runq)) {
        _p_.runq[t%uint32(len(_p_.runq))].set(gp)
        atomic.StoreRel(&_p_.runqtail, t+1) // store-release, makes the item available for consumption
        return
    }
    // 如果本地队列是满了，就把g和一半本地队列 加入全局队列 参考runqputslow
    if runqputslow(_p_, gp, h, t) {
        return
    }
    // the queue is not full, now the put above must succeed
    goto retry
}
```
