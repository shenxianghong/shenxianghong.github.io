---
title: "「 Kata Containers 」源码走读 — containerd-shim-kata-v2"
excerpt: "Containerd Shimv2 API 的实现与 ShimManagement、EventForwarder 等管理相关流程梳理"
cover: https://picsum.photos/0?sig=20230121
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2023-01-21
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

本质上，Kata agent 负责 VM（也称 sandbox、guest 等）中的容器等进程的生命周期的管理。而 containerd-shim-kata-v2 作为 Kata agent 唯一的服务入口（本身是一个可执行程序，运行后即为 shim server），一部分实现了 Containerd shimv2 接口，暴露 shim API 用于与 Containerd 的 gRPC 通信，一部分暴露 HTTP endpoints 用于命令行工具和 Kata monitor 的服务请求处理，内部调用 virtcontainers 的诸多子模块，提供了用户和 Containerd 对 VM 以及容器进程的生命周期管理的能力。

*<u>src/runtime/vendor/github.com/containerd/containerd/runtime/v2/task/shim.pb.go</u>*

```go
// service is the shim implementation of a remote shim over GRPC
type service struct {
	ctx      context.Context
	rootCtx  context.Context // root context for tracing
	rootSpan otelTrace.Span
	sandbox  vc.VCSandbox
	monitor  chan error
	ec       chan exit
    mu       sync.Mutex

	// 配置文件信息
	config *oci.RuntimeConfig

	// 当前 shim service 中的容器信息
	containers map[string]*container

	// 事件消费队列（TaskCreate、TaskStart、TaskDelete、TaskPaused、TaskResumed、TaskOOM、TaskExecAdded 和 TaskExecStarted）
	events      chan interface{}
	eventSendMu sync.Mutex
	
	// 由 container engine 触发的关停函数
	cancel    func()
	// 由 container engine 传入的信息
	namespace string
	id 	      string

	// 原语义为 VM 中的容器 PID，但是在 kata 模型中 shimv2 无法获得，因此这里为 hypervisor 的 PID
	hpid uint32
	// shimv2 的 PID
	pid  uint32
}
```

**main 函数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/containerd-shim-kata-v2/main.go#L24)

1. 如果指定的参数为 --version，则展示其版本、commit 等信息
2. 通过 shim API，初始化并启动 shim server，注册名称为 containerd-shim-kata-v2，其中 NoReaper 和 NoSubreaper 均为 true
   1. 初始化 containerd-kata-shim-v2 模块的 logger，如果没有开启 debug 日志级别，则默认设置为 warn 级别，以此 logger 为基准，设置 virtcontainers 与 katautils 模块的 logger
   2. 校验启动时是否指定了 --namespce 参数（在 Kubernetes 集群中为 k8s.io，Docker 中为 moby，默认为 default）
   3. 启动 goroutine，持续处理 service 中的 exit channel，构建 TaskExit 事件发送至 events channel 中
   4. 初始化 eventForwarder，并启动 goroutine 调用 eventForwarder 的 **forward** 持续上报事件
   5. 返回 service，交给 Containerd 负责针对每个容器启动 shim server

# Service

shim server 对外暴露的 gRPC 服务。

## State

**获取进程的运行时状态**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L628)

1. 请求体和返回体结构如下

   ```go
   type StateRequest struct {
   	ID                   string   
   	ExecID               string      
   }
   ```

   ```go
   type StateResponse struct {
   	ID                   string      
   	Bundle               string      
   	Pid                  uint32      
   	Status               task.Status 
   	Stdin                string      
   	Stdout               string     
   	Stderr               string      
   	Terminal             bool       
   	ExitStatus           uint32      
   	ExitedAt             time.Time   
   	ExecID               string           
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 根据容器中的属性构建返回消息，如果 r.ExecID 不为空，则通过 r.ExecID 获取维护在 container.execs 中的 exec 中的属性为准

## Create

**使用底层 OCI 运行时创建一个新的容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L381)

1. 请求体和返回体结构如下

   ```go
   type CreateTaskRequest struct {
   	ID                   string         
   	Bundle               string         
   	Rootfs               []*types.Mount 
   	Terminal             bool
   	Stdin                string
   	Stdout               string
   	Stderr               string
   	Checkpoint           string
   	ParentCheckpoint     string
   	Options              *types1.Any    
   }
   ```

   ```go
   type CreateTaskResponse struct {
   	// shimv2 无法从 VM 获取容器进程 PID，因此后续对于需要 PID 的返回值，直接返回 hypervisor 的 PID 即可
   	Pid                  uint32  
   }
   ```

2. 校验 r.ID 是否不为空，且正则匹配满足 ^\[a-zA-Z0-9][a-zA-Z0-9_.-]+$
3. 调用 **create**，创建容器，将返回的容器状态设为 created，并维护在 service.containers 中
4. 发送 TaskCreate 事件至 service.events 中

### create

**基于 service 实例和请求消息创建容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/create.go#L51)

1. 初始化空的 rootfs 对象，如果请求中 r.Rootfs 已经提供了一个，则以此为准，丰富 rootfs 对象
2. 将 r.Bundle 下的 config.json 文件（例如 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/config.json，该文件即为 [OCI spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md)，由 container manager 事先创建），解析为 OCI spec 结构
   1. 校验 r.ID 与 r.Bundle 是否不为空，并且 r.Bundle 存在且为目录结构，获取 r.Bundle 符号链接（如果是）指向的路径名
   2. 获取 r.Bundle 目录下的 config.json，读取其内容，解构成 compatOCISpec 结构（为了兼容 v1.0.0-rc4 和 v1.0.0-rc5，参考：https://github.com/opencontainers/runtime-spec/commit/37391fb）
   3. 不同版本下 spec.Process.Capabilities 字段属性不同，通过类型断言，将其以及 compatOCISpec 转换成对应的 OCI spec（实际上，容器运行后，spec 信息可以通过 crictl inspect xxx 或者 ctr c info xxx 查看到，也可以查看 config.json 文件），并返回
3. 根据 spec 中的 annotation 信息判断其容器类型，并转换成 virtcontainers 中定义的容器类型
   1. 判断 spec.Annotations 中是否包含 io.kubernetes.cri.container-type、io.kubernetes.cri-o.ContainerType 和 io.kubernetes.docker.type 的 key（分别代表 Containerd、CRI-O 和 Dockershim 三种 CRI）
   2. 获得 key 对应的 value（value 即为容器的类型），分为两类，分别是 sandbox 和 container，区别于 CRI 的不同，具体的名称有所区别（Containerd 和 CRI-O 中称为 sandbox 和 container，而 Dockershim 中称为 podsandbox 和 container）
   3. 根据容器类型，映射出 virtcontainers 中的容器类别，例如 pod_sandbox 和 pod_container，当 spec.Annotations 中未识别到上述三种 key，则视为 single_container，即非 Pod 容器（如通过 ctr，podman 启动运行）
   4. 此外，匹配到 key，但是 value 不符合 sandbox 和 container 的，均视为 unknown_container_type
4. 构建 runtimeConfig（runtimeConfig 聚合了运行时所有的设置信息，后续的操作中不再解析配置文件），优先级依次从 spec.Annotations、shimv2 请求传参和环境变量中获取
   1. 获取 spec.Annotations 中 key 为 io.katacontainers.config_path 的 value 作为 configPath（也就是 Kata Containers 的静态配置文件）
   2. 当 configPath 为空时，尝试从 r.Options 中获取（其中在类型断言时，优先使用 github.com/containerd/containerd/pkg/runtimeoptions/v1 中的 Options 类型，为了兼容 1.4.3、1.4.4 版本的 Containerd，退而使用 github.com/containerd/cri-containerd/pkg/api/runtimeoptions/v1 中的 Options 类型）
   3. 如果此时 configPath 仍为空，则从环境变量 KATA_CONF_FILE 中获取
   4. 如果 configPath 为空，则按序优先读取 /etc/kata-containers/configuration.toml 和 /usr/share/defaults/kata-containers/configuration.toml 配置文件内容，构建 runtimeConfig
5. 校验部分配置项
   1. [runtime].experimental 特性是否支持
   2. [runtime].sandbox_bind_mounts 挂载点是否嵌套（即挂载点的 base 目录存在重复）
   3. 当 [runtime].disable_new_netns 启用时，[runtime].internetworking_model 是否设置为 none
   4. [hypervisor].default_memory 是否不为 0。当 guest 镜像采用 initrd 格式时，[hypervisor].default_memory 必须大于镜像大小（因为 initrd 会完全读到内存中）；当 guest 镜像采用 image 格式时，镜像大小大于 [hypervisor].default_memory 时，会输出警告信息（虽然 image 不会完全读到内存中，但是此情况并非正常现象）
   5. 当 [factory].enable_template 启用时，guest 镜像类型必须为 initrd 格式；当 [factory].vm_cache_number 大于 0 时，hypervisor 类型必须为 QEMU
6. 根据不同的容器类型，基于配置文件内容创建容器

   ***pod_sandbox 或 single_container**（以下统称为 pod_sandbox）*

   1. 当 service.sandbox 不为空时，意味着当前容器环境已经存在一个 sandbox（只有在完成创建容器后才会回写该字段），是无法嵌套创建 sandbox 的
   2. 基于 [runtime].jaeger_endpoint、[runtime].jaeger_user 和 [runtime].jaeger_password 创建 jaeger tracer
   3. 如果容器类型为 pod_sandbox，则基于 spec.Annotations 中的 io.kubernetes.cri.sandbox-memory、io.kubernetes.cri.sandbox-cpu-period 和 io.kubernetes.cri.sandbox-cpu-quota，计算 VM 的资源大小；如果容器类型为 single_container，则基于 spec.Linux.Resources 中的 Memory.Limit、CPU.Quota 和 CPU.Period，计算 VM 的资源大小。两者公式一致：

      > CPU = (cpu-quota * 1000 / cpu-period + 999) / 1000
      >
      > MEM = memory / 1024 /1024

   4. 检查容器的 rootfs 目录是否需要挂载
      1. 如果请求中 r.Rootfs 指定了一个，则判断其是否为块设备，并且 [hypervisor].disable_block_device_use 为 false（disable_block_device_use 禁止块设备用于容器的 rootfs。 在像 devicemapper 这样的存储驱动程序中，容器的 rootfs 由块设备支持，出于性能原因，块设备直接传递给 hypervisor。 这个标志阻止块设备被传递给 hypervisor，而使用 virtio-fs 传递 rootfs）；或者，rootfs 的类型为 fuse.nydus-overlayfs。满足条件之一，即不需要挂载，而是走后续的热插流程
      2. 创建 rootfs 目录（如果不存在），遍历 r.Rootfs，挂载至 rootfs 目录下（即 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/rootfs）

   5. 基于 factory 配置项，尝试获取现有 VM factory，如果获取失败且未启用 VM cache 特性时，会初始化新的 VM factory，并调用 vircontainers 的 **SetFactory**，透传 VM factory
   6. 基于 [hypervisor].rootless 设置 rootless（默认情况下，QEMU 以 root 身份运行。 当设置为 true 时，QEMU 将以非 root 的随机用户运行），如果启用 rootless，则额外执行以下流程
      1. 创建一个用于运行 Kata Containers 的随机用户
      2. 根据用户名获取并设置 UID 和 GID
      3. 创建用户目录，并设置环境变量 XDG_RUNTIME_DIR
      4. 将 KVM GID 添加到 hypervisor 配置项中的补充组中，使 hypervisor 进程可以访问 /dev/kvm 设备
   7. 基于上述构建的 OCI spec、runtimeConfig、rootfs、bundle、containerID 等信息创建 sandbox
      1. 将 OCI spec 和 runtimeConfig 转为 virtcontainers 所需的配置结构（即 sandboxConfig）<br>
         额外注明：a. 部分参数当 OCI spec annotations 中有声明时，会以 annotations 为准，前提是 [hypervisor].enable_annotations 中声明了允许动态加载的配置项且该配置项合法；b. 当启用 [hypervisor].static_sandbox_resource_mgmt，VM 规格会被静态配置，而非启动后热插拔，因此 CPU 资源规格为 [hypervisor].default_vcpus + \<workloadCPUs\>，内存同理
      2. 当 host 启用 FIPS 时（即 /proc/sys/crypto/fips_enabled 为 1），sandboxConfig 中额外追加 kernel 参数
      3. 启动容器前优先创建 netns。当 [runtime].disable_new_netns 启用时（表示 shim 和 hypervisor 会运行在 host netns 中，而非创建新的），则直接跳过后续创建；否则，当 networkID（spec.Linux.Namespace 中 type 为 network 的 path）为空时（表示 netns 并非由 CNI 提前创建好，而是需要由 Kata Containers 创建，比如脱离 K8s 运行 Kata 的场景中），则根据具体是否为 rootless，执行对应的创建网络命名空间流程。如果为提前创建好的 netns，需要判断其是否与当前进程的 netns 不一致（当前进程可以代表 host 网络，而 Kata Containers 是不支持采用 host 网络作为容器网络的）
      4. 执行 spec.Hooks.Prestart（Prestart 是在执行容器进程之前要运行的 hook 列表，现已废弃） 中定义的动作
      5. 执行 spec.Hooks.CreateRuntime（CreateRuntime 是在创建容器之后但在调用 pivot_root 或任何等效操作之前要运行的 hook 列表） 中定义的动作
      6. 调用 VC 的 **CreateSandbox**，根据 sandboxConfig 信息创建 sandbox，并启动 pod_sandbox 容器
      7. 调用 VCSandbox 的 **GetAllContainers**，校验 sandbox 中的容器总数量是否为 1，返回 sandbox
   8. 设置 service.sandbox 为上面返回的 sandbox，调用 VCSandbox 的 **GetHypervisorPid**，获取 hypervisor 的 PID，设置至 service.hpid 中
   9. 启动监听 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 地址的 shim server，注册服务有 /metrics、/agent-url、/direct-volume/stats、/direct-volume/resize、/iptables、/ip6tables 以及 Pprof，注册上报至 Prometheus 的 shim 相关指标以及 sandbox 相关指标

   ***pod_container***

   1. 校验 service.sandbox 是否不为空（因为 pod_container 的运行是要依托于 pod_sandbox）
   2. 检查容器的 rootfs 目录是否需要挂载
      1. 如果请求中 r.Rootfs 指定了一个，则判断其是否为块设备，并且 [hypervisor].disable_block_device_use 为 false（disable_block_device_use 禁止块设备用于容器的 rootfs。 在像 devicemapper 这样的存储驱动程序中，容器的 rootfs 由块设备支持，出于性能原因，块设备直接传递给 hypervisor。 这个标志阻止块设备被传递给 hypervisor，而使用 virtio-fs 传递 rootfs）；或者，rootfs 的类型为 fuse.nydus-overlayfs。满足条件之一，即不需要挂载，而是走后续的热插流程
      2. 创建 rootfs 目录（如果不存在），遍历 r.Rootfs，挂载 rootfs 目录
   3. 基于上述构建的 OCI spec、runtimeConfig、rootfs、bundle、containerID 等信息创建 container 类型容器
      1. 遍历 spec.Mounts，如果挂载源路径由 K8s 临时存储（即路径中有 kubernetes.io\~empty-dir 标识，且文件类型为 tmpfs），则将 spec.Mounts 中对应挂载点的类型设置为 ephemeral；如果挂载源路径由 K8s emptydir（即路径中有 kubernetes.io\~empty-dir 标识，且文件类型不为 tmpfs） 且 runtimeConfig 没有禁用 disable_guest_empty_dir（如果启用，将不会在 guest 的文件系统中创建 emptydir 挂载，而是在 host 上创建，由 virtiofs 共享，性能较差，但是可以实现 host 和 guest 共享文件），则将 spec.Mounts 中对应挂载点的类型设置为 local （对于给定的 Pod，临时卷仅在 VM 内由 tmpfs 支持时创建一次。 对于同一 Pod 的连续容器，将重复使用已经存在的卷）
      2. 将 OCI spec 和 runtimeConfig 转为 virtcontainers 所需的配置结构（即 containerConfig）
      3. 根据 spec.Annotations 中的 io.kubernetes.cri.sandbox-id、io.kubernetes.cri-SandboxID 和 io.kubernetes.sandbox.id 的 key（分别代表 Containerd、CRI-O 和 Dockershim 三种 CRI），获取到 value（value 即为 sandboxID）
      4. 调用 VCSandbox 的 **CreateContainer**，创建容器
      5. 进入到 sandbox 的网络命名空间中（如果有），执行 spec.Hooks.Prestart（Prestart 是在执行容器进程之前要运行的 hook 列表，现已废弃） 中定义的动作

7. 构建并返回容器

## Start

**启动容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L440)

1. 请求体和返回体结构如下

   ```go
   type StartRequest struct {
   	ID                   string
   	ExecID               string
   }
   ```

   ```go
   type StartResponse struct {
   	Pid                  uint32
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 如果 r.ExecID 为空，则调用 **startContainer**，启动容器，并发送 TaskStart 事件至 service.events 中；否则，调用 **startExec**，启动 exec 进程，并发送 TaskExecStart 事件至 service.events 中

### startContainer

**处理启动容器请求**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/start.go#L18)

1. 如果容器类型为 pod_sandbox
   1. 调用 VCSandbox 的 **Start**，启动 sandbox
   2. 调用 VCSandbox 的 **Monitor**，启动 monitor，并设置在 service.monitor 中
   3. 启动 goroutine，监听 service.monitor 中的错误和退出信号。如果监听到（视为异常），则调用 VCSandbox 的 **Stop** 和 **Delete**，关停与销毁 sandbox 并清理相关资源
   4. 移除 rootfs 挂载点（注意：此处清理的为 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/rootfs，而共享目录下的 rootfs 在上述步骤 3 中已经清理）
   5. 启动 goroutine，调用 VCSandbox 的 **GetOOMEvent**（如果是由于 Kata agent 关停，返回类似 ttrpc: closed 或者 Dead agent 的错误时，则不再监听；其他异常情况，仍重新尝试调用接口），监听 OOM 事件。当收到 OOM 事件后，如果 container manager 为 CRI-O 时，则在例如 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\> 目录下，创建名为 oom 的文件，用于通知 CRI-O；如果 container manager 为 Containerd 时，则发送 TaskOOM 事件至 service.events 中
2. 如果容器的类型不为 pod_sandbox，则调用 VCSandbox 的 **StartContainer**，启动容器
3. 进入到 sandbox 的网络命名空间中（如果有），执行 spec.Hooks.Poststart（Poststart 是在容器进程启动后要运行的 hook 列表） 中定义的动作
4. 设置容器状态为 running
5. 调用 VCSandbox 的 **IOStream**，启动 goroutine，实时处理容器中的标准输出等 IO 流
6. 启动 goroutine，处理容器中的退出队列 ，即先等到容器 IO 退出事件后，再调用 VCSandbox 的 **WaitProcess**，进一步等待进程返回退出码后执行后续清理流程：如果容器类型为 pod_sandbox，则关停 monitor，调用 VCSandbox 的 **Stop** 和 **Delete**，关停与销毁 sandbox，并清理相关资源；否则，调用 VCSandbox 的 **StopContainer**，关停容器。设置容器状态为 stopped，记录退出时间，状态码等<br>*例如，当 Pod 启动之后，pod_sandbox 容器会退出，此时可以收到 pod_sandbox 容器的 IO 退出事件，但是在 WaitProcess 时，不会有返回，因此不会执行 sandbox 的关停与销毁流程；而当 Pod 删除时，pod_sandbox 的 WaitProcess 收到结果，伴随着其余的业务容器一起关停并删除*
7. 启动 goroutine，发送退出消息至 service.ec 中

### startExec

**处理进入容器请求**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/start.go#L100)

1. 通过 r.ID 获取维护在 service.containers 中的容器
2. 通过 r.ExecID 获取维护在 container.execs 中的 exec
3. 调用 VCSandbox 的 **EnterContainer**，进入容器内部
4. 设置 exec 状态为 running
5. 调用 VCSandbox 的 **WinsizeProcess**，调整 tty 大小
6. 调用 VCSandbox 的 **IOStream**，启动 goroutine，实时处理容器中的标准输出等 IO 流
7. 启动 goroutine，处理 exec 中的退出队列 ，即先等到容器 IO 退出事件后，再调用 VCSandbox 的 **WaitProcess**，进一步等待进程返回退出码后执行后续流程。设置 exec 状态为 stopped，记录退出时间，状态码等
8. 启动 goroutine，发送退出消息至 service.ec 中

## Delete

**删除容器或者 exec 进程**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L493)

1. 请求体和返回体结构如下

   ```go
   type DeleteRequest struct {
   	ID                   string
   	ExecID               string
   }
   ```

   ```go
   type DeleteResponse struct {
   	Pid                  uint32
   	ExitStatus           uint32
   	ExitedAt             time.Time 
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 如果 r.ExecID 为空，则调用 **deleteContainer**，删除容器，并发送 TaskDelete 事件至 service.events 中；否则，直接删除 container.execs 中 key 为 r.ExecID 的 exec

### deleteContainer

**删除指定容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/delete.go#L17)

1. 如果容器的类型不是 pod_sandbox，则先调用 VCSandbox 的 **StopContainer**，关停容器（如果容器状态不为 stopped），并调用 VCSandbox 的 **DeleteContainer**，删除容器
2. 执行 spec.Hooks.Poststop（Poststop 是在容器进程退出后要运行的 hook 列表）中定义的动作
3. 如果容器文件系统已经挂载完成，则移除 rootfs 挂载点（例如 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/rootfs，其在 host 上以 overlay 挂载点形式存在）
4. 删除 s.containers 中的 key 为 r.ID 的容器

## Pids

**返回容器内的所有进程 ID，对于 Kata Containers 而言，无法从 VM 获取进程 PID，因此只返回 hypervisor 的 PID**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L829)

1. 请求体和返回体结构如下

   ```go
   type PidsRequest struct {
   	ID                   string
   }
   ```

   ```go
   type PidsResponse struct {
   	// task.ProcessInfo{
   	//   Pid: s.hpid,
   	// }
   	Processes            []*task.ProcessInfo
   }
   ```

## Pause

**暂停容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L684)

1. 请求体结构如下

   ```go
   type PauseRequest struct {
   	ID                   string
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 设置容器状态为 pausing
4. 调用 VCSandbox 的 **PauseContainer**，暂停容器
5. 如果暂停成功，则设置容器状态为 paused，并发送 TaskPaused 事件至 service.events 中；否则，调用 VCSandbox 的 **StatusContainer**，查询容器状态（分为 ready、running、paused 和 stopped），如果查询失败，则实际状态视为 unknown，设置容器状态为查询到的实际结果

## Resume

**恢复容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L725)

1. 请求体结构如下

   ```go
   type ResumeRequest struct {
   	ID                   string
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 调用 VCSandbox 的 **ResumeContainer**，恢复容器
4. 如果恢复成功，则设置容器状态为 running，并发送 TaskResumed 事件至 service.events 中；否则，调用 VCSandbox 的 **StatusContainer**，查询容器状态（分为 ready、running、paused 和 stopped），如果查询失败，则实际状态视为 unknown，设置容器状态为查询到的实际结果

## Checkpoint

**创建容器检查点**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L897)

1. 请求体结构如下

   ```go
   type CheckpointTaskRequest struct {
   	ID                   string      
   	Path                 string      
   	Options              *types1.Any
   }
   ```

2. 截至 Kata 3.0，尚未实现该接口

## Kill

**根据指定信号杀死进程**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L764)

1. 请求体结构如下

   ```go
   type KillRequest struct {
   	ID                   string   
   	ExecID               string  
   	Signal               uint32  
   	All                  bool  
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 以容器状态与 ID 作为待杀死进程的信息，如果 r.ExecID 不为空，则以通过 r.ExecID 获取维护在 container.execs 中的 exec 属性为准
4. 如果信号为 SIGKILL 或者 SIGTERM，并且进程状态已经为 stopped，则不做处理，直接返回即可（根据 CRI 规范，Kubelet 在调用 RemovePodSandbox 之前至少会调用一次 StopPodSandbox ，此调用是幂等的，并且如果所有相关资源都已被回收则不得返回错误。 在调用中它会先发送一个 SIGKILL 信号来尝试停止容器，因此一旦容器终止，应该忽略这个信号并直接返回）；否则，调用 VCSandbox 的 **SignalProcess**，杀死进程

## Exec

**在容器中追加一个额外的进程**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L547)

1. 请求体结构如下

   ```go
   type ExecProcessRequest struct {
   	ID                   string  
   	ExecID               string 
   	Terminal             bool        
   	Stdin                string      
   	Stdout               string      
   	Stderr               string   
   	Spec                 *types1.Any
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 校验 r.ExecID 是否在 container.execs 不存在
4. 基于 r.Stdin、r.Stdout、r.Stderr、r.Terminal 和 r.Spec 构建 exec，维护在 container.execs 中，并发送 TaskExecAdded 事件至 service.events 中

## ResizePty

**调整进程的 pty 大小**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L587)

1. 请求体结构如下

   ```go
   type ResizePtyRequest struct {
   	ID                   string 
   	ExecID               string 
   	Width                uint32  
   	Height               uint32
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 以 container.ID 作为待处理进程的 ID，如果 r.ExecID 不为空，则通过 r.ExecID 获取维护在 container.execs 中的 exec.ID 为准，并更新 r.Width 和 r.Height 至 exec 中
4. 调用 VCSandbox 的 **WinsizeProcess**，调整待处理进程的 pty 大小

## CloseIO

**关闭进程的 IO 流**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L854)

1. 请求体结构如下

   ```go
   type CloseIORequest struct {
   	ID                   string
   	ExecID               string   
   	Stdin                bool    
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 以 container.stdinPipe 和 container.stdinCloser 作为待处理进程 IO 的信息，如果 r.ExecID 不为空，则以通过 r.ExecID 获取维护在 container.execs 中的 exec 信息为准
4. 直至 stdinCloser channel 不再阻塞，调用 stdinPipe 的 Close 方法关闭 IO 流

## Update

**更新容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L1012)

1. 请求体结构如下

   ```go
   type UpdateTaskRequest struct {
   	ID                   string            
   	Resources            *types1.Any      
   	Annotations          map[string]string 
   }
   ```

2. 调用 VCSandbox 的 **UpdateContainer**，更新容器的资源规格

## Wait

**等待进程退出**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L1046)

1. 请求体和返回体结构如下

   ```go
   type WaitRequest struct {
   	ID                   string
   	ExecID               string
   }
   ```

   ```go
   type WaitResponse struct {
   	ExitStatus           uint32   
   	ExitedAt             time.Time
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 从 container.exitCh 中获取退出状态码，并重新回填至 container.exitCh（用容器进程的退出代码重新填充 exitCh，以防此进程有其他等待），如果 r.ExecID 不为空，则以通过 r.ExecID 获取维护在 container.execs 中的 exec.exitCh 为准

## Stats

**获取容器的统计信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L981)

1. 请求体和返回体结构如下

   ```go
   type StatsRequest struct {
   	ID                   string 
   }
   ```

   ```go
   type StatsResponse struct {
   	Stats                *types1.Any
   }
   ```

2. 通过 r.ID 获取维护在 service.containers 中的容器
3. 调用 VCSandbox 的 **StatsContainer**，获取容器的 Hugetlb、Pids、CPU、Memory、Blkio 和 Network 统计信息

## Connect

**返回 shim 相关信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L913)

1. 请求体和返回体结构如下

   ```go
   type ConnectRequest struct {
   	ID                   string
   }
   ```

   ```go
   type ConnectResponse struct {
   	// 即 service.pid
   	ShimPid              uint32
   	// 即 service.hpid
   	TaskPid              uint32   
   	Version              string  
   }
   ```

## Shutdown

**关闭 shim server**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L935)

1. 请求体结构如下

   ```go
   type ShutdownRequest struct {
   	ID                   string
   	Now                  bool
   }
   ```

2. 如果 service.containers 中仍有元素，代表 shim server 仍然管理容器中，因此仅关闭 tracing，不作其他处理，直接返回
3. 调用 service.cancel 退出 shim server（cancel 由 Containerd 服务注册声明）
4. 向 service.hpid 发送 SIGKILL 信号（由于只是在执行 stopSandbox 时向 QEMU 发送了一个 shutdown qmp 命令，并没有等到 QEMU 进程退出，这里最好确保它在 shim server 终止时已经退出。 因此，这里要对 hypervisor 进行最后的清理）
5. 调用 os.Exit(0)，退出程序

## Cleanup

**清理容器相关资源**

*Cleanup 并未用于实现 shimv2 API，而是 Service 的功能扩展，用于在执行 containerd-shim-kata-v2 delete 操作时触发容器清理流程*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/service.go#L323)

1. 返回体结构如下

   ```go
   type DeleteResponse struct {
   	Pid                  uint32
   	ExitStatus           uint32
   	ExitedAt             time.Time
   }
   ```

2. 设置日志输出至 stderr 中（因此，日志信息并不会出现在 Kata 服务中）
3. 获取当前目录下（例如 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\> 目录，流程需要在此路径下执行 ）的 config.json 文件，解析成 OCI spec 格式，判断其容器类型
4. 如果容器类型是 pod_container，则通过 spec.Annotation 中获取到 sandboxID；如果容器类型是 pod_sandbox 或者 single_container，sandboxID 即为 service.id
5. 调用 VC 的 **CleanupContainer**，清理容器
6. 移除 rootfs 挂载点（例如 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/rootfs）

# ShimManagement

shim server 对外暴露的 HTTP 服务。

## agentURL

**处理 /agent-url 请求，返回 agent 的地址**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/shim_management.go#L56)

1. 调用 VCSandbox 的 **GetAgentURL**，获取 agent 地址并返回

## serveMetrics

**处理 /metrics 请求，返回 guest、shim 和 agent 相关的指标**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/shim_management.go#L68)

1. 调用 VCSandbox 的 **UpdateRuntimeMetrics**，更新 guest 指标（更新就是重新获取指标，重新设置在 Prometheus 中）
2. 更新当前进程（即 shimPID）的指标
   1. 获取 /proc/\<shimPID\>/fd 目录下的文件数量，上报 kata_shim_fds 指标
   2. 解析 /proc/\<shimPID\>/stat 文件内容，上报 kata_shim_proc_stat 指标
   3. 解析 /proc/\<shimPID\>/status 文件内容，上报 kata_shim_proc_status 指标
   4. 解析 /proc/\<shimPID\>/io 文件内容，上报 kata_shim_io 指标
3. 如果使用旧版本的 agent（现阶段均为新版本），则不支持获取 agent 指标，直接返回 VM 和 shim 指标即可
4. 调用 VCSandbox 的 **GetAgentMetrics**，获取 agent 指标（在这里，如果获取不到，则视为当前使用旧版本的 agent，后续则由步骤 3 直接返回）
5. 启动 goroutine，上报 pod_overhead_cpu 和 pod_overhead_memory_in_bytes 至 Prometheus（收集 Pod overhead 指标需要 sleep 来获取 cpu/memory 资源使用的变化，所以这里只触发 collect 操作，下次从 Prometheus server 收集请求时收集数据）
   1. 调用 VCSandbox 的 **Stats**，获取 sandbox 的 cgroup 相关信息；调用 VCSandbox 的 **GetAllContainers**，获取所有容器，并逐一调用 VCSandbox 的 **StatsContainer**，获取容器的 cgroup 相关信息
   2. 间隔 1 秒钟，重复步骤 1，再次获取 sandbox 和容器的 cgroup 相关信息
   3. 根据两次数据信息以及总耗时，计算 overhead 指标

## serveVolumeStats

**处理 /direct-volume/stats 请求，返回 guest 中指定卷的信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/shim_management.go#L147)

1. 校验请求中 path 参数是否不为空
2. 调用 VCSandbox 的 **GuestVolumeStats**，获取 guest 中指定卷的信息并返回 

## serveVolumeResize

**处理 /direct-volume/resize 请求，扩容 guest 中指定卷的大小**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/shim_management.go#L175)

1. 读取请求体，并解构成 VCSandbox 所需的格式
2. 调用 VCSandbox 的 **ResizeGuestVolume**，扩容 guest 中指定卷的大小

## ipTablesHandler、ip6TablesHandler

**处理 /iptables 和 /ip6tables请求，操作 guest 中的 iptables 信息**

[source code (ipTablesHandler)](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/shim_management.go#L206)<br>[source code (ip6TablesHandler)](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/shim_management.go#L202)

1. 两者本质相似，区别在于 ip6TablesHandler 的 isIPv6 参数为 true，后续调用接口时，会传递该参数
2. 判断请求方法，目前仅支持 PUT 和 GET 两种，其余返回状态码 501。如果为 PUT 请求，则读取请求体，调用 VCSandbox 的 **SetIPTables**，设置 guest 中的 iptables 信息；如果为 GET 请求，调用 VCSandbox 的 **GetIPTables**，获取 guest 中的 iptables 信息

# EventForwarder

*<u>src/runtime/pkg/containerd-shim-v2/event_forwarder.go</u>*

EventForwarder 为事件上报模块，其中事件源自于 forwarder 中的 service.events。forwarder 包括两类：log 与 containerd，取决于事件最终上报的地点。每一个事件默认上报超时时间为 5 秒钟，超出会被取消上报。

```go
type logForwarder struct {
	s *service
}
```

```go
type containerdForwarder struct {
	s         *service
	ctx       context.Context
	publisher events.Publisher
}
```

其中，publisher 由 Containerd 调用时提供。

**工厂函数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/event_forwarder.go#L71)

1. 如果环境变量中声明了 TTRPC_ADDRESS，则初始化 containerdForwarder，否则初始化 logForwarder

## forward

**处理事件上报**

### logForwarder

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/event_forwarder.go#L40)

1. 监听 service.events 中的事件，断言其的类型，获得事件的 topic
2. 在 containerd-kata-shim-v2 模块的日志中输出事件内容

### containerdForwarder

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/event_forwarder.go#L56)

1. 监听 service.events 中的事件，断言其的类型，获得事件的 topic
2. 调用 Containerd 的 publisher 模块的 Publish（Containerd 负责实现），上报事件至 Containerd

## forwarderType

**forwarder 的具体类型**

### logForwarder

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/event_forwarder.go#L46)

1. 类型为 log

### containerdForwarder

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/containerd-shim-v2/event_forwarder.go#L67)

1. 类型为 containerd
