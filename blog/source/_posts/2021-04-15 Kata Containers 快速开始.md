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

- 动态配置项是通过 OCI spec 中的 annotations 传递，主流的 CRI 实现支持将 Kubernetes Pod annotations 透传至 Kata 运行时
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
| confidential_guest           | N        | 是否启用机密容器特性。机密容器需要 host 支持 tdxProtection（[Intel Trust Domain Extensions](https://software.intel.com/content/www/us/en/develop/articles/intel-trust-domain-extensions.html)）、sevProtection（[AMD Secure Encrypted Virtualization](https://developer.amd.com/sev/)）、pefProtection（[IBM POWER 9 Protected Execution Facility](https://www.kernel.org/doc/html/latest/powerpc/ultravisor.html)）以及 seProtection（[IBM Secure Execution (IBM Z & LinuxONE)](https://www.kernel.org/doc/html/latest/virt/kvm/s390-pv.html)）。不支持 CPU 和内存的热插拔以及 NVDIMM 设备。不支持 arm64 架构 |
| rootless                     | Y        | 是否以非 root 权限的随机用户启动 QEMU VMM                    |
| enable_annotations           | N        | 允许 hypervisor 动态配置的配置项                             |
| valid_hypervisor_paths       | N        | 以 glob(3) 规则校验 path 参数是否合法的路径集合              |
| kernel_params                | Y        | VM kernel 的附加参数                                         |
| firmware                     | Y        |                                                              |
| firmware_volume              | Y        |                                                              |
| machine_accelerators         | Y        | 机器加速器参数                                               |
| seccompsandbox               | N        | seccomp 参数。QEMU seccomp sandbox 是 QEMU VM 中的一种安全特性，通过限制 QEMU 进程的系统调用，以提高 VM 的安全性。它使用了 Linux 内核提供的 seccomp 机制，将 QEMU 进程限制在一组安全的系统调用中，从而降低 VM 遭受攻击的风险。推荐启用 /proc/sys/net/core/bpf_jit_enable，以降低该特性带来的性能下降 |
| cpu_features                 | Y        | CPU 特性参数，例如配置文件中默认的 pmu=off 参数用于禁用 VM 中的性能监视器单元（Performance Monitoring Unit，PMU）。PMU 是一种硬件设备，用于监控 CPU 的性能指标，如指令执行次数、缓存命中率等。在某些情况下，PMU 可能会被用于进行侧信道攻击或窃取敏感信息 |
| default_vcpus                | Y        | VM 默认的 CPU 数量，默认为 1，最大为 host CPU 数量           |
| default_maxvcpus             | Y        | VM 最大的 CPU 数量，默认为 host CPU 数量，具体能否使用到 host CPU 数量，还需要视 hypervisor 限制而定。过大的 CPU 数量会影响到 VM 的性能以及内存占比 |
| default_bridges              | N        | VM 默认的 PCI 桥数量，默认为 1，最大为 5。目前，仅支持 PCI bridge，每个 PCI bridge 最多支持 30 个设备的热插拔，每个 VM 最多支持 5 个 PCI bridge（这可能是 QEMU 或内核中的一个 bug） |
| default_memory               | Y        | VM 默认的内存总量，默认为 1，最大为 host 内存总量            |
| memory_slots                 | Y        | VM 默认的内存插槽数量，默认为 10，即内存热添加数量的上限为 10 |
| default_maxmemory            | Y        | VM 最大的内存总量，默认为  host 内存总量                     |
| memory_offset                | Y        | VM 内存偏移量，用于描述 NVDIMM 设备的内存空间，当 block_device_driver 为 nvdimm 时，需要设置此参数，最终会追加到 default_maxmemory 中 |
| enable_virtio_mem            | Y        |                                                              |
| disable_block_device_use     |          |                                                              |
| shared_fs                    |          |                                                              |
| virtio_fs_daemon             |          |                                                              |
| valid_virtio_fs_daemon_paths |          |                                                              |
| virtio_fs_cache_size         |          |                                                              |
| virtio_fs_extra_args         |          |                                                              |
| virtio_fs_cache              |          |                                                              |
| block_device_driver          |          |                                                              |
| block_device_aio             |          |                                                              |
| block_device_cache_set       |          |                                                              |
| block_device_cache_direct    |          |                                                              |
| block_device_cache_noflush   |          |                                                              |
| enable_iothreads             |          |                                                              |
| enable_mem_prealloc          |          |                                                              |
| enable_hugepages             |          |                                                              |
| enable_vhost_user_store      |          |                                                              |
| vhost_user_store_path        |          |                                                              |
| enable_iommu                 |          |                                                              |
| enable_iommu_platform        |          |                                                              |
| valid_vhost_user_store_paths |          |                                                              |
| file_mem_backend             |          |                                                              |
| valid_file_mem_backends      |          |                                                              |
| pflashes                     |          |                                                              |
| enable_debug                 |          |                                                              |
| disable_nesting_checks       |          |                                                              |
| msize_9p                     |          |                                                              |
| disable_image_nvdimm         |          |                                                              |
| hotplug_vfio_on_root_bus     |          |                                                              |
| pcie_root_port               |          |                                                              |
| disable_vhost_net            |          |                                                              |
| entropy_source               |          |                                                              |
| valid_entropy_sources        |          |                                                              |
| guest_hook_path              |          |                                                              |
| rx_rate_limiter_max_rate     |          |                                                              |
| tx_rate_limiter_max_rate     |          |                                                              |
| guest_memory_dump_path       |          |                                                              |
| guest_memory_dump_paging     |          |                                                              |
| enable_guest_swap            |          |                                                              |
| use_legacy_serial            |          |                                                              |
| disable_selinux              |          |                                                              |

# CRI 配置

Kata Containers 在与 Kubernetes 集成时，默认支持 Containerd 和 CRI-O 作为 CRI，不支持使用 docker-shim 作为 CRI。

## Containerd

*/etc/containerd/config.toml*

在 Docker（docker-shim）作为 CRI 的场景下，Containerd 本身也是 Docker 的组件之一，但是禁用了 Containerd 作为 CRI。

**非 CRI**

默认安装 Docker 服务时，会自动安装 Containerd，配置文件如下：

```toml
#   Copyright 2018-2022 Docker Inc.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

disabled_plugins = ["cri"]

#root = "/var/lib/containerd"
#state = "/run/containerd"
#subreaper = true
#oom_score = 0

#[grpc]
#  address = "/run/containerd/containerd.sock"
#  uid = 0
#  gid = 0

#[debug]
#  address = "/run/containerd/debug.sock"
#  uid = 0
#  gid = 0
#  level = "info"
```

**CRI**

借助 Containerd 自带的配置生成能力，创建其作为 CRI 的配置文件：

```shell
$ sudo mkdir -p /etc/containerd
$ containerd config default | sudo tee /etc/containerd/config.toml
```

```toml
disabled_plugins = []
imports = []
oom_score = 0
plugin_dir = ""
required_plugins = []
root = "/var/lib/containerd"
state = "/run/containerd"
temp = ""
version = 2

[cgroup]
  path = ""

[debug]
  address = ""
  format = ""
  gid = 0
  level = ""
  uid = 0

[grpc]
  address = "/run/containerd/containerd.sock"
  gid = 0
  max_recv_message_size = 16777216
  max_send_message_size = 16777216
  tcp_address = ""
  tcp_tls_ca = ""
  tcp_tls_cert = ""
  tcp_tls_key = ""
  uid = 0

[metrics]
  address = ""
  grpc_histogram = false

[plugins]

  [plugins."io.containerd.gc.v1.scheduler"]
    deletion_threshold = 0
    mutation_threshold = 100
    pause_threshold = 0.02
    schedule_delay = "0s"
    startup_delay = "100ms"

  [plugins."io.containerd.grpc.v1.cri"]
    device_ownership_from_security_context = false
    disable_apparmor = false
    disable_cgroup = false
    disable_hugetlb_controller = true
    disable_proc_mount = false
    disable_tcp_service = true
    enable_selinux = false
    enable_tls_streaming = false
    enable_unprivileged_icmp = false
    enable_unprivileged_ports = false
    ignore_image_defined_volumes = false
    max_concurrent_downloads = 3
    max_container_log_line_size = 16384
    netns_mounts_under_state_dir = false
    restrict_oom_score_adj = false
    sandbox_image = "registry.k8s.io/pause:3.6"
    selinux_category_range = 1024
    stats_collect_period = 10
    stream_idle_timeout = "4h0m0s"
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    systemd_cgroup = false
    tolerate_missing_hugetlb_controller = true
    unset_seccomp_profile = ""

    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      conf_template = ""
      ip_pref = ""
      max_conf_num = 1

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      disable_snapshot_annotations = true
      discard_unpacked_layers = false
      ignore_rdt_not_enabled_errors = false
      no_pivot = false
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        base_runtime_spec = ""
        cni_conf_dir = ""
        cni_max_conf_num = 0
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        runtime_engine = ""
        runtime_path = ""
        runtime_root = ""
        runtime_type = ""

        [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime.options]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          base_runtime_spec = ""
          cni_conf_dir = ""
          cni_max_conf_num = 0
          container_annotations = []
          pod_annotations = []
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_path = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = ""
            CriuImagePath = ""
            CriuPath = ""
            CriuWorkPath = ""
            IoGid = 0
            IoUid = 0
            NoNewKeyring = false
            NoPivotRoot = false
            Root = ""
            ShimCgroup = ""
            SystemdCgroup = false

      [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime]
        base_runtime_spec = ""
        cni_conf_dir = ""
        cni_max_conf_num = 0
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        runtime_engine = ""
        runtime_path = ""
        runtime_root = ""
        runtime_type = ""

        [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime.options]

    [plugins."io.containerd.grpc.v1.cri".image_decryption]
      key_model = "node"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = ""

      [plugins."io.containerd.grpc.v1.cri".registry.auths]

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

      [plugins."io.containerd.grpc.v1.cri".registry.headers]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]

    [plugins."io.containerd.grpc.v1.cri".x509_key_pair_streaming]
      tls_cert_file = ""
      tls_key_file = ""

  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"

  [plugins."io.containerd.internal.v1.restart"]
    interval = "10s"

  [plugins."io.containerd.internal.v1.tracing"]
    sampling_ratio = 1.0
    service_name = "containerd"

  [plugins."io.containerd.metadata.v1.bolt"]
    content_sharing_policy = "shared"

  [plugins."io.containerd.monitor.v1.cgroups"]
    no_prometheus = false

  [plugins."io.containerd.runtime.v1.linux"]
    no_shim = false
    runtime = "runc"
    runtime_root = ""
    shim = "containerd-shim"
    shim_debug = false

  [plugins."io.containerd.runtime.v2.task"]
    platforms = ["linux/amd64"]
    sched_core = false

  [plugins."io.containerd.service.v1.diff-service"]
    default = ["walking"]

  [plugins."io.containerd.service.v1.tasks-service"]
    rdt_config_file = ""

  [plugins."io.containerd.snapshotter.v1.aufs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.btrfs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.devmapper"]
    async_remove = false
    base_image_size = ""
    discard_blocks = false
    fs_options = ""
    fs_type = ""
    pool_name = ""
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.native"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.overlayfs"]
    root_path = ""
    upperdir_label = false

  [plugins."io.containerd.snapshotter.v1.zfs"]
    root_path = ""

  [plugins."io.containerd.tracing.processor.v1.otlp"]
    endpoint = ""
    insecure = false
    protocol = ""

[proxy_plugins]

[stream_processors]

  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar"]
    accepts = ["application/vnd.oci.image.layer.v1.tar+encrypted"]
    args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
    env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
    path = "ctd-decoder"
    returns = "application/vnd.oci.image.layer.v1.tar"

  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar.gzip"]
    accepts = ["application/vnd.oci.image.layer.v1.tar+gzip+encrypted"]
    args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
    env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
    path = "ctd-decoder"
    returns = "application/vnd.oci.image.layer.v1.tar+gzip"

[timeouts]
  "io.containerd.timeout.bolt.open" = "0s"
  "io.containerd.timeout.shim.cleanup" = "5s"
  "io.containerd.timeout.shim.load" = "5s"
  "io.containerd.timeout.shim.shutdown" = "3s"
  "io.containerd.timeout.task.state" = "2s"

[ttrpc]
  address = ""
  gid = 0
  uid = 0
```

可以看到，Containerd 的默认 OCI 运行时为 runC，可以通过新增以下内容，用于对 Kata Containers 的支持：

```toml
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
        runtime_type = "io.containerd.kata.v2"
        privileged_without_host_devices = true
        pod_annotations = ["io.katacontainers.*"]
        container_annotations = ["io.katacontainers.*"]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata.options]
           ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration.toml"、
```

## CRI-O

TODO

# RuntimeClass

RuntimeClass 是一个用于选择容器运行时配置的特性，容器运行时配置用于运行 Pod 中的容器。

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

需要和 CRI 中注册的 handler（HANDLER_NAME） 保持一致，用于声明由具体实现的 runtime。

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

在 Kubernetes 中，Pod 的开销是根据与 Pod 的 [RuntimeClass](https://kubernetes.io/zh/docs/concepts/containers/runtime-class/) 相关联的开销在[准入](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks)时设置的。

如果启用了 Pod Overhead，在调度 Pod 时，除了考虑容器资源请求的总和外，还要考虑 Pod 开销。 类似地，kubelet 将在确定 Pod cgroups 的大小和执行 Pod 驱逐排序时也会考虑 Pod 开销。

# Pod

### 定制化的 annotation

Kata Containers 可以通过 Pod annotation 的方式实现定制化每一个 Pod 的底层 Kata 参数。需要做的是上层 CRI 将 Pod annotation 透传至底层 runtime，同时 Kata Containers 开启识别特定的 Pod annotation，并且 CRI 需要支持此功能（如 Containerd 依赖 1.4.x 以上的版本才可以，且对应的 runtime 配置中新增相关 annotations 支持；CRI-O 默认透传所有参数，无需额外配置）

**全局配置**

| Key                                        | Value Type | Comments                                                     |
| ------------------------------------------ | ---------- | ------------------------------------------------------------ |
| `io.katacontainers.config_path`            | string     | Kata config file location that overrides the default config paths |
| `io.katacontainers.pkg.oci.bundle_path`    | string     | OCI bundle path                                              |
| `io.katacontainers.pkg.oci.container_type` | string     | OCI container type. Only accepts `pod_container` and `pod_sandbox` |

**Runtime 配置**

| Key                                                      | Value Type | Comments                                                     |
| -------------------------------------------------------- | ---------- | ------------------------------------------------------------ |
| `io.katacontainers.config.runtime.experimental`          | `boolean`  | determines if experimental features enabled                  |
| `io.katacontainers.config.runtime.disable_guest_seccomp` | `boolean`  | determines if `seccomp` should be applied inside guest       |
| `io.katacontainers.config.runtime.disable_new_netns`     | `boolean`  | determines if a new netns is created for the hypervisor process |
| `io.katacontainers.config.runtime.internetworking_model` | string     | determines how the VM should be connected to the container network interface. Valid values are `macvtap`, `tcfilter` and `none` |
| `io.katacontainers.config.runtime.sandbox_cgroup_only`   | `boolean`  | determines if Kata processes are managed only in sandbox cgroup |
| `io.katacontainers.config.runtime.enable_pprof`          | `boolean`  | enables Golang `pprof` for `containerd-shim-kata-v2` process |

**Agent 配置**

| Key                                                  | Value Type | Comments                                                     |
| ---------------------------------------------------- | ---------- | ------------------------------------------------------------ |
| `io.katacontainers.config.agent.enable_tracing`      | `boolean`  | enable tracing for the agent                                 |
| `io.katacontainers.config.agent.container_pipe_size` | uint32     | specify the size of the std(in/out) pipes created for containers |
| `io.katacontainers.config.agent.kernel_modules`      | string     | the list of kernel modules and their parameters that will be loaded in the guest kernel. Semicolon separated list of kernel modules and their parameters. These modules will be loaded in the guest kernel using `modprobe`(8). E.g., `e1000e InterruptThrottleRate=3000,3000,3000 EEE=1; i915 enable_ppgtt=0` |

**Hypervisor 配置**

| Key                                                          | Value Type                                                   | Comments                                                     |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `io.katacontainers.config.hypervisor.asset_hash_type`        | string                                                       | the hash type used for assets verification, default is `sha512` |
| `io.katacontainers.config.hypervisor.block_device_cache_direct` | `boolean`                                                    | Denotes whether use of `O_DIRECT` (bypass the host page cache) is enabled |
| `io.katacontainers.config.hypervisor.block_device_cache_noflush` | `boolean`                                                    | Denotes whether flush requests for the device are ignored    |
| `io.katacontainers.config.hypervisor.block_device_cache_set` | `boolean`                                                    | cache-related options will be set to block devices or not    |
| `io.katacontainers.config.hypervisor.block_device_driver`    | string                                                       | the driver to be used for block device, valid values are `virtio-blk`, `virtio-scsi`, `nvdimm` |
| `io.katacontainers.config.hypervisor.cpu_features`           | `string`                                                     | Comma-separated list of CPU features to pass to the CPU (QEMU) |
| `io.katacontainers.config.hypervisor.ctlpath` (R)            | `string`                                                     | Path to the `acrnctl`binary for the ACRN hypervisor          |
| `io.katacontainers.config.hypervisor.default_max_vcpus`      | uint32                                                       | the maximum number of vCPUs allocated for the VM by the hypervisor |
| `io.katacontainers.config.hypervisor.default_memory`         | uint32                                                       | the memory assigned for a VM by the hypervisor in `MiB`      |
| `io.katacontainers.config.hypervisor.default_vcpus`          | uint32                                                       | the default vCPUs assigned for a VM by the hypervisor        |
| `io.katacontainers.config.hypervisor.disable_block_device_use` | `boolean`                                                    | disallow a block device from being used                      |
| `io.katacontainers.config.hypervisor.disable_image_nvdimm`   | `boolean`                                                    | specify if a `nvdimm` device should be used as rootfs for the guest (QEMU) |
| `io.katacontainers.config.hypervisor.disable_vhost_net`      | `boolean`                                                    | specify if `vhost-net` is not available on the host          |
| `io.katacontainers.config.hypervisor.enable_hugepages`       | `boolean`                                                    | if the memory should be `pre-allocated` from huge pages      |
| `io.katacontainers.config.hypervisor.enable_iommu_platform`  | `boolean`                                                    | enable `iommu` on CCW devices (QEMU s390x)                   |
| `io.katacontainers.config.hypervisor.enable_iommu`           | `boolean`                                                    | enable `iommu` on Q35 (QEMU x86_64)                          |
| `io.katacontainers.config.hypervisor.enable_iothreads`       | `boolean`                                                    | enable IO to be processed in a separate thread. Supported currently for virtio-`scsi` driver |
| `io.katacontainers.config.hypervisor.enable_mem_prealloc`    | `boolean`                                                    | the memory space used for `nvdimm` device by the hypervisor  |
| `io.katacontainers.config.hypervisor.enable_vhost_user_store` | `boolean`                                                    | enable vhost-user storage device (QEMU)                      |
| `io.katacontainers.config.hypervisor.enable_virtio_mem`      | `boolean`                                                    | enable virtio-mem (QEMU)                                     |
| `io.katacontainers.config.hypervisor.entropy_source` (R)     | string                                                       | the path to a host source of entropy (`/dev/random`, `/dev/urandom` or real hardware RNG device) |
| `io.katacontainers.config.hypervisor.file_mem_backend` (R)   | string                                                       | file based memory backend root directory                     |
| `io.katacontainers.config.hypervisor.firmware_hash`          | string                                                       | container firmware SHA-512 hash value                        |
| `io.katacontainers.config.hypervisor.firmware`               | string                                                       | the guest firmware that will run the container VM            |
| `io.katacontainers.config.hypervisor.firmware_volume_hash`   | string                                                       | container firmware volume SHA-512 hash value                 |
| `io.katacontainers.config.hypervisor.firmware_volume`        | string                                                       | the guest firmware volume that will be passed to the container VM |
| `io.katacontainers.config.hypervisor.guest_hook_path`        | string                                                       | the path within the VM that will be used for drop in hooks   |
| `io.katacontainers.config.hypervisor.hotplug_vfio_on_root_bus` | `boolean`                                                    | indicate if devices need to be hotplugged on the root bus instead of a bridge |
| `io.katacontainers.config.hypervisor.hypervisor_hash`        | string                                                       | container hypervisor binary SHA-512 hash value               |
| `io.katacontainers.config.hypervisor.image_hash`             | string                                                       | container guest image SHA-512 hash value                     |
| `io.katacontainers.config.hypervisor.image`                  | string                                                       | the guest image that will run in the container VM            |
| `io.katacontainers.config.hypervisor.initrd_hash`            | string                                                       | container guest initrd SHA-512 hash value                    |
| `io.katacontainers.config.hypervisor.initrd`                 | string                                                       | the guest initrd image that will run in the container VM     |
| `io.katacontainers.config.hypervisor.jailer_hash`            | string                                                       | container jailer SHA-512 hash value                          |
| `io.katacontainers.config.hypervisor.jailer_path` (R)        | string                                                       | the jailer that will constrain the container VM              |
| `io.katacontainers.config.hypervisor.kernel_hash`            | string                                                       | container kernel image SHA-512 hash value                    |
| `io.katacontainers.config.hypervisor.kernel_params`          | string                                                       | additional guest kernel parameters                           |
| `io.katacontainers.config.hypervisor.kernel`                 | string                                                       | the kernel used to boot the container VM                     |
| `io.katacontainers.config.hypervisor.machine_accelerators`   | string                                                       | machine specific accelerators for the hypervisor             |
| `io.katacontainers.config.hypervisor.machine_type`           | string                                                       | the type of machine being emulated by the hypervisor         |
| `io.katacontainers.config.hypervisor.memory_offset`          | uint64                                                       | the memory space used for `nvdimm` device by the hypervisor  |
| `io.katacontainers.config.hypervisor.memory_slots`           | uint32                                                       | the memory slots assigned to the VM by the hypervisor        |
| `io.katacontainers.config.hypervisor.msize_9p`               | uint32                                                       | the `msize` for 9p shares                                    |
| `io.katacontainers.config.hypervisor.path`                   | string                                                       | the hypervisor that will run the container VM                |
| `io.katacontainers.config.hypervisor.pcie_root_port`         | specify the number of PCIe Root Port devices. The PCIe Root Port device is used to hot-plug a PCIe device (QEMU) |                                                              |
| `io.katacontainers.config.hypervisor.shared_fs`              | string                                                       | the shared file system type, either `virtio-9p` or `virtio-fs` |
| `io.katacontainers.config.hypervisor.use_vsock`              | `boolean`                                                    | specify use of `vsock` for agent communication               |
| `io.katacontainers.config.hypervisor.vhost_user_store_path` (R) | `string`                                                     | specify the directory path where vhost-user devices related folders, sockets and device nodes should be (QEMU) |
| `io.katacontainers.config.hypervisor.virtio_fs_cache_size`   | uint32                                                       | virtio-fs DAX cache size in `MiB`                            |
| `io.katacontainers.config.hypervisor.virtio_fs_cache`        | string                                                       | the cache mode for virtio-fs, valid values are `always`, `auto` and `none` |
| `io.katacontainers.config.hypervisor.virtio_fs_daemon`       | string                                                       | virtio-fs `vhost-user`daemon path                            |
| `io.katacontainers.config.hypervisor.virtio_fs_extra_args`   | string                                                       | extra options passed to `virtiofs` daemon                    |
| `io.katacontainers.config.hypervisor.enable_guest_swap`      | `boolean`                                                    | enable swap in the guest                                     |
| `io.katacontainers.config.hypervisor.use_legacy_serial`      | `boolean`                                                    | uses legacy serial device for guest's console (QEMU)         |

**Container 配置**

| Key                                                   | Value Type | Comments                                  |
| ----------------------------------------------------- | ---------- | ----------------------------------------- |
| `io.katacontainers.container.resource.swappiness"`    | `uint64`   | specify the `Resources.Memory.Swappiness` |
| `io.katacontainers.container.resource.swap_in_bytes"` | `uint64`   | specify the `Resources.Memory.Swap`       |

例如，通过 Pod Annotation 启动一个忽略底层默认大小的，具有 5C 的 VM

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test
  annotations:
    io.katacontainers.config.hypervisor.default_vcpus: "5"
spec:
  runtimeClassName: kata-containers
  containers:
  - name: uname-kata
    image: busybox
    command: ["/bin/sh", "-c", "uname -r && tail -f /dev/null"]
```

# VMCache

VMCache 是一项新功能，可在使用前将 VM 创建为缓存。它有助于加快新容器的创建。

该功能由借助 Unix Socket 通信的一个 Server 和一些 Client 组成。该协议是 protocols/cache/cache.proto 中的 gRPC。

VMCache Server 将创建一些 VM 并通过 factory cache 缓存它们。它将 VM 转换为 gRPC 格式并在收到 client 请求时传输它。

grpccache Factory 是 VMCache 客户端。它将请求 gRPC 格式的 VM 并将其转换回 VM。如果启用了 VMCache 功能，kata-runtime 在创建新的 sandbox 时会向 grpccache 请求 VM。

**与 VM Tmplating 的区别**

VM Tmplating 和 VMCache 都有助于加快新容器的创建。

当启用 VM 模板时，通过从预先创建的模板 VM 克隆来创建新的 VM，它们将以只读模式共享相同的 initramfs、内核和 agent 内存。因此，如果在同一台主机上运行许多 Kata 容器，它会节省大量内存。

VMCache 不容易受到共享内存 CVE 的影响，因为每个 VM 不共享内存。

**如何启用 VM Cache**

配置文件中修改以下配置项：

- vm_cache_number 指定 VMCache 缓存的个数，不指定或者为 0 时代表 VMCache 被禁用；> 0 时即为缓存个数
- vm_cache_endpoint 指定 socket 地址

通过以下命令创建一个 VM 模板供以后使用，通过 CTRL+C 退出

```shell
$ kata-runtime factory init
```

**已知限制**

- 无法与 VM Templating 共存
- 仅支持 QEMU 作为 hypervisor

# VM Templating

VM Templating 是 Kata Containers 的一项功能，可以借助克隆技术创建新的 VM。启用后，通过从预先创建的模板 VM 克隆创建新的 VM，它们将以只读模式共享相同的 initramfs、内核和 agent 内存。类似于内核的 fork 进程操作，这里 fork 的是 VM。

**与 VMCache 的区别**

VMCache 和 VM Templating 都有助于加快新容器的创建。

启用 VMCache 后，VMCache 服务器会创建新的 VM。所以它不容易受到共享内存 CVE 的攻击，因为每个 VM 都不共享内存。

如果在同一台主机上运行许多 Kata 容器，VM Templating 可以节省大量内存

**优势**

如果在同一主机上运行许多 Kata 容器，VM Templating 有助于加快新容器的创建并节省大量内存。如果正在运行高密度工作负载，或者非常关心容器启动速度，VM Templating 可能非常有用。

在一个示例中，创建了 100 个 Kata 容器，每个容器都拥有 128MB 的 Guest 内存，并且在启用 VM Templating 特性时最终总共节省了 9GB 的内存，这大约是 Guest 内存总量的 72%。[完整结果参考](https://github.com/kata-containers/runtime/pull/303#issuecomment-395846767)。

在另一个示例中，使用 containerd shimv2 创建了 10 个 Kata 容器，并计算了每个容器的平均启动速度。结果表明，VM Templating 将 Kata 容器的创建速度提高了 38.68%。[完整结果参考](https://gist.github.com/bergwolf/06974a3c5981494a40e2c408681c085d)。

**不足**

VM Templating 的一个缺点是它无法避免跨 VM 侧通道攻击，例如最初针对 Linux KSM 功能的 CVE-2015-2877。得出的结论是，“相互不信任的租户之间用于内存保护的共享直到写入的方法本质上是可检测的信息泄露，并且可以归类为潜在的被误解的行为而不是漏洞。”如果对此敏感，不要使用 VM Templating 或 KSM。

**如何启用 VM Templating**

配置文件中修改以下配置项：

- hypervisor 为 qemu，且版本为 v4.1.0 以上
- enable_template 设为 true
- VM 镜像为 initrd 类型
- shared_fs 不为 virtio-fs

通过以下命令创建一个VM 模板供以后使用

```go
$ kata-runtime factory init
vm factory initialized
```

创建的模板位于

```go
$ ls /run/vc/vm/template
memory  state
```

通过以下命令销毁

```go
$ kata-runtime factory destroy
vm factory destroyed
```

如果不想手动调用 kata-runtime factory init，默认创建的第一个 Kata 容器将自动创建一个 VM 模板。

# kata-runtime

## check (kata-check)

```shell
$ kata-runtime check --verbose
INFO[0000] Looking for releases                          arch=amd64 name=kata-runtime pid=33900 source=runtime url="https://api.github.com/repos/kata-containers/kata-containers/releases"
Newer major release available: 3.0.0 (url: https://github.com/kata-containers/kata-containers/releases/download/3.0.0/kata-containers-3.0.0-vendor.tar.gz, date: 2022-10-09T09:48:18Z)
INFO[0002] CPU property found                            arch=amd64 description="Intel Architecture CPU" name=GenuineIntel pid=33900 source=runtime type=attribute
INFO[0002] CPU property found                            arch=amd64 description="Virtualization support" name=vmx pid=33900 source=runtime type=flag
INFO[0002] CPU property found                            arch=amd64 description="64Bit CPU" name=lm pid=33900 source=runtime type=flag
INFO[0002] CPU property found                            arch=amd64 description=SSE4.1 name=sse4_1 pid=33900 source=runtime type=flag
INFO[0002] kernel property found                         arch=amd64 description="Host kernel accelerator for virtio" name=vhost pid=33900 source=runtime type=module
INFO[0002] kernel property found                         arch=amd64 description="Host kernel accelerator for virtio network" name=vhost_net pid=33900 source=runtime type=module
INFO[0002] kernel property found                         arch=amd64 description="Host Support for Linux VM Sockets" name=vhost_vsock pid=33900 source=runtime type=module
INFO[0002] kernel property found                         arch=amd64 description="Intel KVM" name=kvm_intel pid=33900 source=runtime type=module
INFO[0002] kernel property found                         arch=amd64 description="Kernel-based Virtual Machine" name=kvm pid=33900 source=runtime type=module
System is capable of running Kata Containers
```

可选的 flags 包括

| 名称                   | 含义                                                         |
| ---------------------- | ------------------------------------------------------------ |
| --check-version-only   | 仅对比前使用版本和最新可用版本（需要网络支持，且非 root 用户） |
| --include-all-releases | 包含过滤预发布的版本                                         |
| --no-network-checks    | 不借助网络执行检测                                           |
| --only-list-releases   | 仅列出较新的可用版本（需要网络支持，且非 root 用户）         |
| --strict               | 进行严格检查                                                 |
| --verbose              | 展示详细的检查项                                             |

## env (kata-env)

```shell
$ kata-runtime env 
[Kernel]
  Path = "/opt/kata/share/kata-containers/vmlinux.container"
  Parameters = "systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.debug_console agent.debug_console_vport=1026"

[Meta]
  Version = "1.0.26"

[Image]
  Path = "/opt/kata/share/kata-containers/kata-containers.img"

[Initrd]
  Path = ""

[Hypervisor]
  MachineType = "q35"
  Version = "QEMU emulator version 6.2.0 (kata-static)\nCopyright (c) 2003-2021 Fabrice Bellard and the QEMU Project developers"
  Path = "/opt/kata/bin/qemu-system-x86_64"
  BlockDeviceDriver = "virtio-scsi"
  EntropySource = "/dev/urandom"
  SharedFS = "virtio-fs"
  VirtioFSDaemon = "/opt/kata/libexec/kata-qemu/virtiofsd"
  SocketPath = "<<unknown>>"
  Msize9p = 8192
  MemorySlots = 10
  PCIeRootPort = 0
  HotplugVFIOOnRootBus = false
  Debug = false

[Runtime]
  Path = "/usr/bin/kata-runtime"
  Debug = false
  Trace = false
  DisableGuestSeccomp = true
  DisableNewNetNs = false
  SandboxCgroupOnly = true
  [Runtime.Config]
    Path = "/etc/kata-containers/configuration.toml"
  [Runtime.Version]
    OCI = "1.0.2-dev"
    [Runtime.Version.Version]
      Semver = "2.4.3"
      Commit = "fcad969e5200607df3b0b31983cc64488e156e99"
      Major = 2
      Minor = 4
      Patch = 3

[Host]
  Kernel = "3.10.0-957.10.5.el7.x86_64"
  Architecture = "amd64"
  VMContainerCapable = true
  SupportVSocks = true
  [Host.Distro]
    Name = "ArcherOS OS"
    Version = "1.6"
  [Host.CPU]
    Vendor = "GenuineIntel"
    Model = "Intel(R) Xeon(R) CPU E5-2650 v4 @ 2.20GHz"
    CPUs = 48
  [Host.Memory]
    Total = 131447232
    Free = 62496172
    Available = 63926992

[Agent]
  Debug = false
  Trace = false
```

可选的 flags 包括

| 名称   | 含义             |
| ------ | ---------------- |
| --json | 以 JSON 格式展示 |

## exec

```shell
# 对于 Pod 而言是其 SandboxID
$ kata-runtime exec 27ab74433f11c0b64e404a841d5e2f8296a723ebfa4e598b4d9d32871173b82c
```

可选的 flags 包括

| 名称              | 含义                                         |
| ----------------- | -------------------------------------------- |
| --kata-debug-port | debug console 监听的端口，默认为 1026 或者 0 |

## metrics

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

### add

```shell
$ kata-runtime direct-volume add --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount --mount-info \{\"volume-type\":\"block\",\"device\":\"/dev/sdm\",\"fstype\":\"xfs\"\}
```

可选的 flags 包括

| 名称          | 含义                 |
| ------------- | -------------------- |
| --volume-path | 待操作的目标卷路径   |
| --mount-info  | 管理卷挂载的详情信息 |

### remove

```shell
$ kata-runtime direct-volume delete --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount
```

可选的 flags 包括

| 名称          | 含义               |
| ------------- | ------------------ |
| --volume-path | 待操作的目标卷路径 |

### stats

```shell
$ kata-runtime direct-volume stats --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount
```

可选的 flags 包括

| 名称          | 含义               |
| ------------- | ------------------ |
| --volume-path | 待操作的目标卷路径 |

### resize

*截至 Kata Containers 2.4.3，社区仍未实现*

```shell
$ kata-runtime direct-volume resize --volume-path /var/lib/kubelet/pods/8c3d29ad-84b8-45f0-9fcc-8e16778cb3cb/volumes/kubernetes.io~csi/pvc-a950ed68-622c-4ec4-81fa-506f16de2196/mount --size 1756519562
```

可选的 flags 包括

| 名称          | 含义                               |
| ------------- | ---------------------------------- |
| --volume-path | 待操作的目标卷路径                 |
| --size        | 调整后的预期卷大小（单位为：Byte） |

## factory

### init

```go
$ kata-runtime factory init
vm factory initialized
```

### status

```go
$ kata-runtime factory status
vm factory is on
```

### destroy

```go
$ kata-runtime factory destroy
vm factory destroyed
```

## iptables

### get

```shell
$ kata-runtime iptables get --sandbox-id xxx --v6
```

可选的 flags 包括

| 名称         | 含义                  |
| ------------ | --------------------- |
| --sandbox-id | 待操作的 Sandbox ID   |
| --v6         | 获取 IPV6 的 iptables |

### set

```shell
$ kata-runtime iptables set --sandbox-id xxx --v6 ./iptables
```

可选的 flags 包括

| 名称         | 含义                  |
| ------------ | --------------------- |
| --sandbox-id | 待操作的 Sandbox ID   |
| --v6         | 设置 IPV6 的 iptables |

# kata-monitor

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
