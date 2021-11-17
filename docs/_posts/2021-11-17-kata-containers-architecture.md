---
layout: post
title:  "Kata Containers \n 1.Architecture"
date:   2021-04-06
excerpt: "Kata Containers 1.x 与 2.x 架构概述"
project: true
tag:
- Cloud Native
- Kubernetes
- Kata Containers
- Container Runtime
comments: false
---

* [Overview](#overview)
* [Glossary](#glossary)
   * [Guest](#guest)
      * [Kernel](#kernel)
      * [Image](#image)
   * [osbuilder](#osbuilder)
      * [Rootfs](#rootfs)
      * [Initrd](#initrd)
   * [Virtualization](#virtualization)
      * [Qemu](#qemu)
      * [Virtio](#virtio)
      * [Vhost](#vhost)
      * [Vhost-user](#vhost-user)
      * [Virtio-fs](#virtio-fs)
      * [VSOCK](#vsock)
      * [Networking](#networking)
   * [Kata-Runtime (v1)](#kata-runtime-v1)
   * [Kata-Agent (v1 &amp; v2)](#kata-agent-v1--v2)
   * [Kata-Proxy (v1)](#kata-proxy-v1)
   * [Kata-Shim (v1)](#kata-shim-v1)
   * [Containerd-Shim-Kata-V2 (v1 &amp; v2)](#containerd-shim-kata-v2-v1--v2)
* [Archiecture](#archiecture)

# Overview

Kata Containers 是一个开源项目，它采用**轻量化虚拟机**作为容器的隔离来构建一个安全容器运行时，而其虚拟化技术作为容器的二层保护为负载提供了更好的隔离性，这使得 Kata Containers 兼具**传统容器的形态和虚拟机的安全性**。 早在 2015 年，来自英特尔开源技术中心的工程师就开始探索采用 英特尔® 虚拟技术(英特尔® Virtualization Technology，英特尔® VT)来提高容器的安全隔离性，并以此发起了**英特尔® Clear Containers** 开源项目，与此同时，来自 Hyper.sh（一家中国的高科技初创公司）的工程师也发起了 **runV10** 开源项目，这两个项目采用的技术和目的都非常相似，都是为了将容器置于一个安全“沙箱“，以便进一步促进该技术发展和成熟。随后在 2017 年，英特尔和 Hyper.sh 团队将这两个开源项目在社区合并成了一个新的项目 Kata Containers。 传统虚拟机 (VMs) 可提供硬件隔离，而容器可快速响应，且占用空间相对较小，Kata Containers 将这两者的优势完美结合了起来。 每个 container 或 container pod 都在自己单独的虚拟机中启动， 并不再能够访问主机内核，杜绝了恶意代码侵入其它相临容器的可能。由于 Kata Containers 同时具备硬件隔离，也使得互不信任的租户，甚至于生产应用或前生产应用都能够在同一集群内安全运行，从而使得在裸机上运行容器即服务（Containers as a Service, CaaS）成为可能。

# Glossary

## Guest 

Hypervisor（VMM） 会启动一个包含最简 **Guest Kernel** 和 **Guest Image** 的虚拟机（VM）。

### Kernel

Guest 内核传递到 Hypervisor ，用于引导虚拟机。 **Kata Containers 中提供了一个对虚机启动时间和内存占用做了高度优化的默认内核，仅提供了容器工作负载所需的必要服务**。该内核是基于最新的上游 Linux 内核做的定制化。

### Image

VM 的运行除了依赖内核之外，还需要一个操作系统，也就是 Guest OS，这个镜像指的就是 Guest OS disk image。Kata Containers 支持基于 Initrd 和 Rootfs 两种 Guest 镜像。

## osbuilder

osbuilder 本身是 Kata Containers 项目中的一个模块，**主要负责构建 Guest OS 的引导镜像**。

**Kata Containers 支持两种引导镜像：rootfs 和 initrd**。无论哪种方式，默认都会将 Kata-Agent 编译到镜像中，在对 Kata-Agent 有定制化需求的场景下，可以手动编译后添加到镜像中。

### Rootfs

可以直接将 rootfs 构建成引导镜像，或者进一步生成 initrd 后，再对 initrd 构建成引导镜像。

rootfs 的构建过程中至少需要两个必要组件：

- 用于管理 VM 中的容器和容器中的进程服务的 **Kata Agent**，位于 /usr/bin/kata-agent
- 类似于 systemd 用于在 VM 中启动 Kata Agent 的 **init** 进程，位于 /sbin/init

所以在以 rootfs 作为镜像引导的 Guest OS 中是可以看到完整的进程关系大致为：

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/rootfs.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/rootfs.png"></a>
</figure>

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

### Initrd

initrd 的构建前置条件也是要构建 rootfs 文件系统。

initrd 的构建过程中至少需要一个必要组件：

- 用于管理 VM 中的容器和容器中的进程服务的 **Kata Agent**，kata agent 同时作为 init 进程，位于 /sbin/init

所以在以 initrd 作为镜像引导的 Guest OS 中是可以看到完整的进程关系大致为：

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/initrd.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/initrd.png"></a>
</figure>

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

## Virtualization

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtual-map.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtual-map.png"></a>
</figure>

### Qemu

*Kata Containers 本身支持多种 hypervisor  工具，如 qemu，cloud-hypervisor，firecracker，ACRN。*

Kata Containers 使用的 Qemu 社区的上游分支，明确指定了具体的版本支持，并且针对社区 Qemu 做了[定制化的 patch 补丁](https://github.com/kata-containers/kata-containers/tree/main/tools/packaging/qemu/patches)。Kata Containers 在调用 Qemu 创建 Kata VM 时，会根据 Kata 的配置中将 Qemu 需要的参数透传下去，比如 VM 的内存大小，CPU 个数，内核参数等等。

### Virtio

virtio 是一种 Linux 中 I/O **半虚拟化**的解决方案，是一套通用 I/O 设备虚拟化的程序，是对 Hypervisor 中的一组通用 I/O 设备的抽象。提供了一套上层应用与各 Hypervisor 虚拟化设备之间的通信框架和编程接口，减少跨平台所带来的兼容性问题，大大提高驱动程序开发效率。

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtio.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtio.png"></a>
</figure>

### Vhost

在 virtio 的机制中，Guest 与 Hypervisor 通信，会造成多次的数据拷贝和 CPU 上下文切换，带来性能上的损耗。

vhost 正是在这样的背景下提出的一种改善方案，是 virtio 的一种后端实现形式，位于 Host kernel 的一个模块，用于和 Guest 直接通信，数据交换直接在 Guest 和 Host kernel 之间通过 virtqueue 来进行，Qemu 不参与通信，仅负责一些控制层面的事，如 virtio 设备的适配模拟，负责用户空间某些管理控制事件的处理。

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/vhost.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/vhost.png"></a>
</figure>

### Vhost-user

在 vhost 的方案中，由于 vhost 实现在内核中，Guest 与 vhost 的通信，相较于原生的 virtio 方式性能上有了一定程度的提升，从 Guest 到 Hypervisor 的交互只有一次用户态的切换以及数据拷贝。但是对于某些用户态进程间的通信，比如数据面的通信方案，openvswitch 和 SDN 的解决方案，Guest 需要和 Host 用户态的 vswitch 进行数据交换，如果采用 vhost 的方案，Guest 和 Host 之间又存在多次的上下文切换和数据拷贝，为了避免这种情况，业界将 vhost 从内核态移到用户态。这就是 vhost-user 的实现。

### Virtio-fs

Kata Containers 使用轻量级虚拟机和硬件虚拟化技术来提供更强隔离，以构建安全的容器运行时。但也正是因为使用了虚拟机，容器的根文件系统无法像 runC 那样直接使用主机上构建好的目录，而需要有一种方法把 Host 上的目录共享给 Guest。

在此之前，有两种方法能够透传 Host 目录或者数据给 Guest，**一种是基于 file 的方案**，**一个是基于 block 的方案**。而这两种方案各有利弊，这里分别以 9pfs 和 devicemapper 为例来说明：

|      | 9pfs                                                         | devicemapper                                  |
| ---- | ------------------------------------------------------------ | --------------------------------------------- |
| 优势 | 使用 host 的 overlayfs，充分利用 host page cache             | 性能较好，POSIX 语义兼容性较好                |
| ----                                                                                                             |
| 痛点 | 基于网络协议，未对虚拟化场景做优化，性能较差；POSIX 语义兼容性不好 | 无法利用 host page cache，需要维护 lvm volume |

针对以上两个方案的痛点和优势，virtio-fs 在某种程度上做了很好的互补，在 Kata Containers 中，支持两种文件共享方式：**virtio-fs** 和 **virtio-9p**，在 Kata Containers 2.x 之后，**virtio-fs 作为默认且推荐的方案选择**。

virtio-fs 本身采用类似于 CS 的架构，**选择 FUSE 作为文件系统**，而非网络文件系统协议。**server 端是位于 host 上的 virtiofsd**，用于向 guest 提供 fuse 服务；**client 端是把 guest kernel 抽象成一个 fuse client**，用于挂载 host 上导出的目录。两者之间通过 vhost_user 建立连接。

最大的特点是利用了 VM 和 VMM 同时部署在一个 host 上的，**数据的共享访问都是通过共享内存**的方式，避免了 VM 和 VMM 之间的网络通讯，共享内存访问比基于网络文件系统协议访问要更轻量级也有更好的本地文件系统语义和一致性。在面对多 guest 要 mmap 同一个文件的时候，virtio-fs 会将该文件 mmap 到 qmeu 的进程空间里，其余的 guest 通过 **DAX** 直接访问。

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtiofs.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtiofs.png"></a>
</figure>

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtiofs-detail.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/virtiofs-detail.png"></a>
</figure>

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

### VSOCK

虚拟机中的进程可以通过两种不同的方式与主机中的进程进行通信。
第一种是使用串口，虚拟机中的进程可以在串口设备读/写数据，主机中的进程可以从在 Unix 套接字读/写数据。但是，串行链接一次限制对一个进程的读/写访问。
第二种更新、更简单的方法是 VSOCK，它可以接受来自多个客户端的连接。

**在 Kata Containers 2.x 中实现默认采用 Vsock 的方式**

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

- **高密度 Pod**

  在 shimV1 使用 Kata Proxy 建立 VM 和主机之间的连接，每一个 Pod 的内存大小大概是 4.5 MB 左右，在高密度 Pod 的集群中，内存的消耗过大。

- **可靠性**

  Kata Proxy 负责虚拟机和主机进程之间的连接，如果 Kata Proxy 异常，所有连接都会中断，尽管容器仍在运行。由于通过 VSOCK 的通信是直接的，与容器失去通信的唯一场景是 VM 本身或 containerd-shim-kata-v2 异常停止，但是在这种情况下，容器也会被自动删除。

### Networking

Kata Containers 受限于 hypervisor 的功能，没有直接采用 docker 默认的 bridge 网络方案，而是**采用的 MACVTAP 方案**。Kata Containers 本身是支持 CNI 管理网络的，网络方面相比容器，虽有额外开销但兼容性不差。

docker 默认采用的容器网络方案是基于 network namespace + bridge + veth pairs 的，即在 host 上创建一个 network namespace，在 docker0 bridge 上连接 veth pairs 的一端，再去 network namespace 中连上另一端，打通容器和 host 之间的网络。
这种方案得益于 namespace 技术，而许多 hypervisor 比如 Qemu 不能处理 veth interfaces。所以 Kata Containers 为 VM 创建了 TAP interfaces 来打通 VM 和 host 之间的网络。传统的 container engine 比如 Docker，会为容器创建 network namespace 和 veth pair，然后 Kata-runtime 会将 veth pair 的一端连上 TAP，即 MACVTAP 方案。

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/networking.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/networking.png"></a>
</figure>

Kata Containers 网络由 network namespaces、tap 和 tc 打通，创建 sandbox 之前首先创建网络命名空间，里面有 veth-pair 和 tap 两种网络接口，eth0 属于 veth-pair 类型接口，一端接入 cni 创建的网络命名空间，一端接入宿主机；tap0_kata 属于 tap 类型接口，一端接入 cni 创建的网络命名空间，一端接入 qemu 创建的 hypervisor，并且在 cni 创建的网络命名空间使用 tc 策略打通 eth0 网络接口和 tap0_kata 网络接口，相当于把 eth0 和 tap0_kata 两个网络接口连成一条线。

Sandbox 环境中只有 eth0 网络接口，这个接口是 qemu 和 tap 模拟出的接口，mac、ip、掩码都和宿主机中 cni 创建的网络命名空间中 eth0 的配置一样。

Container 运行在 Sandbox 环境中，Container 采用共享宿主机网络命名空间方式创建容器，所以在 Container 中看到的网络配置和 Sandbox 一样。

**网络流量走向：**
流量进入宿主机后首先由物理网络通过网桥或者路由接入到网络命名空间，网络命名空间中在使用 tc 策略牵引流量到 tap 网络接口，然后再通过 tap 网络接口把流量送入虚拟化环境中，最后虚拟化环境中的容器共享宿主机网络命名空间后就可以在容器中拿到网络流量。

```shell
# 通过 Container ID 获取到 PID 进程
[root@archcnstcm6190 ~]# crictl inspect 084b20c34a8d2  | grep pid
# --target 指向上文的 PID 进程
[root@archcnstcm6190 ~]# nsenter --target 29695  --mount  --uts --ipc  --net --pid
[root@archcnstcm6190 /]# ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: tunl0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN group default qlen 1000
    link/ipip 0.0.0.0 brd 0.0.0.0
3: tap0_kata: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 56:09:b7:50:c1:a0 brd ff:ff:ff:ff:ff:ff
    inet6 fe80::5409:b7ff:fe50:c1a0/64 scope link
       valid_lft forever preferred_lft forever
2796: eth0@if2797: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc noqueue state UP group default qlen 1000
    link/ether 02:ae:3b:10:26:43 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 10.244.0.16/16 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fe80::c11:e2ff:fe18:c6d8/64 scope link
       valid_lft forever preferred_lft forever
```

## Kata-Runtime (v1)

Kata-Runtime **实现 OCI 运行时标准**，负责处理 OCI 标准命令，并启动 Kata-Shim 实例。

## Kata-Agent (v1 & v2)

Kata-Agent 是运行在 Kata 创建的 **VM 中的管理程序**，使用 libcontainer 管理容器和容器中的进程服务。具体来说，Kata-Agent 借助 QEMU、VIRTIO serial 或 **VSOCK interface** 的形式在 host 上暴露一个 socket 文件，并在 VM 内运行一个 gRPC server 和 Kata 其他组件交互，runtime（Kata-Runtime & Containerd-Shim-Kata-V2）会通过 gRPC 来与 Kata-Agent 通信，来管理 VM 中的容器。

## Kata-Proxy (v1)

**可选进程**，在支持 VSOCK 的环境可以不需要。Kata-Proxy 给多个 Kata-Shim 和 Kata-Runtime 提供对 **Kata-Agent 访问入口**，负责路由 I/O 流和信号。Kata-Proxy 连接到 Kata-Agent 的 socket 上。一般情况下，Kata-Runtime 会通过 Kata-Proxy 来与 VM 内的 Kata-Agent 通信，管理 VM 内容器进程。

## Kata-Shim (v1)

Kata-Shim 的出现主要是考虑了 VM 内有多个容器的情况。在此之前，每个容器进程的回收由外层的一个 Reaper 负责。而 Kata Containers 方案中，容器运行在一个 VM 内，runtime 是无法监控、控制和回收这些 VM 内的容器，最多就是看到 QEMU 等进程，所以就设计了 Kata-Shim，**用来监控容器进程，处理容器的所有 I/O 流，以及转发所有的要发送出去的信号**。Kata-Runtime 会为每个容器创建一个对应的 Kata-Shim，每个 Pod sandbox（infra）也会有一个 Kata-Shim。

## Containerd-Shim-Kata-V2 (v1 & v2)

在 **Kata Containers v1.5 版本之后**，整合了原本的 Kata-Runtime、Kata-Shim、Kata-Proxy 以及 Reaper 的功能。

在原方案（v1）中，每个 Pod 需要 2N + 1 个 shim（N 代表容器，每个容器需要一个 Containerd-Shim 和 Kata-Shim，而每一个 Pod sandbox 也需要一个 Kata-Shim）。而 Containerd-Shim-Kata-V2 实现了 [Containerd Runtime V2 (Shim API， 用于 runtime 和 Containerd 集成)](https://github.com/containerd/containerd/tree/master/runtime/v2)，K8s 只需要为每个 Pod、包括其内部的多个容器创建一个 shimv2 就够了。除此之外，无论 Kata-Agent 的 gRPC server 是否使用 VSOCK 暴露到 host 上，都不再需要单独的 Kata-Proxy。

# Archiecture

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/architecture.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/architecture.png"></a>
</figure>

- *蓝色区域代表的是 Kubernetes CRI 的组件；红色区域代表的是 Kata Containers 的组件；黄色区域代表的是 Kata Containers 的 VM*
- *ShimV1 中 CRI 的流程只会通过 Kata-Proxy （非 Vsock 环境）和 VM 通信管理容器进程等*
- *runc cmdline 就是实现了 OCI 标准的命令行工具*
- *在 Kata 1.5 之后版本中 Kata-Runtime 得以保留，但是仅用作命令行工具判断 Kata Containers 的运行环境等，真正的 runtime 为 containerd-shim-kata-v2*

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

<figure>
	<a href="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/process.png"><img src="https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/process.png"></a>
</figure>

