---
layout: post
title:  "[ Kata Containers ] 3 基础使用"
date:   2021-04-22
excerpt: "OCI 和 CRI 介绍以及 Kata Containers 在 Kubernetes 的基础使用示例"
photos:
- https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg
tag:
- Cloud Native
- Kubernetes
- Kata Containers
- Container Runtime
categories:
- Kata Containers
---

* [Overview](#overview)
* [High-Level Runtime &amp; Low-Level Runtime](#high-level-runtime--low-level-runtime)
   * [OCI](#oci)
   * [CRI](#cri)
* [CRI Configuration](#cri-configuration)
   * [Containerd](#containerd)
      * [Chain](#chain)
      * [Configuration](#configuration)
         * [Basic](#basic)
         * [Custom](#custom)
      * [CRI-O](#cri-o)
* [RuntimeClass](#runtimeclass)
   * [handler](#handler)
   * [schedule](#schedule)
   * [Overhead](#overhead)

# Overview

![](https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/with-kubernetes.png)

# High-Level Runtime & Low-Level Runtime

![](https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/CRIs.png)


docker 中的 containerd 和单独的 containerd 项目等价，但是 docker 中的 containerd 组件会禁用其作为 CRI 的功能，这部分功能由 kubelet docker-shim 完成。

```shell
$ cat /etc/containerd/config.toml
disabled_plugins = ["cri"]
```

*此外，Kata Containers 2.x 未支持 dockershim 作为 high-level runtime，社区也主推 Containerd 与 CRI-O。*

## OCI

OCI（Open Container Initiative），是一个轻量级，开放的治理结构项目，在 Linux 基金会的支持下成立，致力于围绕**容器格式和运行时创建开放的行业标准**。

OCI 规范中包括三个部分：

- [Runtime Spec](https://github.com/opencontainers/runtime-spec)
- [Image Spec](https://github.com/opencontainers/image-spec)
- [Distribution Spec](https://github.com/opencontainers/distribution-spec)

## CRI

*kubernetes\staging\src\k8s.io\cri-api\pkg\apis\runtime\v1alpha2\api.proto*

```go
// Runtime service defines the public APIs for remote container runtimes
service RuntimeService {
    // Version returns the runtime name, runtime version, and runtime API version.
    rpc Version(VersionRequest) returns (VersionResponse) {}
 
    // RunPodSandbox creates and starts a pod-level sandbox. Runtimes must ensure
    // the sandbox is in the ready state on success.
    rpc RunPodSandbox(RunPodSandboxRequest) returns (RunPodSandboxResponse) {}
    // StopPodSandbox stops any running process that is part of the sandbox and
    // reclaims network resources (e.g., IP addresses) allocated to the sandbox.
    // If there are any running containers in the sandbox, they must be forcibly
    // terminated.
    // This call is idempotent, and must not return an error if all relevant
    // resources have already been reclaimed. kubelet will call StopPodSandbox
    // at least once before calling RemovePodSandbox. It will also attempt to
    // reclaim resources eagerly, as soon as a sandbox is not needed. Hence,
    // multiple StopPodSandbox calls are expected.
    rpc StopPodSandbox(StopPodSandboxRequest) returns (StopPodSandboxResponse) {}
    // RemovePodSandbox removes the sandbox. If there are any running containers
    // in the sandbox, they must be forcibly terminated and removed.
    // This call is idempotent, and must not return an error if the sandbox has
    // already been removed.
    rpc RemovePodSandbox(RemovePodSandboxRequest) returns (RemovePodSandboxResponse) {}
    // PodSandboxStatus returns the status of the PodSandbox. If the PodSandbox is not
    // present, returns an error.
    rpc PodSandboxStatus(PodSandboxStatusRequest) returns (PodSandboxStatusResponse) {}
    // ListPodSandbox returns a list of PodSandboxes.
    rpc ListPodSandbox(ListPodSandboxRequest) returns (ListPodSandboxResponse) {}
 
    // CreateContainer creates a new container in specified PodSandbox
    rpc CreateContainer(CreateContainerRequest) returns (CreateContainerResponse) {}
    // StartContainer starts the container.
    rpc StartContainer(StartContainerRequest) returns (StartContainerResponse) {}
    // StopContainer stops a running container with a grace period (i.e., timeout).
    // This call is idempotent, and must not return an error if the container has
    // already been stopped.
    // TODO: what must the runtime do after the grace period is reached?
    rpc StopContainer(StopContainerRequest) returns (StopContainerResponse) {}
    // RemoveContainer removes the container. If the container is running, the
    // container must be forcibly removed.
    // This call is idempotent, and must not return an error if the container has
    // already been removed.
    rpc RemoveContainer(RemoveContainerRequest) returns (RemoveContainerResponse) {}
    // ListContainers lists all containers by filters.
    rpc ListContainers(ListContainersRequest) returns (ListContainersResponse) {}
    // ContainerStatus returns status of the container. If the container is not
    // present, returns an error.
    rpc ContainerStatus(ContainerStatusRequest) returns (ContainerStatusResponse) {}
    // UpdateContainerResources updates ContainerConfig of the container.
    rpc UpdateContainerResources(UpdateContainerResourcesRequest) returns (UpdateContainerResourcesResponse) {}
    // ReopenContainerLog asks runtime to reopen the stdout/stderr log file
    // for the container. This is often called after the log file has been
    // rotated. If the container is not running, container runtime can choose
    // to either create a new log file and return nil, or return an error.
    // Once it returns error, new container log file MUST NOT be created.
    rpc ReopenContainerLog(ReopenContainerLogRequest) returns (ReopenContainerLogResponse) {}
 
    // ExecSync runs a command in a container synchronously.
    rpc ExecSync(ExecSyncRequest) returns (ExecSyncResponse) {}
    // Exec prepares a streaming endpoint to execute a command in the container.
    rpc Exec(ExecRequest) returns (ExecResponse) {}
    // Attach prepares a streaming endpoint to attach to a running container.
    rpc Attach(AttachRequest) returns (AttachResponse) {}
    // PortForward prepares a streaming endpoint to forward ports from a PodSandbox.
    rpc PortForward(PortForwardRequest) returns (PortForwardResponse) {}
 
    // ContainerStats returns stats of the container. If the container does not
    // exist, the call returns an error.
    rpc ContainerStats(ContainerStatsRequest) returns (ContainerStatsResponse) {}
    // ListContainerStats returns stats of all running containers.
    rpc ListContainerStats(ListContainerStatsRequest) returns (ListContainerStatsResponse) {}
 
    // UpdateRuntimeConfig updates the runtime configuration based on the given request.
    rpc UpdateRuntimeConfig(UpdateRuntimeConfigRequest) returns (UpdateRuntimeConfigResponse) {}
 
    // Status returns the status of the runtime.
    rpc Status(StatusRequest) returns (StatusResponse) {}
}
 
// ImageService defines the public APIs for managing images.
service ImageService {
    // ListImages lists existing images.
    rpc ListImages(ListImagesRequest) returns (ListImagesResponse) {}
    // ImageStatus returns the status of the image. If the image is not
    // present, returns a response with ImageStatusResponse.Image set to
    // nil.
    rpc ImageStatus(ImageStatusRequest) returns (ImageStatusResponse) {}
    // PullImage pulls an image with authentication config.
    rpc PullImage(PullImageRequest) returns (PullImageResponse) {}
    // RemoveImage removes the image.
    // This call is idempotent, and must not return an error if the image has
    // already been removed.
    rpc RemoveImage(RemoveImageRequest) returns (RemoveImageResponse) {}
    // ImageFSInfo returns information of the filesystem that is used to store images.
    rpc ImageFsInfo(ImageFsInfoRequest) returns (ImageFsInfoResponse) {}
}
```

需要注意的是，从 ImageService 可以看出来，关于 image 的 push、build 等，以及 container 的 restart、pause 等并不是 CRI 的标准规范，因此对于部分 CRI 的实现来讲，未必支持：

|                   | Containerd | CRI-O     |
| ----------------- | ---------- | --------- |
| image push        | support    | unsupport |
| image build       | unsupport  | unsupport |
| container pause   | unsupport  | unsupport |
| container restart | unsupport  | unsupport |

![](https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/oci-runtime.png)


- High-Level Runtime 和 Low-Level Runtime 在交互的时候使用的就是 shim 类的组件，如 Containerd 或者 docker 中的 containerd-shim，cri-o 中的 conmon
- Low-Level Runtime **实现了 OCI 开放容器标准**，负责**容器的生命周期管理**
- High-Level Runtime 除了管理容器之外，还提供了一系列的上层高级特性，比如**管理镜像，与 Shim 交互**等

# CRI Configuration

## Containerd

*/etc/containerd/config.toml*

### Chain

![](https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/containerd-chain.png)

Containerd 在 1.0 及以前版本将 dockershim 和 docker daemon 替换为 cri-containerd + containerd，而在 1.1 版本直接将 cri-containerd 内置在 Containerd 中，简化为一个 CRI 插件。

![](https://raw.githubusercontent.com/shenxianghong/shenxianghong.github.io/main/docs/_posts/assert/img/kata-containers/containerd-chain-detail.png)

Containerd 内置的 CRI 插件实现了 Kubelet CRI 接口中的 Image Service 和 Runtime Service，通过内部接口管理镜像和容器环境，并通过 CNI 插件给 Pod 配置网络。

### Configuration

#### Basic

Containerd 的配置取决于它所处的角色**非 CRI** 时，配置文件大致为：

```toml
#   Copyright 2018-2020 Docker Inc.

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

可以通过 Containerd 提供的方式，生成默认的配置文件，配置文件的格式取决于 Containerd 版本

```shell
$ sudo mkdir -p /etc/containerd
$ containerd config default | sudo tee /etc/containerd/config.toml
```

**低版本**

在 Containerd 版本为 1.2.x 时，Containerd 的配置为

```toml
root = "/var/lib/containerd"
state = "/run/containerd"
oom_score = 0

[grpc]
  address = "/run/containerd/containerd.sock"
  uid = 0
  gid = 0
  max_recv_message_size = 16777216
  max_send_message_size = 16777216

[debug]
  address = ""
  uid = 0
  gid = 0
  level = ""

[metrics]
  address = ""
  grpc_histogram = false

[cgroup]
  path = ""

[plugins]
  [plugins.cgroups]
    no_prometheus = false
  [plugins.cri]
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    enable_selinux = false
    sandbox_image = "k8s.gcr.io/pause:3.1"
    stats_collect_period = 10
    systemd_cgroup = false
    enable_tls_streaming = false
    max_container_log_line_size = 16384
    [plugins.cri.containerd]
      snapshotter = "overlayfs"
      no_pivot = false
      [plugins.cri.containerd.default_runtime]
        runtime_type = "io.containerd.runtime.v1.linux"
        runtime_engine = ""
        runtime_root = ""
      [plugins.cri.containerd.untrusted_workload_runtime]
        runtime_type = ""
        runtime_engine = ""
        runtime_root = ""
    [plugins.cri.cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      conf_template = ""
    [plugins.cri.registry]
      [plugins.cri.registry.mirrors]
        [plugins.cri.registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]
    [plugins.cri.x509_key_pair_streaming]
      tls_cert_file = ""
      tls_key_file = ""
  [plugins.diff-service]
    default = ["walking"]
  [plugins.linux]
    shim = "containerd-shim"
    runtime = "runc"
    runtime_root = ""
    no_shim = false
    shim_debug = false
  [plugins.opt]
    path = "/opt/containerd"
  [plugins.restart]
    interval = "10s"
  [plugins.scheduler]
    pause_threshold = 0.02
    deletion_threshold = 0
    mutation_threshold = 100
    schedule_delay = "0s"
    startup_delay = "100ms"
```

**高版本**

```toml
version = 2
root = "/var/lib/containerd"
state = "/run/containerd"
plugin_dir = ""
disabled_plugins = []
required_plugins = []
oom_score = 0

[grpc]
  address = "/run/containerd/containerd.sock"
  tcp_address = ""
  tcp_tls_cert = ""
  tcp_tls_key = ""
  uid = 0
  gid = 0
  max_recv_message_size = 16777216
  max_send_message_size = 16777216

[ttrpc]
  address = ""
  uid = 0
  gid = 0

[debug]
  address = ""
  uid = 0
  gid = 0
  level = ""

[metrics]
  address = ""
  grpc_histogram = false

[cgroup]
  path = ""

[timeouts]
  "io.containerd.timeout.shim.cleanup" = "5s"
  "io.containerd.timeout.shim.load" = "5s"
  "io.containerd.timeout.shim.shutdown" = "3s"
  "io.containerd.timeout.task.state" = "2s"

[plugins]
  [plugins."io.containerd.gc.v1.scheduler"]
    pause_threshold = 0.02
    deletion_threshold = 0
    mutation_threshold = 100
    schedule_delay = "0s"
    startup_delay = "100ms"
  [plugins."io.containerd.grpc.v1.cri"]
    disable_tcp_service = true
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    stream_idle_timeout = "4h0m0s"
    enable_selinux = false
    selinux_category_range = 1024
    sandbox_image = "k8s.gcr.io/pause:3.2"
    stats_collect_period = 10
    systemd_cgroup = false
    enable_tls_streaming = false
    max_container_log_line_size = 16384
    disable_cgroup = false
    disable_apparmor = false
    restrict_oom_score_adj = false
    max_concurrent_downloads = 3
    disable_proc_mount = false
    unset_seccomp_profile = ""
    tolerate_missing_hugetlb_controller = true
    disable_hugetlb_controller = true
    ignore_image_defined_volumes = false
    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "overlayfs"
      default_runtime_name = "runc"
      no_pivot = false
      disable_snapshot_annotations = true
      discard_unpacked_layers = false
      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        runtime_type = ""
        runtime_engine = ""
        runtime_root = ""
        privileged_without_host_devices = false
        base_runtime_spec = ""
      [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime]
        runtime_type = ""
        runtime_engine = ""
        runtime_root = ""
        privileged_without_host_devices = false
        base_runtime_spec = ""
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          runtime_engine = ""
          runtime_root = ""
          privileged_without_host_devices = false
          base_runtime_spec = ""
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      max_conf_num = 1
      conf_template = ""
    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]
    [plugins."io.containerd.grpc.v1.cri".image_decryption]
      key_model = ""
    [plugins."io.containerd.grpc.v1.cri".x509_key_pair_streaming]
      tls_cert_file = ""
      tls_key_file = ""
  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"
  [plugins."io.containerd.internal.v1.restart"]
    interval = "10s"
  [plugins."io.containerd.metadata.v1.bolt"]
    content_sharing_policy = "shared"
  [plugins."io.containerd.monitor.v1.cgroups"]
    no_prometheus = false
  [plugins."io.containerd.runtime.v1.linux"]
    shim = "containerd-shim"
    runtime = "runc"
    runtime_root = ""
    no_shim = false
    shim_debug = false
  [plugins."io.containerd.runtime.v2.task"]
    platforms = ["linux/amd64"]
  [plugins."io.containerd.service.v1.diff-service"]
    default = ["walking"]
  [plugins."io.containerd.snapshotter.v1.devmapper"]
    root_path = ""
    pool_name = ""
    base_image_size = ""
    async_remove = false
```

其中，Containerd 默认的 runtime 为 runC。无论版本的高低，新增对 Kata Containers 的支持都类似，以**高版本**为例

```toml
# 新增 Kata Containers 支持
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-runtime]
        # 固定值
        runtime_type = "io.containerd.kata.v2"
        # 开启允许特权容器
        privileged_without_host_devices = true
```

在低版本中，Containerd 拉取 harbor 认证仓库的权限问题，需要通过手动下载 ssl 证书，高版本中可以通过配置文件的方式。

```shell
curl -o /etc/ssl/certs/ca.crt http://<mirror-address>/ca.crt
```

#### Custom

Kata Containers 可以通过 Pod annotation 的方式实现定制化每一个 Pod 的底层 Kata 参数。需要做的是上层 CRI 将 Pod annotation 透传至底层 runtime，同时 Kata Containers 开启识别特定的 Pod annotation，并且 CRI 需要支持此功能（如 Containerd 依赖 1.4.x 以上的版本才可以）

**Kata 支持的 annotation 配置**

*Global Options*

| Key                                        | Value Type | Comments                                                     |
| ------------------------------------------ | ---------- | ------------------------------------------------------------ |
| `io.katacontainers.config_path`            | string     | Kata config file location that overrides the default config paths |
| `io.katacontainers.pkg.oci.bundle_path`    | string     | OCI bundle path                                              |
| `io.katacontainers.pkg.oci.container_type` | string     | OCI container type. Only accepts `pod_container` and `pod_sandbox` |

*Runtime Options*

| Key                                                      | Value Type | Comments                                                     |
| -------------------------------------------------------- | ---------- | ------------------------------------------------------------ |
| `io.katacontainers.config.runtime.experimental`          | `boolean`  | determines if experimental features enabled                  |
| `io.katacontainers.config.runtime.disable_guest_seccomp` | `boolean`  | determines if `seccomp` should be applied inside guest       |
| `io.katacontainers.config.runtime.disable_new_netns`     | `boolean`  | determines if a new netns is created for the hypervisor process |
| `io.katacontainers.config.runtime.internetworking_model` | string     | determines how the VM should be connected to the container network interface. Valid values are `macvtap`, `tcfilter` and `none` |
| `io.katacontainers.config.runtime.sandbox_cgroup_only`   | `boolean`  | determines if Kata processes are managed only in sandbox cgroup |
| `io.katacontainers.config.runtime.enable_pprof`          | `boolean`  | enables Golang `pprof` for `containerd-shim-kata-v2` process |

*Agent Options*

| Key                                                  | Value Type | Comments                                                     |
| ---------------------------------------------------- | ---------- | ------------------------------------------------------------ |
| `io.katacontainers.config.agent.enable_tracing`      | `boolean`  | enable tracing for the agent                                 |
| `io.katacontainers.config.agent.container_pipe_size` | uint32     | specify the size of the std(in/out) pipes created for containers |
| `io.katacontainers.config.agent.kernel_modules`      | string     | the list of kernel modules and their parameters that will be loaded in the guest kernel. Semicolon separated list of kernel modules and their parameters. These modules will be loaded in the guest kernel using `modprobe`(8). E.g., `e1000e InterruptThrottleRate=3000,3000,3000 EEE=1; i915 enable_ppgtt=0` |

*Hypervisor Options*

| Key                                                          | Value Type                                                   | Comments                                                     |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `io.katacontainers.config.hypervisor.asset_hash_type`        | string                                                       | the hash type used for assets verification, default is `sha512` |
| `io.katacontainers.config.hypervisor.block_device_cache_direct` | `boolean`                                                    | Denotes whether use of `O_DIRECT` (bypass the host page cache) is enabled |
| `io.katacontainers.config.hypervisor.block_device_cache_noflush` | `boolean`                                                    | Denotes whether flush requests for the device are ignored    |
| `io.katacontainers.config.hypervisor.block_device_cache_set` | `boolean`                                                    | cache-related options will be set to block devices or not    |
| `io.katacontainers.config.hypervisor.block_device_driver`    | string                                                       | the driver to be used for block device, valid values are `virtio-blk`, `virtio-scsi`, `nvdimm` |
| `io.katacontainers.config.hypervisor.cpu_features`           | `string`                                                     | Comma-separated list of CPU features to pass to the CPU (QEMU) |
| `io.katacontainers.config.hypervisor.ctlpath` (R)            | `string`                                                     | Path to the `acrnctl` binary for the ACRN hypervisor         |
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
| `io.katacontainers.config.hypervisor.enable_swap`            | `boolean`                                                    | enable swap of VM memory                                     |
| `io.katacontainers.config.hypervisor.enable_vhost_user_store` | `boolean`                                                    | enable vhost-user storage device (QEMU)                      |
| `io.katacontainers.config.hypervisor.enable_virtio_mem`      | `boolean`                                                    | enable virtio-mem (QEMU)                                     |
| `io.katacontainers.config.hypervisor.entropy_source` (R)     | string                                                       | the path to a host source of entropy (`/dev/random`, `/dev/urandom` or real hardware RNG device) |
| `io.katacontainers.config.hypervisor.file_mem_backend` (R)   | string                                                       | file based memory backend root directory                     |
| `io.katacontainers.config.hypervisor.firmware_hash`          | string                                                       | container firmware SHA-512 hash value                        |
| `io.katacontainers.config.hypervisor.firmware`               | string                                                       | the guest firmware that will run the container VM            |
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
| `io.katacontainers.config.hypervisor.virtio_fs_daemon`       | string                                                       | virtio-fs `vhost-user` daemon path                           |
| `io.katacontainers.config.hypervisor.virtio_fs_extra_args`   | string                                                       | extra options passed to `virtiofs` daemon                    |
| `io.katacontainers.config.hypervisor.enable_guest_swap`      | `boolean`                                                    | enable swap in the guest                                     |

*Container Options*

| Key                                                   | Value Type | Comments                                  |
| ----------------------------------------------------- | ---------- | ----------------------------------------- |
| `io.katacontainers.container.resource.swappiness"`    | `uint64`   | specify the `Resources.Memory.Swappiness` |
| `io.katacontainers.container.resource.swap_in_bytes"` | `uint64`   | specify the `Resources.Memory.Swap`       |

**example**

示例是通过 Pod Annotation 启动一个忽略底层默认大小的，具有 5C 的容器

*Kata Containers*

```toml
[hypervisor.qemu]
path = "/opt/kata/bin/qemu-system-x86_64"
kernel = "/opt/kata/share/kata-containers/vmlinux.container"
initrd = "/opt/kata/share/kata-containers/kata-containers-initrd.img"
machine_type = "pc"

# List of valid annotation names for the hypervisor
# Each member of the list is a regular expression, which is the base name
# of the annotation, e.g. "path" for io.katacontainers.config.hypervisor.path"
enable_annotations = ["default_vcpus"]
default_vcpus = 1
```

*Containerd*

```toml
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
          runtime_type = "io.containerd.kata.v2"
          privileged_without_host_devices = true
          shim_debug = true
          container_annotations = ["io.katacontainers.*"]
          pod_annotations = ["io.katacontainers.*"]  
```

*Pod*

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

### CRI-O

*/etc/crio/crio.conf*

TODO

# RuntimeClass

RuntimeClass 是一个用于选择容器运行时配置的特性，容器运行时配置用于运行 Pod 中的容器。

```yaml
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: kata-containers
handler: kata-containers
overhead:
  podFixed:
    memory: "140Mi"
    cpu: "250m"
scheduling:
  nodeSelector:
    runtime: kata
```

## handler

需要和 CRI 中注册的 handler 保持一致

**Containerd**

```toml
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.${HANDLER_NAME}]
```

**CRI-O**

```toml
[crio.runtime.runtimes.${HANDLER_NAME}]
  runtime_path = "${PATH_TO_BINARY}"
```

## schedule

通过为 RuntimeClass 指定 `scheduling` 字段， 可以通过设置约束，确保运行该 RuntimeClass 的 Pod 被调度到支持该 RuntimeClass 的节点上。 如果未设置 `scheduling`，则假定所有节点均支持此 RuntimeClass 。

为了确保 pod 会被调度到支持指定运行时的 node 上，每个 node 需要设置一个通用的 label 用于被 `runtimeclass.scheduling.nodeSelector` 挑选。在 admission 阶段，RuntimeClass 的 nodeSelector 将会与 pod 的 nodeSelector 合并，取二者的交集。如果有冲突，pod 将会被拒绝。

如果 node 需要阻止某些需要特定 RuntimeClass 的 pod，可以在 `tolerations` 中指定。 与 `nodeSelector` 一样，tolerations 也在 admission 阶段与 pod 的 tolerations 合并，取二者的并集。

## overhead

在节点上运行 Pod 时，Pod 本身占用大量系统资源。这些资源是运行 Pod 内容器所需资源的附加资源。Overhead 是一个特性，用于计算 Pod 基础设施在容器请求和限制之上消耗的资源。

在 Kubernetes 中，Pod 的开销是根据与 Pod 的 [RuntimeClass](https://kubernetes.io/zh/docs/concepts/containers/runtime-class/) 相关联的开销在 [准入](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks) 时设置的。

如果启用了 Pod Overhead，在调度 Pod 时，除了考虑容器资源请求的总和外，还要考虑 Pod 开销。 类似地，kubelet 将在确定 Pod cgroups 的大小和执行 Pod 驱逐排序时也会考虑 Pod 开销。

*关于 overhead 的更多介绍参考 https://shenxianghong.github.io/kata-containers-resource-limation/*

