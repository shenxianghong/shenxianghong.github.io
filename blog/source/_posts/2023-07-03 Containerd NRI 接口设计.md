---
title: "「 Containerd 」NRI 接口设计"
excerpt: "Containerd 中 Node Resource Interface 插件设计理念与简单使用示例"
cover: https://picsum.photos/0?sig=20230703
thumbnail: /gallery/containerd/thumbnail.svg
date: 2023-07-03
toc: true
categories:
- Container Runtime
tag:
- Containerd
---

<div align=center><img width="200" style="border: 0px" src="/gallery/containerd/logo.svg"></div>

------

> based on **v0.3.0**

# 目标

NRI（Node Resource Interface）是 Containerd 的一个子项目，允许将自定义逻辑插入到 OCI 兼容的运行时中，从而实现在容器生命周期的某些特定时间点对容器进行更改操作或执行 OCI 规范之外的额外操作。例如，用于改进设备和其他容器资源的分配和管理。NRI 本身对任何容器运行时的内部实现细节是不感知的。它为 CRI 运行时提供了一个适配库，用于集成 NRI 和扩展插件进行交互。

NRI 提供了接口定义和基础组件，可以实现可插拔的 CRI 运行时插件，这些插件就是 NRI 插件。这些 NRI 插件是与运行时类型无关的，插件既可以应用于 Containerd，也可以应用于 CRI-O。原则上，任何 NRI 插件都应该能够和启用 NRI 的运行时正常协作。

NRI 插件是一个类似守护进程的实例。插件的单个实例会处理 NRI 所有的事件和请求，使用 Unix-domain socket 来进行数据传输和通信，NRI 定义了一套基于protobuf 的协议：NRI plugin protocal，并通过 ttRPC 进行实现。这样可以通过降低每条信息的开销提高通信效率，并且可以实现有状态的 NRI 插件。

# 组件

NRI 实现包含了两个核心组件：[NRI 协议](https://github.com/containerd/nri/tree/main/pkg/api)和 [NRI 运行时适配器](https://github.com/containerd/nri/tree/main/pkg/adaptation)。

这些组件一起建立了运行时如何与 NRI 交互以及插件如何通过 NRI 与运行时中的容器交互的模型。它们还定义了插件可以在哪些条件下对容器进行更改以及这些更改的程度。

其余的组件是 [NRI 插件 stub 库](https://github.com/containerd/nri/tree/main/pkg/stub)和一些 [NRI 示例插件](https://github.com/containerd/nri/tree/main/plugins)。其中，一些插件在实际应用场景中实现了有用的功能，另外一些插件则用于调试。所有示例插件都可以作为如何使用 stub 库实现 NRI 插件的示例。

## API 协议

NRI API 协议中定义了两个服务：runtime 和 plugin

- runtime 服务是运行时向 NRI 插件暴漏的接口。在此接口上的所有请求都由插件发起。该接口提供以下功能：

  - 启动插件注册

  - 请求对容器的更新

- plugin 服务是 NRI 用于与插件交互的公共接口。在此接口上的所有请求都由 NRI 或运行时发起。该接口提供以下功能：

  - 配置插件

  - 获取已存在的 Pod 和容器的初始列表

  - 将插件 hook 到 Pod/container 的生命周期事件中

  - 关闭插件

插件需要向 NRI 注册，用于接收和处理容器事件。在注册过程中，插件和 NRI 执行以下步骤的顺序：

- 插件注册至运行时
- NRI 向插件下发特定的配置数据
- 插件订阅 Pod 和容器生命周期事件
- NRI 向插件发送已存在的 Pod 和容器列表
- 插件请求对现有容器的更新

通过插件名称和插件索引向 NRI 注册插件。NRI 通过插件索引来确定所有插件在 hook Pod 和容器的生命周期事件的处理顺序。

NRI 插件名称用于 NRI 服务从默认插件配置路径 `/etc/nri/conf.d` 中选择对应插件的配置文件发送给 NRI 插件。只有当对应的 NRI 插件被 NRI 服务内部调用时，才会读取对应的配置文件。如果 NRI 插件是从外部启动的，那么它也可以通过其他方式获取配置。NRI 插件可以根据需要订阅 Pod 和容器的生命周期，并且返回修改后的配置。NRI 插件如果采用预注册的方式运行时，需要将可执行文件的命名规则需要符合 `xx-plugin_name`，例如 01-logger。其中 xx 必须为两位数字，作为 NRI 插件的索引，决定了插件的的执行顺序。

在注册和握手的最后一步，NRI 发送 CRI 运行时已知的所有的 Pod和容器的信息。此时插件可以对任何已经存在的 Pod 和容器进行更新。一旦握手结束，并且 NRI 插件成功向 NRI 服务注册之后，它将开始根据自己的订阅接收 Pod 和容器的生命周期事件。

## 运行时适配器

NRI 运行时适配包是用于集成到 NRI 并与 NRI 插件交互的接口运行时。它实现了插件发现，启动和配置。它还提供了将 NRI 插件插入到 CRI 运行时的 Pod 和容器的生命周期事件中的必要功能。

运行时适配器实现了多个 NRI 插件可能在处理同一个 Pod 或者容器的生命周期事件。它负责按照索引顺序依次调用插件，并把插件的修改内容合并后返回。在合并插件修改的 OCI spec 时，当检测到到多个 NRI 插件对同一个容器产生了冲突的修改，就会返回一个错误。

## 其他组件

NRI 还包含一个 NRI 插件 stub 库，为 NRI 插件的实现提供了一个简洁易用的框架。stub 库屏蔽了 NRI 插件的底层实现细节，它负责连接建立、插件注册、配置和事件订阅。

同时 NRI 也提供了一些 NRI 插件的示例，这些示例都是结合实际使用场景创建的，其中一些示例非常适合调试场景。目前，NRI 提供的所有示例插件都基于 stub 库实现的。这些示例插件的实现都可以用作学习使用 stub 库的教程。

另外，NRI 还包含一个 OCI 规范生成器主要用于 NRI 插件用来调整和更新 OCI spec，然后更新到容器。

# 可订阅事件

```go
// Handlers for NRI plugin event and request.
type handlers struct {
	Configure           func(context.Context, string, string, string) (api.EventMask, error)
	Synchronize         func(context.Context, []*api.PodSandbox, []*api.Container) ([]*api.ContainerUpdate, error)
	Shutdown            func(context.Context)

	// Pod 事件
	RunPodSandbox       func(context.Context, *api.PodSandbox) error
	StopPodSandbox      func(context.Context, *api.PodSandbox) error
	RemovePodSandbox    func(context.Context, *api.PodSandbox) error

	// 容器事件
	CreateContainer     func(context.Context, *api.PodSandbox, *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error)
	StartContainer      func(context.Context, *api.PodSandbox, *api.Container) error
	UpdateContainer     func(context.Context, *api.PodSandbox, *api.Container, *api.LinuxResources) ([]*api.ContainerUpdate, error)
	StopContainer       func(context.Context, *api.PodSandbox, *api.Container) ([]*api.ContainerUpdate, error)
	RemoveContainer     func(context.Context, *api.PodSandbox, *api.Container) error
	PostCreateContainer func(context.Context, *api.PodSandbox, *api.Container) error
	PostStartContainer  func(context.Context, *api.PodSandbox, *api.Container) error
	PostUpdateContainer func(context.Context, *api.PodSandbox, *api.Container) error
}
```

在事件中可以获得的 Pod 元信息：

- ID
- name
- UID
- namespace
- labels
- annotations
- cgroup parent directory
- runtime handler name

在事件中可以获得的容器元数据：

- ID
- pod ID
- name
- state
- labels
- annotations
- command line arguments
- environment variables
- mounts
- OCI hooks
- linux
  - namespace IDs
  - devices
  - resources
    - memory
      - limit
      - reservation
      - swap limit
      - kernel limit
      - kernel TCP limit
      - swappiness
      - OOM disabled flag
      - hierarchical accounting flag
      - hugepage limits
    - CPU
      - shares
      - quota
      - period
      - realtime runtime
      - realtime period
      - cpuset CPUs
      - cpuset memory
    - Block I/O class
    - RDT class

**容器调整**

在容器创建过程中可以调整容器的参数，在容器创建后，任何生命周期事件都可以更新容器的参数，但是调整参数和更新参数的范围是不同的，容器创建时支持更多的参数设置，容器创建完成后，只有部分参数可以修改。其中 ID、pod ID、name、state、labels、command line arguments、OCI hooks 和 linux.namespace IDs 信息不可修改。

**容器更新**

容器创建完成后，NRI 插件可以对容器进行更新。这个更新操作也可以由其他任何容器创建、更新或者停止的事件触发，或者可以主动更新容器参数。更新过程中，可以改的容器的参数要少于创建时可修改的参数，其中仅 linux.resources 信息可修改。

# 安全性

从安全角度来看，应该将 NRI 插件视为容器运行时的一部分。NRI 没有实现对其提供的功能的细粒度访问控制。访问 NRI 是通过限制对系统范围的 NRI socket 的访问来控制的。如果进程可以连接到 NRI socket 并发送数据，则可以访问通过 NRI 可用的完整功能范围。

特别是包括：

- 注入OCI hook，允许以与容器运行时相同的特权级别执行任意进程
- 对挂载点进行任意更改，包括新的绑定挂载点、更改 proc、sys、mqueue、shm 和 tmpfs 挂载点
- 添加或删除任意设备
- 对可用内存、CPU、block I/O 和 RDT 资源的限制进行任意更改，包括通过设置非常低的限制来拒绝服务

保护 NRI socket 的注意事项和原则与保护运行时本身的 socket 相同。除非它已经存在，否则 NRI 本身会创建目录来保存其 socket，该目录具有仅允许运行时进程的用户 ID 访问的权限。默认情况下，这限制 NRI 访问以 root UID 0 身份运行的进程。强烈建议不要更改默认 socket 权限。如果没有对容器安全的全部影响和潜在后果的充分理解，就永远不应该对 NRI 启用更宽松的访问控制。

当运行时管理 Kubernetes 集群中的 Pod 和容器时，使用 Kubernetes DaemonSets 可以方便地部署和管理 NRI 插件。除此之外，这需要将 NRI socket 挂载到运行插件的特权容器的文件系统中。对于保护 NRI socket 和 NRI 插件，应采取与 Kubelet Device Manager socket 和 Kubernetes device-plugin 类似的手段。

集群配置应确保未经授权的用户无法挂载 host 目录并创建特权容器来访问这些 socket 并充当 NRI 或 device-plugin。

# 与运行时集成

## Containerd

<div align=center><img width="800" style="border: 0px" src="/gallery/containerd/nri-integration.png"></div>

Containerd 在 v1.7.0 版本中新增对 NRI 特性的支持，通过在 Containerd 配置文件的 `[plugins."io.containerd.nri.v1.nri"]` 部分中配置：

```toml
  [plugins."io.containerd.nri.v1.nri"]
    # Enable NRI support in containerd.
    disable = false
    # Allow connections from externally launched NRI plugins.
    disable_connections = false
    # plugin_config_path is the directory to search for plugin-specific configuration.
    plugin_config_path = "/etc/nri/conf.d"
    # plugin_path is the directory to search for plugins to launch on startup.
    plugin_path = "/opt/nri/plugins"
    # plugin_registration_timeout is the timeout for a plugin to register after connection.
    plugin_registration_timeout = "5s"
    # plugin_requst_timeout is the timeout for a plugin to handle an event/request.
    plugin_request_timeout = "2s"
    # socket_path is the path of the NRI socket to create for plugins to connect to.
    socket_path = "/var/run/nri/nri.sock"
```

有两种方法可以启动 NRI 插件：

- 预注册（pre-connected）：当 NRI 适配器实例化时，NRI 插件就会自动启动。预注册就是将 NRI 的可执行文件放置到 NRI 插件的指定路径中，默认路径通过 plugin_path 指定，当 Containerd 启动时，就会自动加载并运行在该路径下注册的 NRI 插件

- 外部运行（external）：NRI 插件进程可以由 systemd 创建，或者运行在 Pod 中。只要保证 NRI 插件可以通过 NRI socket 和 Containerd 进行通信即可，默认的 NRI socket 存储路径通过 socket_path 指定

预注册的插件是通过一个预先连接到 NRI 的 socket 启动，外部运行的插件通过 NRI socket 向 NRI 适配器注册自己。预注册插件和外部启动插件，这两种运行方式唯一的不同点就是如何启动以及如何连接到 NRI。一旦建立了连接，所有的 NRI 插件都是相同的。

NRI 可以通过 disable_connections 选项禁用外部运行插件的连接，在这种情况下 NRI socket 将不会被创建。

# 简单示例

*以 [NRI logger](https://github.com/containerd/nri/tree/main/plugins/logger) 插件为例。*

**插件编译**

```shell
$ git clone https://github.com/containerd/nri.git
$ cd plugins/logger/

# 命名格式必须为“索引-名称”，其中索引必须为 2 位数字，否则无法通过校验
# FATAL  [0000] failed to create plugin stub: invalid plugin index "nri", must be 2 digits
$ go build -o 01-logger nri-logger
$ cp 01-logger /opt/nri/plugins/00-logger
```

logger 插件逻辑就是在各个 Pod 和容器生命周期节点格式化输出元数据信息。此外， CreateContainer 阶段中会设置环境变量和注解等信息：

```go
func (p *plugin) CreateContainer(_ context.Context, pod *api.PodSandbox, container *api.Container) (*api.ContainerAdjustment, []*api.ContainerUpdate, error) {
	dump("CreateContainer", "pod", pod, "container", container)

	adjust := &api.ContainerAdjustment{}

	if cfg.AddAnnotation != "" {
		adjust.AddAnnotation(cfg.AddAnnotation, fmt.Sprintf("logger-pid-%d", os.Getpid()))
	}
	if cfg.SetAnnotation != "" {
		adjust.RemoveAnnotation(cfg.SetAnnotation)
		adjust.AddAnnotation(cfg.SetAnnotation, fmt.Sprintf("logger-pid-%d", os.Getpid()))
	}
	if cfg.AddEnv != "" {
		adjust.AddEnv(cfg.AddEnv, fmt.Sprintf("logger-pid-%d", os.Getpid()))
	}
	if cfg.SetEnv != "" {
		adjust.RemoveEnv(cfg.SetEnv)
		adjust.AddEnv(cfg.SetEnv, fmt.Sprintf("logger-pid-%d", os.Getpid()))
	}

	return adjust, nil, nil
}
```

**配置启动**

这里采用外部启动（00-logger）和预先配置（01-logger）两种启动方式

- 00-logger：设置环境变量 logger-env 与注解 logger-annotation，值为 logger-pid-\<PID\>
- 01-logger：设置环境变量 logger-env，值为 logger-pid-\<PID\>

重启 Containerd 发现预先配置的 01-logger 插件：

```shell
$ journalctl -xeu containerd
Jul 05 17:35:43 wnx containerd[84985]: time="2023-07-05T17:35:43.730685674+08:00" level=info msg="using experimental NRI integration - disable nri plugin to prevent this"
...
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.019786972+08:00" level=info msg="starting plugins..."
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.019915270+08:00" level=info msg="discovered plugin 01-logger"
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.019929420+08:00" level=info msg="starting plugin \"logger\"..."
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.049078817+08:00" level=info msg="plugin \"pre-connected:01-logger[84985]\" registered as \"01-logger\""
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.050249456+08:00" level=info msg="plugin invocation order"
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.050284541+08:00" level=info msg="  #1: \"01-logger\" (pre-connected:01-logger[84985])"
Jul 05 17:35:44 wnx containerd[84985]: time="2023-07-05T17:35:44.050528623+08:00" level=info msg="containerd successfully booted in 1.616308s"
```

00-logger 插件注册后，订阅了所有的 Pod 和容器事件；插件启动后，即收到了运行时所有的 Pod 和容器信息：

```shell
$ ./00-logger --set-annotation logger-annotation
INFO   [0000] Created plugin 00-logger (00-logger, handles RunPodSandbox,StopPodSandbox,RemovePodSandbox,CreateContainer,PostCreateContainer,StartContainer,PostStartContainer,UpdateContainer,PostUpdateContainer,StopContainer,RemoveContainer) 
INFO   [0000] Registering plugin 00-logger...              
INFO   [0000] Configuring plugin 00-logger for runtime containerd/v1.7.2... 
INFO   [0000] got configuration data: "" from runtime containerd v1.7.2 
INFO   [0000] Subscribing plugin 00-logger (00-logger) for events RunPodSandbox,StopPodSandbox,RemovePodSandbox,CreateContainer,PostCreateContainer,StartContainer,PostStartContainer,UpdateContainer,PostUpdateContainer,StopContainer,RemoveContainer 
INFO   [0000] Started plugin 00-logger...                  
INFO   [0000] Synchronize: pods:                           
INFO   [0000] Synchronize:    - annotations:               
INFO   [0000] Synchronize:        io.kubernetes.cri.container-type: sandbox 
INFO   [0000] Synchronize:        io.kubernetes.cri.sandbox-cpu-period: "100000" 
INFO   [0000] Synchronize:        io.kubernetes.cri.sandbox-cpu-quota: "100000" 
INFO   [0000] Synchronize:        io.kubernetes.cri.sandbox-cpu-shares: "1024" 
...
```

Containerd 服务也收到了来自外部启动的 00-logger 插件信息：

```shell
$ journalctl -xeu containerd
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.236103390+08:00" level=info msg="plugin \"external:00-logger[87525]\" registered as \"00-logger\""
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.237601881+08:00" level=info msg="Synchronizing NRI (plugin) with current runtime state"
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.328071227+08:00" level=info msg="synchronizing plugin 00-logger"
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.797028952+08:00" level=info msg="plugin invocation order"
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.797164886+08:00" level=info msg="  #1: \"00-logger\" (external:00-logger[87525])"
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.797197017+08:00" level=info msg="  #2: \"01-logger\" (pre-connected:01-logger[84985])"
Jul 05 17:38:49 wnx containerd[84985]: time="2023-07-05T17:38:49.797226434+08:00" level=info msg="plugin \"00-logger\" connected"
```

多 NRI 插件是按照索引顺序执行，因此 01-logger 会重新设置 00-logger 设置的 logger-env 环境变量：

```shell
$ crictl inspect 6577fa85ac7e6 | grep logger
          "logger-env=logger-pid-85262"
        "logger-annotation": "logger-pid-87525"
```

# 更多价值

为了满足不同业务应用场景的需求，特别是在在线任务与离线任务混布的场景下，在提高资源利用率的同时，也要保证延迟敏感服务可以得到充分的资源保证，这就需要 Kubernetes 提供更加细粒度的资源管理功能，增强容器的隔离性，减少容器之间的互相干扰。例如，CPU 编排，内存分层，缓存管理，IO 管理等。目前有很多方案，但是都有其一定的局限性。

截至目前，Kubernetes 并没有提供一个非常完善的资源管理方案，很多 Kubernetes 周边的开源项目通过一些自己的方式修改 Pod 的部署和管理流程，实现资源分配的细粒度管理。例如 [cri-resource-manager](https://github.com/intel/cri-resource-manager)、[Koordinator](https://github.com/koordinator-sh/koordinator)、[Crane](https://github.com/gocrane/crane) 等项目。

这些项目对 Kubernetes 创建和更新 Pod 的流程的优化可以大致分为两种模式，一种是 proxy 模式，一种是 standalone 模式。

<div align=center><img width="800" style="border: 0px" src="/gallery/containerd/proxy-standalone.png"></div>

在目前的 K8s 架构中（如图 a）Kubelet 通过调用 CRI 兼容的容器运行时创建和管理 Pod。CRI 运行时再通过调用 OCI 兼容的 low-level 运行时创建容器。

Proxy 模式（如图 b）则是在客户端 Kubelet 和 CRI 运行时之间增加一个 CRI proxy 中继请求和响应，在 proxy 中劫持 Pod 以及容器的创建/更新/删除事件，对Pod spec 进行修改或者完善，将硬件感知的资源分配策略应用于容器中。

Standalone 模式（如图 c）则是在每一个工作节点上创建一个 agent，当这个 agent 监听到在本节点的 Pod 创建或者修改事件的时候，再根据 Pod spec 中的注解等扩展信息，转换成细粒度资源配置的 spec，然后调用 CRI 运行时实现对 Pod 的更新。

这两种方式在满足特定业务需求的同时也存在一定的缺点, 两种方式都需要依赖额外的组件，来捕获 Pod 的生命周期事件。proxy 模式增加了 Pod 创建管理流程的链路以及部署和维护成本，standalone 模式是在侦听到 Pod 创建以及修改的事件后，才会对 Pod 进行更新，会有一定的延迟。

使用 NRI 可以将 Kubelet 的 Resource Manager 下沉到 CRI 运行时层进行管理。Kubelet 当前不适合处理多种需求的扩展，在 Kubelet 层增加细粒度的资源分配会导致 Kubelet 和 CRI 的界限越来越模糊。而 NRI，则是在 CRI 生命周期间做调用，更适合做资源绑定和节点的拓扑感知。并且在 CRI 内部做插件定义和迭代，可以做到上层 Kubenetes 以最小的代价来适配变化。

到现在为止，已经有越来越多的节点资源细粒度管理方案开始探索使用 NRI 实现的可能性。当 NRI 成为节点细粒度资源分配管理方案后，可以进一步提高资源管理方案的标准化，提高相关组件的可复用性。参考：https://github.com/containers/nri-plugins。
