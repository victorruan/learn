[原文地址](https://zhuanlan.zhihu.com/p/102562318)

# 源码角度解析Go语言并发[1]---M,P,G的定义，状态转换及一些"边角料"

## 1. Go程序入口——m0、g0
linux下,可以通过readelf工具和dlv查找Go程序的入口
```
# 编译文本文件成为可执行文件
go build hello.go
# 通过readelf观察整个程序的Entry point address
readelf -h ./hello | grep 'Entry point address'
# 通过dlv给可执行程序打断定的方式，找到程序入口
dlv exec ./hello
b *0x45bc80
# /path/to/golang/src/runtime/rt0_linux_amd64.s:8
```

通过工具找到入口rt0_linux_amd64.s
### rt0_linux_amd64.s
启动文件没什么,主要是跳转到 `_rt0_amd64`
```
TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
	JMP	_rt0_amd64(SB)
```

### asm_amd64.s
go源代码全局搜索 `TEXT rt0_amd64(SB)` 并找到以`_amd64.s`后缀的文件 asm_amd64.s
可以看到注释:
> _rt0_amd64 是大多数`amd64`系统通用**启动代码**
> 
> 这里是可执行程序的`entry point`
> 
> 栈顶储存了参数数量`argc`以及C风格的`argv`

```
// _rt0_amd64 is common startup code for most amd64 systems when using
// internal linking. This is the entry point for the program from the
// kernel for an ordinary -buildmode=exe program. The stack holds the
// number of arguments and the C-style argv.
TEXT _rt0_amd64(SB),NOSPLIT,$-8
	MOVQ	0(SP), DI	// argc
	LEAQ	8(SP), SI	// argv
	JMP	runtime·rt0_go(SB)
```
注意到这里仅仅是将参数准备好,真实的执行入口是 `runtime·rt0_go(SB)`
我们把`runtime·rt0_go(SB)`的代码截取一部分看看，它做了什么
```
TEXT runtime·rt0_go(SB),NOSPLIT|TOPFRAME,$0
	// copy arguments forward on an even stack
	MOVQ	DI, AX		// argc
	MOVQ	SI, BX		// argv
	SUBQ	$(4*8+7), SP		// 2args 2auto
	ANDQ	$~15, SP
	MOVQ	AX, 16(SP)
	MOVQ	BX, 24(SP)

```
上面的代码是参数拷贝,我们可以不用关心


```
	// create istack out of the given (operating system) stack.
	// _cgo_init may update stackguard.
	MOVQ	$runtime·g0(SB), DI            ;; DI=&g0 //`DI寄存器`存的就是g0的地址
	LEAQ	(-64*1024+104)(SP), BX         
	MOVQ	BX, g_stackguard0(DI)
	MOVQ	BX, g_stackguard1(DI)
	MOVQ	BX, (g_stack+stack_lo)(DI)
	MOVQ	SP, (g_stack+stack_hi)(DI)
```
上面这段代码比较有意思了，应该是初始化g0的一部分，注意目前 `DI寄存器`存的就是g0的地址

分别是 `g0.stackguard0`、`g0.stackguard1`、`g.stack`

注释里还强调 _cgo_init 时，可能会更改 stackguard ，不过这个不是我们关注的重点

```
	// find out information about the processor we're on
	MOVL	$0, AX
	CPUID
	MOVL	AX, SI
	CMPL	AX, $0
	JE	nocpuinfo

	CMPL	BX, $0x756E6547  // "Genu"
	JNE	notintel
	CMPL	DX, $0x49656E69  // "ineI"
	JNE	notintel
	CMPL	CX, $0x6C65746E  // "ntel"
	JNE	notintel
	MOVB	$1, runtime·isIntel(SB)
	
```

上面的这段代码的目的是找到cpu相关的信息，

不过这个不是我们关注的重点

如果是Intel芯片,runtime·isIntel(SB)会被置为1

```
notintel:

	// Load EAX=1 cpuid flags
	MOVL	$1, AX
	CPUID
	MOVL	AX, runtime·processorVersionInfo(SB)

```
设置非intel的处理器信息 `runtime·processorVersionInfo(SB)`

```
	// update stackguard after _cgo_init
	MOVQ	$runtime·g0(SB), CX
	MOVQ	(g_stack+stack_lo)(CX), AX
	ADDQ	$const__StackGuard, AX
	MOVQ	AX, g_stackguard0(CX)
	MOVQ	AX, g_stackguard1(CX)

```
执行完_cgo_init后
更新stackguard
```

	LEAQ	runtime·m0+m_tls(SB), DI
	CALL	runtime·settls(SB)

	// store through it, to make sure it works
	get_tls(BX)
	MOVQ	$0x123, g(BX)
	MOVQ	runtime·m0+m_tls(SB), AX
	CMPQ	AX, $0x123
	JEQ 2(PC)
	CALL	runtime·abort(SB)
	
```
设置m0.tls的相关信息，当然有很多操作系统并不需要tls信息
这样就会直接跳转到下面的代码

```
ok:
	// set the per-goroutine and per-mach "registers"
	get_tls(BX)
	LEAQ	runtime·g0(SB), CX
	MOVQ	CX, g(BX)
	LEAQ	runtime·m0(SB), AX

	// save m->g0 = g0
	MOVQ	CX, m_g0(AX)
	// save m0 to g0->m
	MOVQ	AX, g_m(CX)

	CLD				// convention is D is always left cleared
	CALL	runtime·check(SB)
```
将m0与g0互相绑定

```
	MOVL	16(SP), AX		// copy argc
	MOVL	AX, 0(SP)
	MOVQ	24(SP), AX		// copy argv
	MOVQ	AX, 8(SP)
	CALL	runtime·args(SB)
	CALL	runtime·osinit(SB)
	CALL	runtime·schedinit(SB)

	// create a new goroutine to start program
	MOVQ	$runtime·mainPC(SB), AX		// entry
	PUSHQ	AX
	CALL	runtime·newproc(SB)
	POPQ	AX

	// start this M
	CALL	runtime·mstart(SB)

	CALL	runtime·abort(SB)	// mstart should never return
	RET

	// Prevent dead-code elimination of debugCallV2, which is
	// intended to be called by debuggers.
	MOVQ	$runtime·debugCallV2<ABIInternal>(SB), AX
	RET
```
一系列初始化，并创建一个新g去执行 runtime.main
- CALL	runtime·args(SB) // 参数准备
- CALL	runtime·osinit(SB) //
- CALL	runtime·schedinit(SB) // 调度器初始化
- CALL	runtime·newproc(SB) // 创建g 
- CALL	runtime·mstart(SB) // 创建m 用上面的g栈 执行runtime.main


## 2. M,P,G

## 3. 调度——框架