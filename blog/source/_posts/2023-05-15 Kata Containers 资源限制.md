---
title: "「 Kata Containers 」资源限制"
excerpt: "Kata Containers 在 Kubernetes 集群场景中资源限制与实践验证"
cover: https://picsum.photos/0?sig=20230515
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-05-15
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

# cgroup 管理

Kata Containers 目前支持 cgroups v1 和 v2。

Kata Containers 中，工作负载是在 VM 中运行，VM 由运行在 host 上的 VMM（virtual machine monitor）管理。因此，Kata Containers 运行在两层 cgroup 之上：一层为工作负载所在的 guest，另一层为运行 VMM 和相关线程的 host。

容器 cgroup 路径的配置是在 [OCI runtime spec](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md) 中声明的 cgroupsPath 字段，可用于控制容器的 cgroup 层次结构以及在容器中运行的进程。在 Kubernetes 场景中，Pod 的 cgroup 是由 Kubelet 管理，而容器的 cgroup 是由运行时管理。 Kubelet 将根据容器资源需求调整 Pod 的 cgroup 大小，其中包含 Pod spec.Overhead 中声明的资源。

Kata Containers 的设计为 sandbox 引入了不可忽略的资源开销。通常，与基于进程级别的容器运行时相比，Kata shim（即 containerd-shim-kata-v2）会调用底层 VMM 创建额外的线程，例如半虚拟化 I/O 后端、VMM 实例以及 Kata shim 进程。这些 host 进程消耗的内存和 CPU 资源是不与容器中的工作负载直接相关，而是属于引入 sandbox 带来的额外开销。为了使 Kata 工作负载在不显着降低性能的情况下运行，必须相应地配置其 sandbox 的开销。因此，可能有两种情况：

- 上层编排器在调整 Pod cgroup 大小时考虑运行 sandbox 的额外开销。例如，Kubernetes 的 Pod Overhead 特性允许编排器将 sandbox 的额外开销计入其所有容器资源的总和中。在这种情况下，Kata 创建的所有进程都将在 Pod 的 cgroup 约束和限制下运行
- 上层编排器不考虑 sandbox 的额外开销，因此 Pod 的 cgroup 大小可能无法满足运行 Kata 创建的所有进程。在这种情况下，将所有 Kata 相关进程附加到 Pod 的 cgroup 中可能会导致不可忽略的工作负载性能下降。因此，Kata Containers 会将除 vCPU 线程之外的所有进程移动到名为 /kata_overhead 下的子 cgroup 中。 Kata 运行时不会对该 cgroup 作出任何约束或限制，而由集群管理员选择性设置

Kata Containers 并不会动态检测这两种情况，而是通过配置文件中的 [runtime].sandbox_cgroup_only 选项决定的。

**cgroup 种类**

- Pod cgroup

  位于 /kubepods 层级下的子 cgroup，命名为 /kubepods/\<PodUID\>，由 Kubelet 管理

  - sandbox cgroup

    位于 /kubepods/\<PodUID\> 层级下的子 cgroup，命名为 /kata\_\<sandboxID\>，由运行时管理

- overhead cgroup

  位于 /kata_overhead 层级下的子 cgroup，命名为 /kata_overhead/\<sandboxID\>，由运行时管理

## sandbox_cgroup_only = true

sandbox_cgroup_only 设置为 true 意味着 Kubelet 在设置 Pod cgroup 的大小时会将 Pod 的额外开销考虑在内（Kubernetes 1.16 起，借助 Pod Overhead 特性）。相对而言，这种方式较为推荐，Kata Containers 所有相关进程都可以简单地放置在给定的 cgroup 路径中。

```shell
┌─────────────────────────────────────────┐
│  ┌──────────────────────────────────┐   │
│  │ ┌─────────────────────────────┐  │   │
│  │ │ ┌─────────────────────┐     │  │   │
│  │ │ │ vCPU threads        │     │  │   │
│  │ │ │ I/O threads         │     │  │   │
│  │ │ │ VMM                 │     │  │   │
│  │ │ │ Kata Shim           │     │  │   │
│  │ │ │                     │     │  │   │
│  │ │ │ /kata_<sandboxID>   │     │  │   │
│  │ │ └─────────────────────┘     │  │   │
│  │ │Pod                          │  │   │
│  │ └─────────────────────────────┘  │   │
│  │/kubepods                         │   │
│  └──────────────────────────────────┘   │
│ Node                                    │
└─────────────────────────────────────────┘
```

### 实现细节

当启用 sandbox_cgroup_only 时，Kata shim 将在 Pod cgroup 下创建一个名为 /kata\_<sandboxID\> 的子 cgroup，即 sandbox cgroup。大多数情况下，sandbox cgroup 不作单独约束和限制，而是自继承父 cgroup。cpuset 和 devices cgroup 子系统除外，它们是由 Kata shim 管理。

```shell
# ======= host =======
# Kata 的 cgroup 层级与限制
└── /kubepods/pod08ae4074-5398-439b-93ae-a63035cbd3ae
	├── tasks				(空)
    ├── cpu.cfs_period_us	(-> 100000)
    ├── cpu.cfs_quota_us	(-> 100000)
    └── kata_dc5e4c1588ba3cdeb4fe1dffcb2420997408f42ad2545ddc792724b3bbfb7654	(infra 容器)
    	├── tasks				(containerd-shim-kata-v2、virtiofsd、vhost 和 qemu-system 虚拟化进程)
    	├── cpu.cfs_period_us	(-> 100000)
    	└── cpu.cfs_quota_us	(-> -1)

# runC 的 cgroup 层级与限制
└── /kubepods/pod505eb17b-78d4-4dce-bfb2-60085f629344
	├── tasks				(空)
    ├── cpu.cfs_period_us	(-> 100000)
    ├── cpu.cfs_quota_us	(-> 100000)
    ├── 499316b3661bc989f0999dd51901d2afaad0dda0aa614a2ebcd39f2517e7c56b	(业务容器)
    |	├── tasks				(业务进程)
    | 	├── cpu.cfs_period_us	(-> 100000)
    |	└── cpu.cfs_quota_us	(-> 100000)
    └──	fa6545c433f02a1c712db11cb58bb100a013f9622d725c6a41c60500c20031c5	(infra 容器)
    	├── tasks				(pause 进程)
    	├── cpu.cfs_period_us	(-> 100000)
    	└── cpu.cfs_quota_us	(-> -1)
    	
# ======= guest =======
# Kata VM 中的 cgroup 层级与限制
└── /kubepods/pod08ae4074-5398-439b-93ae-a63035cbd3ae
	├── tasks				(空)
    ├── cpu.cfs_period_us	(-> 100000)
    ├── cpu.cfs_quota_us	(-> -1)
    ├── 8d0a3396afc32d47276b4b25e76e23cdf80dfc51ca980846fb3c847effbe84f9	(业务容器)
    |	├── tasks				(业务进程)
    |	├── cpu.cfs_period_us	(-> 100000)
    |	└── cpu.cfs_quota_us	(-> 100000)
    └──	dc5e4c1588ba3cdeb4fe1dffcb2420997408f42ad2545ddc792724b3bbfb7654	(infra 容器)
    	├── tasks				(pause 进程)
    	├── cpu.cfs_period_us	(-> 100000)
    	└── cpu.cfs_quota_us	(-> -1)
```

创建 sandbox cgroup 之后，Kata shim 会在 VM 启动之前将其自身加入到该 cgroup 中。因此，随后由 Kata shim 创建的所有进程（VMM 本身，以及所有 vCPU 和 I/O 相关线程）都将受 sandbox cgroup 约束。

### sandbox cgroup 的价值

为什么不直接将 sandbox、shim 等 Kata 相关进程添加到 Pod cgroup？

Kata shim 实现了 per-sandbox cgroup （即每一个 sandbox 都有一个对应的 sandbox cgroup）来支持 Docker 场景。尽管 Docker 没有 Pod 的概念，但 Kata Containers 仍然创建了一个 sandbox 来支持 Docker 实现的无 Pod、单一容器用例（即 single_container）。为了简化使用，Kata Containers 选择一个独立的 sandbox cgroup，而不是构建容器和 sandbox 之间的 cgroup 映射关系。

### 优点

将 Kata Containers 所有进程放置在适当大小的 Pod cgroup 中可以简化控制流程，有助于收集准确的指标统计数据并防止 Kata 工作负载产生近邻干扰（noisy neighbor），具体为：

**Pod 资源统计**

如果想获取 Kata 容器在 host 上的资源使用情况，可以从 Pod cgroup 中获取指标统计信息。其中，cgroup 的统计数据包括 Kata 的额外开销，提供了在 Pod 级别和容器级别收集使用静态信息的能力。

**更好的 host 资源隔离**

Kata 运行时会将所有 Kata 进程放在 Pod cgroup 中，所以对 Pod cgroup 设置的资源限制将作用于 host 中属于 Kata sandbox 的所有进程（例如 qemu-system、virtiofsd 等），从而可以改善 host 中的隔离，防止 Kata 产生近邻干扰（noisy neighbor）。

## sandbox_cgroup_only = false

如果提供给 Kata 容器的 Pod cgroup 大小不合适，Kata 组件将消耗实际容器工作负载期望使用的资源，导致不稳定和性能下降。

为避免这种情况，Kata Containers 创建了一个名为 /kata_overhead 的 cgroup，即 overhead cgroup，并将所有与工作负载无关的进程（除 vCPU 线程外的任何进程）移至其中。

Kata Containers 不对 overhead cgroup 作任何约束或限制，因此可以

- 预先创建并规划 overhead cgroup 的限制条件，Kata Containers 不会再额外创建，而是将所有与工作负载无关的进程移动到其中
- 让 Kata Containers 创建 overhead cgroup，让其不受约束或事后调整大小

```shell
┌────────────────────────────────────────────────────────────────────┐
│  ┌─────────────────────────────┐    ┌───────────────────────────┐  │
│  │   ┌─────────────────────────┼────┼─────────────────────────┐ │  │
│  │   │ ┌─────────────────────┐ │    │ ┌─────────────────────┐ │ │  │
│  │   │ │  vCPU threads       │ │    │ │  VMM                │ │ │  │
│  │   │ │                     │ │    │ │  I/O threads        │ │ │  │
│  │   │ │                     │ │    │ │  Kata Shim          │ │ │  │
│  │   │ │                     │ │    │ │                     │ │ │  │
│  │   │ │ /kata_<sandboxID>   │ │    │ │ /<sandboxID>        │ │ │  │
│  │   │ └─────────────────────┘ │    │ └─────────────────────┘ │ │  │
│  │   │  Pod                    │    │                         │ │  │
│  │   └─────────────────────────┼────┼─────────────────────────┘ │  │
│  │ /kubepods                   │    │ /kata_overhead            │  │
│  └─────────────────────────────┘    └───────────────────────────┘  │
│ Node                                                               │
└────────────────────────────────────────────────────────────────────┘
```

### 实现细节

当 sandbox_cgroup_only 被禁用时，Kata shim 将在 Pod cgroup 下创建 sandbox cgroup 子 cgroup，并在 overhead cgroup 下创建一个名为 /\<sandboxID\> 的子 cgroup。

TODO

与启用 sandbox_cgroup_only 时不同，Kata shim 将其自身加入到 overhead cgroup 中，然后将 vCPU 线程移动到 sandbox cgroup 中。除 vCPU 线程外的其他 Kata 进程和线程都将在 overhead cgroup 下运行。

在禁用 sandbox_cgroup_only 的情况下，Kata Containers 假定 Pod cgroup 的大小仅能满足容器工作负载进程。VMM 创建的 vCPU 线程是唯一在 Pod cgroup 下运行的 Kata 相关线程，降低了 VMM、Kata shim 和 I/O 线程 OOM 的风险。

### 优缺点

在不受约束的 overhead cgroup 下运行所有非 vCPU 线程可能会导致工作负载潜在地消耗大量 host 资源。

另一方面，由于 overhead cgroup 的专用性，在 overhead cgroup 下运行所有非 vCPU 线程可以获取 Kata Container Pod 额外开销的准确指标，以此更合理的调整 overhead cgroup 大小和约束。
