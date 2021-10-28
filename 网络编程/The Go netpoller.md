# [Go网络轮询器（The Go netpoller）](https://morsmachine.dk/netpoller)

##  简介（Introduction）

> I'm bored again or I have something more [important](http://www.structuredprocrastination.com/) to do, so it's time for another blog post about the Go runtime. This time I'm gonna take a look at how Go handles network I/O.

本文我会简单的分析下Go是如何处理网络IO的。

## 阻塞（Blocking）

> In Go, all I/O is blocking. 

在Go中，所有的IO都是阻塞的。

> The Go ecosystem is built around the idea that you write against a blocking interface and then handle concurrency through goroutines and channels rather than callbacks and futures. 

先设计一个阻塞的接口，然后结合goroutines 和 channels来实现并发。Go生态就是基于这样一个理念来构建的。

> An example is the HTTP server in the "net/http" package. 
>
> Whenever it accepts a connection, it will create a new goroutine to handle all the requests that will happen on that connection. This construct means that the request handler can be written in a very straightforward manner. First do this, then do that. Unfortunately, using the blocking I/O provided by the operating system isn't suitable for constructing our own blocking I/O interface.

"net/http" package 中的 HTTP server 就是一个简单的例子。当它接收了一个连接，它就会新创建一个goroutine来处理这个连接上的所有请求。这种设计让我们可以用非常简单的方式来处理请求。先做什么，再做什么。然而不幸的是，使用操作系统提供阻塞IO，并不能很好的适配上述设计。



> In my [previous post](http://morsmachine.dk/go-scheduler) about the Go runtime, I covered how the Go scheduler handles syscalls. To handle a blocking syscall, we need a thread that can be blocked inside the operating system. If we were to build our blocking I/O on top of the OS' blocking I/O, we'd spawn a new thread for every client stuck in a syscall. This becomes really expensive once you have 10,000 client threads, all stuck in a syscall waiting for their I/O operation to succeed.

去处理一个阻塞的系统调用，我们需要一个线程真正的阻塞在操作系统。如果我们基于操作系统的阻塞IO来构建我们自己的阻塞IO，

我们就必需经常性的为每一次卡顿的系统调用新建1个线程。当你有1万个线程时，所有的线程都会变的卡顿，直到IO操作完成。

> Go gets around this problem by using the asynchronous interfaces that the OS provides, but blocking the goroutines that are performing I/O.

Go通过操作系统提供异步IO接口和阻塞协程来解决这个问题，

## 网络轮询器（The netpoller）

> The part that converts asynchronous I/O into blocking I/O is called the netpoller. 

将异步IO转换成阻塞IO的部分就叫网络轮询器。

> It sits in its own thread, receiving events from goroutines wishing to do network I/O. The netpoller uses whichever interface the OS provides to do polling of network sockets. On Linux, it uses epoll, on the BSDs and Darwin, it uses kqueue and on Windows it uses IoCompletionPort. These interfaces all have in common that they provide user space a way to efficiently poll for the status of network I/O.

网络轮询器占用一个线程，帮助其他正在使用网络IO的协程去接收网络事件。它基于不同操作系统提供的多路复用机制，来实现。虽然提供的接口不同，但是它们都有一个共性，那就是提供用户空间一种更加有效的的方式去处理网络IO

> Whenever you open or accept a connection in Go, the file descriptor that backs it is set to non-blocking mode. This means that if you try to do I/O on it and the file descriptor isn't ready, it will return an error code saying so. Whenever a goroutine tries to read or write to a connection, the networking code will do the operation until it receives such an error, then call into the netpoller, telling it to notify the goroutine when it is ready to perform I/O again. The goroutine is then scheduled out of the thread it's running on and another goroutine is run in its place.

当你在Go中建立一个连接时，连接背后的文件描述符就会被设置成`非阻塞模式`。也就是说，当文件描述符还没有准备好数据时，你做了IO操作，操作系统会返回一个错误Code给你，告诉你它还没准备好。当一个协程读或写一个连接时，网络代码会执行IO操作，如果遇到了错误Code（not ready），就会托管给网络轮询器，网络轮询器会在此IO准备好数据时，负责通知这个协程。托管后，协程会让出线程资源，该线程资源会去执行其他协程。

> When the netpoller receives notification from the OS that it can perform I/O on a file descriptor, it will look through its internal data structure, see if there are any goroutines that are blocked on that file and notify them if there are any. The goroutine can then retry the I/O operation that caused it to block and succeed in doing so.

当netpoller从操作系统收到通知，它可以对某个文件描述符进行操作的时候。netpoller会查询它的内部数据结构，看看是否有goroutines阻塞在该文件描述符上，并通知它们。这些goroutine之后会重试之前导致阻塞的IO操作，并成功执行。

> If this is sounding a lot like using the old select and poll Unix system calls to do I/O, it's because it is. But instead of looking up a function pointer and a struct containing a bunch of state variables, the netpoller looks up a goroutine that can be scheduled in. This frees you from managing all that state, rechecking whether you received enough data on the last go around and juggling function pointers like you would do with traditional Unix networking I/O.

如果这听着有点像select或poll，那是因为本质上他就是。只不过netpoller是查询可以调度的goroutine，而前者需要处理函数指针和一堆变量。netpoller让你从传统Unix网络编程的管理状态，反复检查和函数选择中解放出来。

## 相关文章（Related articles）

- [Using Causal Profiling to Optimize the Go HTTP/2 Server](https://morsmachine.dk/http2-causalprof)
- [A Causal Profiling update](https://morsmachine.dk/causalprof-update)
- [Causal Profiling for Go](https://morsmachine.dk/causalprof)
- [Effective error handling in Go.](https://morsmachine.dk/error-handling)
- [The Go scheduler](https://morsmachine.dk/go-scheduler)

