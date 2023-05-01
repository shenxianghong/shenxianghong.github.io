---
title: "「 Kata Containers 」 3.4 源码走读 — virtcontainers"
excerpt: "virtcontainers 库中 VC、VCSandbox 和 VCContainer 模块源码走读"
cover: https://picsum.photos/0?sig=20230130
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-01-30
toc: true
categories:
- Code Walkthrough
tag:
- Kata Containers
---

<div align=center><img width="300" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> Based on **v3.0.0**

virtcontainers 本质上不是一个独立组建，而是一个用于构建硬件虚拟化容器运行时的 Golang 库。

现有的少数部分基于 VM 的容器运行时都共享相同的硬件虚拟化语义，但是使用不同的代码库来实现，virtcontainers 的目标就是将这部分封装成一个通用的 Golang 库。

理想情况下，基于 VM 的容器运行时，会从将它们实现的运行时规范（例如 OCI spec 或 Kubernetes CRI）转换成 virtcontainers API。

virtcontainers API 大致受到 Kubernetes CRI 的启发。然而，尽管这两个项目之间的 API 相似，但 virtcontainers 的目标不是构建 CRI 实现，而是提供一个通用的、运行时规范不可知的、硬件虚拟化的容器库，其他项目可以利用它来自己实现 CRI。

****

# VC

*<u>src/runtime/virtcontainers/interfaces.go</u>*

virtcontainers 库的入口模块，VC 初始化 VCSandbox 模块管理 sandbox，进而初始化 VCContainer 模块管理容器。

```go
type VCImpl struct {
	factory Factory
}
```

*工厂函数为参数赋值初始化，无复杂逻辑，不作详述。*

VC 中声明的 **SetLogger** 和 **SetFactory** 均为参数赋值，无复杂逻辑，不作详述。

## CreateSandbox

**创建 sandbox 与 pod_sandbox 容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/implementation.go#L34)

1. 创建 sandbox
   1. 校验 sandboxConfig 的 annotation 中自定义运行时配置（称为 assets ）的合法性并设置<br>*annotation 例如 io.katacontainers.hypervisor.kernel 为用户上层通过 Pod annotation 定义，CRI 会透传给底层运行时*
   2. 初始化 VCSandbox，准备所需环境
   3. 如果 sandbox 中状态信息（即 sandbox.state）已经存在，则表明不是新创建的 pod_sandbox 容器，无需后续动作，仅用于状态更新维护，直接返回 sandbox 即可
   4. 调用 fsShare 的 **Prepare**，准备 sandbox 所需的共享文件系统目录
   5. 调用 agent 的 **createSandbox**，准备 sandbox 所需环境
   6. 设置 sandbox 状态为 ready
2. 创建 sandbox 网络环境
   1. 如果 [runtime].disable_new_netns 未启用并且不是 VM factory 场景（在 VM factory 场景下，网卡是在 VM 启动后热插进去的），则扫描容器环境中 netns 下现有的网卡信息（比如 eth0 网卡）
   2. 将容器环境的网卡 attach 到 VM 中
   3. 根据 [runtime].internetworking_model 的类型，给 VM 添加特定类型的 tap0_kata 网卡，配置 TC 策略（如果 internetworking_model 为 TcFilter），打通 CNI 网络和 VM 网络之间的连通性
3. 调用 resCtrl 的 **setupResourceController**，将当前进程加入 cgroup 中管理
4. 启动 VM（在 1-2 步骤中的 VCSandbox 初始化流程中，已经创建了 VM）
   1. 如果 [hypervisor].enable_debug 启用（用于输出  hypervisor 和 kernel 产生的消息），则调用 hypervisor 的 **GetVMConsole**，获取 VM console 地址（/run/vc/vm/\<sandboxID\>/console.sock）
   2. 在 VM factory 场景下，获取 factory 中缓存的 VM，调用 agent 的 **reuseAgent**，更新 agent 实例，并创建软链接 /run/vc/vm/\<sandboxID\> 指向 /run/vc/vm/\<vmID\>；否则，调用 hypervisor 的 **StartVM**，启动 VM 进程
   3. 在 VM factory 场景下，扫描容器环境中 netns 下现有的网卡信息，热插到 VM 中
   4. 如果 [hypervisor].enable_debug 启用，实时读取 VM console 地址获取其实时内容，并以 debug 级别日志形式输出
   5. 调用 agent 的 **startSandbox**
5. 关闭 veth-pair（如 br0_kata）位于 host 侧的 vhost_net 句柄（/dev/vhost-net）
6. 调用 agent 的 **getGuestDetails**，获取 guest 信息详情，更新至 sandbox 中
7. 创建 sandbox 中的每一个容器（其实，此时 sandbox 中仅有一个容器，就是 pod_sandbox 容器本身）
   1. 初始化 VCContainer，准备容器所需环境
   1. 根据 [hypervisor].disable_block_device_use、agent 是否具备使用块设备能力以及 hypervisor 是否允许块设备热插拔，判断是否当前支持块设备，并且容器的 rootfs 类型不是 fuse.nydus-overlayfs，也就是 rootfs 是基于块设备创建的
      1. 通过 /sys/dev/block/\<major\>-\<minor\>/dm 的存在性，判断是否为 devicemapper 块设备
      1. 如果是 devicemapper 块设备，则调用 devManager **NewDevice**，初始化设备，并调用 devManager 的 **AttachDevice**，热插到 VM 中 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/containers/\<sandboxID\> 路径
   1. 针对容器中的每一个设备，调用 devManager 的 **AttachDevice**，attach 到 VM 中
   1. 调用 agent 的 **createContainer**，创建 pod_sandbox 容器
   1. 设置容器状态为 ready
   1. 调用 store 的 **ToDisk**，保存状态数据到文件中
   1. 更新维护在 sandbox 中的容器信息
8. 调用 **updateResources**，热更新 VM 的资源规格（由于该流程中仅为创建 pod_sandbox，不涉及 pod_container，因此为配置中声明的 [hypervisor].default_vcpus 和 [hypervisor].default_memory）
9. 调用 resCtrl 的 **resourceControllerUpdate**，更新 sandbox 的 cgroup
10. 调用 store 的 **ToDisk**，保存状态数据到文件中


## CleanupContainer

**关停、删除容器并销毁 sandbox 环境**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/implementation.go#L41)

1. 调用 store 的 **FromDisk**，读取 sandbox 状态信息
2. 获取 sandbox 并更新其中的容器（更新的意义在于后续的删除操作以文件内容为准）
3. 调用 VCSandbox 的 **StopContainer** 和 **DeleteContainer**，关停并删除该容器
4. 调用 VCSandbox 的 **GetAllContainers**，获取 sandbox 中的所有容器，如果仍大于 0（说明当前 sandbox 仍有容器存在，需要保留 sandbox 环境），否则调用 VCSandbox 的 **Stop**，关停 sandbox，并调用 VCSandbox 的 **Delete**，删除 sandbox

****

# VCSandbox

*<u>src/runtime/virtcontainers/interfaces.go</u>*

virtcontainers 库中用于管理 sandbox 的模块，同时调用 VCContainer 模块间接管理容器。

```go
type Sandbox struct {
	ctx        context.Context
	devManager api.DeviceManager
	factory    Factory
	hypervisor Hypervisor
	agent      agent
	store      persistapi.PersistDriver
	fsShare    FilesystemSharer

	swapDevices []*config.BlockDrive
	volumes     []types.Volume

	monitor         *monitor
	config          *SandboxConfig
	annotationsLock *sync.RWMutex
	wg              *sync.WaitGroup
	cw              *consoleWatcher

	sandboxController  resCtrl.ResourceController
	overheadController resCtrl.ResourceController

	containers map[string]*Container

	id string

	network Network

	state types.SandboxState

	sync.Mutex

	swapSizeBytes int64
	shmSize       uint64
	swapDeviceNum uint

	sharePidNs        bool
	seccompSupported  bool
	disableVMShutdown bool
}
```

**工厂函数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L527)

1. 校验 [runtime].experimental 是否为可支持的特性
2. 初始化 hypervisor、agent、store、fsSharer、devManager 等
3. 初始化 resourceController（[runtime].sandbox_cgroup_only 为 true 表示 Pod 所有的线程全部由 sandboxController 管理；反之，仅 vCPU 线程由 sandboxController 管理，而其余的由 overheadController 管理）
   1. 获取到 spec.Linux.CgroupsPath（缺省为 /vc），如果 cgroup 不是由 systemd 纳管（通过 cgroupPath 格式判断），则最后一级路径新增 kata_ 前缀
   2. 获取 spec.Linux.Resources.Devices 中 /dev/null 和 /dev/urandom 设备信息（如果未声明，则构建）
   3. 调用 devManager 的 **GetAllDevices**，获取所有的设备；进一步调用 device 的 **GetHostPath**，获取设备位于 host 上的路径，将其构建成 cgroup 管理形式
   4. 调用 resCtrl 的 **NewSandboxResourceController**，初始化 sandboxController（cgroupPath 为步骤 1 处理后的结果）
   5. 如果 [runtime].sandbox_cgroup_only 为 false，则调用 resCtrl 的 **NewResourceController**，初始化 overheadController （cgroupPath 为 /kata_overhead/\<sandboxID\>）
4. 从文件恢复 sandbox 状态信息
   1. 调用 store 的 **FromDisk**，尝试从文件中恢复 sandbox 状态信息<br>*当 sandbox 新创建的时候，并没有状态文件，因此必然失败，但是会忽略错误信息*
   2. 调用 hypervisor 的 **Load**，加载 hypervisor 信息
   3. 调用 devManager 的 **LoadDevices**，加载设备信息
   4. 调用 agent 的 **load**，加载 agent 信息
   5. 调用 endpoint 的 **load**，加载网络 endpoint 信息
5. 校验 sandboxConfig 中的 hypervisor 配置的合法性，其中包括 [hypervisor].kernel 是否不为空，[hypervisor].image 和 [hypervisor].initrd 有且仅有一个。并设置 [hypervisor].default_vcpus 缺省时为 1，[hypervisor].default_memory 缺省时为 2048
6. 调用 hypervisor 的 **CreateVM**，创建 VM
7. 调用 agent 的 **init**，准备 agent 环境

VCSandbox 中声明的 **Annotations**、**GetNetNs**、**GetAllContainers**、**GetAnnotations**、**GetContainer**、**ID** 和 **SetAnnotations** 均为参数获取与赋值，无复杂逻辑，不作详述。

## updateResources

**热更新 VM 的资源规格**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1966)

1. 如果 [runtime].static_sandbox_resource_mgmt 启用时（Kata 将在启动虚拟机之前确定合适的 sandbox 内存和 CPU 大小，而非动态更新。 作为 hypervisor 不支持 CPU、内存热插拔时的解决方案），则不触发更新操作，直接返回
2. 计算 sandbox 中所有状态非 stopped 容器的 CPU 、内存和 SWAP 总量（前提是 [hypervisor].enable_guest_swap 开启，并且 memory.swappiness 大于 0，才会考虑 SWAP 资源），加上基础 VM 大小，得出最后预期的 VM 大小，公式为

   > CPU = (C1.cpu-quota * 1000 / C1.cpu-period + 999) / 1000 + C2... + [hypervisor].default_vcpus
   >
   > MEM = C1.memory-limit + C1.hugepages-limit + C2... + [hypervisor].default_memory
   >
   > SWAP = 
   >
   > | `io.katacontainers.container.resource.swap_in_bytes` | `memory_limit_in_bytes` | swap size                                                    |
   > | ---------------------------------------------------- | ----------------------- | ------------------------------------------------------------ |
   > | set                                                  | set                     | `io.katacontainers.container.resource.swap_in_bytes`- `memory_limit_in_bytes` |
   > | not set                                              | set                     | `memory_limit_in_bytes`                                      |
   > | not set                                              | not set                 | `io.katacontainers.config.hypervisor.default_memory`         |
   > | set                                                  | not set                 | cgroup doesn't support this usage                            |

   *截至 Kata 3.0，在 K8s 场景下，SWAP 功能仍存在异常：参考 https://github.com/kata-containers/kata-containers/issues/5627*
3. 如果预期 SWAP 比当前 sandbox 的 SWAP 多，则需要新增一个大小为两者差值的 SWAP 文件
   1. 创建 /run/kata-containers/shared/sandboxes/swap\<ID\> 文件（sandbox 中的 SWAP 序号从 0 递增）
   2. 调整文件的大小为差值和 10 倍内存分页中最大值（小于 10 倍内页分页的 SWAP 会被 mkswap 拒绝：mkswap: error: swap area needs to be at least 40 KiB，内存分页：4096），并额外追加一个内存分页大小（SWAP 文件需要一个内存分页大小储存元数据）
   3. 调用系统命令 mkswap，转换为 SWAP 文件，并构建 raw 格式的块设备类型
   4. 调用 hypervisor 的 **HotplugAddDevice**，热添加 SWAP 文件到 VM 中
   5. 调用 agent 的 **addSwap**，配置 SWAP 文件
4. 调用 hypervisor 的 **ResizeVCPUs**，调整 VM CPU 数量
5. 如果为新增调整，则调用 agent 的 **onlineCPUMem**，通知 agent 上线热添加部分的 CPU
6. 循环调用 hypervisor 的 **GetTotalMemoryMB**，获取当前 VM 的内存数量，比对预期 VM 的内存数量，在不超出最大热添加内存数量限制的前提下（部分场景下，例如 ACPI 热插拔，内存的单次热添加有最大数量限制；而在 virtio-mem 下，即 [hypervisor].enable_virtio_mem 启用， 且 /proc/sys/vm/overcommit_memory 文件内容为 1，则没有最大热添加数量限制），调用 hypervisor 的 **ResizeMemory**，分批热添加 VM 内存
7. 调用 agent 的 **memHotplugByProbe**，通知 agent 内存热插事件（如果 guest 内核支持内存热添加探测），并调用 agent 的 **onlineCPUMem**，通知 agent 上线热添加部分的内存

## Stats

**获取 sandbox 的统计信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1547)

1. 调用 sandboxController 的 **Stat**，获取 sandbox 全部的 cgroup 统计信息（截至当前，并未聚合 kata_overhead 的统计信息，即 overheadController 部分）
2. 调用 hypervisor 的 **GetThreadIDs**，获取 hypervisor 使用的 CPU 数量
3. 聚合以上信息并返回

## Start

**启动 sandbox 与 pod_sandbox 容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1662)

1. 校验 sandbox 状态是否为 ready、paused 或 stopped
2. 设置 sandbox 状态为 running
3. 针对 sandbox 中的每一个容器，调用 VCContainer 的 **start**，启动容器（其实，此时 sandbox 中仅有一个容器，就是 pod_sandbox 容器本身）
4. 调用 store 的 **ToDisk**，保存状态数据到文件中

## Stop

**关停 sandbox 与容器，并清理相关资源**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1687)

1. 如果 sandbox 状态已经为 stopped，则不做任何操作
2. 校验 sandbox 状态是否为 ready、running 或者 paused
3. 针对 sandbox 中的每一个容器，调用 VCContainer 的 **stop**，关停容器
4. 调用 agent 的 **stopSandbox**，关停 sandbox
5. 调用 hypervisor 的 **StopVM**，关停 VM
6. 如果 [hypervisor]. enable_debug 启用，则关闭 VM console
7. 设置 sandbox 状态为 stopped
8. 移除 host 上的 sandbox 网络资源
9. 调用 store 的 **ToDisk**，保存状态数据到文件中
10. 调用 agent 的 **disconnect**，关闭与 agent 的连接
11. 移除 host 上的 /run/kata-containers/shared/sandboxes/swap\<ID\> 文件（sandbox 中的 SWAP 序号从 0 递增）

## Delete

**销毁 sandbox 与容器，并清理相关资源**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L798)

1. 校验 sandbox 的状态是否为 ready、paused 和 stopped
2. 针对 sandbox 中的每一个容器，调用 VCContainer 的 **delete**，删除容器
3. 如果在 root 权限下，则调用 resCtrl 的 **resourceControllerDelete**，删除相关的 resourceController 以及 cgroup 资源
4. 关停 sandbox 的 monitor
5. 调用 hypervisor 的 **Cleanup**，清理 hypervisor 资源
6. 调用 fsShare 的 **Cleanup**，清理 sandbox 的共享文件系统
7. 调用 store 的 **Destroy**，删除状态数据目录

## Status

**获取 sandbox 与容器的详细信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L333)

1. 针对 sandbox 中的每一个容器，获取其状态信息（例如：ID、rootfs、状态、启动时间、annotation、PID 等）
2. 结合 sandbox 的状态信息（例如 ID、状态、hypervisor 类别、hypervisor 配置、annotation 等）返回

## CreateContainer

**创建 pod_container 容器并热更新 sandbox 规格**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1300)

1. 初始化 VCContainer，准备容器环境，挂载设备等
2. 根据 [hypervisor].disable_block_device_use、agent 是否具备使用块设备能力以及 hypervisor 是否允许块设备热插拔，判断是否当前支持块设备，并且容器的 rootfs 类型不是 fuse.nydus-overlayfs，也就是 rootfs 是基于块设备创建的
   1. 通过 /sys/dev/block/\<major\>-\<minor\>/dm 的存在性，判断是否为 devicemapper 块设备
   1. 如果是 devicemapper 块设备，则调用 devManager **NewDevice**，初始化设备，并调用 devManager 的 **AttachDevice**，热插到 VM 中 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/containers/\<sandboxID\> 路径
3. 针对容器中的每一个设备，调用 devManager 的 **AttachDevice**，热插到 VM 中
4. 调用 agent 的 **createContainer**，创建 pod_container 容器
5. 设置容器状态为 ready
6. 调用 store 的 **ToDisk**，保存状态数据到文件中
7. 更新维护在 sandbox 中的容器信息
8. 调用 **updateResources**，热更新 VM 的资源规格
9. 调用 resCtrl 的 **resourceControllerUpdate**，更新 sandbox 的 cgroup
10. 调用 store 的 **ToDisk**，保存状态数据到文件中

## DeleteContainer

**删除 sandbox 中的指定容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1431)

1. 获取 sandbox 中的指定容器，并调用 VCContainer 的 **delete**，删除维护的容器信息
2. 调用 resCtrl 的 **resourceControllerUpdate**，更新 sandbox 的 cgroup
3. 调用 store 的 **ToDisk**，保存状态数据到文件中

## StartContainer

**启动 sandbox 中的 pod_container 容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1364)

1. 获取 sandbox 中的指定容器，并调用 VCContainer 的 **start**，启动容器
2. 调用 store 的 **ToDisk**，保存状态数据到文件中
3. 调用 **updateResources**，热更新 VM 的资源信息

## StopContainer

**关停 sandbox 中的指定容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1392)

1. 获取 sandbox 中的指定容器，并调用 VCContainer 的 **stop**，关停容器并清理相关资源
2. 调用 store 的 **ToDisk**，保存状态数据到文件中

## KillContainer

**杀死 sandbox 中的指定容器进程**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1411)

1. 获取 sandbox 中的指定容器，并调用 VCContainer 的 **signalProcess**，发送 kill 信号（理论上是这样，然而并未有真实调用）

## StatusContainer

**获取 sandbox 中指定容器的状态**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1467)

1. 获取 sandbox 中的指定容器，获取其状态信息（例如：ID、rootfs、状态、启动时间、annotation、PID 等）

## StatsContainer

**获取 sandbox 中指定容器的统计信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1532)

1. 获取 sandbox 中的指定容器，校验其状态是否为 running
2. 调用 agent 的 **statsContainer**，获取容器状态信息

## PauseContainer

**暂停 sandbox 中的指定容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1576)

1. 获取 sandbox 中的指定容器，校验其状态是否为 running
2. 调用 agent 的 **pauseContainer**，暂停容器
3. 设置容器状态为 paused
4. 调用 store 的 **ToDisk**，保存状态数据到文件中

## ResumeContainer

**恢复 sandbox 中的指定容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1595)

1. 获取 sandbox 中的指定容器，校验其状态是否为 running
2. 调用 agent 的 **resumeContainer**，恢复容器
3. 设置容器状态为 running
4. 调用 store 的 **ToDisk**，保存状态数据到文件中

## EnterContainer

**在 sandbox 中的指定容器中执行命令**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1493)

1. 获取 sandbox 中的指定容器，校验其状态是否为 running
2. 调用 agent 的 **exec**，进入容器执行指定命令

## UpdateContainer

**更新 sandbox 中的指定容器资源规格**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1510)

1. 获取 sandbox 中的指定容器，校验其状态是否为 running
2. 调用 **updateResources**，热更新 VM 的资源规格
3. 调用 agent 的 **updateContainer**，更新容器
4. 调用 resCtrl 的 **resourceControllerUpdate**，更新 sandbox 的 cgroup
5. 调用 store 的 **ToDisk**，保存状态数据到文件中

## WaitProcess

**等待 sandbox 中的指定容器进程返回退出码**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L377)

1. 校验 sandbox 的状态是否为 running
2. 获取 sandbox 中的指定容器，校验其状态是否为 ready 或 running
3. 调用 agent 的 **waitProcess**，等待进程返回退出码

## SignalProcess

**向 sandbox 中的指定容器进程发送指定信号**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L392)

1. 校验 sandbox 的状态是否为 running
2. 获取 sandbox 中的指定容器，调用 VCContainer 的 **signalProcess**，向容器进程发送指定信号

## WinsizeProcess

**设置 sandbox 中的指定容器 tty 大小**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L406)

1. 校验 sandbox 的状态是否为 running
2. 获取 sandbox 中的指定容器，校验其状态是否为 ready 或 running
3. 调用 agent 的 **winsizeProcess**，设置进程的 tty 大小

## IOStream

**获取 sandbox 中的指定容器 IO 流**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L420)

1. 校验 sandbox 的状态是否为 running
2. 获取 sandbox 中的指定容器，校验其状态是否为 ready 或 running
3. 初始化 IO 流，返回其 stdin、stdout 和 stderr

## AddDevice

**向 sandbox 中添加设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1932)

1. 调用 devManager 的 **NewDevice**，初始化对应类型的设备
2. 调用 devManager 的 **AttachDevice**，添加该设备

## AddInterface

**向 sandbox 中添加网卡**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L918)

1. 将 rpc 请求体转换成网卡信息结构
2. 调用 network 的 **AddEndpoints**，添加该网卡
3. 获取网卡 PCI 地址，调用 agent 的 **updateInterface**，更新网卡信息
4. 调用 store 的 **ToDisk**，保存状态数据到文件中

## RemoveInterface

**移除 sandbox 中的指定网卡**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L956)

1. 调用 network 的 **RemoveEndpoints**，移除指定网卡
2. 调用 store 的 **ToDisk**，保存状态数据到文件中

## ListInterfaces

**获取 sandbox 的所有网卡配置**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L975)

1. 调用 agent 的 **listInterfaces**，获取所有网卡配置

## UpdateRoutes

**更新 sandbox 的路由表**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L980)

1. 调用 agent 的 **updateRoutes**，更新路由表

## ListRoutes

**获取 sandbox 的所有路由配置**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L985)

1. 调用 agent 的 **listRoutes**，获取所有路由配置

## GetOOMEvent

**获取 sandbox 的 OOM 事件信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2320)

1. 调用 agent 的 **getOOMEvent**，获取 OOM 事件信息

## GetHypervisorPid

**获取 sandbox 的 hypervisor PID**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L292)

1. 调用 hypervisor 的 **GetPids**，获取所有 PID 列表
2. 返回 PID 列表首位（因为首位是 hypervisor PID，次位为 virtiofsd PID）

## UpdateRuntimeMetrics

**更新 sandbox 的 hypervisor 相关指标**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox_metrics.go#L134)

1. 调用 hypervisor 的 **GetPids**，获取 hypervisor 的 PID
2. 获取 /proc/\<hypervisorPID\>/fd 目录下的文件数量（进程打开的所有文件描述符，这些文件描述符是指向实际文件的一个符号链接，例如 0 表示 stdin、1 表示 stdout、2 表示 stderr 等等），上报 kata_hypervisor_fds 指标
3. 解析 /proc/\<hypervisorPID\>/net/dev 文件内容（网络设备状态信息，例如 eth0、lo、tap0_kata 和 tunl0 接受和发送的数据包、错误和冲突的数量以及其他基本统计，参考 ifconfig 命令结果），上报 kata_hypervisor_netdev 指标
4. 解析 /proc/\<hypervisorPID\>/stat 文件内容（进程的状态信息，参考 ps 命令结果），上报 kata_hypervisor_proc_stat 指标
5. 解析 /proc/\<hypervisorPID\>/status 文件内容（进程的状态信息，相较于 /proc/\<hypervisorPID\>/stat 更易读），上报 kata_hypervisor_proc_status 指标
6. 解析 /proc/\<hypervisorPID\>/io 文件内容（进程的 IO 统计信息），上报 kata_hypervisor_io 指标
7. 调用 hypervisor 的 **GetVirtioFsPid**，获取 virtiofsd 的 PID
8. 获取 /proc/\<virtiofsdPID\>/fd 目录下的文件数量，上报 kata_virtiofsd_fds 指标
9. 解析 /proc/\<virtiofsdPID\>/stat 文件内容，上报 kata_virtiofsd_proc_stat 指标
10. 解析 /proc/\<hypervisorPID\>/status 文件内容，上报 kata_virtiofsd_proc_status 指标
11. 解析 /proc/\<hypervisorPID\>/io 文件内容，上报 kata_virtiofsd_io 指标

## GetAgentMetrics

**获取 sandbox 的 agent 相关指标**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox_metrics.go#L221)

1. 调用 agent 的 **getAgentMetrics**，获取 agent 的指标信息

## GetAgentURL

**获取 sandbox 的 agent URI 信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2324)

1. 调用 agent 的 **getAgentURl**，获取 URI 信息

## GuestVolumeStats

**获取 sandbox 中的指定挂载卷信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2339)

1. 校验卷路径是否存在
2. 遍历所有容器的所有挂载点，获取挂载源为指定卷目录的 sandbox 内的挂载点
3. 调用 agent 的 **getGuestVolumeStats**，获取卷信息

## ResizeGuestVolume

**调整 sandbox 中的指定挂载卷大小**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2348)

1. 校验卷路径是否存在
2. 遍历所有容器的所有挂载点，获取挂载源为指定卷目录的 sandbox 内的挂载点
3. 调用 agent 的 **resizeGuestVolume**，调整卷大小

## GetIPTables

**获取 sandbox 的 iptables 信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox_metrics.go#L2329)

1. 调用 agent 的 **getIPTables**，获取 iptables 信息

## SetIPTables

**设置 sandbox 的 iptables 信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox_metrics.go#L2334)

1. 调用 agent 的 **setIPTables**，设置 iptables 信息

****

# VCContainer

*<u>src/runtime/virtcontainers/interfaces.go</u>*

virtcontainers 库中用于管理容器的模块。

```go
type Container struct {
	ctx context.Context

	config  *ContainerConfig
	sandbox *Sandbox

	id            string
	sandboxID     string
	containerPath string
	rootfsSuffix  string

	mounts []Mount

	devices []ContainerDevice

	state types.ContainerState

	process Process

	rootFs RootFs

	systemMountsInfo SystemMountsInfo
}
```

**工厂函数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/container.go#L714)

1. 检验 containerConfig 配置是否合法（containerConfig 取自于 sandboxConfig 中的相关配置）
3. 校验 annotation 中 SWAP 资源声明是否合法（io.katacontainers.container.resource.swappiness 必须小于 200），并透传设置（区别于 CPU 和内存等资源，SWAP 无法通过 spec.Containers.Resources 的方式声明，而需要通过 annotation 声明）
4. 调用 store 的 **FromDisk**，获取 sandbox 和容器的状态信息。如果成功获取则表明不是新创建的容器，无需后续动作，仅用于状态更新维护，直接返回容器实例即可
5. 如果挂载点是块设备，则需要交由 device manager 维护
   1. 根据 hypervisor 配置项是否允许使用块设备，agent 是否具备使用块设备能力以及 hypervisor 是否允许块设备热插拔判断是否当前支持块设备
   2. 遍历容器中的所有挂载信息，执行后续步骤
      1. 如果 mount.BlockDeviceID 已经存在，则表明已经有一个设备和挂载点相关联，因此不需要创建设备，跳过即可
      2. 如果挂载类型不是 bind，跳过即可
      3. 获取 /run/kata-containers/shared/direct-volumes/\<base64 mount.Source\>/mountInfo.json 文件，如果存在，表明当前挂载设备需要以直通卷的方式处理
         *mount.Source 格式为 /var/lib/kubelet/pods/\<podUID\>/volumes/kubernetes.io~csi/\<pvName\>/mount<br>mountInfo.json 中 device 字段的格式为 /dev/sda（取决于 host 上的具体设备）*
      4. 创建 /run/kata-containers/shared/direct-volumes/\<base64 mount.Source\>/\<sandboxID\> 文件
      5. 替换 mount.Source、mount.Type、mount.Options、mount.FSGroup 和 mount.FSGroupChangePolicy 为 mountInfo.json 的对应字段
   3. 针对挂载信息中的块设备类型（传统块设备和 PMEM 设备），调用 devManager 的  **NewDevice**，初始化设备，回写 mount.BlockDeviceID 字段信息
6. 过滤容器中的 CDROM 和 floppy 类型的设备，回写至 container.devices 中

VCContainer 中声明的 **GetAnnotations**、**GetPid**、**GetToken**、**ID**、**Sandbox** 以及 **Process** 均为参数获取与赋值，无复杂逻辑，不作详述。

## start

**启动容器**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/container.go#L940)

1. 校验 sandbox 状态是否为 running
2. 校验容器状态是否为 ready 或 stopped
3. 调用 agent 的 **startContainer**，启动容器
4. 如果启动失败，则调用 **stop**，执行回滚操作；否则，设置容器状态为 running，并调用 store 的 **ToDisk**，保存 sandbox 和容器的状态数据到文件中

## stop

**关停容器，并清理相关资源**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/container.go#L966)

1. 如果容器状态已经为 stopped，则不做任何操作
2. 校验容器状态是否为 ready、running 或 paused
3. 调用 **signalProcess**，向 sandbox 中的容器进程发送 kill 信号
4. 调用 agent 的 **waitProcess**，确保容器进程已退出
5. 调用 agent 的 **stopContainer**，关停容器
6. 针对容器中每一个挂载信息，调用 fsShare 的 **UnshareFile**，移除 host 侧的 sandbox 共享文件
7. 调用 fsShare 的 **UnshareRootFilesystem**，移除 sandbox 中的容器 rootfs 共享挂载
8. 调用 devManager 的 **DetachDevice** 和 **RemoveDevice**，detach 并移除 sandbox 中的所有设备（含块设备）
9. 如果容器的 rootfs 是块设备，则调用 devManager 的 **DetachDevice** 和 **RemoveDevice**，detach 并移除容器的 rootfs 块设备
10. 设置容器状态为 stopped，并调用 store 的 **ToDisk**，保存 sandbox 和容器的状态数据到文件中

## delete

**删除容器信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/container.go#L896)

1. 校验容器状态是否为 ready 或 stopped
2. 删除维护在 sandbox 中的容器信息
3. 调用 store 的 **ToDisk**，保存 sandbox 和容器的状态数据到文件中

## signalProcess

**向容器进程发送指定信号**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/container.go#L1074)

1. 校验 sandbox 状态是否为 ready 或者 running
2. 校验容器状态是否为 ready、running 或者 paused
3. 调用 agent 的 **signalProcess**，向 sandbox 中的指定容器进程发送信号（由于 Containerd 和 CRIO 并不会处理 `ESRCH: No such process` 错误，因此 Kata runtime 在这里做了特殊操作，针对此报错仅输出 warning 日志，不作返回）
