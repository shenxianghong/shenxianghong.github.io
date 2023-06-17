---
title: "「 Kata Containers 」架构演进"
excerpt: "Kata Containers 3.x 较 2.x 版本架构发展演进与设计初衷"
cover: https://picsum.photos/0?sig=20221114
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2022-11-14
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

# 概览

在云原生场景中，对容器启动速度、资源消耗、稳定性和安全性的需求不断增加，目前 Kata Containers 运行时相对于其他运行时面临挑战。为了解决这一点，社区提出了一个可靠的、经过现场测试的、安全的 Rust 版本的 kata-runtime。

## 特性

- Turn key solution with builtin `Dragonball` Sandbox
- 异步 I/O 以减少资源消耗
- 用于多种服务、运行时和 hypervisor 的可扩展框架
- sandbox 和容器相关联资源的生命周期管理

## 选择 Rust 的理由

之所以选择 Rust，是因为它被设计为一种注重效率的系统语言。与 Go 相比，Rust 进行了各种设计权衡以获得良好的执行性能，其创新技术与 C 或 C++ 相比，提供了针对常见内存错误（缓冲区溢出、无效指针、范围错误）的合理保护、错误检查（确保错误得到处理）、线程安全、资源所有权等。

当 Kata agent 用 Rust 重写时，这些优点得到了验证。基于 Rust 的实现显着减少了内存使用量。

# 设计

## 架构

<div align=center><img width="800" style="border: 0px" src="/gallery/kata-containers/architecture.png"></div>

## 内置的 VMM

### 当前 Kata 2.x 架构

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/not_built_in_vmm.png"></div>

如图所示，runtime 和 VMM 是独立的进程。runtime 进程 fork 出 VMM 进程并通过 RPC 进程间通信。通常，进程间交互比进程内交互开销更大且效率更低。同时，还要考虑资源维护成本。例如，在异常情况下进行资源回收时，任何进程的异常都必须被其他组件检测到，并触发相应的资源回收进程。如果有额外的过程，恢复变得更加困难。

### 如何支持内置的 VMM

社区提供了 Dragonball Sandbox，通过将 VMM 的功能集成到 Rust 库中来启用内置的 VMM。可以通过使用该库来执行与 VMM 相关的功能。因为 runtime 和 VMM 在同一个进程中，所以在消息处理速度和 API 同步方面有所改善。还可以保证 runtime 和 VMM 生命周期的一致性，减少资源回收和异常处理维护的复杂度，如图所示：

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/built_in_vmm.png"></div>

## 支持异步

### 为什么需要异步

**Async 已经为稳定版 Rust 特性**

> 参考：[Why Async](https://rust-lang.github.io/async-book/01_getting_started/02_why_async.html) 和 [The State of Asynchronous Rust](https://rust-lang.github.io/async-book/01_getting_started/03_state_of_async_rust.html)

- Async 显著降低了 CPU 和内存开销，尤其是对于具有大量 I/O 绑定类型的任务负载
- Async 在 Rust 中是零成本的，这意味着只需应用层面开销，可以不用堆分配和动态调度，大大提高效率

**如果使用 Sync Rust 实现 kata-runtime 可能会出现的几个问题**

- TTRPC 连接线程数太多（TTRPC threads: reaper thread(1) + listener thread(1) + client handler(2)）
- 每个容器有三个 I/O 线程
- 在 Sync 模式下，实现超时机制具有挑战性。比如 TTRPC API 交互中，超时机制很难和 Golang 对齐

**如何支持 Async**

kata-runtime 由 TOKIO_RUNTIME_WORKER_THREADS 控制运行 OS 线程，默认为 2 个线程。TTRPC 和容器相关的线程统一运行在tokio 线程中，Timer，File，Netlink 等相关的依赖调用需要切换到 Async。借助 Async，可以轻松支持无阻塞 I/O 和定时器。目前，仅将 Async 用于 kata-runtime。内置的 VMM 保留了 OS 线程，因为它可以保证线程的可控性。

## 可扩展框架

Kata 3.x runtime 设计了 service、runtime、hypervisor 的扩展，结合配置满足不同场景的需求。目前服务提供注册机制，支持多种服务。服务可以通过消息与 runtime 交互。此外，runtime handler 处理来自服务的消息。为了满足二进制支持多个 runtime 和 hypervisor 的需求，启动必须通过配置获取 runtime handler 类型和 hypervisor 类型。

<div align=center><img width="800" style="border: 0px" src="/gallery/kata-containers/framework.png"></div>

## 资源管理器

在实际使用中，会有各种各样的资源，每个资源都有几个子类型。特别是对于 virtcontainers，资源的每个子类型都有不同的操作。并且可能存在依赖关系，例如 share-fs rootfs 和 share-fs volume 会使用 share-fs 资源将文件共享到 VM。目前，network、share-fs 被视为沙盒资源，rootfs、volume、cgroup 被视为容器资源。此外，社区为每个资源抽象出一个公共接口，并使用子类操作来评估不同子类型之间的差异。

<div align=center><img width="800" style="border: 0px" src="/gallery/kata-containers/resourceManager.png"></div>

# 路线图

- 阶段 1（截至 2022.06）：提供基础特性
- 阶段 2（截至 2022.09）：提供常用特性
- 阶段 3：提供全量特性

| **Class**                  | **Sub-Class**     | **Development Stage** | **Status** |
| -------------------------- | ----------------- | --------------------- | ---------- |
| Service                    | task service      | Stage 1               | ✅          |
|                            | extend service    | Stage 3               | 🚫          |
|                            | image service     | Stage 3               | 🚫          |
| Runtime handler            | Virt-Container    | Stage 1               | ✅          |
| Endpoint                   | VETH Endpoint     | Stage 1               | ✅          |
|                            | Physical Endpoint | Stage 2               | ✅          |
|                            | Tap Endpoint      | Stage 2               | ✅          |
|                            | Tuntap Endpoint   | Stage 2               | ✅          |
|                            | IPVlan Endpoint   | Stage 2               | ✅          |
|                            | MacVlan Endpoint  | Stage 2               | ✅          |
|                            | MACVTAP Endpoint  | Stage 3               | 🚫          |
|                            | VhostUserEndpoint | Stage 3               | 🚫          |
| Network Interworking Model | Tc filter         | Stage 1               | ✅          |
|                            | MacVtap           | Stage 3               | 🚧          |
| Storage                    | Virtio-fs         | Stage 1               | ✅          |
|                            | nydus             | Stage 2               | 🚧          |
|                            | device mapper     | Stage 2               | 🚫          |
| Cgroup V2                  |                   | Stage 2               | 🚧          |
| Hypervisor                 | Dragonball        | Stage 1               | 🚧          |
|                            | QEMU              | Stage 2               | 🚫          |
|                            | ACRN              | Stage 3               | 🚫          |
|                            | Cloud Hypervisor  | Stage 3               | 🚫          |
|                            | Firecracker       | Stage 3               | 🚫          |

# FAQ

- service、message dispatcher 和 runtime handler 都是 Kata 3.x runtime 二进制的一部分吗？

  是的。它们是 Kata 3.x 运行时中的组件。它们将被打包成一个二进制文件

  - service 是一个接口，负责处理任务服务、镜像服务等多种服务
  - message dispatcher 用于匹配来自服务模块的多个请求
  - runtime handler 用于处理对沙箱和容器的操作

- Kata 3.x runtime 二进制的名称是什么？

  由于 containerd-shim-v2-kata 已经被使用了，目前在内部，社区使用 containerd-shim-v2-rund

- Kata 3.x 设计是否与 containerd shimv2 架构兼容？

  是的。它旨在遵循 go 版本 kata 的功能。它实现了 containerd shim v2 接口/协议

- 用户将如何迁移到 Kata 3.x 架构？

  迁移计划将在 Kata 3.x 合并到主分支之前提供

- Dragonball 是不是仅限于自己内置的 VMM？ Dragonball 系统是否可以配置为使用外部 Dragonball VMM/hypervisor 工作？

  Dragonball 可以用作外部管理程序。然而，在这种情况下，稳定性和性能具有挑战性。内置 VMM 可以优化容器开销，易于维护稳定性。runD 是 runC 的 containerd-shim-v2 对应物，可以运行 Pod/容器。 Dragonball 是一种 microvm/VMM，旨在运行容器工作负载。有时将其称为安全沙箱，而不是 microvm/VMM

- QEMU、Cloud Hypervisor 和 Firecracker 支持已在计划中，但如何运作。它们在不同的进程中工作吗？

  是的。它们无法像 VMM 中内置的那样工作

- upcall 是什么？

  upcall 用于热插拔 CPU/内存/MMIO 设备，它解决了两个问题：

  - 避免依赖 PCI/ACPI

  - 避免在 guest 中依赖 udevd 并获得热插拔操作的确定性结果。所以 upcall 是基于 ACPI 的 CPU/内存/设备热插拔的替代方案。如果需要，Kata 社区会与相关社区合作添加对基于 ACPI 的 CPU/内存/设备热插拔的支持

  Dbs-upcall 是 VMM 和 guest 之间基于 vsock 的直接通信工具。 upcall 的服务器端是 guest 内核中的驱动程序（此功能需要内核补丁），一旦内核启动，它将开始为请求提供服务。而客户端在 VMM 中，它将是一个线程，通过 uds 与 VSOCK 通信。通过upcall 直接实现了设备的热插拔，避免了ACPI 的虚拟化，将虚拟机的开销降到最低。通过这种直接的通信手段，可能还有许多其他用途。现目前已经开源：https://github.com/openanolis/dragonball-sandbox/tree/main/crates/dbs-upcall

- 内核补丁适用于 4.19，但它们也适用于 5.15+ 吗？

  向前兼容应该是可以实现的，社区已经将它移植到基于 5.10 的内核

- 这些补丁是否特定于平台，或者它们是否适用于支持 VSOCK 的任何架构？

  它几乎与平台无关，但一些与 CPU 热插拔相关的是与平台相关的

- 是否可以使用 loopback VSOCK 将内核驱动程序替换为 guest 中的用户态守护程序？

  需要为热添加的 CPU/内存/设备创建设备节点，因此用户空间守护进程执行这些任务并不容易

- upcall 允许 VMM 和 guest 之间进行通信的事实表明，此架构可能与 https://github.com/confidential-containers 不兼容，其中 VMM 应该不知道 VM 内部发生的事情

  - TDX 还不支持 CPU/内存热插拔
  - 对于基于 ACPI 的设备热插拔，它依赖于 ACPI DSDT 表，guest 内核将在处理这些热插拔事件时执行 ASL 代码来处理。与 ACPI ASL 方法相比，审计基于 VSOCK 的通信应该更容易

- 单体与内置 VMM 的安全边界是什么？

  它具有虚拟化的安全边界。更多细节将在下一阶段提供
