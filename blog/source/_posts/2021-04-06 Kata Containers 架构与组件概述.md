---
title: "「 Kata Containers 」架构与组件概述"
excerpt: "Kata Containers 2.x 与 1.x 版本架构差异对比与组件功能概述"
cover: https://picsum.photos/0?sig=20210406
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2021-04-06
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **2.1.1**

# 概述

Kata Containers 是一个开源项目，它采用轻量化虚拟机作为容器的隔离来构建一个安全容器运行时，而其虚拟化技术作为容器的二层保护为负载提供了更好的隔离性，这使得 Kata Containers 兼具传统容器的形态和虚拟机的安全性。 早在 2015 年，来自英特尔开源技术中心的工程师就开始探索采用 英特尔® 虚拟技术（英特尔® Virtualization Technology，英特尔® VT）来提高容器的安全隔离性，并以此发起了英特尔® Clear Containers 开源项目，与此同时，来自 Hyper.sh（一家中国的高科技初创公司）的工程师也发起了 runV10 开源项目，这两个项目采用的技术和目的都非常相似，都是为了将容器置于一个安全“沙箱“，以便进一步促进该技术发展和成熟。随后在 2017 年，英特尔和 Hyper.sh 团队将这两个开源项目在社区合并成了一个新的项目 Kata Containers。 传统虚拟机（VMs）可提供硬件隔离，而容器可快速响应，且占用空间相对较小，Kata Containers 将这两者的优势完美结合了起来。 每个容器或 Pod 都在自己单独的虚拟机中启动， 并不再能够访问主机内核，杜绝了恶意代码侵入其它相临容器的可能。由于 Kata Containers 同时具备硬件隔离，也使得互不信任的租户，甚至于生产应用或前生产应用都能够在同一集群内安全运行，从而使得在裸机上运行容器即服务（Containers as a Service, CaaS）成为可能。

# Assets

Kata Containers 创建一个 VM，在其中运行一个或多个容器。需要通过启动 hypervisor 创建虚拟机来实现这一点。hypervisor 需要两个 assets 来完成这项任务：一个 Linux 内核和一个用于引导 VM 的小型根文件系统镜像。

## kernel

guest 内核传递到 hypervisor 用于引导虚拟机。 Kata Containers 中提供了一个对虚机启动时间和内存占用做了高度优化的默认内核，仅提供了容器工作负载所需的必要服务。该内核是基于最新的上游 Linux 内核做的定制化。

## image

hypervisor 使用一个镜像文件，该文件提供了一个最小的根文件系统，供 guest 内核用来启动 VM 和托管 Kata 容器。 Kata Containers 支持基于 initrd 和 rootfs 的最小 guest 镜像（但是，并非所有的 hypervisor 均支持）。默认包同时提供 image 和 initrd，两者都是使用 osbuilder 工具创建的。

### rootfs

默认打包的 rootfs 映像，也称 mini O/S，是一个高度优化的容器引导系统。

使用此镜像启动 Kata 容器的背后流程为：

1. 运行时将启动 hypervisor
2. hypervisor 将使用 guest 内核启动 rootfs 镜像
3. 内核将在 VM 根环境中以 PID 1（systemd）启动 init 守护进程
4. 在 rootfs 上下文中运行的 systemd 将在 VM 的根上下文中启动 kata-agent
5. kata-agent 将创建一个新的容器环境，将其根文件系统设置为用户请求的文件系统（例如 Ubuntu、busybox 等）
6. kata-agent 将在新容器内执行容器启动命令

下表总结了默认的 rootfs，显示了创建的环境、在这些环境中运行的服务（适用于所有平台）以及每个服务使用的根文件系统：

| Process                                                      | Environment  | systemd service? | rootfs                                                       | User accessible                                              | Notes                                      |
| ------------------------------------------------------------ | ------------ | ---------------- | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------ |
| systemd                                                      | VM root      | n/a              | [VM guest image](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/guest-assets.md#guest-image) | [debug console](https://github.com/kata-containers/kata-containers/blob/main/docs/Developer-Guide.md#connect-to-debug-console) | The init daemon, running as PID 1          |
| [Agent](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/README.md#agent) | VM root      | yes              | [VM guest image](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/guest-assets.md#guest-image) | [debug console](https://github.com/kata-containers/kata-containers/blob/main/docs/Developer-Guide.md#connect-to-debug-console) | Runs as a systemd service                  |
| `chronyd`                                                    | VM root      | yes              | [VM guest image](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/guest-assets.md#guest-image) | [debug console](https://github.com/kata-containers/kata-containers/blob/main/docs/Developer-Guide.md#connect-to-debug-console) | Used to synchronise the time with the host |
| container workload (`sh(1)` in [the example](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/example-command.md)) | VM container | no               | User specified (Ubuntu in [the example](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/example-command.md)) | [exec command](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/README.md#exec-command) | Managed by the agent                       |

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/rootfs.png"></div>

```shell
$ ps -ef
>>>
UID        PID  PPID  C STIME TTY          TIME CMD
root         1     0  0 11:53 ?        00:00:00 /sbin/init
root         2     0  0 11:53 ?        00:00:00 [kthreadd]
<skip...>
root        61     1  0 11:53 ?        00:00:00 /usr/bin/kata-agent
root        71    61  0 11:53 ?        00:00:00 /pause
root        73    61  0 11:53 ?        00:00:00 tail -f /dev/null
root        75    61  0 11:55 pts/0    00:00:00 [bash]
root        77    75  0 11:55 pts/0    00:00:00 ps -ef
```

```shell
$ ./usr/bin/kata-agent
>>>
{"msg":"announce","level":"INFO","ts":"2021-07-14T14:56:42.558066805+08:00","source":"agent","pid":"88325","subsystem":"root","name":"kata-agent","version":"0.1.0","api-version":"0.0.1","agent-version":"2.1.0","config":"AgentConfig { debug_console: false, dev_mode: false, log_level: Info, hotplug_timeout: 3s, debug_console_vport: 0, log_vport: 0, container_pipe_size: 0, server_addr: \"vsock://-1:1024\", unified_cgroup_hierarchy: false }","agent-type":"rust","agent-commit":"2.1.0-645e950b8e0e238886adbff695a793126afb584f"}
{"msg":"starting uevents handler","level":"INFO","ts":"2021-07-14T14:56:42.558356885+08:00","name":"kata-agent","source":"agent","subsystem":"uevent","pid":"88325","version":"0.1.0"}
{"msg":"ttRPC server started","level":"INFO","ts":"2021-07-14T14:56:42.558522099+08:00","name":"kata-agent","source":"agent","version":"0.1.0","subsystem":"rpc","pid":"88325","address":"vsock://-1:1024"}
```

### initrd

initrd 镜像是一个压缩的 cpio(1) 归档文件，它是从加载到内存中的 rootfs 创建的，并用作 Linux 启动过程的一部分。在启动过程中，内核将其解压到一个特殊的 tmpfs 挂载实例中，该实例成为初始根文件系统。

使用此镜像启动 Kata 容器的背后流程为：

1. 运行时将启动 hypervisor
2. hypervisor 将使用 guest 内核启动 initrd 镜像
3. 内核将在 VM 根环境中以 PID 1（kata-agent）启动 init 守护进程
4. kata-agent 将创建一个新的容器环境，将其根文件系统设置为用户请求的文件系统（例如 Ubuntu、busybox 等）
5. kata-agent 将在新容器内执行容器启动命令

下表总结了默认的 initrd，显示了创建的环境、在这些环境中运行的服务（适用于所有平台）以及每个服务使用的根文件系统：

| Process                                                      | Environment  | rootfs                                                       | User accessible                                              | Notes                           |
| ------------------------------------------------------------ | ------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------- |
| [Agent](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/README.md#agent) | VM root      | [VM guest image](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/guest-assets.md#guest-image) | [debug console](https://github.com/kata-containers/kata-containers/blob/main/docs/Developer-Guide.md#connect-to-debug-console) | Runs as the init daemon (PID 1) |
| container workload                                           | VM container | User specified (Ubuntu in this example)                      | [exec command](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/README.md#exec-command) | Managed by the agent            |

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/initrd.png"></div>

```shell
$ ps -ef
>>>
UID        PID  PPID  C STIME TTY          TIME CMD
root         1     0  0 06:23 hvc0     00:00:02 /init
root         2     0  0 06:23 ?        00:00:00 [kthreadd]
<skip...>
root        41     1  0 06:23 hvc0     00:00:00 /pause
root        43     1  0 06:23 hvc0     00:00:00 tail -f /dev/null
root        45     1  0 06:24 pts/0    00:00:00 [bash]
root        58    45  0 06:27 pts/0    00:00:00 ps -ef
```

```shell
$ ./sbin/init
>>>
{"msg":"announce","level":"INFO","ts":"2021-07-14T14:58:37.454291069+08:00","source":"agent","pid":"66236","name":"kata-agent","subsystem":"root","version":"0.1.0","api-version":"0.0.1","agent-type":"rust","agent-commit":"2.1.0-645e950b8e0e238886adbff695a793126afb584f","agent-version":"2.1.0","config":"AgentConfig { debug_console: false, dev_mode: false, log_level: Info, hotplug_timeout: 3s, debug_console_vport: 0, log_vport: 0, container_pipe_size: 0, server_addr: \"vsock://-1:1024\", unified_cgroup_hierarchy: false }"}
{"msg":"starting uevents handler","level":"INFO","ts":"2021-07-14T14:58:37.455243334+08:00","version":"0.1.0","subsystem":"uevent","name":"kata-agent","pid":"66236","source":"agent"}
{"msg":"ttRPC server started","level":"INFO","ts":"2021-07-14T14:58:37.455325746+08:00","version":"0.1.0","pid":"66236","subsystem":"rpc","source":"agent","name":"kata-agent","address":"vsock://-1:1024"}
```

**总结**

| Image type                                                   | Default distro                                              | Init daemon                                                  | Reason                               | Notes                      |
| ------------------------------------------------------------ | ----------------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------ | -------------------------- |
| [image](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/background.md#root-filesystem-image) | [Clear Linux](https://clearlinux.org/) (for x86_64 systems) | systemd                                                      | Minimal and highly optimized         | systemd offers flexibility |
| [initrd](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/guest-assets.md#initrd-image) | [Alpine Linux](https://alpinelinux.org/)                    | Kata [agent](https://github.com/kata-containers/kata-containers/blob/main/docs/design/architecture/README.md#agent) (as no systemd support) | Security hardened and tiny C library |                            |

## osbuilder

osbuilder 本身是 Kata Containers 项目中的一个模块，主要负责构建 guest OS 的引导镜像。

Kata Containers 支持两种引导镜像：rootfs 和 initrd。无论哪种方式，默认都会将 kata-agent 编译到镜像中，在对 kata-agent 有定制化需求的场景下，可以手动编译后添加到镜像中。

# Virtualization

Kata 容器是在传统 namespace 隔离之上创建的以硬件虚拟化为基础的第二层隔离。 Kata 启动一个轻量级虚拟机，并使用 guest 中特供的内核来承载容器工作负载。

## 接口映射

Kata 容器的典型部署场景是借助 CRI 实现在 Kubernetes 中进行。在每个节点上，Kubelet 将与 CRI 实现者（如 Containerd 或 CRI-O 等）交互，CRI 实现者将与 Kata Containers（基于 OCI 规范的底层运行时）交互。

<div align=center><img width="700" style="border: 0px" src="/gallery/kata-containers/virtual-map.png"></div>

## hypervisor（VMM）

Kata Containers 本身支持多种 hypervisor 工具，如 QEMU、cloud-hypervisor、firecracker、ACRN 和 Dragonball（Kata 3.0 引入）。

| Hypervisor                                                   | Written in | Architectures   | Type                                                         | Configuration file            |
| ------------------------------------------------------------ | ---------- | --------------- | ------------------------------------------------------------ | ----------------------------- |
| [ACRN](https://projectacrn.org/)                             | C          | x86_64          | Type 1 (bare metal)                                          | configuration-acrn.toml       |
| [Cloud Hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor) | rust       | aarch64, x86_64 | Type 2 ([KVM](https://en.wikipedia.org/wiki/Kernel-based_Virtual_Machine)) | configuration-clh.toml        |
| [Firecracker](https://github.com/firecracker-microvm/firecracker) | rust       | aarch64, x86_64 | Type 2 ([KVM](https://en.wikipedia.org/wiki/Kernel-based_Virtual_Machine)) | configuration-fc.toml         |
| [QEMU](http://www.qemu-project.org/)                         | C          | all             | Type 2 ([KVM](https://en.wikipedia.org/wiki/Kernel-based_Virtual_Machine)) | configuration-qemu.toml       |
| [Dragonball](https://github.com/openanolis/dragonball-sandbox) | rust       | aarch64, x86_64 | Type 2 ([KVM](https://en.wikipedia.org/wiki/Kernel-based_Virtual_Machine)) | configuration-dragonball.toml |

**异同点参考**

| Hypervisor                                                   | Summary                                                   | Features          | Limitations                      | Container Creation speed | Memory density | Use cases                               | Comment                                   |
| ------------------------------------------------------------ | --------------------------------------------------------- | ----------------- | -------------------------------- | ------------------------ | -------------- | --------------------------------------- | ----------------------------------------- |
| [ACRN](https://projectacrn.org/)                             | Safety critical and real-time workloads                   |                   |                                  | excellent                | excellent      | Embedded and IOT systems                | For advanced users                        |
| [Cloud Hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor) | Low latency, small memory footprint, small attack surface | Minimal           |                                  | excellent                | excellent      | High performance modern cloud workloads |                                           |
| [Firecracker](https://github.com/firecracker-microvm/firecracker) | Very slimline                                             | Extremely minimal | Doesn't support all device types | excellent                | excellent      | Serverless / FaaS                       |                                           |
| [QEMU](http://www.qemu-project.org/)                         | Lots of features                                          | Lots              |                                  | good                     | good           | Good option for most users              |                                           |
| [Dragonball](https://github.com/openanolis/dragonball-sandbox) | Built-in VMM, low CPU and memory overhead                 | Minimal           |                                  | excellent                | excellent      | Optimized for most container workloads  | out-of-the-box Kata Containers experience |

### QEMU/KVM

Kata Containers with QEMU 与 Kubernetes 完全兼容（此外，Kata 社区对 QEMU 作了[定制化的 patch 补丁](https://github.com/kata-containers/kata-containers/tree/main/tools/packaging/qemu/patches)）

取决于不同的 host 架构，Kata Containers 支持各种机器类型（machine），例如 x86 系统上的 q35、ARM 系统上的 virt 和 IBM Power 系统上的 pseries。

使用到的设备和特性有：

- virtio VSOCK or virtio serial
- virtio block or virtio SCSI
- [virtio net](https://www.redhat.com/en/virtio-networking-series)
- virtio fs or virtio 9p (recommend: virtio fs)
- VFIO
- hotplug
- machine accelerators

Kata 容器中使用加速器（accelerators）和热插拔来管理资源限制、缩短启动时间并减少内存占用。

**加速器**

加速器是特定于体系结构的，可用于提高性能并启用机器类型的特定功能。 Kata 容器中支持以下机器加速器：

- NVDIMM

  此机器加速器特定于 x86，并且仅支持 q35 机器类型。 nvdimm 用于将根文件系统作为持久内存设备提供给 VM

**设备热插拔**

Kata Containers VM 为了更快的启动时间和减少内存占用，往往是以最少的资源启动。在容器启动过程中，设备会热插拔到 VM 中。例如，当指定了额外 CPU 时，便是通过热添加的方式追加资源。 

Kata Containers 支持热添加以下设备：

- Virtio block
- Virtio SCSI
- VFIO
- CPU

### Firecracker/KVM

Firecracker 是基于 [rust-VMM](https://github.com/rust-vmm) 的衍生项目，支持的设备类型有限，但能提供更轻的体量和攻击面，专注于 FaaS 场景。因此，带有 Firecracker VMM 的 Kata 容器支持 CRI API 的一个子集。 Firecracker 不支持文件系统共享，仅支持基于块存储驱动程序。 Firecracker 不支持设备热插拔，也不支持 VFIO。因此，带有 Firecracker VMM 的 Kata Containers 不支持在启动后更新容器资源，也不支持设备透传。

支持的设备类型：

- virtio VSOCK
- virtio block
- virtio net

### Cloud Hypervisor/KVM

Cloud Hypervisor 同样是基于 [rust-VMM](https://github.com/rust-vmm) 的衍生项目，旨在为运行现代云工作负载提供更小的占用空间和更小的攻击面。具有 Cloud Hypervisor 的 Kata Containers 提供与 Kubernetes 的几乎完全兼容性，与 QEMU 能力相当。从 Kata Containers 1.12 和 2.0.0 版本开始，Cloud Hypervisor 配置支持 CPU 和内存大小调整、设备热插拔（磁盘和 VFIO）、通过 virtio-fs 共享文件系统、基于块的卷、从 VM 镜像启动由 pmem 设备支持，并为每个 VMM 线程（例如所有 virtio 设备工作线程）提供细粒度的 seccomp 过滤器。

支持的设备类型与特性：

- virtio VSOCK or virtio serial
- virtio block
- virtio net
- virtio fs
- virtio pmem
- VFIO
- hotplug
- seccomp filters
- [HTTP OpenAPI](https://github.com/cloud-hypervisor/cloud-hypervisor/blob/main/vmm/src/api/openapi/cloud-hypervisor.yaml)

**总结**

| Solution         | release introduced | brief summary                                                |
| ---------------- | ------------------ | ------------------------------------------------------------ |
| Cloud Hypervisor | 1.10               | upstream Cloud Hypervisor with rich feature support, e.g. hotplug, VFIO and FS sharing |
| Firecracker      | 1.5                | upstream Firecracker, rust-VMM based, no VFIO, no FS sharing, no memory/CPU hotplug |
| QEMU             | 1.0                | upstream QEMU, with support for hotplug and filesystem sharing |

# Storage

Kata Containers 与现有标准运行时兼容。从存储的角度来看，这意味着容器工作负载可能使用的存储量没有限制。由于 cgroups 无法设置存储分配限制，如果希望限制容器使用的存储量，请考虑使用现有设施，例如 quota(1) 限制或 device mapper 限制。

## virtio SCSI

virtio-scsi 用于将工作负载镜像（例如 busybox:latest）共享到 VM 内的容器环境中。现阶段，Kata Containers 支持 virtio SCSI 和 virtio BLK，后者由于较多限制已不做推荐。

## virtio FS

virtio-fs（VIRTIO）覆盖文件系统挂载点来共享工作负载镜像。kata-agent 使用此挂载点作为容器进程的根文件系统。

对于 virtio-fs，运行时为每个创建的 VM 启动一个 virtiofsd 守护进程（在主机上下文中运行）。

Kata Containers 使用轻量级虚拟机和硬件虚拟化技术来提供更强隔离，以构建安全的容器运行时。但也正是因为使用了虚拟机，容器的根文件系统无法像 runC 那样直接使用主机上构建好的目录，而需要有一种方法把 host 上的目录共享给 guest。

在此之前，有两种方法能够透传 host 目录或者数据给 guest，一种是基于 file 的方案，一个是基于 block 的方案。而这两种方案各有利弊，这里分别以 9pfs 和 devicemapper 为例来说明：

|      | 9pfs                                                         | devicemapper                                  |
| ---- | ------------------------------------------------------------ | --------------------------------------------- |
| 优势 | 使用 host 的 overlayfs，充分利用 host page cache             | 性能较好，POSIX 语义兼容性较好                |
| 痛点 | 基于网络协议，未对虚拟化场景做优化，性能较差；POSIX 语义兼容性不好 | 无法利用 host page cache，需要维护 lvm volume |

针对以上两个方案的痛点和优势，virtio-fs 在某种程度上做了很好的互补，在 Kata Containers 中，支持两种文件共享方式：virtio-fs 和 virtio-9p，在 Kata Containers 2.x 之后，virtio-fs 作为默认且推荐的方案选择。

virtio-fs 本身采用类似于 CS 的架构，选择 FUSE 作为文件系统，而非网络文件系统协议。server 端是位于 host 上的 virtiofsd，用于向 guest 提供 fuse 服务；client 端是把 guest kernel 抽象成一个 fuse client，用于挂载 host 上导出的目录。两者之间通过 vhost_user 建立连接。

最大的特点是利用了 VM 和 VMM 同时部署在一个 host 上的，数据的共享访问都是通过共享内存的方式，避免了 VM 和 VMM 之间的网络通讯，共享内存访问比基于网络文件系统协议访问要更轻量级也有更好的本地文件系统语义和一致性。在面对多 guest 要 mmap 同一个文件的时候，virtio-fs 会将该文件 mmap 到 QEMU 的进程空间里，其余的 guest 通过 DAX 直接访问。

<div align=center><img width="400" style="border: 0px" src="/gallery/kata-containers/virtiofs.png"></div>

<div align=center><img width="700" style="border: 0px" src="/gallery/kata-containers/virtiofs-detail.png"></div>

```shell
# qemu 进程参数节选
$ ps -ef | grep qemu
>>>
/usr/bin/qemu-system-x86_64
-name sandbox-f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9
-uuid 50041ac8-a9ed-4a60-9db1-44a55b2343d8
-machine pc,accel=kvm,kernel_irqchip
-cpu host,pmu=off
 
-device vhost-vsock-pci,disable-modern=false,vhostfd=3,id=vsock-117410659,guest-cid=117410659 -chardev socket,id=char-25f51af992a053e1,path=/run/vc/vm/f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9/vhost-fs.sock -device vhost-user-fs-pci,chardev=char-25f51af992a053e1,tag=kataShared
 
-kernel /usr/share/kata-containers/vmlinux-5.10.25-85
-initrd /usr/share/kata-containers/kata-containers-initrd-2021-07-14-11:02:27.932339999+0800-645e950
```

```shell
# 宿主机共享目录
$ ll /run/kata-containers/shared/sandboxes/f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9/shared
>>>
total 16
drwxr-xr-x 3 root root  60 Jul 19 14:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e
            drwxr-xr-x 1 root root 40 Jul 19 14:57 rootfs
-rw-rw-rw- 1 root root   0 Jul 19 14:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-1b95530f54b2fab0-termination-log
drwxrwxrwt 3 root root 140 Jul 19 14:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-395f94e69275ce07-serviceaccount
            lrwxrwxrwx 1 root root 13 Jul 19 14:57 ca.crt -> ..data/ca.crt
            lrwxrwxrwx 1 root root 16 Jul 19 14:57 namespace -> ..data/namespace
            lrwxrwxrwx 1 root root 12 Jul 19 14:57 token -> ..data/token
-rw-r--r-- 1 root root 212 Jul 19 14:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-45f076005d889842-hosts
-rw-r--r-- 1 root root 103 Jul 19 14:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-70624933bdd41bd7-resolv.conf
-rw-r--r-- 1 root root  15 Jul 19 14:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-c4568deafa816abf-hostname
drwxr-xr-x 3 root root  60 Jul 19 14:57 f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9
            drwxr-xr-x 1 root root 40 Jul 19 14:57 rootfs
-rw-r--r-- 1 root root 103 Jul 19 14:57 f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9-172f4c5d001a82b4-resolv.conf
```

```shell
# 虚拟机 mount 点
$ mount | grep kataShared
>>>
kataShared on /run/kata-containers/shared/containers type virtiofs (rw,relatime)
kataShared on /run/kata-containers/f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9/rootfs type virtiofs (rw,relatime)
kataShared on /run/kata-containers/6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e/rootfs type virtiofs (rw,relatime)
```

```shell
# 虚拟机共享目录
$ ls -l /run/kata-containers/shared/containers
>>>
total 16
drwxr-xr-x 3 root root  60 Jul 19 06:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e
-rw-rw-rw- 1 root root   0 Jul 19 06:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-1b95530f54b2fab0-termination-log
drwxrwxrwt 3 root root 140 Jul 19 06:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-395f94e69275ce07-serviceaccount
-rw-r--r-- 1 root root 212 Jul 19 06:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-45f076005d889842-hosts
-rw-r--r-- 1 root root 103 Jul 19 06:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-70624933bdd41bd7-resolv.conf
-rw-r--r-- 1 root root  15 Jul 19 06:57 6cc73ba11330cfbdb54bf40c77613d5f832aad01413d566ff8dabbf4e29d748e-c4568deafa816abf-hostname
drwxr-xr-x 3 root root  60 Jul 19 06:57 f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9
-rw-r--r-- 1 root root 103 Jul 19 06:57 f13846d4f1d58e82b2d3f461c3f2296c57992d415e32d7b41f689cf1126ee8d9-172f4c5d001a82b4-resolv.conf
```

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/storage-compare.png"></div>

## Devicemapper

devicemapper snapshotter 是一个特例。snapshotter 使用专用的块设备而不是格式化的文件系统，并且在块级别而不是文件级别运行。用于容器根文件系统直接使用底层块设备而不是覆盖文件系统。块设备映射到覆盖层的顶部读写层。与使用 virtio-fs 共享容器文件系统相比，这种方法提供了更好的 I/O 性能。

Kata Containers 具有热插拔添加和热插拔移除块设备的能力。这使得在 VM 启动后启动的容器可以使用块设备。

用户可以通过在容器内调用 mount(8) 来检查容器是否使用 devicemapper 块设备作为其 rootfs。如果使用 devicemapper 块设备，根文件系统（/）将从 /dev/vda 挂载。用户可以通过运行时配置禁止直接挂载底层块设备。

# VSOCKs

虚拟机中的进程可以通过以下两种方式与主机中的进程进行通信：

- 使用串口，虚拟机中的进程可以在串口设备读/写数据，主机中的进程可以从在 Unix socket 读/写数据。但是，串行链接一次限制对一个进程的读/写访问
- 更新、更简单的方法是 VSOCK，它可以接受来自多个客户端的连接

在 Kata Containers 2.x 中实现默认采用 VSOCK 的方式（依赖 4.8 以上版本内核和 vhost_vsock 内核模块）

```
.----------------------.
| .------------------. |
| | .-----.  .-----. | |
| | |cont1|  |cont2| | |
| | `-----'  `-----' | |
| |       |   |      | |
| |    .---------.   | |
| |    |  agent  |   | |
| |    `---------'   | |
| |       |   |      | |
| | POD .-------.    | |
| `-----| vsock |----' |
|       `-------'      |
|         |   |        |
|  .------.   .------. |
|  | shim |   | shim | |
|  `------'   `------' |
| Host                 |
`----------------------'
```

**优势**

- 高密度 Pod

  在 shimv1 使用 kata-proxy 建立 VM 和主机之间的连接，每一个 Pod 的内存大小大概是 4.5 MB 左右，在高密度 Pod 的集群中，内存的消耗过大。

- 可靠性

  kata-proxy 负责虚拟机和主机进程之间的连接，如果 kata-proxy 异常，所有连接都会中断，尽管容器仍在运行。由于通过 VSOCK 的通信是直接的，与容器失去通信的唯一场景是 VM 本身或 containerd-shim-kata-v2 异常停止，但是在这种情况下，容器也会被自动删除。

# Networking

Kata Containers 受限于 hypervisor 的功能，没有直接采用 Docker 默认的 Bridge 网络方案，而是采用的 macvtap 或者 tcfilter（使用 tc rules 将 veth 的 ingress 和 egress 队列分别对接 tap 的 egress 和 ingress 队列实现 veth 和 tap 的直连）方案。Kata Containers 本身是支持 CNI 管理网络的，网络方面相比容器，虽有额外开销但兼容性不差。

Docker 默认采用的容器网络方案是基于 network namespace + bridge + veth pairs 的，即在 host 上创建一个 network namespace，在 docker0 网桥上连接 veth pairs 的一端，再去 network namespace 中连上另一端，打通容器和 host 之间的网络。
这种方案得益于 namespace 技术，而许多 hypervisor 比如 QEMU 不能处理 veth interfaces。所以 Kata Containers 为 VM 创建了 TAP interfaces 来打通 VM 和 host 之间的网络。传统的 Container Engine 比如 Docker，会为容器创建 network namespace 和 veth pair，然后 Kata 会将 veth pair 的一端连上 TAP，即 macvtap 方案。

<div align=center><img width="700" style="border: 0px" src="/gallery/kata-containers/networking.png"></div>

Kata Containers 网络由 network namespaces、tap 和 tc 打通，创建 sandbox 之前首先创建网络命名空间，里面有 veth-pair 和 tap 两种网络接口，eth0 属于 veth-pair 类型接口，一端接入 CNI 创建的网络命名空间，一端接入宿主机；tap0_kata 属于 tap 类型接口，一端接入 cni 创建的网络命名空间，一端接入 QEMU 创建的 hypervisor，并且在 CNI 创建的网络命名空间使用 tc 策略打通 eth0 网络接口和 tap0_kata 网络接口，相当于把 eth0 和 tap0_kata 两个网络接口连成一条线。

sandbox 环境中只有 eth0 网络接口，这个接口是 QEMU 和 tap 模拟出的接口，mac、ip、掩码都和宿主机中 CNI 创建的网络命名空间中 eth0 的配置一样。

容器运行在 sandbox 环境中，容器采用共享宿主机网络命名空间方式创建容器，所以在容器中看到的网络配置和 sandbox 一样。

**网络流量走向：**
流量进入宿主机后首先由物理网络通过网桥或者路由接入到网络命名空间，网络命名空间中在使用 tc 策略牵引流量到 tap 网络接口，然后再通过 tap 网络接口把流量送入虚拟化环境中，最后虚拟化环境中的容器共享宿主机网络命名空间后就可以在容器中拿到网络流量。

```shell
[root@node1 kata]# ip netns exec cni-d27eff58-b9c9-a258-3a1e-a34528d9796f ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: tunl0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN group default qlen 1000
    link/ipip 0.0.0.0 brd 0.0.0.0
4: eth0@if29: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc noqueue state UP group default qlen 1000
    link/ether fe:68:1c:e3:47:da brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 10.244.166.150/32 brd 10.244.166.150 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fe80::fc68:1cff:fee3:47da/64 scope link
       valid_lft forever preferred_lft forever
5: tap0_kata: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 76:c7:1b:ab:30:64 brd ff:ff:ff:ff:ff:ff
    inet6 fe80::74c7:1bff:feab:3064/64 scope link
       valid_lft forever preferred_lft forever
```

```shell
[root@node1 kata]# ip netns exec cni-d27eff58-b9c9-a258-3a1e-a34528d9796f tc -s qdisc show dev eth0
qdisc noqueue 0: root refcnt 2
 Sent 0 bytes 0 pkt (dropped 0, overlimits 0 requeues 0)
 backlog 0b 0p requeues 0
qdisc ingress ffff: parent ffff:fff1 ----------------
 Sent 480 bytes 5 pkt (dropped 0, overlimits 0 requeues 0)
 backlog 0b 0p requeues 0
 
[root@node1 kata]# ip netns exec cni-d27eff58-b9c9-a258-3a1e-a34528d9796f tc -s filter show dev eth0 ingress
filter protocol all pref 49152 u32
filter protocol all pref 49152 u32 fh 800: ht divisor 1
filter protocol all pref 49152 u32 fh 800::800 order 2048 key ht 800 bkt 0 terminal flowid ??? not_in_hw  (rule hit 5 success 5)
  match 00000000/00000000 at 0 (success 5 )
        action order 1: mirred (Egress Redirect to device tap0_kata) stolen
        index 1 ref 1 bind 1 installed 439 sec used 437 sec
        Action statistics:
        Sent 480 bytes 5 pkt (dropped 0, overlimits 0 requeues 0)
        backlog 0b 0p requeues 0
 
[root@node1 kata]# ip netns exec cni-d27eff58-b9c9-a258-3a1e-a34528d9796f tc -s filter show dev tap0_kata ingress
filter protocol all pref 49152 u32
filter protocol all pref 49152 u32 fh 800: ht divisor 1
filter protocol all pref 49152 u32 fh 800::800 order 2048 key ht 800 bkt 0 terminal flowid ??? not_in_hw  (rule hit 12 success 12)
  match 00000000/00000000 at 0 (success 12 )
        action order 1: mirred (Egress Redirect to device eth0) stolen
        index 2 ref 1 bind 1 installed 451 sec used 165 sec
        Action statistics:
        Sent 768 bytes 12 pkt (dropped 0, overlimits 0 requeues 0)
        backlog 0b 0p requeues 0
```
<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/networking2.png"></div>

# Kata Containers

## kata-runtime (v1)

kata-runtime 实现 OCI 运行时标准，负责处理 OCI 标准命令，并启动 kata-shim 实例。

## kata-agent (v1 & v2)

kata-agent 是运行在 Kata 创建的 VM 中的管理程序，使用 libcontainer 管理容器和容器中的进程服务。具体来说，kata-agent 借助 QEMU、VIRTIO serial 或 VSOCK interface 的形式在 host 上暴露一个 socket 文件，并在 VM 内运行一个 gRPC server 和 Kata 其他组件交互，runtime（kata-runtime & containerd-shim-kata-v2）会通过 gRPC 来与 kata-agent 通信，来管理 VM 中的容器。

## kata-proxy (v1)

可选进程，在支持 VSOCK 的环境可以不需要。kata-proxy 给多个 kata-shim 和 kata-runtime 提供对 kata-agent 访问入口，负责路由 I/O 流和信号。kata-proxy 连接到 kata-agent 的 socket 上。一般情况下，kata-runtime 会通过 kata-proxy 来与 VM 内的 kata-agent 通信，管理 VM 内容器进程。

## kata-shim (v1)

kata-shim 的出现主要是考虑了 VM 内有多个容器的情况。在此之前，每个容器进程的回收由外层的一个 Reaper 负责。而 Kata Containers 方案中，容器运行在一个 VM 内，runtime 是无法监控、控制和回收这些 VM 内的容器，最多就是看到 QEMU 等进程，所以就设计了 kata-shim，用来监控容器进程，处理容器的所有 I/O 流，以及转发所有的要发送出去的信号。kata-runtime 会为每个容器创建一个对应的 kata-shim，每个 Pod sandbox（infra）也会有一个 kata-shim。

## containerd-shim-kata-v2 (v1 & v2)

在 Kata Containers v1.5 版本之后，整合了原本的 kata-runtime、kata-shim、kata-proxy 以及 reaper 的功能。

在原方案（v1）中，每个 Pod 需要 2N + 1 个 shim（N 代表容器，每个容器需要一个 containerd-shim 和 kata-shim，而每一个 Pod sandbox 也需要一个 kata-shim）。而 containerd-shim-kata-v2 实现了 [Containerd Runtime V2 (Shim API， 用于 runtime 和 Containerd 集成)](https://github.com/containerd/containerd/tree/master/runtime/v2)，K8s 只需要为每个 Pod、包括其内部的多个容器创建一个 shimv2 就够了。除此之外，无论 kata-agent 的 gRPC server 是否使用 VSOCK 暴露到 host 上，都不再需要单独的 kata-proxy。

## 整体架构

<div align=center><img width="800" style="border: 0px" src="/gallery/kata-containers/shimv2.svg"></div>

- 蓝色区域代表的是 Kubernetes CRI 的组件；红色区域代表的是 Kata Containers 的组件；黄色区域代表的是 Kata Containers 的 VM
- shimV1 中 CRI 的流程只会通过 kata-proxy （非 Vsock 环境）和 VM 通信管理容器进程等
- runc cmdline 就是实现了 OCI 标准的命令行工具
- 在 Kata 1.5 之后版本中 kata-runtime 得以保留，但是仅用作命令行工具判断 Kata Containers 的运行环境等，真正的 runtime 为 containerd-shim-kata-v2

**Kata Containers 1.x**

| Component                                                    | Type           | Description                                                  |
| ------------------------------------------------------------ | -------------- | ------------------------------------------------------------ |
| [agent](https://github.com/kata-containers/agent)            | core           | Management process running inside the virtual machine / POD that sets up the container environment. |
| [documentation](https://github.com/kata-containers/documentation) | documentation  | Documentation common to all components (such as design and install documentation). |
| [KSM throttler](https://github.com/kata-containers/ksm-throttler) | optional core  | Daemon that monitors containers and deduplicates memory to maximize container density on the host. |
| [osbuilder](https://github.com/kata-containers/osbuilder)    | infrastructure | Tool to create "mini O/S" rootfs and initrd images for the hypervisor. |
| [packaging](https://github.com/kata-containers/packaging)    | infrastructure | Scripts and metadata for producing packaged binaries (components, hypervisors, kernel and rootfs). |
| [proxy](https://github.com/kata-containers/proxy)            | core           | Multiplexes communications between the shims, agent and runtime. |
| [runtime](https://github.com/kata-containers/runtime)        | core           | Main component run by a container manager and providing a containerd shimv2 runtime implementation. |
| [shim](https://github.com/kata-containers/shim)              | core           | Handles standard I/O and signals on behalf of the container process. |

**Kata Containers 2.x**

| Component                                                    | Type           | Description                                                  |
| ------------------------------------------------------------ | -------------- | ------------------------------------------------------------ |
| [agent-ctl](https://github.com/kata-containers/kata-containers/blob/main/tools/agent-ctl) | utility        | Tool that provides low-level access for testing the agent.   |
| [agent](https://github.com/kata-containers/kata-containers/blob/main/src/agent) | core           | Management process running inside the virtual machine / POD that sets up the container environment. |
| [documentation](https://github.com/kata-containers/kata-containers/blob/main/docs) | documentation  | Documentation common to all components (such as design and install documentation). |
| [osbuilder](https://github.com/kata-containers/kata-containers/blob/main/tools/osbuilder) | infrastructure | Tool to create "mini O/S" rootfs and initrd images for the hypervisor. |
| [packaging](https://github.com/kata-containers/kata-containers/blob/main/tools/packaging) | infrastructure | Scripts and metadata for producing packaged binaries (components, hypervisors, kernel and rootfs). |
| [runtime](https://github.com/kata-containers/kata-containers/blob/main/src/runtime) | core           | Main component run by a container manager and providing a containerd shimv2 runtime implementation. |
| [trace-forwarder](https://github.com/kata-containers/kata-containers/blob/main/src/trace-forwarder) | utility        | Agent tracing helper.                                        |

**与 Kubernetes 集成架构**

<div align=center><img width="800" style="border: 0px" src="/gallery/kata-containers/with-kubernetes.png"></div>

# 流程示例

以容器创建流程为例，初步理解下 Kata Containers 是如何运作

1. 用户通过类似于 `sudo ctr run --runtime "io.containerd.kata.v2" --rm -t "quay.io/libpod/ubuntu:latest" foo sh` 命令请求 Container Manager 创建容器
2. Container Manager 守护进程启动 Kata 运行时的单个实例，即 containerd-shim-kata-v2
3. Kata 运行时加载配置文件
4. Container Manager 调用一组 shimv2 的 API
5. Kata 运行时启动配置好的 hypervisor
6. hypervisor 使用 guest 资源配置创建并启动（引导）VM
   1. hypervisor DAX 将 guest 镜像共享到 VM 中成为 VM rootfs（安装在 /dev/pmem* 设备上），即 VM 根环境
   2. hypervisor 使用 virtio FS 将 OCI bundle 安装到 VM 的 rootfs 内的容器特定目录中（这个容器特定目录将成为容器 rootfs，称为容器环境）
7. Kata agent 作为 VM 启动的一部分
8. 运行时调用 Kata agent 的 CreateSandbox API 来请求 agent 创建容器
   1. Kata agent 在包含容器 rootfs 的特定目录中创建容器环境（容器环境在容器 rootfs 目录中托管工作负载）（agent 创建的容器环境相当于 runc OCI 运行时创建的容器环境；Linux cgroups 和命名空间由 guest 内核在 VM 内创建，用于将工作负载与创建容器的 VM 环境隔离开来）
   2. Kata agent 在容器环境中生成工作负载
9. Container Manager 将容器的控制权返回给运行 ctr 命令的用户

