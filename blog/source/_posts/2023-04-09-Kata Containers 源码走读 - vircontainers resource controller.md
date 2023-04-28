---
title: "「 Kata Containers 」 4.4.4 源码走读 — virtcontainers resource controller"
excerpt: "virtcontainers 库中 resource controller 模块源码走读"
cover: https://unsplash.it/0/0?random&sig=1234
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-04-09
toc: true
categories:
  - Kata Containers
  - virtcontainers
tag:
  - Cloud Native
  - Container Runtime
  - Kata Containers
---

<div align=center><img width="300" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> Based on **v3.0.0**

# ResourceController

*<u>src/runtime/pkg/resourcecontrol/controller.go</u>*

ResourceController 在 Linux 上的实现为 LinuxCgroup，而 LinuxCgroup 具体体现为两种：sandboxController 和 overheadController。当 [runtime].sandbox_cgroup_only 开启时，顾名思义仅有 sandboxController，用于管理 Pod 所有的线程资源；当未开启时，资源分为两类，其中 vCPU 线程资源会由 sandboxController 管理，其余资源由 overheadController 管理。

*具体执行标准参考 OCI runtime-spec：https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md*

```go
type LinuxCgroup struct {
	cgroup  interface{}
	path    string
	cpusets *specs.LinuxCPU
	devices []specs.LinuxDeviceCgroup

	sync.Mutex
}
```

**工厂函数**

工厂函数有以下三种实现方式：

**NewResourceController**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/resourcecontrol/cgroups.go#L133)

*用于构建 overheadController 或仅 sandboxController 存在的场景下，用于构建 sandboxController*

1. 简单调用 `github.com/containerd/cgroups`，根据 cgroup 的类型，创建对应版本的 cgroup，初始化 LinuxCgroup

**NewSandboxResourceController**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/resourcecontrol/cgroups.go#L168)

*overheadController 和 sandboxController 同时存在的场景下，用于构建 sandboxController*

1. 准备待纳管的 sandbox 资源信息<br>*除了创建时指定的 sandbox 设备之外，还会追加以下设备：*<br>*默认设备：/dev/null、/dev/random、/dev/full、/dev/tty、/dev/zero、/dev/urandom 和 /dev/console*<br>*虚拟化设备：/dev/kvm、/dev/vhost-net、/dev/vfio/vfio 和 /dev/vhost-vsock*<br>*wildcard 设备（通过手动指定 major、minor、access 和 type 属性构造的设备）：tuntap、/dev/pts 等*
2. 如果 cgroup 不是由 systemd 纳管（通过 cgroupPath 格式判断）或者 [runtime].sandbox_cgroup_only 为 false，则以第一种工厂函数流程处理；否则，调用 systemd 创建对应版本的 cgroup，加载 cgroup，追加 sandbox 资源信息，初始化 LinuxCgroup<br>*github.com/containerd/cgroups 不支持针对 systemd 创建具有 v1 和 v2 cgroup 的 scope，因此直接与 systemd 交互创建 cgroup，然后使用 containerd 的 api 加载它。 添加运行时进程，无需调用 setupCgroups*

**LoadResourceController**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/resourcecontrol/cgroups.go#L229)

*根据 cgroupsPath，回溯得到对应的 sandboxController 或 overheadController*

1. 解析 cgroupsPath 的路径，得到现存的 cgroup 信息，初始化 LinuxCgroup

LinuxCgroup 的实现方式本质上是封装了 `github.com/containerd/cgroups ` 库，用于处理 cgroup 资源。因此，**Type**、**ID**、**Parent**、**Delete**、**Stat**、**AddProcess**、**AddThread**、**Update**、**MoveTo**、**AddDevice**、**RemoveDevice** 和 **UpdateCpuSet** 均为该库针对 cgroup v1 和 v2 不同版本下统一入口的二次封装。

*此外，以下的函数声明并非 ResourceController 的接口声明，而是 VCSandbox 的扩展封装，为了便于理解，将其归类至 ResourceController 下。*

## setupResourceController

**配置 resourceController**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2275)

1. 调用 sandboxController 或 overheadController（取决于 sandbox 资源是否分开管理，即 [runtime].sandbox_cgroup_only 的配置）的 **AddProcess**，将当前进程 ID 加入到 cgroup 中管理<br>*确保运行时的任何子进程（即服务于 Kata Pod 的所有进程）都将存在于 resourceController 中，且如果有 overheadController 则由 overheadController 管理此类进程以及子进程*

## resourceControllerUpdate

**更新 resourceController 以及 cgroup 信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2191)

1. 聚合 sandbox 中所有容器的 CPUSet 和 MEMSet 信息，调用 sandboxController 的 **UpdateCpuSet**，更新到 cgroup 中
2. 如果 sandbox 的资源分开管理（即存在 overheadController），则调用 hypervisor 的 **GetThreadIDs**，获取 vCPU 线程，并调用 sandboxController 的 **AddThread**，将 vCPU 线程的加入到 cgroup 中管理<br>*因为当有 overheadController 时，意味着会产生新的 vCPU 线程会作为 hypervisor 的子线程，所以需要并入统一的 cgroup 中管理*

## resourceControllerDelete

**删除 resourceController 以及 cgroup 信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L2216)

1. 调用 **LoadResourceController**，根据 sandboxController 的 cgroupPath 获取 sandboxController
2. 调用 sandboxController 的 **Parent**，获取父级信息；调用 sandboxController 的 **MoveTo**，将其管理的进程移至父级；并调用 sandboxController 的 **Delete**，删除 sandboxController 的 cgroup
3. 如果 sandbox 的资源分开管理（即存在 overheadController），则执行同样的操作
