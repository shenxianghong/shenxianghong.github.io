---
title: "「 Kata Containers 」源码走读 — virtcontainers/agent"
excerpt: "virtcontainers 中与 Kata-agent 组件相关的流程梳理"
cover: https://picsum.photos/0?sig=20230726
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2023-07-26
toc: true
categories:
- Container Runtime
tag:
- Kata Containers

---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

# agent

*<u>src/runtime/virtcontainers/agent.go</u>*

从代码结构来看，agent 是一个典型的 CS 架构，客户端部分位于 virtcontainers 库中，与位于 guest VM 中运行的服务端进程进行 RPC 通信（实际为 ttrpc），管理 VM 中容器进程的生命周期。

目前 agent 的实现方式仅有一种：kataAgent。

```go
type kataAgent struct {
	ctx      context.Context

	// 包含两种类型：VSock 和 HybridVSock
    // VSock：vsock://<contextID>:1024，QEMU 支持
    // HybridVSock：hvsock://<udsPath>:1024，CloudHypervisor 和 Firecracker 支持
	vmSocket interface{}

	client *kataclient.AgentClient

	// lock protects the client pointer
	sync.Mutex

	state KataAgentState

	reqHandlers map[string]reqFunc
    
	// [agent].kernel_modules
	kmodules    []string

	// [agent].dial_timeout
	dialTimout uint32

	// 固定为 true，如果为 false，代表不是长连接，需要调用 disconnect 手动断开连接
	keepConn bool
	dead     bool
}
```

agent 中声明的 **longLiveConn**、**getAgentURL**、**setAgentURL**、**reuseAgent** 均为参数获取与赋值，无复杂逻辑，不作详述。

## init

**初始化 agent 服务**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L348)

1. disableVMShutdown 是否为 true 取决于是否启用了 [agent].enable_tracing<br>*当 disableVMShutdown 为 true，也就意味着在关停 VM 时，会向 QMP 服务发送 quit 命令，请求关闭 VM 实例；否则，直接 syscall kill 掉 QEMU 进程，不会等待 VM 关闭，因此在启用 [agent].enable_tracing 时，VM 的关停时间会有所增加*
2. 初始化 agent 实现，返回 disableVMShutdown，后续决定在调用 Hypervisor 的 **Stop** 操作时的 VM 关闭方式

## capabilities

**获取 agent 支持的特性**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L376)

1. 设置并返回 agent 默认支持特性，包括块设备特性支持

## check

**检查 agent 是否存活**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1827)

1. 请求 agent server 的 grpc.Health 接口的 Check 方法，检测 agent server 的存活性

## disconnect

**断开与 agent 的连接**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1805)

1. 关闭 agent 客户端，重置 gRPC 路由映射表

## createSandbox

**sandbox 运行前的准备工作**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L449)

1. 调用 Hypervisor 的 **GenerateSocket**，生成用于 host 和 guest 通信的 socket 地址
2. 调用 Hypervisor 的 **AddDevice**，根据生成的 socket 的类型，为 VM 添加对应类型的设备（例如 vhost-vsock 或 virtio-vsock）
3. 调用 Hypervisor 的 **Capabilities**，检验 hypervisor 是否支持文件系统共享特性
   1. 创建 <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/\<containerID\>/shared 目录
   2. 调用 Hypervisor 的 **AddDevice**，为 VM 添加 filesystem 类型的设备，其中，挂载标签为 kataShared，挂载源为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/\<containerID\>/shared

## exec

**在运行的容器中执行命令**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L533)

1. 请求 agent server 的 grpc.AgentService 接口的 ExecProcess 方法，进入指定容器执行命令<br>*期间涉及到 types.Cmd 和 grpc.Process 的转换，两者都包含要在容器中运行的命令的主要信息，包括工作目录、用户、主组、参数和环境变量等。*

## startSandbox

**启动 sandbox 中的所有容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L705)

1. 调用 agent 的 **setAgentURL**，设置通信 URL
2. 读取 sandbox OCI spec 中挂载点为 /etc/resolv.conf 的挂载源文件内容（即位于 host 上用于挂载到容器中的 DNS 配置文件）
3. 调用 agent 的 **check**，检测 agent server 的存活性
4. 调用 Network 的 **Endpoints**，获取 sandbox 中所有网卡，进而调用 Endpoint 的 **Properties**，获取网卡属性等信息，构建 RPC 通信所需的网卡接口、路由和 ARP neighbor 信息
   1. 请求 agent server 的 grpc.AgentService 接口的 UpdateInterface 方法，更新 VM 中网卡信息
   2. 请求 agent server 的 grpc.AgentService 接口的 UpdateRoutes 方法，更新 VM 中路由信息 
   3. 请求 agent server 的 grpc.AgentService 接口的 AddARPNeighbors 方法，更新 VM 中 ARP neighbor 信息
5. 调用 Hypervisor 的 **Capabilities**，检验 hypervisor 是否支持文件系统共享特性
   1. 如果 [hypervisor].shared_fs 为 virtio-fs 或者 virtio-fs-nydus
      1. 当 [hypervisor].virtio_fs_cache 不为 none 且 [hypervisor].virtio_fs_cache_size 不为 0 时，挂载参数中会追加 dax<br>*如果 virtio-fs 使用 auto 或者 always，则可以使用选项 dax 挂载 guest 目录，从而允许它直接映射来自 host 的内容。 当设置为 none 时，挂载选项不应包含 dax，以免 virtio-fs 守护进程因无效地址引用而崩溃。*
      2. 生成一个类型为 virtiofs、挂载源为 kataShared、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/（当 [hypervisor].shared_fs 为 virtio-fs-nydus 时， 为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/）以及含上述挂载参数的 virtio-fs 挂载信息
   2. 如果 [hypervisor].shared_fs 为 virtio-9p，则生成一个类型为 9p、挂载源为 kataShared，挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/、挂载参数为 msize=\<[hypervisor].msize_9p\> 的 9p 挂载信息
6. 如果 shmSize 大于 0，则生成一个类型为 tmpfs、挂载源为 shm、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/sandbox/shm、挂载参数为 size=\<shmSize\>,noexec,nosuid,nodev,mode=1777 的 ephemeral 挂载信息<br>*shmSize 为 sandbox OCI spec 中 destination 为 /dev/shm，type 为 bind 的挂载点的 source 大小*
7. 请求 agent server 的 grpc.AgentService 接口的 CreateSandbox 方法，启动 sandbox 中的所有容器<br>*其中参数包含 [hypervisor].guest_hook_path，表示 VM 中 hook 脚本路径，hook 必须按照其 hook 类型存储在 guest_hook_path 的子目录中，例如 guest_hook_path/{prestart,poststart,poststop}。Kata agent 将扫描这些目录查找可执行文件，按字母顺序将其添加到容器的生命周期中，并在 VM 运行时命名空间中执行。*

## stopSandbox

**关停 sandbox 中的所有容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L856)

1. 请求 agent server 的 grpc.AgentService 接口的 DestroySandbox 方法，关停 sandbox 中的所有容器

## createContainer

**创建容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1151)

1. 调用 FilesystemSharer 的 **ShareRootFilesystem**，创建 VM 中容器 rootfs 的共享挂载

## startContainer

**启动容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1625)

1. 请求 agent server 的 grpc.AgentService 接口的 StartContainer 方法，启动指定容器

## stopContainer

**关停容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1637)

1. 请求 agent server 的 grpc.AgentService 接口的 RemoveContainer 方法，关停指定容器

## signalProcess

**向指定进程发送信号**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1645)

1. 请求 agent server 的 grpc.AgentService 接口的 SignalProcess 方法，根据函数入参 all 是否为 true 决定是否向所有进程发送信号

## winsizeProcess

**设置进程的 tty 大小**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1661)

1. 请求 agent server 的 grpc.AgentService 接口的 TtyWinResize 方法，设置指定进程的 tty 大小

## writeProcessStdin

**将内容写入至进程的标准输入流中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1850)

1. 请求 agent server 的 grpc.AgentService 接口的 WriteStdin 方法，将内容写入至指定进程的标准输入流中

## closeProcessStdin

**关闭进程的标准输入流**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1850)

1. 请求 agent server 的 grpc.AgentService 接口的 CloseStdin 方法，关闭指定进程的标准输入流

## readProcessStdout

**读取进程的标准输出流内容**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1850)

1. 请求 agent server 的 grpc.AgentService 接口的 ReadStdout 方法，将请求返回体数据拷贝至入参的接收数据中

## readProcessStderr

**读取进程的标准错误流内容**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1850)

1. 请求 agent server 的 grpc.AgentService 接口的 ReadStderr 方法，将请求返回体数据拷贝至入参的接收数据中

## updateContainer

**更新容器资源配置**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1673)

1. 请求 agent server 的 grpc.AgentService 接口的 UpdateContainer 方法，更新指定容器的资源配置

## waitProcess

**等待进程返回退出码**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1835)

1. 请求 agent server 的 grpc.AgentService 接口的 WaitProcess 方法，等待指定进程返回退出码

## onlineCPUMem

**通知 CPU 和内存上线**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1730)

1. 请求 agent server 的 grpc.AgentService 接口的 OnlineCPUMem 方法，通知上线指定 CPU 和内存

## memHotplugByProbe

## statsContainer

## pauseContainer

## resumeContainer

## configure

## configureFromGrpc

## reseedRNG

## updateInterface

## listInterfaces

## updateRoutes

## listRoutes

## getGuestDetails

## setGuestDateTime

## copyFile

## addSwap

## markDead

## cleanup

## save

## load

## getOOMEvent

## getAgentMetrics

## getGuestVolumeStats

## resizeGuestVolume

## getIPTables

## setIPTables
