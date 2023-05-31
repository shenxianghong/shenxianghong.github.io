---
title: "「 Kata Containers 」快速开始"
excerpt: "Kata Containers 在 Kubernetes 集群场景中的配置与基础使用示例"
cover: https://picsum.photos/0?sig=20210415
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2021-04-15
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

# 安装

Kata Containers 社区提供了 x86 架构制品，arm64 架构制品需要手动编译。这里以 x86 架构的社区制品安装为例：

```shell
$ wget https://github.com/kata-containers/kata-containers/releases/download/3.0.0/kata-static-3.0.0-x86_64.tar.xz
$ tar -xvf kata-static-3.0.0-x86_64.tar.xz
$ tree opt/kata/
opt/kata/
├── bin # 可执行的二进制文件与脚本
│   ├── cloud-hypervisor
│   ├── containerd-shim-kata-v2
│   ├── firecracker
│   ├── jailer
│   ├── kata-collect-data.sh
│   ├── kata-monitor
│   ├── kata-runtime
│   └── qemu-system-x86_64
├── libexec # 可执行的二进制文件
│   └── virtiofsd
├── runtime-rs # Rust 实现下的 shimv2
│   └── bin
│       └── containerd-shim-kata-v2
└── share
    ├── bash-completion # 命令行补全脚本
    │   └── completions
    │       └── kata-runtime
    ├── defaults # 不同 hypervisor 实现下的静态配置文件
    │   └── kata-containers
    │       ├── configuration-acrn.toml
    │       ├── configuration-clh.toml
    │       ├── configuration-dragonball.toml
    │       ├── configuration-fc.toml
    │       ├── configuration-qemu.toml
    │       └── configuration.toml -> configuration-qemu.toml
    ├── kata-containers # 内核与 guest 镜像
    │   ├── config-5.19.2
    │   ├── kata-alpine-3.15.initrd
    │   ├── kata-clearlinux-latest.image
    │   ├── kata-containers.img -> kata-clearlinux-latest.image
    │   ├── kata-containers-initrd.img -> kata-alpine-3.15.initrd
    │   ├── vmlinux-5.19.2-96
    │   ├── vmlinux.container -> vmlinux-5.19.2-96
    │   ├── vmlinuz-5.19.2-96
    │   └── vmlinuz.container -> vmlinuz-5.19.2-96
    └── kata-qemu # QEMU 依赖
        └── qemu
            ├── bios-256k.bin
            ├── bios.bin
            ├── bios-microvm.bin
            ├── edk2-aarch64-code.fd
            ├── edk2-arm-code.fd
            ├── edk2-arm-vars.fd
            ├── edk2-i386-code.fd
            ├── edk2-i386-secure-code.fd
            ├── edk2-i386-vars.fd
            ├── edk2-licenses.txt
            ├── edk2-x86_64-code.fd
            ├── edk2-x86_64-secure-code.fd
            ├── efi-virtio.rom
            ├── firmware
            │   ├── 50-edk2-i386-secure.json
            │   ├── 50-edk2-x86_64-secure.json
            │   ├── 60-edk2-aarch64.json
            │   ├── 60-edk2-arm.json
            │   ├── 60-edk2-i386.json
            │   └── 60-edk2-x86_64.json
            ├── hppa-firmware.img
            ├── kvmvapic.bin
            ├── linuxboot.bin
            ├── linuxboot_dma.bin
            ├── multiboot_dma.bin
            ├── pvh.bin
            ├── qboot.rom
            ├── qemu-nsis.bmp
            ├── s390-ccw.img
            └── s390-netboot.img
```

# 配置参数

Kata Containers 中配置的优先级为：动态配置项 > 静态配置项 > 默认值

- 动态配置项是通过 OCI spec 中的 annotations 传递，主流的 Container Engine 均实现支持将容器 annotations 透传至 Kata 运行时
- 各个动态与静态配置项支持与否视 hypervisor 具体实现的能力有所区别

## hypervisor

动态配置项的前缀为 io.katacontainers.hypervisor.\<静态配置项\>

### QEMU

| 静态配置项                   | 动态配置 | 含义                                                         |
| ---------------------------- | -------- | ------------------------------------------------------------ |
| path                         | Y        | hypervisor 可执行文件的路径                                  |
| kernel                       | Y        | VM 内核路径                                                  |
| image                        | Y        | VM rootfs 镜像路径，与 initrd 有且仅有一个                   |
| initrd                       | Y        | VM rootfs 镜像路径，与 image 有且仅有一个                    |
| machine_type                 | Y        | QEMU 机器类型，例如 amd64 架构下为 q35、arm64 架构下为 virt  |
| confidential_guest           | N        | 是否启用机密容器特性。机密容器需要 host 支持 tdxProtection（[Intel Trust Domain Extensions](https://software.intel.com/content/www/us/en/develop/articles/intel-trust-domain-extensions.html)）、sevProtection（[AMD Secure Encrypted Virtualization](https://developer.amd.com/sev/)）、pefProtection（[IBM POWER 9 Protected Execution Facility](https://www.kernel.org/doc/html/latest/powerpc/ultravisor.html)）以及 seProtection（[IBM Secure Execution (IBM Z & LinuxONE)](https://www.kernel.org/doc/html/latest/virt/kvm/s390-pv.html)）。不支持 CPU 和内存的热插拔以及 NVDIMM 设备，不支持 arm64 架构 |
| rootless                     | Y        | 是否以非 root 权限的随机用户启动 QEMU VMM，默认为 false      |
| enable_annotations           | N        | 允许 hypervisor 动态配置的配置项                             |
| valid_hypervisor_paths       | N        | 以 glob(3) 规则校验 path 参数是否为合法的路径集合            |
| kernel_params                | Y        | VM kernel 的额外附加参数，默认为空                           |
| firmware                     | Y        | 固件路径，默认为空                                           |
| firmware_volume              | Y        | 固件卷路径，默认为空                                         |
| machine_accelerators         | Y        | 机器加速器参数，默认为空                                     |
| seccompsandbox               | N        | seccomp 参数。QEMU seccomp sandbox 是 QEMU VM 中的一种安全特性，通过限制 QEMU 进程的系统调用，以提高 VM 的安全性。它使用了 Linux 内核提供的 seccomp 机制，将 QEMU 进程限制在一组安全的系统调用中，从而降低 VM 遭受攻击的风险。推荐设置 /proc/sys/net/core/bpf_jit_enable 文件内容为 1，以降低该特性带来的性能下降 |
| cpu_features                 | Y        | CPU 特性参数，默认为空                                       |
| default_vcpus                | Y        | VM 默认的 CPU 数量，默认为 1，最大为 host CPU 数量           |
| default_maxvcpus             | Y        | VM 最大的 CPU 数量，默认为 host CPU 数量，具体能否使用到 host CPU 数量，还需要视 hypervisor 限制而定。过大的 CPU 数量会影响到 VM 的性能以及内存占比 |
| default_bridges              | N        | VM 默认的 PCI 桥数量，默认为 1，最大为 5。目前，仅支持 PCI bridge，每个 PCI bridge 最多支持 30 个设备的热插拔，每个 VM 最多支持 5 个 PCI bridge（这可能是 QEMU 或内核中的一个 bug） |
| default_memory               | Y        | VM 默认的内存总量，默认为 1，最大为 host 内存总量            |
| memory_slots                 | Y        | VM 默认的内存插槽数量，默认为 10，即内存热添加次数上限为 10  |
| default_maxmemory            | Y        | VM 最大的内存总量，默认为  host 内存总量                     |
| memory_offset                | Y        | VM 内存偏移量，用于描述 NVDIMM 设备的内存空间，当 block_device_driver 为 nvdimm 时，需要设置此参数，最终会追加到 default_maxmemory 中 |
| enable_virtio_mem            | Y        | 是否启用 virtio-mem 设备，默认为 false。virtio-mem 设备可以提高 VM 的内存性能。它通过在 host 和 VM 之间共享内存，使 VM 可以直接访问 host 内存，而无需通过复制或传输数据。这种直接访问可显著降低内存访问延迟和 CPU 使用率，并提高 VM 的性能和吞吐量。推荐设置 /proc/sys/vm/overcommit_memory 文件内容为 1 |
| disable_block_device_use     | Y        | 禁止块设备用于容器的 rootfs。例如 devicemapper 之类的存储驱动程序中，容器的 rootfs 由块设备支持，出于性能原因，块设备默认直接传递给 hypervisor。 禁用传递时，会用 virtio-fs 传递 rootfs |
| shared_fs                    | Y        | host 和 VM 之间共享文件系统类型，默认为 virtio-fs，此外支持 virtio-9p 和 virtio-fs-nydus |
| virtio_fs_daemon             | Y        | vhost-user-fs 可执行文件的路径                               |
| valid_virtio_fs_daemon_paths | N        | 以 glob(3) 规则校验 virtio_fs_daemon 参数是否为合法的路径集合 |
| virtio_fs_cache_size         | Y        | DAX 缓存大小，默认为 0 MiB。virtio_fs 支持 DAX（Direct Access）模式，这意味着 VM 可以直接访问 host 的文件系统缓存，从而提高了读取和写入数据的速度 |
| virtio_fs_extra_args         | Y        | vhost-user-fs 的额外附加参数                                 |
| virtio_fs_cache              | Y        | virtio-fs 文件系统在 VM 和 host 之间共享文件时的缓存模式，默认是 auto，此外支持 none 和 always。none 表示 VM 中不缓存文件系统的元数据、数据和路径名查找，所有这些信息都需要从 host 中获取。在这种模式下，任何对文件的修改都会立即被推送到 host；alway 则截然相反，表示 VM 中的文件系统元数据、数据和路径名查找都会被缓存，并且永不过期；而 auto 表示 VM 中的元数据和路径名查找缓存会在一定时间后过期（默认为 1 秒），而数据则会在文件打开时缓存（即 close-to-open 一致性）。在这种模式下，VM 会根据需要从 host 中获取文件信息，而不是每次都从 host 获取 |
| block_device_driver          | Y        | hypervisor 用于管理容器 rootfs 的块存储驱动程序，默认为 virtio-scsi，此外支持 virtio-blk 和 nvdimm。virtio-scsi 是一种基于SCSI 协议的存储虚拟化技术；virtio-blk 则是一种用于块设备的存储虚拟化技术，在使用 virtio-scsi 和 virtio-blk 时，host 上的块设备可以被 VM 视为本地的块设备，从而可以在 VM 中进行读写操作；nvdimm 是一种用于非易失性内存（NVM）的存储技术。它允许将内存作为块设备使用，并提供了与传统块设备相似的可靠性和数据完整性保护 |
| block_device_aio             | Y        | QEMU 使用的块设备异步 I/O 机制，默认为 io_uring，此外支持 threads 和 native。threads 表示 QEMU 使用基于 pthread 的磁盘 I/O 机制，这种机制是在用户空间实现的，可以在多个线程之间共享 CPU 时间，但是性能比较一般；native 表示 QEMU 使用本地的 Linux I/O 机制。这种机制是在内核空间实现的，可以获得更好的性能，但是需要特权；io_uring 表示 QEMU 使用 Linux io_uring API 来实现异步 I/O，这种机制提供了 Linux 中最快的 I/O 操作，可以在 QEMU 5.0 及以上版本中使用，但需要 Linux 内核版本大于 5.1，io_uring 机制可以减少 CPU 的上下文切换次数，提高 I/O 操作的效率 |
| block_device_cache_set       | Y        | 是否将缓存相关选项设置给块设备，默认为 false。该参数影响到 block_device_cache_direct 和 block_device_cache_noflush 是否生效 |
| block_device_cache_direct    | Y        | 是否启用 O_DIRECT 选项，默认为 false。O_DIRECT 是一种 Linux 系统提供的选项，可以绕过 host 页缓存，直接访问块设备，从而提高存储 I/O 的性能。受 block_device_cache_set 参数设置影响 |
| block_device_cache_noflush   | Y        | 是否忽略块设备的缓存刷盘请求，默认为 false。受 block_device_cache_set 参数设置影响 |
| enable_iothreads             | Y        | 是否启用独立的 I/O 线程，默认为 false。启用时，块设备的 I/O 操作将在一个单独的 I/O 线程中处理，而非 QEMU 的主线程中进行处理，可以减少了主线程的阻塞时间，提高 VM 的 I/O 性能 |
| enable_mem_prealloc          | Y        | 是否启用 VM 内存预分配，默认为 false。启用 VM 内存的预分配可以使内存分配更加稳定和可预测，从而提高 VM 的性能。但是，预分配内存也会占用更多的系统资源，降低容器密度 |
| enable_hugepages             | Y        | 是否启用 VM 大页内存，默认为 false。Huge Pages 的特点是将内存分配成固定大小的页（通常为 2MB 或 1GB），从而降低了页表的大小和操作系统内核的开销，使用 Huge Pages 分配 VM 内存可以提升性能。在启用大页内存时，内存预分配（enable_mem_prealloc）会被强制设置启用 |
| enable_vhost_user_store      | Y        | 是否启用 vhost-user 存储设备，默认为 false。启用 vhost-user 存储设备可以将 host 上的块设备虚拟化为一种可以在 VM 中使用的设备，通过 vhost-user 协议在 host 和 VM 之间传输数据，从而提高 VM 的存储性能。在启用 vhost-user 存储设备时，Linux 中的一些保留块类型（Major Range 240-254）将被选择用于表示 vhost-user 设备 |
| vhost_user_store_path        | Y        | vhost-user 设备的目录，默认为 /var/run/kata-containers/vhost-user。在该目录下，"block" 子目录用于存储块设备，"block/sockets" 子目录用于存储 vhost-user sockets，"block/devices" 子目录用于存储模拟的块设备节点 |
| enable_iommu                 | Y        | 是否启用 vIOMMU 设备，默认为 false。vIOMMU 用于将 VM 的 I/O 操作隔离在一个独立的内存地址空间中，以提高 VM 的安全性和性能。此外，vIOMMU 还可以提供更好的 I/O 性能，因为它可以减少 VM 和 host 之间的数据传输次数 |
| enable_iommu_platform        | Y        | 是否启用 IOMMU_PLATFORM 设备，默认为 false。IOMMU_PLATFORM 用于设备 DMA（Direct Memory Access）操作隔离在一个独立的内存地址空间中，以提高系统的安全性和性能。此外，IOMMU_PLATFORM 还可以提供更好的 DMA 性能，因为它可以减少系统和设备之间的数据传输次数。 |
| valid_vhost_user_store_paths | N        | 以 glob(3) 规则校验 vhost_user_store_path 参数是否为合法的路径集合 |
| file_mem_backend             | Y        | 基于文件的内存支持的路径，默认为空。基于文件的 VM 内存支持是一种将 VM 内存保存在文件中的技术，而不是保存在 host 的物理内存中。此外，使用基于文件的 VM 内存还可以减少 VM 和 host 之间的数据传输，从而提高 VM 的性能。在使用 virtio-fs 时，该选项会自动启用，并使用 "/dev/shm" 作为后端文件 |
| valid_file_mem_backends      | N        | 以 glob(3) 规则校验 file_mem_backend 参数是否为合法的路径集合 |
| pflashes                     | N        | 向 VM 中添加的镜像文件路径，默认为空。镜像文件通常用于模拟系统中的 BIOS 或 UEFI  固件等。例如，arm64 架构下的内存热插拔则需要提供一对 pflash |
| enable_debug                 | N        | 是否启用 hypervisor 和内核的 debug 参数，默认为 false        |
| disable_nesting_checks       | N        | 是否禁止嵌套虚拟化环境检查，默认为 false。禁用嵌套检查可以从运行时的行为与在裸机上相同 |
| msize_9p                     | Y        | virtio-9p 共享文件系统中描述 9p 数据包有效载荷的字节数量，默认为 8192 |
| disable_image_nvdimm         | Y        | 是否禁止使用 NVDIMM 设备挂载 VM 镜像，默认为 false。在未禁用且支持 NVDIMM 设备时，VM 镜像会借助 NVDIMM 设备热添加，否则，使用 virtio-block 设备 |
| hotplug_vfio_on_root_bus     | Y        | 是否允许 VFIO 设备在 root 总线上热插拔，默认为 true。VFIO 是一种用于虚拟化环境中的设备直通技术，它允许将物理设备直接分配给 VM，从而提高 VM 的性能和可靠性。然而，在桥接设备上进行 VFIO 设备的热插拔存在一些限制，特别是对于具有大型 PCI 条的设备。因此，通过将该选项设置为 true，可以在 root 总线上启用 VFIO 设备的热插拔，从而解决这些限制问题 |
| pcie_root_port               | Y        | pcie_root_port 设备数量，默认为 0。在热插拔 PCIe 设备之前需要添加 pcie_root_port 设备，主要针对使用一些大型 PCI 条设备（如 Nvidia GPU）的情况。仅在启用 hotplug_vfio_on_root_bus 且 machine_type 为 q35 时生效 |
| disable_vhost_net            | Y        | 是否禁用 vhost-net 作为 virtio-net 的后端，默认为 false。使用 vhost-net 时意味着在提高网络 I/O 性能的同时，会牺牲一定的安全性（因为 vhost-net 运行在 ring0 模式下，具有最高的权限和特权） |
| entropy_source               | Y        | 熵源路径，默认为 /dev/urandom，用于生成随机数的来源。/dev/random 是一个阻塞的熵源，如果 host 的熵池用尽，VM 的启动时间会增加，可能会导致启动超时。相比之下，/dev/urandom 是一个非阻塞的熵源，可以适用于大多数场景 |
| valid_entropy_sources        | N        | 以 glob(3) 规则校验 entropy_source 参数是否为合法的路径集合  |
| guest_hook_path              | Y        | VM 中 hook 脚本路径，默认为空。hook 必须按照其 hook 类型存储在 guest_hook_path 的子目录中，例如 "guest_hook_path/{prestart,poststart,poststop}"。Kata agent 将扫描这些目录查找可执行文件，按字母顺序将其添加到容器的生命周期中，并在 VM 运行时命名空间中执行 |
| rx_rate_limiter_max_rate     | Y        | 网络 I/O inbound 带宽限制，默认为 0，即不作限制。在 QEMU 中，借助 HTB(Hierarchy Token Bucket) 限制管理 |
| tx_rate_limiter_max_rate     | Y        | 网络 I/O outbound 带宽限制，默认为 0，即不作限制。在 QEMU 中，借助 HTB(Hierarchy Token Bucket) 限制管理 |
| guest_memory_dump_path       | N        | VM 内存转储文件路径，默认为空。在出现 GUEST_PANICKED 事件时，VM 的内存将被转储到 host 文件系统下的指定目录中（如果该目录不存在，会自动创建）。被转储的文件（也称为 vmcore 文件）可以使用 crash 或 gdb 等工具进行处理。注意，转储 VM 内存可能需要很长时间，具体取决于 VM 内存的大小，并且会占用大量磁盘空间 |
| guest_memory_dump_paging     | N        | 是否启用 VM 内存分页，默认为 false。在 VM 内存转储时，将使用分页机制来处理虚拟地址和物理地址之间的映射关系。如果禁用该选项，则将使用物理地址而不是虚拟地址来进行转储。比如，如果希望使用 gdb 工具而不是 crash 工具，或者需要在 ELF vmcore 中使用VM 的虚拟地址，那么则需要启用内存分页功能 |
| enable_guest_swap            | Y        | 是否启用 VM 中的交换空间，默认为 false。启用时，会将一个 raw 格式的设备添加到 VM 中作为 SWAP 设备。如果 annotations["io.katacontainers.container.resource.swappiness"] 大于 0，则根据 annotations["io.katacontainers.container.resource.swap_in_bytes"] 计算 SWAP 设备大小：默认为 swap_in_bytes - memory_limit_in_bytes；如果 swap_in_bytes 未设置，则为 memory_limit_in_bytes，如果均未设置，则为 default_memory |
| use_legacy_serial            | Y        | 是否使用传统的串行接口作为 VM 控制台设备，默认为 false       |
| disable_selinux              | N        | 是否禁用在 hypervisor 上应用 SELinux，默认为 false           |

## factory

不支持动态配置项

| 静态配置项        | 含义                                                         |
| ----------------- | ------------------------------------------------------------ |
| enable_template   | 是否启用 VM 模板，默认为 false。 启用后，从模板克隆创建新的 VM。 它们将通过只读映射共享相同的内核、initramfs 和 Kata agent 内存。 如果在同一 host 上运行许多 Kata 容器，VM 模板有助于加快容器的创建并节省大量内存。仅支持镜像类型为 initrd |
| template_path     | VM 模板保存的路径，默认为 /run/vc/vm/template                |
| vm_cache_number   | VMCache 的数量，默认为 0，表示禁用 VMCache。VMCache 是一种在使用之前将 VM 创建为缓存的功能，有助于加快容器的创建。 该功能由服务器和通过 Unix socket 进行通信的客户端组成，服务器将创建一些 VM 并缓存起来。如果启用了 VMCache 功能，kata-runtime 在创建新的 sandbox 时会向 VMCache 服务器请求 VM |
| vm_cache_endpoint | VMCache 服务器的 socket 地址，默认为 /var/run/kata-containers/cache.sock |

## runtime

动态配置项的前缀为 io.katacontainers.config.runtime.\<静态配置项\>

| 静态配置项                   | 动态配置 | 含义                                                         |
| ---------------------------- | -------- | ------------------------------------------------------------ |
| enable_debug                 | N        | 是否启用 containerd-shim-kata-v2 的 debug 参数，默认为 false |
| internetworking_model        | Y        | VM 与容器网络的连通方式，默认为 tcfilter，此外支持 tcfilter、macvtap 和 none。无论哪种方式，tap 设备都是创建的，区别在于 tap 设备和容器网络是如何打通的 |
| disable_guest_seccomp        | Y        | 是否在 VM 中启用 seccomp 特性，默认为 false。启用时，seccomp 配置文件会由 Kata agent 传递到 VM 中并应用，用于提供额外的安全层 |
| enable_tracing               | N        | 是否启用 opentracing 的 traces 和 spans，默认为 false        |
| jaeger_endpoint              | N        | Jaeger 服务地址，默认为 `http://localhost:14268/api/traces`  |
| jaeger_user                  | N        | Jaeger 服务账号，默认为空                                    |
| jaeger_password              | N        | Jaeger 服务密码，默认为空                                    |
| disable_new_netns            | Y        | 是否禁止为 shim 和 hypervisor 进程创建网络命名空间，默认为 false。适用于 internetworking_model 为 none，此时 tap 设备将位于 host 网络命名空间中，并可以直接连接到 bridge（如 OVS） |
| sandbox_cgroup_only          | Y        | 是否仅启用 sandboxCgroup，默认为 false。启用时，cgroups 仅有一个 sandboxCgroup，用于限制所有的 Kata 进程；禁用时，cgroups 分为 sandboxCgroup 和 overheadCgroup，除 vCPU 线程外的其他 Kata 进程和线程都将在 overheadCgroup 下运行 |
| static_sandbox_resource_mgmt | N        | 是否启用静态资源管理，默认为 false。启用时，Kata Containers 将在 VM 启动之前尝试确定适当的资源大小，而非动态更新 VM 中的内存和 CPU 数量，用作不支持 CPU 和内存热插拔的硬件架构或 hypervisor 解决方案 |
| sandbox_bind_mounts          | N        | VM 中待挂载 host 的文件路径，默认为空。启用时，host 的该路径文件会被以只读的形式挂载到 VM 的 /run/kata-containers/shared/containers/sandbox-mounts 路径中，不会暴露给容器工作负载，仅为潜在的 VM 服务提供 |
| vfio_mode                    | Y        | VFIO 的模式，默认为 guest-kernel，可选的有 vfio 和 guest-kernel。vfio 与 runC 的行为相近，在容器中，VFIO 设备将显示为 VFIO 字符设备，位于 /dev/vfio 下，确切的名称可能与 host 不同（需要匹配 VM 的 IOMMU 组号，而不是 host 的）；guest-kernel 是 Kata 特有的行为，VFIO 设备由 VM 内核中的驱动程序管理，意味着它将显示为一个或多个设备节点或网络接口，具体取决于设备的特性。这种模式要求容器内的工作负载具有显式支持 VM 内设备的代码或逻辑 |
| disable_guest_empty_dir      | N        | 是否禁用在 VM 文件系统创建 emptyDir 挂载点，默认为 false。禁用时，Kata Containers 将不会在 VM 文件系统上创建 Kubernetes emptyDir 挂载点，而是在 host 上创建 emptyDir 挂载点，并通过 virtio-fs 共享，虽然更慢一些，但允许从 host 共享文件到 VM 中 |
| experimental                 | Y        | 体验特性，默认为空。*暂未有支持的体验特性*                   |
| enable_pprof                 | Y        | 是否启用 pprof，默认为 false。启用后，可以通过 kata-monitor 运行 pprof 工具来分析 shim 进程 |

## annotation 参数扩展

Kata Containers 可以通过 annotation 的方式定制化每一个 Kata 容器的底层运行时参数：

- 上层容器运行时将 annotation 透传至底层运行时（例如 Containerd 1.4.x 以上的版本支持 annotation 透传；CRI-O 默认透传所有 annotation，无需额外配置。*具体参考 Container Manager 集成*）
- Kata Containers 配置中开启识别特定的 annotation（[hypervisor].enable_annotations）

此外，Kata Containers 支持 OCI 和容器级别的配置，例如：

**OCI 配置**

| 配置项                                   | 含义                                                |
| ---------------------------------------- | --------------------------------------------------- |
| io.katacontainers.config_path            | Kata Containers 配置文件路径                        |
| io.katacontainers.pkg.oci.bundle_path    | OCI bundle 路径                                     |
| io.katacontainers.pkg.oci.container_type | OCI 容器类型，可选的有 pod_container 和 pod_sandbox |

**容器配置**

| 配置项                                             | 含义                                                         |
| -------------------------------------------------- | ------------------------------------------------------------ |
| io.katacontainers.container.resource.swappiness    | 即 Resources.Memory.Swappiness，用于配置容器内存管理器在何时将内存页面写入 SWAP 空间的一个相对度量。该参数的值介于 0 和 100 之间，表示内存页面的使用频率 |
| io.katacontainers.container.resource.swap_in_bytes | 即 Resources.Memory.Swap，用于配置容器可以使用的 SWAP 空间的大小 |

例如，通过 annotation 启动一个忽略底层默认大小，具有 5 CPUs 的 VM：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kata
  annotations:
    io.katacontainers.config.hypervisor.default_vcpus: "5"
spec:
  runtimeClassName: kata
  containers:
  - name: kata
    image: busybox
    command: ["/bin/sh", "-c", "tail -f /dev/null"]
```

# 与 Container Manager 集成

## Docker

*TODO：Docker 23.0.0 版本，新增了运行时 shim 的支持，也就支持了 Kata Containers*

## Containerd

```shell
# 生成 Containerd 默认的配置文件
$ sudo mkdir -p /etc/containerd
$ containerd config default | sudo tee /etc/containerd/config.toml
```

可以看到，Containerd 的默认底层运行时为 runC，新增以下内容支持 Kata Containers：

```toml
 [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
        runtime_type = "io.containerd.kata.v2"
        privileged_without_host_devices = true
        pod_annotations = ["io.katacontainers.*"]
        container_annotations = ["io.katacontainers.*"]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata.options]
           ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration.toml"
```

## CRI-O

*TODO*

至此，可以单独通过 Container Manager 各自的命令行运行 Kata Containers，以 Containerd 为例：

```shell
$ sudo ctr image pull docker.io/library/ubuntu:latest
$ sudo ctr run --runtime io.containerd.run.kata.v2 -t --rm docker.io/library/ubuntu:latest hello sh -c "free -h"
$ sudo ctr run --runtime io.containerd.run.kata.v2 -t --memory-limit 536870912 --rm docker.io/library/ubuntu:latest hello sh -c "free -h"
```

# 与 Kubernetes 集成

Kubernetes 中对于运行时的集成是通过 [RuntimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/) 资源对象，例如

```yaml
kind: RuntimeClass
apiVersion: node.k8s.io/v1
metadata:
  name: kata-containers
handler: kata
overhead:
  podFixed:
    memory: "140Mi"
    cpu: "250m"
scheduling:
  nodeSelector:
    runtime: kata
```

## handler

需要和 CRI 中注册的 handler（HANDLER_NAME） 保持一致。

**Containerd**

```toml
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.${HANDLER_NAME}]
```

**CRI-O**

```toml
[crio.runtime.runtimes.${HANDLER_NAME}]
```

## scheduling

通过为 RuntimeClass 指定 scheduling 字段， 可以通过设置约束，确保运行该 RuntimeClass 的 Pod 被调度到支持该 RuntimeClass 的节点上。 如果未设置 scheduling，则假定所有节点均支持此 RuntimeClass 。

为了确保 Pod 会被调度到支持指定运行时的节点上，每个节点需要设置一个通用的 label 用于被 runtimeclass.scheduling.nodeSelector 挑选。在 admission 阶段，RuntimeClass 的 nodeSelector 将会与 Pod 的 nodeSelector 合并，取二者的交集。如果有冲突，Pod 将会被拒绝。

如果节点需要阻止某些需要特定 RuntimeClass 的 Pod，可以在 tolerations 中指定。 与 nodeSelector 一样，tolerations 也在 admission 阶段与 Pod 的 tolerations 合并，取二者的并集。

## overhead

在节点上运行 Pod 时，Pod 本身占用大量系统资源。这些资源是运行 Pod 内容器所需资源的附加资源。Overhead 是一个特性，用于计算 Pod 基础设施在容器请求和限制之上消耗的资源。

在 Kubernetes 中，Pod 的开销是根据与 Pod 的 RuntimeClass 相关联的开销在准入控制时设置的。

如果启用了 Pod Overhead，在调度 Pod 时，除了考虑容器资源请求的总和外，还要考虑 Pod 开销。 类似地，Kubelet 将在确定 Pod cgroups 的大小和执行 Pod 驱逐排序时也会考虑 Pod 开销。

# VM factory

## VMCache

VMCache 是一项新功能，可在使用前将 VM 创建为缓存。它有助于加快新容器的创建。

该功能由借助 Unix socket 通信的一个 gRPC Server 和一些 Client 组成。

VMCache server 将事先创建并缓存一些 VM，它将 VM 转换为 gRPC 格式并在收到 client 请求时返回；grpccache factory 是 VMCache 客户端，它将请求到的 gRPC 格式的 VM 并将其转换回 VM。如果启用了 VMCache 功能，Kata 运行时在创建新的 sandbox 时会向 grpccache 请求获取 VM。

**与 VM tmplating 的区别**

VM tmplating 和 VMCache 都有助于加快新容器的创建。

当启用 VM tmplating 时，通过从预先创建的模板 VM 克隆来创建新的 VM，它们将以只读模式共享相同的 initramfs、内核和 agent 内存。因此，如果在同一主机上运行许多 Kata 容器，它会节省大量内存。

而 VMCache 不容易受到共享内存 CVE 的影响，因为每个 VM 不共享内存。

**如何启用 VM Cache**

配置文件中修改以下配置项：

- [factory].vm_cache_number 指定 VM 缓存的个数
- [factory].vm_cache_endpoint 指定 socket 地址（自动创建），默认为 /var/run/kata-containers/cache.sock

通过以下命令创建一个 VM 模板供以后使用，通过 CTRL+C 退出：

```shell
$ kata-runtime factory init
```

区别于 VM templating，VMCache 创建的 VM 是处于运行状态，而非保存在 [factory].template_path 目录下

```shell
$ kata-runtime factory status
VM cache server pid = 38308
VM pid = 38334 Cpu = 1 Memory = 2048MiB
VM pid = 38331 Cpu = 1 Memory = 2048MiB
VM pid = 38332 Cpu = 1 Memory = 2048MiB
vm factory not enabled

$ ls -la /run/vc/vm
a78a9744-5984-4e54-bda9-9b6280bf9a3f
41648333-a4a3-48ee-b80b-f7c19e3081b1
57d2dd69-0e73-4779-b23f-68ee8e4f66de
template
```

```shell
$ kata-runtime factory destroy
vm factory destroyed
```

**已知限制**

- 无法与 VM templating 共存
- 仅支持 QEMU 作为 hypervisor
- [hypervisor].shared_fs 为 virtio-9p（社区有支持 virtio-fs 的提案 https://github.com/kata-containers/kata-containers/pull/4522，但截至 Kata 3.0.0 暂未合入）

*经验证，截至 Kata 3.0.0，VMCache 并不能开箱即用，在 VMCache 流程中部分变量缺少赋值，导致代码报错*

## VM templating

VM templating 是 Kata Containers 的一项功能，可以借助克隆技术创建新的 VM。启用后，新的 VM 将通过从预先创建的模板进行克隆来创建，它们将以只读模式共享相同的 initramfs、内核和 agent 内存。类似于内核的 fork 进程操作，这里 fork 的是 VM。

**与 VMCache 的区别**

VMCache 和 VM templating 都有助于加快新容器的创建。

启用 VMCache 后，VMCache 服务器会创建新的 VM。所以它不容易受到共享内存 CVE 的攻击，因为每个 VM 都不共享内存。

如果在同一主机上运行许多 Kata 容器，VM templating 可以节省大量内存。

**优势**

如果在同一主机上运行许多 Kata 容器，VM templating 有助于加快新容器的创建并节省大量内存。如果正在运行高密度工作负载，或者非常关心容器启动速度，VM templating 可能非常有用。

在一个[示例](https://github.com/kata-containers/runtime/pull/303#issuecomment-395846767)中，创建了 100 个 Kata 容器，每个容器都拥有 128MB 的 VM 内存，并且在启用 VM templating 特性时最终总共节省了 9GB 的内存，这大约是 VM 内存总量的 72%。

在另一个[示例](https://gist.github.com/bergwolf/06974a3c5981494a40e2c408681c085d)中，创建了 10 个 Kata 容器，并计算了每个容器的平均启动速度。结果表明，VM templating 将 Kata 容器的创建速度提高了 38.68%。

**不足**

VM templating 的一个缺点是它无法避免跨 VM 侧通道攻击，例如最初针对 Linux KSM 功能的 CVE-2015-2877。得出的结论是，“相互不信任的租户之间用于内存保护的共享直到写入的方法本质上是可检测的信息泄露，并且可以归类为潜在的被误解的行为而不是漏洞。”如果对此敏感，不要使用 VM templating 或 KSM。

**如何启用 VM templating**

配置文件中修改以下配置项：

- hypervisor 为 qemu，且版本为 v4.1.0 以上
- [factory].enable_template 设为 true
- VM 镜像为 initrd 类型，即为 [hypervisor].initrd
- [hypervisor].shared_fs 为 virtio-9p

通过以下命令创建一个 VM 模板：

```shell
$ kata-runtime factory init
vm factory initialized
```

创建的模板默认保存在 /run/vc/vm/template，可以通过 [factory].template_path 指定：

```shell
$ ls /run/vc/vm/template
memory  state
```

模板通过以下命令销毁：

```shell
$ kata-runtime factory destroy
vm factory destroyed
```

如果不想手动调用 kata-runtime factory init，在启用 VM templating 后，默认创建的第一个 Kata 容器将自动创建一个 VM 模板。

# kata-runtime

kata-runtime 是一个命令行工具，支持以下功能：

## check (kata-check)

检测当前环境是否可以运行 Kata Containers 以及版本是否正确。

```shell
$ kata-runtime check --verbose
INFO[0000] IOMMUPlatform is disabled by default.        
WARN[0000] Not running network checks as super user      arch=amd64 name=kata-runtime pid=29825 source=runtime
INFO[0000] CPU property found                            arch=amd64 description="Intel Architecture CPU" name=GenuineIntel pid=29825 source=runtime type=attribute
INFO[0000] CPU property found                            arch=amd64 description="Virtualization support" name=vmx pid=29825 source=runtime type=flag
INFO[0000] CPU property found                            arch=amd64 description="64Bit CPU" name=lm pid=29825 source=runtime type=flag
INFO[0000] CPU property found                            arch=amd64 description=SSE4.1 name=sse4_1 pid=29825 source=runtime type=flag
INFO[0000] kernel property found                         arch=amd64 description="Intel KVM" name=kvm_intel pid=29825 source=runtime type=module
INFO[0000] kernel property found                         arch=amd64 description="Kernel-based Virtual Machine" name=kvm pid=29825 source=runtime type=module
INFO[0000] kernel property found                         arch=amd64 description="Host kernel accelerator for virtio" name=vhost pid=29825 source=runtime type=module
INFO[0000] kernel property found                         arch=amd64 description="Host kernel accelerator for virtio network" name=vhost_net pid=29825 source=runtime type=module
INFO[0000] kernel property found                         arch=amd64 description="Host Support for Linux VM Sockets" name=vhost_vsock pid=29825 source=runtime type=module
System is capable of running Kata Containers
INFO[0000] device available                              arch=amd64 check-type=full device=/dev/kvm name=kata-runtime pid=29825 source=runtime
INFO[0000] feature available                             arch=amd64 check-type=full feature=create-vm name=kata-runtime pid=29825 source=runtime
System can currently create Kata Containers
```

可选的 flags 包括：

| 名称                    | 含义                                                         |
| ----------------------- | ------------------------------------------------------------ |
| --check-version-only    | 仅对比前使用版本和最新可用版本（需要网络支持，且非 root 用户） |
| --include-all-releases  | 包含过滤预发布的版本                                         |
| --no-network-checks, -n | 不借助网络执行检测，该参数等价于设置 KATA_CHECK_NO_NETWORK 环境变量 |
| --only-list-releases    | 仅列出较新的可用版本（需要网络支持，且非 root 用户）         |
| --strict, -s            | 进行严格检查                                                 |
| --verbose, -v           | 展示详细的检查项                                             |

## env (kata-env)

Kata Containers 配置展示，默认输出格式为 TOML。

```shell
$ kata-runtime env 
[Kernel]
  Path = "/opt/kata/share/kata-containers/vmlinux-5.19.2-96"
  Parameters = "systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.log=debug agent.debug_console agent.debug_console_vport=1026"

[Meta]
  Version = "1.0.26"

[Image]
  Path = "/opt/kata/share/kata-containers/kata-clearlinux-latest.image"

[Initrd]
  Path = ""

[Hypervisor]
  MachineType = "q35"
  Version = "QEMU emulator version 6.2.0 (kata-static)\nCopyright (c) 2003-2021 Fabrice Bellard and the QEMU Project developers"
  Path = "/opt/kata/bin/qemu-system-x86_64"
  BlockDeviceDriver = "virtio-scsi"
  EntropySource = "/dev/urandom"
  SharedFS = "virtio-fs"
  VirtioFSDaemon = "/opt/kata/libexec/virtiofsd"
  SocketPath = ""
  Msize9p = 8192
  MemorySlots = 10
  PCIeRootPort = 2
  HotplugVFIOOnRootBus = true
  Debug = true

[Runtime]
  Path = "/usr/local/bin/kata-runtime"
  Debug = true
  Trace = false
  DisableGuestSeccomp = true
  DisableNewNetNs = false
  SandboxCgroupOnly = false
  [Runtime.Config]
    Path = "/etc/kata-containers/configuration.toml"
  [Runtime.Version]
    OCI = "1.0.2-dev"
    [Runtime.Version.Version]
      Semver = "3.0.0"
      Commit = "e2a8815ba46360acb8bf89a2894b0d437dc8548a-dirty"
      Major = 3
      Minor = 0
      Patch = 0

[Host]
  Kernel = "4.18.0-305.43.25.ar.el7.x86_64"
  Architecture = "amd64"
  VMContainerCapable = true
  SupportVSocks = true
  [Host.Distro]
    Name = "CentOS Linux"
    Version = "7"
  [Host.CPU]
    Vendor = "GenuineIntel"
    Model = "QEMU Virtual CPU version (cpu64-rhel6)"
    CPUs = 8
  [Host.Memory]
    Total = 12057632
    Free = 3352124
    Available = 8508112

[Agent]
  Debug = true
  Trace = false
```

可选的 flags 包括

| 名称   | 含义             |
| ------ | ---------------- |
| --json | 以 JSON 格式展示 |

## exec

借助 debug console，进入 VM 控制台，需要 [agent].debug_console_enabled 设置为 true。

```shell
# 对于 Pod 而言是其 SandboxID
$ kata-runtime exec 27ab74433f11c0b64e404a841d5e2f8296a723ebfa4e598b4d9d32871173b82c
```

可选的 flags 包括

| 名称              | 含义                                         |
| ----------------- | -------------------------------------------- |
| --kata-debug-port | debug console 监听的端口，默认为 1026 或者 0 |

## metrics

收集与用于运行 sandbox 的基础设施相关的指标，例如 runtime、agent、hypervisor 等。

```shell
# 对于 Pod 而言是其 SandboxID
$ kata-runtime metrics 27ab74433f11c0b64e404a841d5e2f8296a723ebfa4e598b4d9d32871173b82c
# HELP kata_hypervisor_fds Open FDs for hypervisor.
# TYPE kata_hypervisor_fds gauge
kata_hypervisor_fds 122
# HELP kata_hypervisor_io_stat Process IO statistics.
# TYPE kata_hypervisor_io_stat gauge
kata_hypervisor_io_stat{item="cancelledwritebytes"} 0
kata_hypervisor_io_stat{item="rchar"} 5.915546e+06
kata_hypervisor_io_stat{item="readbytes"} 1.1665408e+07
kata_hypervisor_io_stat{item="syscr"} 95522
kata_hypervisor_io_stat{item="syscw"} 202276
kata_hypervisor_io_stat{item="wchar"} 3.715404e+06
kata_hypervisor_io_stat{item="writebytes"} 2.097152e+06
```

## direct-volume

管理 Kata Containers 的直通卷。*具体使用方式参考 **Kata Containers Block Volume 直通**说明。*

**add**

```shell
$ kata-runtime direct-volume add --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount --mount-info \{\"volume-type\":\"block\",\"device\":\"/dev/sdm\",\"fstype\":\"xfs\"\}
```

可选的 flags 包括

| 名称          | 含义                 |
| ------------- | -------------------- |
| --volume-path | 待操作的目标卷路径   |
| --mount-info  | 管理卷挂载的详情信息 |

**remove**

```shell
$ kata-runtime direct-volume delete --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount
```

可选的 flags 包括

| 名称          | 含义               |
| ------------- | ------------------ |
| --volume-path | 待操作的目标卷路径 |

**stats**

```shell
$ kata-runtime direct-volume stats --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount
```

可选的 flags 包括

| 名称          | 含义               |
| ------------- | ------------------ |
| --volume-path | 待操作的目标卷路径 |

**resize**

*截至 Kata Containers 3.0.0，社区仍未实现 VM 中 Kata agent 的逻辑*

```shell
$ kata-runtime direct-volume resize --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount --size 1756519562
```

可选的 flags 包括

| 名称          | 含义                               |
| ------------- | ---------------------------------- |
| --volume-path | 待操作的目标卷路径                 |
| --size        | 调整后的预期卷大小（单位为：Byte） |

## factory

管理 Kata Containers 的 VM factory。*具体使用方式参考 **VM factory** 说明。*

**init**

```shell
$ kata-runtime factory init
vm factory initialized
```

**status**

```shell
$ kata-runtime factory status
vm factory is on
```

**destroy**

```shell
$ kata-runtime factory destroy
vm factory destroyed
```

## iptables

管理 VM 中的 iptables 信息。

**get**

```shell
$ kata-runtime iptables get --sandbox-id xxx --v6
```

可选的 flags 包括

| 名称         | 含义                  |
| ------------ | --------------------- |
| --sandbox-id | 待操作的 Sandbox ID   |
| --v6         | 获取 IPV6 的 iptables |

**set**

```shell
$ kata-runtime iptables set --sandbox-id xxx --v6 ./iptables
```

可选的 flags 包括

| 名称         | 含义                  |
| ------------ | --------------------- |
| --sandbox-id | 待操作的 Sandbox ID   |
| --v6         | 设置 IPV6 的 iptables |

# kata-monitor

Kata monitor 是一个守护进程，能够收集和暴露在同一 host 上运行的所有 Kata 容器工作负载相关的指标。

```shell
$ kata-monitor
INFO[0000] announce                                      app=kata-monitor arch=amd64 git-commit=fcad969e5200607df3b0b31983cc64488e156e99 go-version=go1.16.10 listen-address="127.0.0.1:8090" log-level=info os=linux runtime-endpoint=/run/containerd/containerd.sock version=0.3.0
```

可选的 flags 包括

| 名称               | 含义                                                         |
| ------------------ | ------------------------------------------------------------ |
| --listen-address   | 监听 HTTP 请求的地址，默认为 127.0.0.1:8090                  |
| --log-level        | 服务日志级别，可选有 trace/debug/info/warn/error/fatal/panic，默认为 info |
| --runtime-endpoint | CRI 容器运行时服务的 socket 地址，默认为 /run/containerd/containerd.sock |
