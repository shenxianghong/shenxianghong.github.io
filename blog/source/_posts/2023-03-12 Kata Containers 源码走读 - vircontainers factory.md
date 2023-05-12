---
title: "「 Kata Containers 」源码走读 — virtcontainers/factory"
excerpt: "virtcontainers 中与 Factory、CacheServer 等工厂模式相关的流程梳理"
cover: https://picsum.photos/0?sig=20230312
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-03-12
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

# Factory

*<u>src/runtime/virtcontainers/factory.go</u>*

Factory 继承自 FactoryBase，两者的区别在于 FactoryBase 用于创建 base VM（即为模版 VM），创建后会将其暂停，而 Factory 会在 VM 使用时将其恢复，并热更新 VM 以满足运行时的规格要求。

FactoryBase 有四种实现：direct、template、grpccache 和 cache。但是它们并不会对外暴露使用，而是在 Factory 的工厂函数中根据具体的配置细节初始化对应的实现，作为统一的 Factory 对外提供接口调用，即 factory。

```go
type direct struct {
	config vc.VMConfig
}
```

```go
type grpccache struct {
	conn   *grpc.ClientConn
	config *vc.VMConfig
}
```

```go
type template struct {
	// [factory].template_path
	statePath string
	config    vc.VMConfig
}
```

```go
type cache struct {
	// cache factory 的初始化必须基于 template factory 或者 direct factory。
	base base.FactoryBase
	cacheCh chan *vc.VM
	closed  chan<- int
	vmm map[*vc.VM]interface{}
	wg        sync.WaitGroup
	closeOnce sync.Once
	vmmLock sync.RWMutex
}
```

```go
type factory struct {
	base base.FactoryBase
}
```

**工厂函数**

*目前来看，grpccache 初始化的条件应该不存在。此外，当启用 VM factory 时，必然是 VM template 和 VM cache 二选一，所以 direct 不会作为 factory 直接对外使用，而是进一步初始化成 cache factory。*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/factory.go#L51)

1. 校验 VMConfig 配置的合法性，其中包括 [hypervisor].kernel 是否不为空，[hypervisor].image 和 [hypervisor].initrd 有且仅有一个。并设置 [hypervisor].default_vcpus 缺省时为 1（单位：Core），[hypervisor].default_memory 缺省时为 2048（单位：MiB）
2. 当启用 VM template 时（即 [factory].enable_template 为 true），则初始化 template factory
   1. 如果 fetchOnly 为 true，则校验 [factory].template_path 目录（默认为 /run/vc/vm/template）下是否存在 state 和 memory 文件
   2. 如果 fetchOnly 为 false，则初始化 template factory
      1. 校验 [factory].template_path 目录下是否不存在 state 和 memory 文件
      2. 创建 [factory].template_path 目录，将 tmps 挂载到此目录下，大小为 [hypervisor].default_memory + 8 MiB（amd64 架构下为 8 MiB；arm64 架构下为 300 MiB），并在此目录下创建 memory 文件
      3. 调用 **NewVM**，基于 VMConfig 创建 VM（创建后则作为模版 VM）
      4. 调用 agent 的 **disconnect**，断开与 agent 的链接
      5. 调用 hypervisor 的 **PauseVM**，暂停 VM
      6. 调用 hypervisor 的 **SaveVM**，保存 VM 到磁盘文件
      7. 调用 hypervisor 的 **StopVM**，关停 VM
      8. 调用 store 的 **Destroy**，删除状态数据目录
3. 当启用 VM cache 时（即 [factory].vm_cache_number 大于 0），则初始化 cache factory
   1. 初始化 direct factory（cache factory 的初始化必须依赖其他 factory）
   2. 反复调用 **GetBaseVM**（direct.GetBaseVM），直至创建暂停状态的 VM 数量等于 [factory].vm_cache_number
   3. 将这些事先创建好的 base VM 维护在 cache 中<br>*后续需要时通过 **GetBaseVM**（cache.GetBaseVM），获取 base VM，并通过 **GetVM** 热更新*

## NewVM

**基于 VMConfig 创建 VM**

*NewVM 并非 FactoryBase 定义接口，而是 virtcontainers 提供的一个基于 VMConfig 创建 VM 的工厂函数，仅用于 factory 相关流程*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/vm.go#L85)

1. 校验 VMConfig 配置的合法性，其中包括 [hypervisor].kernel 是否不为空，[hypervisor].image 和 [hypervisor].initrd 有且仅有一个。并设置 [hypervisor].default_vcpus 缺省时为 1，[hypervisor].default_memory 缺省时为 2048
2. 初始化 hypervisor 和 agent
3. 调用 hypervisor 的 **CreateVM**，创建一个不含网络信息的 VM
4. 调用 agent 的 **configure** 和 **setAgentURL**，配置 agent 相关信息
5. 调用 hypervisor 的 **StartVM**，启动 VM
6. 如果 VM 不是从 template 启动（因为从 template 启动的 VM，会进入 pause 状态），则调用 agent 的 **check**，检测服务存活性

## Config

**获取 base factory 的配置信息**

*cache、direct、grpccache 和 template 实现方式相同。* 

1. 返回 VMConfig

## GetVMStatus

**获取 base VM 的状态信息**

*direct、grpccache 和 template 实现下均不支持此接口，会触发 panic。*

### cache

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/cache/cache.go#L98)

1. 针对 cache 中缓存的每一个 VM，获取其 CPU 和内存大小（即配置中声明的默认大小），并调用 hypervisor 的 **GetPids**，获取相关的 PID

## GetBaseVM

**获取 base VM**

### direct

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/direct/direct.go#L32)

1. 调用 **NewVM**，创建 VM
1. 调用 hypervisor 的 **PauseVM**，将其暂停并返回

### template

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/template/template.go#L76)

1. 调用 **NewVM**，创建 VM（该 VM 基于模版创建，创建后不作为模版 VM）

### grpccache

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/grpccache/grpccache.go#L53)

1. gRPC 调用 cache server 的 **GetBaseVM**，获取 VM
1. 调用 hypervisor 的 **fromGrpc**，配置 hypervisor 信息
1. 调用 agent 的 **configureFromGrpc**，配置 agent 信息
1. 基于配置后的 hypervisor、agent 和 gRPC 返回体构建并返回 VM

### cache

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/cache/cache.go#L112)

1. 从缓存的 base VM 中返回一个

## CloseFactory

**关闭并销毁 factory**

*direct、grpccache 实现下此接口不做任何处理，直接返回即可。*

### template

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/template/template.go#L81)

1. 移除 [factory].template_path 挂载点（默认为 /run/vc/vm/template），并删除此目录

### cache

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/cache/cache.go#L121)

1. 调用 base factory 的 **ClostFactory**，关闭 factory<br>*如上所述，cache factory 也就是调用 direct factory 的 CloseFactory*

## GetVM

**热更新 base VM 以满足需求**

*GetVM 不是 base factory 的接口，而是 factory 的接口<br>GetVM 接受一个 VMConfig 类型的参数，该参数描述了预期的 VM 配置（下称 newConfig），而 factory 实现中的 VMConfig 是 base VM 的配置（下称 baseConfig），两者的差异点补齐便是 GetVM 操作的核心逻辑*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/factory/factory.go#L145)

1. 校验 newConfig 配置的合法性，其中包括 [hypervisor].kernel 是否不为空，[hypervisor].image 和 [hypervisor].initrd 有且仅有一个。并设置 [hypervisor].default_vcpus 缺省时为 1，[hypervisor].default_memory 缺省时为 2048
2. 调用 **Config**，获取 baseConfig 信息，校验两个配置信息是否并不冲突
3. 调用 **GetBaseVM**，获取 base VM
4. 调用 hypervisor 的 **ResumeVM**，将 VM 从暂停状态恢复
5. 借助 /dev/urandom 重新生成随机熵，调用 agent 的 **reseedRNG**，为 guest 内存重新生成随机数
6. 为了补齐 VM 的暂停时间，调用 agent 的 **setGuestDateTime**，同步 host 时间至 guest 中
7. 如果 base VM 中的 CPU 数量小于期望配置中的 CPU 数量，则调用 hypervisor 的 **HotplugAddDevice**，热添加差值 CPU；内存同理
8. 当有 CPU 或内存的热添加动作后，调用 agent 的 **onlineCPUMem**，通知 agent 上线资源

****

# Cache Server

*<u>src/runtime/protocols/cachecache.pb.go</u>*

cache server 并非默认启动的 gRPC 服务，而是在 VM Cache 特性启用时，通过 kata-runtime factory init 命令启动。

```go
type cacheServer struct {
	rpc     *grpc.Server
	factory vc.Factory
	done    chan struct{}
}
```

**工厂函数**

 [source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L148)

1. 基于配置，初始化一个新的 factory（即 fetchOnly 为 false）
2. 启动 gRPC 服务，监听 [factory].vm_cache_endpoint 地址（默认为 /var/run/kata-containers/cache.sock）

## Config

**获取 base factory 的配置信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L54)

1. 返回体结构如下

   ```go
   type GrpcVMConfig struct {
   	// VMConfig
   	Data                 []byte
   	// VMConfig.AgentConfig 
   	AgentConfig          []byte
   }
   ```

2. 调用 factory 的 **Config**，获取 base factory 的配置信息<br>*由于 base factory 在 cache server 启动后便固定规格，因此在首次调用后，会保存配置信息，之后的调用直接返回即可*

## GetBaseVM

**获取 base VM**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L69)

1. 返回体结构如下

   ```go
   type GrpcVM struct {
   	// VM.id
   	Id                   string
   	// VM.hypervisor.toGrpc
   	Hypervisor           []byte
   	ProxyPid             int64
   	ProxyURL             string
   	// VM.cpu
   	Cpu                  uint32
   	// VM.memory
   	Memory               uint32
   	// VM.cpuDelta
   	CpuDelta             uint32
   }
   ```

2. 调用 factory 的 **Config**，获取 base factory 的配置信息
3. 调用 factory 的 **GetBaseVM**，获取 base VM

## Status

**获取 base VM 的状态信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L95)

1. 返回体结构如下

   ```go
   type GrpcStatus struct {
   	// 当前进程的 PID
   	Pid                  int64
   	// factory.GetVMStatus 的返回结果
   	Vmstatus             []*GrpcVMStatus
   }
   ```

2. 调用 factory 的 **GetVMStatus**，获取 base VM 的状态信息

## Quit

**关闭 cache server**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L86)

1. 1 秒钟后，关闭 cache gRPC server
