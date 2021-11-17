---
layout: post
title:  "Kata Containers 1.Architecture"
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

### rootfs

可以直接将 rootfs 构建成引导镜像，或者进一步生成 initrd 后，再对 initrd 构建成引导镜像。

rootfs 的构建过程中至少需要两个必要组件：

- 用于管理 VM 中的容器和容器中的进程服务的 **Kata Agent**，位于 /usr/bin/kata-agent
- 类似于 systemd 用于在 VM 中启动 Kata Agent 的 **init** 进程，位于 /sbin/init

所以在以 rootfs 作为镜像引导的 Guest OS 中是可以看到完整的进程关系大致为：

![](.\assert\img\kata-containers\rootfs.png)

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

initrd 的构建前置条件也是要构建 rootfs 文件系统。

initrd 的构建过程中至少需要一个必要组件：

- 用于管理 VM 中的容器和容器中的进程服务的 **Kata Agent**，kata agent 同时作为 init 进程，位于 /sbin/init

所以在以 initrd 作为镜像引导的 Guest OS 中是可以看到完整的进程关系大致为：

![](.\assert\img\kata-containers\initrd.png)

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

Kata Containers 本身支持多种 hypervisor  工具，如 qemu，cloud-hypervisor，firecracker，ACRN。

### Qemu

Kata Containers 使用的 Qemu 社区的上游分支，明确指定了具体的版本支持，并且针对社区 Qemu 做了[定制化的 patch 补丁](https://github.com/kata-containers/kata-containers/tree/main/tools/packaging/qemu/patches)。Kata Containers 在调用 Qemu 创建 Kata VM 时，会根据 Kata 的配置中将 Qemu 需要的参数透传下去，比如 VM 的内存大小，CPU 个数，内核参数等等。

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

![](.\assert\img\kata-containers\virtiofs.png)

![](.\assert\img\kata-containers\virtiofs-detail.png)

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

