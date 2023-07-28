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

1. 调用 **configure**，指定共享目录为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/\<containerID\>/shared

## exec

**在运行的容器中执行命令**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L533)

1. 请求 agent server 的 grpc.AgentService 接口的 ExecProcess 方法，进入指定容器执行命令<br>*期间涉及到 types.Cmd 和 grpc.Process 的转换，两者都包含要在容器中运行的命令的主要信息，包括工作目录、用户、主组、参数和环境变量等*

## startSandbox

**启动 sandbox 中的所有容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L705)

1. 调用 agent 的 **setAgentURL**，设置通信 URL
2. 读取 sandbox OCI spec 中挂载点为 /etc/resolv.conf 的挂载源文件内容（即位于 host 上用于挂载到容器中的 DNS 配置文件）
3. 调用 agent 的 **check**，检测 agent server 的存活性
4. 调用 Network 的 **Endpoints**，获取 sandbox 中所有网卡，进而调用 Endpoint 的 **Properties**，获取网卡属性等信息，构建 RPC 通信所需的网卡接口、路由和 ARP neighbor 信息
   1. 调用 **updateInterface**，更新 VM 的网卡信息
   2. 请求 agent server 的 grpc.AgentService 接口的 UpdateRoutes 方法，更新 VM 中路由信息 
   3. 请求 agent server 的 grpc.AgentService 接口的 AddARPNeighbors 方法，更新 VM 中 ARP neighbor 信息
5. 调用 Hypervisor 的 **Capabilities**，检验 hypervisor 是否支持文件系统共享特性
   1. 如果 [hypervisor].shared_fs 为 virtio-fs 或者 virtio-fs-nydus
      1. 当 [hypervisor].virtio_fs_cache 不为 none 且 [hypervisor].virtio_fs_cache_size 不为 0 时，挂载参数中会追加 dax<br>*如果 virtio-fs 使用 auto 或者 always，则可以使用选项 dax 挂载 guest 目录，从而允许它直接映射来自 host 的内容。 当设置为 none 时，挂载选项不应包含 dax，以免 virtio-fs 守护进程因无效地址引用而崩溃*
      2. 生成一个类型为 virtiofs、挂载源为 kataShared、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/（当 [hypervisor].shared_fs 为 virtio-fs-nydus 时， 为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/）以及含上述挂载参数的 virtio-fs 挂载信息
   2. 如果 [hypervisor].shared_fs 为 virtio-9p，则生成一个类型为 9p、挂载源为 kataShared，挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/、挂载参数为 msize=\<[hypervisor].msize_9p\> 的 9p 挂载信息
6. 如果 shmSize 大于 0，则生成一个类型为 tmpfs、挂载源为 shm、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/sandbox/shm、挂载参数为 size=\<shmSize\>,noexec,nosuid,nodev,mode=1777 的 ephemeral 挂载信息<br>*shmSize 为 sandbox OCI spec 中 destination 为 /dev/shm，type 为 bind 的挂载点的 source 大小*
7. 请求 agent server 的 grpc.AgentService 接口的 CreateSandbox 方法，启动 sandbox 中的所有容器<br>*其中参数包含 [hypervisor].guest_hook_path，表示 VM 中 hook 脚本路径，hook 必须按照其 hook 类型存储在 guest_hook_path 的子目录中，例如 guest_hook_path/{prestart,poststart,poststop}。Kata agent 将扫描这些目录查找可执行文件，按字母顺序将其添加到容器的生命周期中，并在 VM 运行时命名空间中执行*

## stopSandbox

**关停 sandbox 中的所有容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L856)

1. 请求 agent server 的 grpc.AgentService 接口的 DestroySandbox 方法，关停 sandbox 中的所有容器

## createContainer

**创建容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1151)

1. 调用 fsShare 的 **ShareRootFilesystem**，创建容器 rootfs 的共享挂载
2. 针对容器中每一个共享挂载信息
   1. 忽略挂载源为系统类别的，例如 /proc 或者 /sys
   2. 如果待挂载的是块设备类型，则调用 devManager 的 **AttachDevice**，attach 设备，而不会作为共享挂载
   3. 忽略挂载类型不为 bind 的挂载信息
   4. 忽略挂载点为 /dev/shm 的挂载信息<br>*将 /dev/shm 作为一个绑定挂载传递到容器中不是一个合理的方式，因为它不需要从 host 的 9p 挂载中传递。相反，需要在容器内部分配内存来处理 /dev/shm*
   5. 忽略挂载点为 /dev 或者 /dev/ 目录层级下的设备文件、目录等非常规文件
   6. 调用 fsShare 的 **ShareFile**，将位于 host 的挂载点文件共享至 guest 中
   7. 如果挂载源为 Kubernetes ConfigMap 或者 Secret 资源（路径中会包含 kubernetes.io~configmap 和 kubernetes.io~secret 特征）并且调用 Hypervisor 的 **Capabilities**，判断 hypervisor 支持文件系统共享特性，则创建 <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/\<containerID\>/mounts/watchable 目录，并生成一个类型为 bind、挂载源为 \<guestPath\>（即步骤 6 返回的位于 guest 中的文件路径），挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/watchable/\<base guestPath\> 的 watchable-bind 挂载信息，并替换原挂载源为 watchable-bind 的挂载点<br>*virtiofs 不支持 inotify 机制，因此这是一种解决方案，用于让 virtiofs 可以间接感知到 Kubernetes ConfigMap 和 Secret 文件的变化。具体来说，是将这两种资源文件重新挂载，并将原本的 OCI spec 中声明的挂载源替换成新的挂载点*

3. 针对容器中每一个临时挂载信息（即 ephemeral），生成一个类型为 tmpfs、挂载源为 tmpfs、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/sandbox/ephemeral/\<base source\>、挂载参数为 fsgid=\<gid\> 的 ephemeral 挂载信息<br>*如果卷的 gid 不是根组（默认组），这意味着在该本地卷上设置了特定的  fsGroup，那么它应该传递给 guest*
4. 针对容器中每一个本地挂载信息（即 local，用于 VM 中多容器的文件共享），解析 /proc/mounts 文件内容，获取文件系统类型为 hugetlbfs 的挂载参数，用于进一步解析大页大小，生成一个类型为 hugetlbfs、挂载源为 nodev、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/sandbox/ephemeral/\<base source\>、挂载参数为 pagesize=\<pagesize\>,size=\<size\> 的 ephemeral 挂载信息
5. 针对容器中每一个本地挂载信息（即 local，用于 VM 中多容器的文件共享），生成一个类型为 local、挂载源为 local、挂载点为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/\<sandboxID\>/rootfs/local/\<base source\>、挂载参数为 mode=0777,fsgid=\<gid\> 的 local 挂载信息
6. 修正容器 OCI spec 中的信息，更新和忽略其中的挂载点信息<br>*更新和忽略的信息均来自步骤 2，其中如果步骤 2-7 有变更，则需要更新；如果步骤 2-6 调用返回为 nil，则需要忽略*
7. 针对容器的每一个设备信息，调用 devManager 的 **GetDeviceByID**，获取设备对象，进一步根据其设备类型（例如 block、vhost-user-blk-pci 和 vfio），调用 device 的 **GetDeviceInfo**，获取设备信息
8. 针对容器的每一个块设备信息（步骤 2-2 未做处理），将设备的挂载源更新为 /run/kata-containers/sandbox/storage/\<source\>，更新至 OCI spec 中
9. 校验当禁用 [runtime].disable_guest_seccomp 时，Kata agent 是否支持 seccomp 特性
10. 请求 agent server 的 grpc.AgentService 接口的 CreateContainer 方法，聚合上述的挂载点、设备、OCI spec 等信息创建容器

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

**通过探针接口通知 guest 内核有关内存热插拔事件**

*用于热添加内存之后，通知内存上线之前*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1706)

1. 请求 agent server 的 grpc.AgentService 接口的 MemHotplugByProbe 方法，通知内存热插拔事件<br>*内存热插拔是分批的，因此通知事件也是分批执行的，其中会根据内存分片大小涉及到内存偏移量*

## statsContainer

**获取容器详情**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1741)

1. 请求 agent server 的 grpc.AgentService 接口的 StatsContainer 方法，获取容器的详情信息，其中主要关注 hugetlb、blkio、CPU、memory 和 pid 等 cgroup 相关的信息

## pauseContainer

**暂停容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1688)

1. 请求 agent server 的 grpc.AgentService 接口的 PauseContainer 方法，暂停容器

## resumeContainer

**恢复容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1697)

1. 请求 agent server 的 grpc.AgentService 接口的 ResumeContainer 方法，恢复容器

## configure

**设置 agent 配置信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L400)

1. 调用 Hypervisor 的 **GenerateSocket**，生成用于 host 和 guest 通信的 socket 地址
2. 调用 Hypervisor 的 **AddDevice**，根据生成的 socket 的类型，为 VM 添加对应类型的设备（例如 vhost-vsock 或 virtio-vsock）
3. 调用 Hypervisor 的 **Capabilities**，检验 hypervisor 是否支持文件系统共享特性，创建指定共享目录，调用 Hypervisor 的 **AddDevice**，为 VM 添加 filesystem 类型的设备，其中，挂载标签为 kataShared，挂载源为此共享目录，例如 <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/\<containerID\>/shared

## configureFromGrpc

**设置 agent 配置信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L445)

1. 调用 Hypervisor 的 **GenerateSocket**，生成用于 host 和 guest 通信的 socket 地址

## reseedRNG

**重置随机数生成器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L1873)

1. 请求 agent server 的 grpc.AgentService 接口的 ReseedRandomDev 方法，使用新的种子值重置 guest 的随机数生成器

## updateInterface

**更新网卡信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L557)

1. 请求 agent server 的 grpc.AgentService 接口的 UpdateInterface 方法，更新 VM 中网卡信息

## listInterfaces

**获取所有网卡信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L631)

1. 请求 agent server 的 grpc.AgentService 接口的 ListInterfaces 方法，获取 VM 中所有的网卡信息

## updateRoutes

**更新路由信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L584)

1. 请求 agent server 的 grpc.AgentService 接口的 UpdateRoutes 方法，更新 VM 中路由信息

## listRoutes

**获取所有路由信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L644)

1. 请求 agent server 的 grpc.AgentService 接口的 ListRoutes 方法，获取 VM 中所有的路由信息

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
