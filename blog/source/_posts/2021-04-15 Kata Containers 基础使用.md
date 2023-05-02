---
title: "「 Kata Containers 」基础使用"
excerpt: "Kata Containers 基础使用示例"
cover: https://picsum.photos/0?sig=20210415
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2021-04-15
toc: true
categories:
- Getting Started
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> Based on **v3.0.0**

# 安装

Kata Containers 社区仅提供了 x86 架构的制品，arm64 架构需要手动编译。这里以 x86 架构下的社区制品安装为例：

```shell
$ wget https://github.com/kata-containers/kata-containers/releases/download/3.0.0/kata-static-3.0.0-x86_64.tar.xz
$ tar -xvf kata-static-3.0.0-x86_64.tar.xz
$ cp -r ./opt/kata /opt
$ cp -r ./opt/kata/bin/kata-runtime /usr/local/bin
```

**校验**

```shell
$ kata-runtime check
WARN[0000] Not running network checks as super user      arch=amd64 name=kata-runtime pid=18456 source=runtime
System is capable of running Kata Containers
System can currently create Kata Containers
```

```shell
$ kata-runtime env
[Kernel]
  Path = "/opt/kata/share/kata-containers/vmlinux-5.19.2-96"
  Parameters = "systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none"

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
  PCIeRootPort = 0
  HotplugVFIOOnRootBus = false
  Debug = false

[Runtime]
  Path = "/usr/local/bin/kata-runtime"
  Debug = false
  Trace = false
  DisableGuestSeccomp = true
  DisableNewNetNs = false
  SandboxCgroupOnly = false
  [Runtime.Config]
    Path = "/opt/kata/share/defaults/kata-containers/configuration-qemu.toml"
  [Runtime.Version]
    OCI = "1.0.2-dev"
    [Runtime.Version.Version]
      Semver = "3.0.0"
      Commit = "e2a8815ba46360acb8bf89a2894b0d437dc8548a"
      Major = 3
      Minor = 0
      Patch = 0

[Host]
  Kernel = "3.10.0-957.10.4.el7.x86_64"
  Architecture = "amd64"
  VMContainerCapable = true
  SupportVSocks = true
  [Host.Distro]
    Name = "CentOS Linux"
    Version = "7"
  [Host.CPU]
    Vendor = "GenuineIntel"
    Model = "QEMU Virtual CPU version (cpu64-rhel6)"
    CPUs = 4
  [Host.Memory]
    Total = 7989988
    Free = 2159220
    Available = 5454100

[Agent]
  Debug = false
  Trace = false
```

**默认配置**

```toml
# Copyright (c) 2017-2019 Intel Corporation
# Copyright (c) 2021 Adobe Inc.
#
# SPDX-License-Identifier: Apache-2.0
#

# XXX: WARNING: this file is auto-generated.
# XXX:
# XXX: Source file: "config/configuration-qemu.toml.in"
# XXX: Project:
# XXX:   Name: Kata Containers
# XXX:   Type: kata

[hypervisor.qemu]
path = "/opt/kata/bin/qemu-system-x86_64"
kernel = "/opt/kata/share/kata-containers/vmlinux.container"
image = "/opt/kata/share/kata-containers/kata-containers.img"
machine_type = "q35"

# Enable confidential guest support.
# Toggling that setting may trigger different hardware features, ranging
# from memory encryption to both memory and CPU-state encryption and integrity.
# The Kata Containers runtime dynamically detects the available feature set and
# aims at enabling the largest possible one, returning an error if none is
# available, or none is supported by the hypervisor.
#
# Known limitations:
# * Does not work by design:
#   - CPU Hotplug
#   - Memory Hotplug
#   - NVDIMM devices
#
# Default false
# confidential_guest = true

# Enable running QEMU VMM as a non-root user.
# By default QEMU VMM run as root. When this is set to true, QEMU VMM process runs as
# a non-root random user. See documentation for the limitations of this mode.
# rootless = true

# List of valid annotation names for the hypervisor
# Each member of the list is a regular expression, which is the base name
# of the annotation, e.g. "path" for io.katacontainers.config.hypervisor.path"
enable_annotations = ["enable_iommu"]

# List of valid annotations values for the hypervisor
# Each member of the list is a path pattern as described by glob(3).
# The default if not set is empty (all annotations rejected.)
# Your distribution recommends: ["/opt/kata/bin/qemu-system-x86_64"]
valid_hypervisor_paths = ["/opt/kata/bin/qemu-system-x86_64"]

# Optional space-separated list of options to pass to the guest kernel.
# For example, use `kernel_params = "vsyscall=emulate"` if you are having
# trouble running pre-2.15 glibc.
#
# WARNING: - any parameter specified here will take priority over the default
# parameter value of the same name used to start the virtual machine.
# Do not set values here unless you understand the impact of doing so as you
# may stop the virtual machine from booting.
# To see the list of default parameters, enable hypervisor debug, create a
# container and look for 'default-kernel-parameters' log entries.
kernel_params = ""

# Path to the firmware.
# If you want that qemu uses the default firmware leave this option empty
firmware = ""

# Path to the firmware volume.
# firmware TDVF or OVMF can be split into FIRMWARE_VARS.fd (UEFI variables
# as configuration) and FIRMWARE_CODE.fd (UEFI program image). UEFI variables
# can be customized per each user while UEFI code is kept same.
firmware_volume = ""

# Machine accelerators
# comma-separated list of machine accelerators to pass to the hypervisor.
# For example, `machine_accelerators = "nosmm,nosmbus,nosata,nopit,static-prt,nofw"`
machine_accelerators=""

# Qemu seccomp sandbox feature
# comma-separated list of seccomp sandbox features to control the syscall access.
# For example, `seccompsandbox= "on,obsolete=deny,spawn=deny,resourcecontrol=deny"`
# Note: "elevateprivileges=deny" doesn't work with daemonize option, so it's removed from the seccomp sandbox
# Another note: enabling this feature may reduce performance, you may enable
# /proc/sys/net/core/bpf_jit_enable to reduce the impact. see https://man7.org/linux/man-pages/man8/bpfc.8.html
#seccompsandbox="on,obsolete=deny,spawn=deny,resourcecontrol=deny"

# CPU features
# comma-separated list of cpu features to pass to the cpu
# For example, `cpu_features = "pmu=off,vmx=off"
cpu_features="pmu=off"

# Default number of vCPUs per SB/VM:
# unspecified or 0                --> will be set to 1
# < 0                             --> will be set to the actual number of physical cores
# > 0 <= number of physical cores --> will be set to the specified number
# > number of physical cores      --> will be set to the actual number of physical cores
default_vcpus = 1

# Default maximum number of vCPUs per SB/VM:
# unspecified or == 0             --> will be set to the actual number of physical cores or to the maximum number
#                                     of vCPUs supported by KVM if that number is exceeded
# > 0 <= number of physical cores --> will be set to the specified number
# > number of physical cores      --> will be set to the actual number of physical cores or to the maximum number
#                                     of vCPUs supported by KVM if that number is exceeded
# WARNING: Depending of the architecture, the maximum number of vCPUs supported by KVM is used when
# the actual number of physical cores is greater than it.
# WARNING: Be aware that this value impacts the virtual machine's memory footprint and CPU
# the hotplug functionality. For example, `default_maxvcpus = 240` specifies that until 240 vCPUs
# can be added to a SB/VM, but the memory footprint will be big. Another example, with
# `default_maxvcpus = 8` the memory footprint will be small, but 8 will be the maximum number of
# vCPUs supported by the SB/VM. In general, we recommend that you do not edit this variable,
# unless you know what are you doing.
# NOTICE: on arm platform with gicv2 interrupt controller, set it to 8.
default_maxvcpus = 0

# Bridges can be used to hot plug devices.
# Limitations:
# * Currently only pci bridges are supported
# * Until 30 devices per bridge can be hot plugged.
# * Until 5 PCI bridges can be cold plugged per VM.
#   This limitation could be a bug in qemu or in the kernel
# Default number of bridges per SB/VM:
# unspecified or 0   --> will be set to 1
# > 1 <= 5           --> will be set to the specified number
# > 5                --> will be set to 5
default_bridges = 1

# Default memory size in MiB for SB/VM.
# If unspecified then it will be set 2048 MiB.
default_memory = 2048
#
# Default memory slots per SB/VM.
# If unspecified then it will be set 10.
# This is will determine the times that memory will be hotadded to sandbox/VM.
#memory_slots = 10

# Default maximum memory in MiB per SB / VM
# unspecified or == 0           --> will be set to the actual amount of physical RAM
# > 0 <= amount of physical RAM --> will be set to the specified number
# > amount of physical RAM      --> will be set to the actual amount of physical RAM
default_maxmemory = 0

# The size in MiB will be plused to max memory of hypervisor.
# It is the memory address space for the NVDIMM devie.
# If set block storage driver (block_device_driver) to "nvdimm",
# should set memory_offset to the size of block device.
# Default 0
#memory_offset = 0

# Specifies virtio-mem will be enabled or not.
# Please note that this option should be used with the command
# "echo 1 > /proc/sys/vm/overcommit_memory".
# Default false
#enable_virtio_mem = true

# Disable block device from being used for a container's rootfs.
# In case of a storage driver like devicemapper where a container's
# root file system is backed by a block device, the block device is passed
# directly to the hypervisor for performance reasons.
# This flag prevents the block device from being passed to the hypervisor,
# virtio-fs is used instead to pass the rootfs.
disable_block_device_use = false

# Shared file system type:
#   - virtio-fs (default)
#   - virtio-9p
#   - virtio-fs-nydus
shared_fs = "virtio-fs"

# Path to vhost-user-fs daemon.
virtio_fs_daemon = "/opt/kata/libexec/virtiofsd"

# List of valid annotations values for the virtiofs daemon
# The default if not set is empty (all annotations rejected.)
# Your distribution recommends: ["/opt/kata/libexec/virtiofsd"]
valid_virtio_fs_daemon_paths = ["/opt/kata/libexec/virtiofsd"]

# Default size of DAX cache in MiB
virtio_fs_cache_size = 0

# Extra args for virtiofsd daemon
#
# Format example:
#   ["-o", "arg1=xxx,arg2", "-o", "hello world", "--arg3=yyy"]
# Examples:
#   Set virtiofsd log level to debug : ["-o", "log_level=debug"] or ["-d"]
#
# see `virtiofsd -h` for possible options.
virtio_fs_extra_args = ["--thread-pool-size=1", "-o", "announce_submounts"]

# Cache mode:
#
#  - none
#    Metadata, data, and pathname lookup are not cached in guest. They are
#    always fetched from host and any changes are immediately pushed to host.
#
#  - auto
#    Metadata and pathname lookup cache expires after a configured amount of
#    time (default is 1 second). Data is cached while the file is open (close
#    to open consistency).
#
#  - always
#    Metadata, data, and pathname lookup are cached in guest and never expire.
virtio_fs_cache = "auto"

# Block storage driver to be used for the hypervisor in case the container
# rootfs is backed by a block device. This is virtio-scsi, virtio-blk
# or nvdimm.
block_device_driver = "virtio-scsi"

# aio is the I/O mechanism used by qemu
# Options:
#
#   - threads
#     Pthread based disk I/O.
#
#   - native
#     Native Linux I/O.
#
#   - io_uring
#     Linux io_uring API. This provides the fastest I/O operations on Linux, requires kernel>5.1 and
#     qemu >=5.0.
block_device_aio = "io_uring"

# Specifies cache-related options will be set to block devices or not.
# Default false
#block_device_cache_set = true

# Specifies cache-related options for block devices.
# Denotes whether use of O_DIRECT (bypass the host page cache) is enabled.
# Default false
#block_device_cache_direct = true

# Specifies cache-related options for block devices.
# Denotes whether flush requests for the device are ignored.
# Default false
#block_device_cache_noflush = true

# Enable iothreads (data-plane) to be used. This causes IO to be
# handled in a separate IO thread. This is currently only implemented
# for SCSI.
#
enable_iothreads = false

# Enable pre allocation of VM RAM, default false
# Enabling this will result in lower container density
# as all of the memory will be allocated and locked
# This is useful when you want to reserve all the memory
# upfront or in the cases where you want memory latencies
# to be very predictable
# Default false
#enable_mem_prealloc = true

# Enable huge pages for VM RAM, default false
# Enabling this will result in the VM memory
# being allocated using huge pages.
# This is useful when you want to use vhost-user network
# stacks within the container. This will automatically
# result in memory pre allocation
#enable_hugepages = true

# Enable vhost-user storage device, default false
# Enabling this will result in some Linux reserved block type
# major range 240-254 being chosen to represent vhost-user devices.
enable_vhost_user_store = false

# The base directory specifically used for vhost-user devices.
# Its sub-path "block" is used for block devices; "block/sockets" is
# where we expect vhost-user sockets to live; "block/devices" is where
# simulated block device nodes for vhost-user devices to live.
vhost_user_store_path = "/var/run/kata-containers/vhost-user"

# Enable vIOMMU, default false
# Enabling this will result in the VM having a vIOMMU device
# This will also add the following options to the kernel's
# command line: intel_iommu=on,iommu=pt
#enable_iommu = true

# Enable IOMMU_PLATFORM, default false
# Enabling this will result in the VM device having iommu_platform=on set
#enable_iommu_platform = true

# List of valid annotations values for the vhost user store path
# The default if not set is empty (all annotations rejected.)
# Your distribution recommends: ["/var/run/kata-containers/vhost-user"]
valid_vhost_user_store_paths = ["/var/run/kata-containers/vhost-user"]

# Enable file based guest memory support. The default is an empty string which
# will disable this feature. In the case of virtio-fs, this is enabled
# automatically and '/dev/shm' is used as the backing folder.
# This option will be ignored if VM templating is enabled.
#file_mem_backend = ""

# List of valid annotations values for the file_mem_backend annotation
# The default if not set is empty (all annotations rejected.)
# Your distribution recommends: [""]
valid_file_mem_backends = [""]

# -pflash can add image file to VM. The arguments of it should be in format
# of ["/path/to/flash0.img", "/path/to/flash1.img"]
pflashes = []

# This option changes the default hypervisor and kernel parameters
# to enable debug output where available.
#
# Default false
#enable_debug = true

# Disable the customizations done in the runtime when it detects
# that it is running on top a VMM. This will result in the runtime
# behaving as it would when running on bare metal.
#
#disable_nesting_checks = true

# This is the msize used for 9p shares. It is the number of bytes
# used for 9p packet payload.
#msize_9p = 8192

# If false and nvdimm is supported, use nvdimm device to plug guest image.
# Otherwise virtio-block device is used.
#
# nvdimm is not supported when `confidential_guest = true`.
#
# Default is false
#disable_image_nvdimm = true

# VFIO devices are hotplugged on a bridge by default.
# Enable hotplugging on root bus. This may be required for devices with
# a large PCI bar, as this is a current limitation with hotplugging on
# a bridge.
# Default false
#hotplug_vfio_on_root_bus = true

# Before hot plugging a PCIe device, you need to add a pcie_root_port device.
# Use this parameter when using some large PCI bar devices, such as Nvidia GPU
# The value means the number of pcie_root_port
# This value is valid when hotplug_vfio_on_root_bus is true and machine_type is "q35"
# Default 0
#pcie_root_port = 2

# If vhost-net backend for virtio-net is not desired, set to true. Default is false, which trades off
# security (vhost-net runs ring0) for network I/O performance.
#disable_vhost_net = true

#
# Default entropy source.
# The path to a host source of entropy (including a real hardware RNG)
# /dev/urandom and /dev/random are two main options.
# Be aware that /dev/random is a blocking source of entropy.  If the host
# runs out of entropy, the VMs boot time will increase leading to get startup
# timeouts.
# The source of entropy /dev/urandom is non-blocking and provides a
# generally acceptable source of entropy. It should work well for pretty much
# all practical purposes.
#entropy_source= "/dev/urandom"

# List of valid annotations values for entropy_source
# The default if not set is empty (all annotations rejected.)
# Your distribution recommends: ["/dev/urandom","/dev/random",""]
valid_entropy_sources = ["/dev/urandom","/dev/random",""]

# Path to OCI hook binaries in the *guest rootfs*.
# This does not affect host-side hooks which must instead be added to
# the OCI spec passed to the runtime.
#
# You can create a rootfs with hooks by customizing the osbuilder scripts:
# https://github.com/kata-containers/kata-containers/tree/main/tools/osbuilder
#
# Hooks must be stored in a subdirectory of guest_hook_path according to their
# hook type, i.e. "guest_hook_path/{prestart,poststart,poststop}".
# The agent will scan these directories for executable files and add them, in
# lexicographical order, to the lifecycle of the guest container.
# Hooks are executed in the runtime namespace of the guest. See the official documentation:
# https://github.com/opencontainers/runtime-spec/blob/v1.0.1/config.md#posix-platform-hooks
# Warnings will be logged if any error is encountered while scanning for hooks,
# but it will not abort container execution.
#guest_hook_path = "/usr/share/oci/hooks"
#
# Use rx Rate Limiter to control network I/O inbound bandwidth(size in bits/sec for SB/VM).
# In Qemu, we use classful qdiscs HTB(Hierarchy Token Bucket) to discipline traffic.
# Default 0-sized value means unlimited rate.
#rx_rate_limiter_max_rate = 0
# Use tx Rate Limiter to control network I/O outbound bandwidth(size in bits/sec for SB/VM).
# In Qemu, we use classful qdiscs HTB(Hierarchy Token Bucket) and ifb(Intermediate Functional Block)
# to discipline traffic.
# Default 0-sized value means unlimited rate.
#tx_rate_limiter_max_rate = 0

# Set where to save the guest memory dump file.
# If set, when GUEST_PANICKED event occurred,
# guest memeory will be dumped to host filesystem under guest_memory_dump_path,
# This directory will be created automatically if it does not exist.
#
# The dumped file(also called vmcore) can be processed with crash or gdb.
#
# WARNING:
#   Dump guest’s memory can take very long depending on the amount of guest memory
#   and use much disk space.
#guest_memory_dump_path="/var/crash/kata"

# If enable paging.
# Basically, if you want to use "gdb" rather than "crash",
# or need the guest-virtual addresses in the ELF vmcore,
# then you should enable paging.
#
# See: https://www.qemu.org/docs/master/qemu-qmp-ref.html#Dump-guest-memory for details
#guest_memory_dump_paging=false

# Enable swap in the guest. Default false.
# When enable_guest_swap is enabled, insert a raw file to the guest as the swap device
# if the swappiness of a container (set by annotation "io.katacontainers.container.resource.swappiness")
# is bigger than 0.
# The size of the swap device should be
# swap_in_bytes (set by annotation "io.katacontainers.container.resource.swap_in_bytes") - memory_limit_in_bytes.
# If swap_in_bytes is not set, the size should be memory_limit_in_bytes.
# If swap_in_bytes and memory_limit_in_bytes is not set, the size should
# be default_memory.
#enable_guest_swap = true

# use legacy serial for guest console if available and implemented for architecture. Default false
#use_legacy_serial = true

# disable applying SELinux on the VMM process (default false)
disable_selinux=false

[factory]
# VM templating support. Once enabled, new VMs are created from template
# using vm cloning. They will share the same initial kernel, initramfs and
# agent memory by mapping it readonly. It helps speeding up new container
# creation and saves a lot of memory if there are many kata containers running
# on the same host.
#
# When disabled, new VMs are created from scratch.
#
# Note: Requires "initrd=" to be set ("image=" is not supported).
#
# Default false
#enable_template = true

# Specifies the path of template.
#
# Default "/run/vc/vm/template"
#template_path = "/run/vc/vm/template"

# The number of caches of VMCache:
# unspecified or == 0   --> VMCache is disabled
# > 0                   --> will be set to the specified number
#
# VMCache is a function that creates VMs as caches before using it.
# It helps speed up new container creation.
# The function consists of a server and some clients communicating
# through Unix socket.  The protocol is gRPC in protocols/cache/cache.proto.
# The VMCache server will create some VMs and cache them by factory cache.
# It will convert the VM to gRPC format and transport it when gets
# requestion from clients.
# Factory grpccache is the VMCache client.  It will request gRPC format
# VM and convert it back to a VM.  If VMCache function is enabled,
# kata-runtime will request VM from factory grpccache when it creates
# a new sandbox.
#
# Default 0
#vm_cache_number = 0

# Specify the address of the Unix socket that is used by VMCache.
#
# Default /var/run/kata-containers/cache.sock
#vm_cache_endpoint = "/var/run/kata-containers/cache.sock"

[agent.kata]
# If enabled, make the agent display debug-level messages.
# (default: disabled)
#enable_debug = true

# Enable agent tracing.
#
# If enabled, the agent will generate OpenTelemetry trace spans.
#
# Notes:
#
# - If the runtime also has tracing enabled, the agent spans will be
#   associated with the appropriate runtime parent span.
# - If enabled, the runtime will wait for the container to shutdown,
#   increasing the container shutdown time slightly.
#
# (default: disabled)
#enable_tracing = true

# Comma separated list of kernel modules and their parameters.
# These modules will be loaded in the guest kernel using modprobe(8).
# The following example can be used to load two kernel modules with parameters
#  - kernel_modules=["e1000e InterruptThrottleRate=3000,3000,3000 EEE=1", "i915 enable_ppgtt=0"]
# The first word is considered as the module name and the rest as its parameters.
# Container will not be started when:
#  * A kernel module is specified and the modprobe command is not installed in the guest
#    or it fails loading the module.
#  * The module is not available in the guest or it doesn't met the guest kernel
#    requirements, like architecture and version.
#
kernel_modules=[]

# Enable debug console.

# If enabled, user can connect guest OS running inside hypervisor
# through "kata-runtime exec <sandbox-id>" command

#debug_console_enabled = true

# Agent connection dialing timeout value in seconds
# (default: 30)
#dial_timeout = 30

[runtime]
# If enabled, the runtime will log additional debug messages to the
# system log
# (default: disabled)
#enable_debug = true
#
# Internetworking model
# Determines how the VM should be connected to the
# the container network interface
# Options:
#
#   - macvtap
#     Used when the Container network interface can be bridged using
#     macvtap.
#
#   - none
#     Used when customize network. Only creates a tap device. No veth pair.
#
#   - tcfilter
#     Uses tc filter rules to redirect traffic from the network interface
#     provided by plugin to a tap interface connected to the VM.
#
internetworking_model="tcfilter"

# disable guest seccomp
# Determines whether container seccomp profiles are passed to the virtual
# machine and applied by the kata agent. If set to true, seccomp is not applied
# within the guest
# (default: true)
disable_guest_seccomp=true

# If enabled, the runtime will create opentracing.io traces and spans.
# (See https://www.jaegertracing.io/docs/getting-started).
# (default: disabled)
#enable_tracing = true

# Set the full url to the Jaeger HTTP Thrift collector.
# The default if not set will be "http://localhost:14268/api/traces"
#jaeger_endpoint = ""

# Sets the username to be used if basic auth is required for Jaeger.
#jaeger_user = ""

# Sets the password to be used if basic auth is required for Jaeger.
#jaeger_password = ""

# If enabled, the runtime will not create a network namespace for shim and hypervisor processes.
# This option may have some potential impacts to your host. It should only be used when you know what you're doing.
# `disable_new_netns` conflicts with `internetworking_model=tcfilter` and `internetworking_model=macvtap`. It works only
# with `internetworking_model=none`. The tap device will be in the host network namespace and can connect to a bridge
# (like OVS) directly.
# (default: false)
#disable_new_netns = true

# if enabled, the runtime will add all the kata processes inside one dedicated cgroup.
# The container cgroups in the host are not created, just one single cgroup per sandbox.
# The runtime caller is free to restrict or collect cgroup stats of the overall Kata sandbox.
# The sandbox cgroup path is the parent cgroup of a container with the PodSandbox annotation.
# The sandbox cgroup is constrained if there is no container type annotation.
# See: https://pkg.go.dev/github.com/kata-containers/kata-containers/src/runtime/virtcontainers#ContainerType
sandbox_cgroup_only=false

# If enabled, the runtime will attempt to determine appropriate sandbox size (memory, CPU) before booting the virtual machine. In
# this case, the runtime will not dynamically update the amount of memory and CPU in the virtual machine. This is generally helpful
# when a hardware architecture or hypervisor solutions is utilized which does not support CPU and/or memory hotplug.
# Compatibility for determining appropriate sandbox (VM) size:
# - When running with pods, sandbox sizing information will only be available if using Kubernetes >= 1.23 and containerd >= 1.6. CRI-O
#   does not yet support sandbox sizing annotations.
# - When running single containers using a tool like ctr, container sizing information will be available.
static_sandbox_resource_mgmt=false

# If specified, sandbox_bind_mounts identifieds host paths to be mounted (ro) into the sandboxes shared path.
# This is only valid if filesystem sharing is utilized. The provided path(s) will be bindmounted into the shared fs directory.
# If defaults are utilized, these mounts should be available in the guest at `/run/kata-containers/shared/containers/sandbox-mounts`
# These will not be exposed to the container workloads, and are only provided for potential guest services.
sandbox_bind_mounts=[]

# VFIO Mode
# Determines how VFIO devices should be be presented to the container.
# Options:
#
#  - vfio
#    Matches behaviour of OCI runtimes (e.g. runc) as much as
#    possible.  VFIO devices will appear in the container as VFIO
#    character devices under /dev/vfio.  The exact names may differ
#    from the host (they need to match the VM's IOMMU group numbers
#    rather than the host's)
#
#  - guest-kernel
#    This is a Kata-specific behaviour that's useful in certain cases.
#    The VFIO device is managed by whatever driver in the VM kernel
#    claims it.  This means it will appear as one or more device nodes
#    or network interfaces depending on the nature of the device.
#    Using this mode requires specially built workloads that know how
#    to locate the relevant device interfaces within the VM.
#
vfio_mode="guest-kernel"

# If enabled, the runtime will not create Kubernetes emptyDir mounts on the guest filesystem. Instead, emptyDir mounts will
# be created on the host and shared via virtio-fs. This is potentially slower, but allows sharing of files from host to guest.
disable_guest_empty_dir=false

# Enabled experimental feature list, format: ["a", "b"].
# Experimental features are features not stable enough for production,
# they may break compatibility, and are prepared for a big version bump.
# Supported experimental features:
# (default: [])
experimental=[]

# If enabled, user can run pprof tools with shim v2 process through kata-monitor.
# (default: false)
# enable_pprof = true

# WARNING: All the options in the following section have not been implemented yet.
# This section was added as a placeholder. DO NOT USE IT!
[image]
# Container image service.
#
# Offload the CRI image management service to the Kata agent.
# (default: false)
#service_offload = true

# Container image decryption keys provisioning.
# Applies only if service_offload is true.
# Keys can be provisioned locally (e.g. through a special command or
# a local file) or remotely (usually after the guest is remotely attested).
# The provision setting is a complete URL that lets the Kata agent decide
# which method to use in order to fetch the keys.
#
# Keys can be stored in a local file, in a measured and attested initrd:
#provision=data:///local/key/file
#
# Keys could be fetched through a special command or binary from the
# initrd (guest) image, e.g. a firmware call:
#provision=file:///path/to/bin/fetcher/in/guest
#
# Keys can be remotely provisioned. The Kata agent fetches them from e.g.
# a HTTPS URL:
#provision=https://my-key-broker.foo/tenant/<tenant-id>
```

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

# Kata Runtime

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

# Kata Monitor

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
