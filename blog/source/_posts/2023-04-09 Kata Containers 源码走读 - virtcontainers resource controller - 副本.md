---
title: "「 Kata Containers 」源码走读 — virtcontainers/resource controller"
excerpt: "virtcontainers 中与 ResourceController 等资源限制相关的流程梳理"
cover: https://picsum.photos/0?sig=20230409
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2023-04-09
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

# ResourceController

*<u>src/runtime/pkg/resourcecontrol/controller.go</u>*

ResourceController 在 Linux 上的实现为 LinuxCgroup，而 LinuxCgroup 具体体现为两种：sandboxController 和 overheadController：

- 当 [runtime].sandbox_cgroup_only 开启时，顾名思义仅有 sandboxController，用于管理 Pod 所有的线程资源
- 当 [runtime].sandbox_cgroup_only 未开启时，资源分为两类，其中 vCPU 线程资源会由 sandboxController 管理，其余资源由 overheadController 管理

*具体执行标准参考 OCI runtime-spec：https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md。*

```go
type LinuxCgroup struct {
	sync.Mutex
    
	// cgroup 实现，其中类型包括 legacy、hybrid 和 unified，用于区分 cgroups v1 和 v2
	cgroup  interface{}

	// cgroup 路径
	path    string

	// 待限制的 CPU
	cpusets *specs.LinuxCPU
    
	// 待限制的 sandbox 设备
	// 除了创建时指定的 sandbox 设备，还会追加以下设备：
	//   默认设备：/dev/null、/dev/random、/dev/full、/dev/tty、/dev/zero、/dev/urandom 和 /dev/console
	//   虚拟化设备：/dev/kvm、/dev/vhost-net、/dev/vfio/vfio 和 /dev/vhost-vsock
	//   wildcard 设备（通过手动指定 major、minor、access 和 type 属性构造的设备）：tuntap、/dev/pts 等
	devices []specs.LinuxDeviceCgroup
}
```

LinuxCgroup 的实现方式本质上是封装了 `github.com/containerd/cgroups ` 库，用于处理 cgroup 资源。因此，**Type**、**ID**、**Parent**、**Delete**、**Stat**、**AddProcess**、**AddThread**、**Update**、**MoveTo**、**AddDevice**、**RemoveDevice** 和 **UpdateCpuSet** 均为该库针对 cgroup v1 和 v2 不同版本下统一入口的二次封装。*该库不支持针对 systemd 创建具有 v1 和 v2 cgroup 的 scope，因此这部分是直接与 systemd 交互创建 cgroup，然后使用 containerd 的 api 加载它。 添加运行时进程，无需调用 setupCgroups*

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
