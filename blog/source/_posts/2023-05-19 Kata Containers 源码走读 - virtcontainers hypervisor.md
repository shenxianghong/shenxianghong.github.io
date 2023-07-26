---
title: "「 Kata Containers 」源码走读 — virtcontainers/hypervisor"
excerpt: "virtcontainers 中与 Hypervisor 等虚拟化相关的流程梳理"
cover: https://picsum.photos/0?sig=20230519
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2023-05-19
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

# Hypervisor

*<u>src/runtime/virtcontainers/hypervisor.go</u>*

Kata Containers 支持的 hypervisor 有 QEMU、Cloud Hypervisor、Firecracker、ACRN 以及 DragonBall，其中 DragonBall 是 Kata Containers 3.0 为新增的 runtime-rs 组件引入的内置 hypervisor，而 runtime-rs 的整体架构区别于当前的 runtime，不在此详读 DragonBall 实现。

***目前，暂时走读 QEMU 实现，后续补充其他 hypervisor 实现。***

```go
// qemu is an Hypervisor interface implementation for the Linux qemu hypervisor.
type qemu struct {
	// 针对不同 CPU 架构下的 QEMU 配置项，后续会进一步构建成 qemuConfig
	arch qemuArch

	virtiofsDaemon VirtiofsDaemon

	ctx context.Context
	id string
	mu sync.Mutex

	// fds is a list of file descriptors inherited by QEMU process
	// they'll be closed once QEMU process is running
	fds []*os.File

	// HotplugVFIOOnRootBus: [hypervisor].hotplug_vfio_on_root_bus
	// PCIeRootPort: [hypervisor].pcie_root_port
	state QemuState

	// path
	// <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/qmp.sock
	qmpMonitorCh qmpChannel

	// QEMU 进程的配置参数
	qemuConfig govmmQemu.Config

	// QEMU 实现下的 hypervisor 配置
	config HypervisorConfig

	// if in memory dump progress
	memoryDumpFlag sync.Mutex

	// NVDIMM 设备数量
	nvdimmCount int

	stopped bool
}
```

```go
/* 前置说明
 protection
 - amd64：默认为 noneProtection
   如果启用 [hypervisor].confidential_guest，则进一步判断 protection
   - 如果 host 上 /sys/firmware/tdx_seam/ 文件夹存在或者 CPU flags 中包含 tdx，则为 tdxProtection（Intel Trust Domain Extensions）
   - 如果 host 上 /sys/module/kvm_amd/parameters/sev 文件存在且内容为 1 或者 Y 则为 sevProtection（AMD Secure Encrypted Virtualization）
  - arm64：noneProtection 
*/

// Config is the qemu configuration structure.
// It allows for passing custom settings and parameters to the qemu API.
// nolint: govet
type Config struct {
	// Path is the qemu binary path.
	// - amd64: /usr/bin/qemu-system-x86_64
	// - arm64: /usr/bin/qemu-system-aarch64
	Path string

	// Ctx is the context used when launching qemu.
	Ctx context.Context
	// User ID.
	Uid uint32
	// Group ID.
	Gid uint32
	// Supplementary group IDs.
	Groups []uint32

	// Name is the qemu guest name
	// -name 参数
	// sandbox-<sandboxID>
	Name string

	// UUID is the qemu process UUID.
	// -uuid 参数
	// 随机生成
	UUID string

	// CPUModel is the CPU model to be used by qemu.
	// -cpu 参数，例如 -cpu host,pmu=off
	// 默认为 host，如果指定 [hypervisor].cpu_features 则继续追加
	CPUModel string

	// SeccompSandbox is the qemu function which enables the seccomp feature
	// [hypervisor].seccompsandbox
	SeccompSandbox string

	// Machine
	// -machine 参数，例如 -machine q35,accel=kvm,kernel_irqchip=on,nvdimm=on
	// Type: [hypervisor].machine_type
	// - amd64: 默认为 q35
	// - arm64: virt
	// Options: 
	// - amd64: 默认为 accel=kvm,kernel_irqchip=on
	//   如果启用 [hypervisor].confidential_guest 或者启用 hypervisor[enable_iommu]，则覆盖 Options 为 accel=kvm,kernel_irqchip=split
	//   如果 sgxEPCSize 不为 0，则追加 sgx-epc.0.memdev=epc0,sgx-epc.0.node=0
	//   如果启用 [hypervisor].confidential_guest: 
	//   - 如果 protection 为 tdxProtection，则追加 kvm-type=tdx,confidential-guest-support=tdx
	//   - 如果 protection 为 sevProtection，则追加 confidential-guest-support=sev
	//   如果镜像类型为 [hypervisor].image 且 disableNvdimm 为 false，则追加 nvdimm=on
	// - arm64: usb=off,accel=kvm,gic-version=host
	// 如果指定 [hypervisor].machine_accelerators，则继续追加
	Machine Machine

	// QMPSockets is a slice of QMP socket description.
	// -qmp 参数，例如 -qmp unix:/run/vc/vm/<sandboxID>/qmp.sock,server=on,wait=off
	// Type: unix
	// Name: <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/qmp.sock
	// Server: true
	// NoWait: true
	QMPSockets []QMPSocket

	// Devices is a list of devices for qemu to create and drive.
	// -device 参数
	// 
	// =========== Bridge ===========
	// 例如 -device pci-bridge,bus=pcie.0,id=pci-bridge-0,chassis_nr=1,shpc=off,addr=2,io-reserve=4k,mem-reserve=1m,pref64-reserve=1m
	// BridgeDevice（数量等于 [hypervisor].default_bridges）
	//   Type: 默认为 0，即 PCI，如果 bridge 类型为 PCIe，则为 PCIe
	//   Bus: 默认为 pci.0，如果 Machine.Type 为 q35 或者 virt，则为 pcie.0
	//   ID: <bt>-bridge-<idx>，其中 idx 为 0 ~ [hypervisor].default_bridges 的递增索引
	//   - 如果 Machine.Type 为 q35、virt 和 pseries，则 bt 为 pci，容量为 30
	//   - 如果 Machine.Type 为 s390-ccw-virtio，则 bt 为 ccw，容量为 65535
	//   Chassis: idx + 1，其中 idx 为 bridge 列表的索引
	// 	 SHPC: false
	//   Addr: idx + 2，其中 idx 为 bridge 列表的索引
	//   IOReserve: 4k
	//   MemReserve: 1m
	//   Pref64Reserve: 1m
	// 
	// =========== Console ===========
	// - 禁用 [hypervisor].use_legacy_serial
	//   例如 -device virtio-serial-pci,disable-modern=true,id=serial0 -device virtconsole,chardev=charconsole0,id=console0 -chardev socket,id=charconsole0,path=/run/vc/vm/<sandboxID>/console.sock,server=on,wait=off
	//   CharDevice
	//     Driver: virtconsole
	//     Backend: socket
	//     DeviceID: console0
	//     ID: charconsole0
	//     Path: <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/console.sock
	//   SerialDevice
	//     Driver: virtio-serial
	//     ID: serial0
	//     DisableModern: 
	//     - amd64: 当未禁用 [hypervisor].disable_nesting_checks，且 CPU flags 中有 hypervisor，视为 true；否则，为 false
	//     - arm64: false
	//     MaxPorts: 2
	// - 启用 [hypervisor].use_legacy_serial
	//   例如 -serial chardev:charconsole0 -chardev socket,id=charconsole0,path=/run/vc/vm/<sandboxID>/console.sock,server=on,wait=off
	//   CharDevice
	//     Driver: serial
	//     Backend: socket
	//     DeviceID: console0
	//     ID: charconsole0
	//     Path: <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/console.sock
	//   LegacySerialDevice
	//     Chardev: charconsole0
	// 
	// =========== Image（当镜像类型为 [hypervisor].image） ===========
	// - 禁用 [hypervisor].disable_image_nvdimm
	//   例如 -drive id=image-199896efe4d8ad3b,file=/opt/kata/share/kata-containers/kata-clearlinux-latest.image,aio=threads,format=raw,if=none,readonly=on
	//   BlockDrive
	//     File: [hypervisor].image
	//	   Format: raw
	//	   ID: image-<随机字符串>
	//	   ShareRW: true
	//	   ReadOnly: true
	// - 启用 [hypervisor].disable_image_nvdimm
	//   例如 -device nvdimm,id=nv0,memdev=mem0,unarmed=on -object memory-backend-file,id=mem0,mem-path=/opt/kata/share/kata-containers/kata-clearlinux-latest.image,size=134217728,readonly=on
	//   Object
	//     Driver: nvdimm
	//     Type: memory-backend-file
	//     DeviceID: nv0
	//     ID: mem0
	//     MemPath: [hypervisor].image
	//     Size: [hypervisor].image 大小
	//     ReadOnly: true
	// 
	// =========== IOMMU（当启用 [hypervisor].enable_iommu） ===========
	// IommuDev
	//   Intremap: true
	//   DeviceIotlb: true
	//   CachingMode: true
	// 
	// =========== PVPanic（当指定 [hypervisor].guest_memory_dump_path） ===========
	// PVPanicDevice
	//   NoShutdown: true
	// 
	// =========== BlockDeviceDriver（当 [hypervisor].block_device_driver 为 virtio-scsi） ===========
	// 例如 -device virtio-scsi-pci,id=scsi0,disable-modern=true
	// SCSIController
	//   ID: scsi0
	//   DisableModern: 
	//   - amd64: 当未禁用 [hypervisor].disable_nesting_checks，且 CPU flags 中有 hypervisor，视为 true；否则，为 false
	//   - arm64: false
	//   IOThread:（当启用 [hypervisor].enable_iothreads）
	//     ID: iothread-<随机字符串>
	//
	// =========== Protection ===========	
	// Object（当 sgxEPCSize 不为 0 时）
	//   Type: memory-backend-epc
	//   ID: epc0
	//   Prealloc: true
	//   Size: sgxEPCSize
	// Object（当 protection 为 tdxProtection 时）
	//   Driver: loader
	//   Type: tdx-guest
	//   ID: tdx
	//   DeviceID: fd<idx>，其中 idx 为 loader 类型 Driver 的统计数量
	//   Debug: false
	//   File: [hypervisor].firmware
	//   FirmwareVolume: [hypervisor].firmware_volume
	// Object（当 protection 为 sevProtection 时）
	//   Type: sev-guest
	//	 ID: sev
	//   Debug: false
	//   File: [hypervisor].firmware
	//   CBitPos: ebx & 0x3F
	//   ReducedPhysBits: (ebx >> 6) & 0x3F
	//
	// =========== rngDev（当 Machine.Type 不为 s390-ccw-virtio）===========
	// RNGDev
	// 例如 -object rng-random,id=rng0,filename=/dev/urandom
	//   ID: rng0
	//   FileName: [hypervisor].entropy_source
	//
	// =========== PCIe（当 [hypervisor].pcie_root_port 大于 0 且 Machine.Type 为 q35 或 virt）===========
	// PCIeRootPortDevice（数量等于 [hypervisor].pcie_root_port）
	// 例如 -device pcie-root-port,id=rp1,bus=pcie.0,chassis=0,slot=1,multifunction=off,pref64-reserve=2097152B,mem-reserve=4194304B
	//   ID: rp<idx>，其中 idx 为 0 ~ [hypervisor].pcie_root_port 的递增索引
	//   Bus: pcie.0
	//   Chassis: 0
	//   Slot: idx
	//   Multifunction: false
	//   Addr: 0
	//   MemReserve: 默认 4MB，如果累加每个 BAR 的 32 位内存窗口值更大，则以此值为准，并乘以 2
	//   Pref64Reserve: 默认 2MB，如果累加每个 BAR 的 64 位内存窗口值更大，则以此值为准
	Devices []Device

	// RTC is the qemu Real Time Clock configuration
	// -rtc 参数，例如 -rtc base=utc,driftfix=slew,clock=host
	// Base: utc
	// Clock: host
	// DriftFix: slew
	RTC RTC

	// VGA is the qemu VGA mode.
	// -vga 参数
	// none
	VGA string

	// Kernel is the guest kernel configuration.
	// -kernel 参数，例如 -kernel /opt/kata/share/kata-containers/vmlinux-5.19.2-96
	// -initrd 参数，例如 -initrd /opt/kata/share/kata-containers/kata-alpine-3.15.initrd
 	// -append 参数，例如 -append tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 console=hvc0 console=hvc1 debug panic=1 nr_cpus=8 scsi_mod.scan=none agent.log=debug agent.debug_console agent.debug_console_vport=1026
	// Path: [hypervisor].kernel
	// InitrdPath: [hypervisor].initrd，当镜像类型为 [hypervisor].image 时，没有 -initrd 参数
	// Params: 
	// - kernelParams: 
	//   - amd64: 默认为 tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 panic=1 nr_cpus=[hypervisor].default_maxvcpus
	//     如果启用 [hypervisor].enable_iommu，则追加 intel_iommu=on iommu=pt
	//     如果镜像类型为 [hypervisor].image: 
	//     - 如果 disableNvdimm 为 true，则追加 root=/dev/vda1 rootflags=data=ordered errors=remount-ro ro rootfstype=ext4
	//     - 如果 disableNvdimm 为 false: 
	//       - 如果 dax 为 false，则追加 root=/dev/pmem0p1 rootflags=data=ordered errors=remount-ro ro rootfstype=ext4
	//       - 如果 dax 为 true，则追加 root=/dev/pmem0p1 rootflags=dax data=ordered errors=remount-ro ro rootfstype=ext4
	//     如果启用 [hypervisor].use_legacy_serial，则追加 console=ttyS0，否则，则追加 console=hvc0 console=hvc1
	//   - arm64: iommu.passthrough=0 panic=1 nr_cpus=[hypervisor].default_maxvcpus
	// - kernelParamsDebug: 默认为 debug，如果镜像类型为 [hypervisor].image，则追加 systemd.show_status=true systemd.log_level=debug
	// - kernelParamsNonDebug: 默认为 quiet，如果镜像类型为 [hypervisor].image，则追加 systemd.show_status=false
	// 由以上三个参数组成，具体为 kernelParams + kernelParamsDebug/kernelParamsNonDebug（取决于 [hypervisor].enable_debug），如果指定 [hypervisor].kernel_params，则继续追加
	Kernel Kernel

	// Memory is the guest memory configuration.
	// -m 参数，例如 -m 2048M,slots=10,maxmem=12799M
	// Size: [hypervisor].default_memory
	// Slots: [hypervisor].memory_slots
	// MaxMem: 
	// - amd64: [hypervisor].memory_offset + [hypervisor].default_maxmemory
	// - arm64: [hypervisor].default_maxmemory
	// Path: 
	// - 如果为 VM factory 场景，则为 [factory].template_path/memory
	// - 如果 [hypervisor].shared_fs 为 virtio-fs 或者 virtio-fs-nydus, 再或者 annotations["io.katacontainers.config.hypervisor.file_mem_backend"] 不为空，则为 /dev/shm（如果 annotations 传递，则以 annotations 为准）
	Memory Memory

	// SMP is the quest multi processors configuration.
	// -smp 参数，例如 -smp 1,cores=1,threads=1,sockets=8,maxcpus=8
	// CPUs: [hypervisor].default_vcpus
	// Cores: 1
	// Threads: 1
	// Sockets: [hypervisor].default_maxvcpus
	// MaxCPUs: [hypervisor].default_maxvcpus
	SMP SMP

	// GlobalParam is the -global parameter.
	// -global 参数
	// kvm-pit.lost_tick_policy=discard
	GlobalParam string

	// Knobs is a set of qemu boolean settings.
	// -no-user-config -nodefaults -nographic --no-reboot -daemonize 参数
	// NoUserConfig、NoDefaults、NoGraphic、NoReboot、Daemonize: true
	// MemPrealloc: 默认为 [hypervisor].enable_mem_prealloc，如果 [hypervisor].shared_fs 为 virtio-fs 或者 virtio-fs-nydus, 再或者 annotations["io.katacontainers.config.hypervisor.file_mem_backend"] 不为空，并且启用 [hypervisor].enable_hugepages，则为 true
	// HugePages: [hypervisor].enable_hugepages
	// IOMMUPlatform: [hypervisor].enable_iommu_platform
	// FileBackedMem: 
	// - 如果为 VM factory 场景，则为 true
	// - 如果 [hypervisor].shared_fs 为 virtio-fs 或者 virtio-fs-nydus, 再或者 annotations["io.katacontainers.config.hypervisor.file_mem_backend"] 不为空，则为 true
	// MemShared: 
	// - 如果为 VM factory 中的启动为模板场景，则为 true
	// - 如果 [hypervisor].shared_fs 为 virtio-fs 或者 virtio-fs-nydus, 再或者 annotations["io.katacontainers.config.hypervisor.file_mem_backend"] 不为空，则为 true
	// - 如果启用 [hypervisor].enable_vhost_user_store，则为 true
	Knobs Knobs

	// Bios is the -bios parameter
	// -bios 参数
	// [hypervisor].firmware
	Bios string

	// PFlash specifies the parallel flash images (-pflash parameter)
	// -pflash 参数
	// [hypervisor].pflashes
	PFlash []string

	// Incoming controls migration source preparation
	// MigrationType: 如果为 VM factory 中的从模板启动场景，则为 3
	Incoming Incoming

	// fds is a list of open file descriptors to be passed to the spawned qemu process
	fds []*os.File

	// FwCfg is the -fw_cfg parameter
	FwCfg []FwCfg

	// Devices 中 SCSIController.IOThread
	IOThreads []IOThread

	// PidFile is the -pidfile parameter
	// -pidfile 参数
	// <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/pid
	PidFile string

	// LogFile is the -D parameter
	// -D 参数
	// <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/qemu.log
	LogFile string

	// 基于上述的 QEMU 配置项，构建 -name、-uuid、-machine、-cpu、-qmp、-m、-device、-rtc、-global、-pflash 等参数信息
	qemuParams []string
}
```

Hypervisor 中声明的 **HypervisorConfig**、**setConfig**、**GetVirtioFsPid**，**fromGrpc**、**toGrpc**，**Save** 和 **Load** 均为参数获取与赋值，无复杂逻辑，不作详述。

## qmpSetup

**初始化 QMP 服务**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L1106)

1. 如果当前 QMP 服务已经就绪，则直接返回
2. 启动 goroutine，处理 QMP 事件，如果为 GUEST_PANICKED 事件，并且指定了 [hypervisor].guest_memory_dump_path，则转储 VM 的内存信息
   1. 保存 sandbox 元数据信息
      1. 创建 [hypervisor].guest_memory_dump_path/\<sandboxID\>/state 目录（如果不存在）
      2. 将 \<storage.PersistDriver.RunStoragePath\>/\<sandboxID\> 目录下的内容拷贝至 \[hypervisor].guest_memory_dump_path/\<sandboxID\>/state 目录中
      3. 将 hypervisor 的配置信息写入 \[hypervisor].guest_memory_dump_path/\<sandboxID\>/hypervisor.conf 文件中
      4. 执行 qemu-system --version，获取 QEMU 的版本信息，写入 \[hypervisor].guest_memory_dump_path/\<sandboxID\>/hypervisor.version 文件中
   2. 校验 \[hypervisor].guest_memory_dump_path/\<sandboxID\> 目录空间是否为 VM 内存（静态 + 热添加）的两倍以上
   3. 向 QMP 服务发送 dump-guest-memory 命令，将 VM 中的内存内容转储到 \[hypervisor].guest_memory_dump_path/\<sandboxID\>/vmcore-\<currentTime\>.elf 文件中，是否内存分页取决于 [hypervisor].guest_memory_dump_paging
3. 启动 QMP 服务，监听 qmp.sock，校验 QEMU 版本是否大于 5.x
4. 向 QMP 服务发送 qmp_capabilities 命令，从 capabilities negotiation 模式切换至 command 模式，命令无报错则视为 VM 处于正常运行状态

## hotplugDevice

**VM 设备热插拔**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L1799)

***block***

*先调用 blockdev-add 命令是为了创建一个块设备，并将其配置为所需的类型、格式等。这个过程中，QEMU 会加载相应的块设备驱动程序，并为块设备分配所需的资源。然后调用 device_add 命令是为了将该块设备添加到 VM 中，使其成为 VM 的一部分。*

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 如果为热添加
   1. 如果 [hypervisor].block_device_driver 为 nvdimm，或者为 PMEM 设备，则向 QMP 服务发送 object-add 和 device_add 命令，为 VM 添加块设备；否则，向 QMP 服务发送 blockdev-add 命令，准备块设备
   2. 如果 [hypervisor].block_device_driver 为 virtio-blk 或 virtio-blk-ccw 时，会在 bridge 中新增设备信息维护；如果为 virtio-scsi 时，设备会添加到 scsi0.0 总线中
   3. 向 QMP 服务发送 device_add 命令，为 VM 添加块设备
3. 如果为热移除
   1. 如果 [hypervisor].block_device_driver 为 virtio-blk 时，移除 bridge 中维护的设备信息
   2. 向 QMP 服务发送 device_del 和 blockdev-del 命令，移除 VM 中的指定块设备

***CPU***

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 如果为热添加
   1. 如果当前 VM 的 CPU 数量与待热添加的 CPU 数量之和超出 [hypervisor].default_maxvcpus 限制，并不会不报错中断，而是热插至最大数量限制
   2. 向 QMP 服务发送 query-hotpluggable-cpus 命令，获得 host 上可插拔的 CPU 列表
   3. 遍历所有可插拔的 CPU，向 QMP 服务发送 device_add 命令，为 VM 添加未被使用的 CPU（如果 CPU 的 qom-path 不为空，则代表其正在使用中）<br>*添加失败并不会报错，而是尝试其他 CPU，直至满足数量要求或者再无可用的 CPU*
3. 如果为热移除
   1. 只有热添加的 CPU 才可以热移除，因此需要校验期望热移除的 CPU 数量是否小于当前热添加的 CPU 数量
   2. 向 QMP 服务发送 device_del 命令，移除 VM 中最近添加的 CPU（即倒序移除）

***VFIO***

*[hypervisor].hotplug_vfio_on_root_bus 决定是否允许 VFIO 设备在 root 总线上热插拔，默认为 true。VFIO 是一种用于虚拟化环境中的设备直通技术，它允许将物理设备直接分配给 VM，从而提高 VM 的性能和可靠性。然而，在桥接设备上进行 VFIO 设备的热插拔存在一些限制，特别是对于具有大型 PCI 条的设备。因此，通过将该选项设置为 true，可以在 root 总线上启用 VFIO 设备的热插拔，从而解决这些限制问题*

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 如果为热添加
   1. 如果启用 [hypervisor].hotplug_vfio_on_root_bus，则后续的设备添加操作会作用在 root 总线上，否则会作用在 bridge 上
   2. 向 QMP 服务发送 device_add 命令，为 VM 添加 VFIO-CCW、VFIO-PCI 或 VFIO-AP 设备
3. 如果为热移除
   1. 如果未启用 [hypervisor].hotplug_vfio_on_root_bus，则移除 bridge 中维护的设备信息
   2. 向 QMP 服务发送 device_del 命令，移除 VM 中的指定 VFIO 设备

***memory***

*先调用 object-add 命令是为了创建一个内存设备对象，并为其分配内存，以便后续使用。而后调用  device_add 命令是为了将该内存设备对象添加到 VM 中，使其成为 VM 的一部分，从而实现内存的热添加。*

1. 检验 VM protection 模式是否为 noneProtection，其他 VM protection 模式下均不支持内存热插拔特性
2. 调用 **qmpSetup**，初始化 QMP 服务
3. 仅支持热添加内存，不支持热移除
   1. 向 QMP 服务发送 query-memory-devices 命令，查询 VM 中所有的内存设备，用于生成下一个内存设备的 slot 序号
   2. 向 QMP 服务发送 object-add 和 device_add 命令，为 VM 添加内存设备
   5. 如果 VM 内核只支持通过探测接口热添加内存（通过内存设备的 probe 属性判断），则需要额外向 QMP 服务发送 query-memory-devices 命令，查询 VM 中最近的一个内存设备，回写其地址信息

***endpoint***

*netdev_add 添加的是网络前端设备，而 device_add 添加的是一个完整的设备，其中包括前端设备和后端设备。在添加网络设备时，通常需要先添加一个网络前端设备，然后再将它连接到一个网络后端设备上。*

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 如果为热添加
   1. 向 QMP 服务发送 getfd 命令，分别获取 tap 设备的 VMFds 和 VhostFds 的信息
   2. bridge 中新增设备信息维护
   3. 向 QMP 服务发送 netdev_add 和 device_add 命令，为 VM 添加 PCI 或 CCW 类型（[hypervisor].machine 为 s390-ccw-virtio 时）的网络设备
3. 如果为热移除
   1. 移除 bridge 中维护的设备信息
   2. 向 QMP 服务发送 device_del 和 netdev_del 命令，移除 VM 中指定的网络设备


***vhost-user***

*vhost-user 设备需要与 host 的网络堆栈进行通信，而 host 网络堆栈使用字符设备来管理网络连接。因此，要创建一个 vhost-user 设备，需要先创建一个字符设备，然后将其与 vhost-user 设备连接。*

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 如果为热添加，仅支持 vhost-user-blk-pci 类型的设备
   1. 向 QMP 服务发送 chardev-add 和 device_add 命令，为 VM 添加指定的 vhost-user 设备
3. 如果为热移除
   1. 向 QMP 服务发送 device_del 和 chardev-remove 命令，移除 VM 中指定的 vhost-user 设备

## CreateVM

**准备创建 VM 所需的配置信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L490)

1. 根据 QEMU 实现的 hypervisor 配置项初始化对应架构下的 qemu，其中包含了 qemu-system（govmmQemu.Config）和 virtiofsd/nydusd（VirtiofsDaemon）进程的配置参数

## StartVM

**启动 VM**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L800)

1. 以当前用户组信息创建 \<storage.PersistDriver.RunVMStoragePath\>/\<sandboxID\> 目录（如果不存在）
1. 如果启用 [hypervisor].enable_debug，则设置 qemuConfig.LogFile 为 \<storage.PersistDriver.RunVMStoragePath\>/\<sandboxID\>/qemu.log
1. 如果未启用 [hypervisor].disable_selinux，则向 /proc/thread-self/attr/exec （如果其不存在，则为 /proc/self/task/\<PID\>/attr/exec）中写入 OCI spec.Process.SelinuxLabel 中声明的内容，VM 启动之后会重新置空
1. 如果 [hypervisor].shared_fs 为 virtiofs-fs 或者 virtio-fs-nydus，则调用 VirtiofsDaemon 的 **Start**，启动 virtiofsd 进程，回写 virtiofsd PID 至 qemustate 中
1. 构建 QEMU 进程的启动参数、执行命令的文件句柄、属性、标准输出等信息，执行 qemu-system 可执行文件，启动 qemu-system 进程。如果启用 [hypervisor].enable_debug 并且配置中指定了日志文件路径，则读取日志内容，追加错误信息
1. 关停当前的 QMP 服务，执行类似于 **qmpSetup** 的流程，初始化 QMP 服务，无报错即视为 VM 处于正常运行状态
1. 如果 VM 从模板启动

   1. 调用 **qmpSetup**，初始化 QMP 服务
   1. 向 QMP 服务发送 migrate-set-capabilities 命令，设置在迁移过程中忽略共享内存，避免数据的错误修改和不一致性
   1. 向 QMP 服务发送 migrate-incoming 命令，用于将迁移过来的 VM 恢复到 [factory].template_path/state 中
   1. 向 QMP 服务发送 query-migrate 命令，查询迁移进度，直至完成
1. 如果启用 [hypervisor].enable_virtio_mem
   1. virtio-mem 设备后续会添加至 VM 的 root 总线中，获取地址和 bridge 等信息，后续执行 QMP 命令时传递
   1. 则向 QMP 服务发送 object-add 和 device_add 命令，为 VM 添加指定的 virio-mem 设备<br>*如果 QMP 添加设备失败，且报错中包含 Cannot allocate memory，则需要执行 echo 1 > /proc/sys/vm/overcommit_memory 解决*



## StopVM

**关闭 VM**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L972)

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 如果禁止 VM 关闭（agent 的 init 会返回 disableVMShutdown，用作 StopVM 的入参），则调用 **GetPids**，获得所有相关的 PIDs，kill 掉其中的 QEMU 进程（即列表中索引为 0 的 PID）；否则，则向 QMP 服务发送 quit 命令，关闭 QEMU 实例，关闭 VM
3. 如果 [hypervisor].shared_fs 为 virtiofs-fs 或者 virtio-fs-nydus，调用 VirtiofsDaemon 的 **Stop**，关停 virtiofsd 服务

## PauseVM

**暂停 VM**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2038)

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 向 QMP 服务发送 stop 命令，暂停 VM

## SaveVM

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2125)

1. 调用 **qmpSetup**，初始化 QMP 服务
1. 如果 VM 启动后作为模板，则向 QMP 服务发送 migrate-set-capabilities 命令，设置 VM 在迁移过程中忽略共享内存，避免数据的错误修改和不一致性
1. 向 QMP 服务发送 migrate 命令，将 VM 迁移到指定 [factory].template_path/state 中
4. 向 QMP 服务发送 query-migrate 命令，查询迁移进度，直至完成

## ResumeVM

**恢复 VM**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2045)

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 向 QMP 服务发送 cont 命令，恢复 VM

## AddDevice

**向 VM 中添加设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2053)

1. 根据不同设备类型，初始化对应的设备对象 — govmmQemu.Device，追加到 qemuConfig.Devices 中

## HotplugAddDevice

**热添加指定设备至 VM 中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L1824)

1. 调用 **hotplugDevice**，热添加指定设备至 VM 中

## HotplugRemoveDevice

**热移除 VM 中的指定设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L1837)

1. 调用 **hotplugDevice**，热移除 VM 中的指定设备

## ResizeMemory

**调整 VM 内存规格**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2200)

1. 调用 **GetTotalMemoryMB**，获取 VM 当前的内存
2. 调用 **qmpSetup**，初始化 QMP 服务
3. 如果启用 [hypervisor].enable_virtio_mem，向 QMP 服务发送 qom-set 命令，设置 virtiomem0 设备的 requested-size 属性值为待热添加的内存量（即期望的 VM 内存与 [hypervisor].default_memory 的差值），直接返回<br>*virtio-mem 只需要将用于 host 和 guest 内存共享的 virtiomem0 设备内存扩大至预期大小即可，不需要返回内存设备对象，也不会调用 agent 通知内存上线*
4. 调用 **HotplugAddDevice** 或者 **HotplugRemoveDevice**，为 VM 调整内存规格，取决于 VM 当前内存是否大于预期 VM 内存大小<br>*如果期望的 VM 内存超出了 [hypervisor].default_maxmemory 限制，也不会报错中断，而是热插至最大数量限制*

## ResizeVCPUs

**调整 VM CPU 规格**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2440)

1. 调用 **HotplugAddDevice** 或者 **HotplugRemoveDevice**，为 VM 调整 CPU 规格，取决于 VM 当前 CPU 是否大于预期 VM CPU 大小

## GetTotalMemoryMB

**获取 VM 总内存**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2185)

1. 返回 [hypervisor].default_memory 和已热添加内存之和

## GetVMConsole

**获取 VM console 地址**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2113)

1. 返回 \<storage.PersistDriver.RunVMStoragePath\>/\<sandboxID\>/console.sock

## Disconnect

**断开 QMP 连接**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2178)

1. channel 关闭，重置 QMP 对象

## Capabilities

**设置 hypervisor 支持的特性**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L195)

1. 设置 hypervisor 默认支持特性包括：块设备支持、设备多队列和文件系统共享

## GetThreadIDs

**获取 VM 中 CPU 的 threadID 信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2408)

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 向 QMP 服务发送 query-cpus-fast 命令，获取 VM 中所有 CPU 详细信息
3. 遍历所有 CPU，返回其 CPU ID 和 threadID 的映射关系

## Cleanup

**hypervisor 相关资源清理**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2473)

1. 关闭 QEMU 所有相关的文件句柄

## GetPids

**获取 hypervisor 相关的 PID**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2487)

1. 读取 \<storage.PersistDriver.RunVMStoragePath\>/\<sandboxID\>/pid 文件内容，如果 virtiofsd 服务的 PID 不为空，则一并返回

## Check

**VM 状态检查**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2620)

1. 调用 **qmpSetup**，初始化 QMP 服务
2. 向 QMP 服务发送 query-status 命令，查询并校验 VM 状态是否为 internal-error 或 guest-panicked

## GenerateSocket

**生成 host 和 guest 通信的 socket 地址**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2640)

1. 获取 /dev/vhost-vsock 设备的文件句柄
2. 获取一个从 0x3（contextID 中 1 和 2 是内部预留的） 到 0xFFFFFFFF（2^32 - 1）范围内可用的 contextID
3. 返回包含 vhost-vsock 设备的文件句柄、可用的 contextID 以及端口为 1024 的 VSock 对象

## IsRateLimiterBuiltin

**hypervisor 是否原生支持限速特性**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L2644)

1. 返回 false，QEMU 未内置支持限速功能

# VirtiofsDaemon

*<u>src/runtime/virtcontainers/virtiofsd.go</u>*

VirtiofsDaemon 是用于 host 与 guest 的文件共享的进程服务，实现包括 virtiofsd 以及蚂蚁社区提出的 nydusd。

***目前，暂时走读 virtiofsd 实现，后续补充其他 virtiofsd 实现。***

```go
// virtiofsd 进程启动参数中还有
// --syslog：用于将日志发送至系统日志中
// -o no_posix_lock：禁用 POSIX 锁定机制，从而提高文件系统的性能。但是，这也可能会导致在多个进程同时对同一个文件进行写操作时出现数据损坏的风险
type virtiofsd struct {
	// Neded by tracing
	ctx context.Context

	// PID process ID of virtiosd process
	// --fd 参数，例如 --fd=3
	// --fd 参数从 3 开始，0 为 stdin、1 为 stdout、2 为 stderr，具体取决于从 socketPath 中读取的 socket 文件句柄个数
	PID int

	// path to virtiofsd daemon
	// [hypervisor].shared_fs
	path string

	// socketPath where daemon will serve
	// <storage.PersistDriver.RunVMStoragePath>/<sandboxID>/vhost-fs.sock
	socketPath string

	// cache size for virtiofsd
	// -o 参数，例如 -o cache=auto
	// [hypervisor].virtio_fs_cache
	cache string

	// sourcePath path that daemon will help to share
	// -o 参数，例如 -o source=/run/kata-containers/shared/sandboxes/<sandboxID>/shared
	// <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/<containerID>/shared
	sourcePath string

	// extraArgs list of extra args to append to virtiofsd command
	// [hypervisor].virtio_fs_extra_args
	extraArgs []string
}
```

## Start

**启动 virtiofsd 服务**

1. 检验 virtiofsd 服务相关参数是否为空以及 <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/\<containerID\>/shared 路径是否存在
2. 获取 <storage.PersistDriver.RunVMStoragePath>/\<sandboxID\>/vhost-fs.sock 的文件句柄，并将其权限设置为 root<br>*这里区别于 QEMU 进程，QEMU 可以以非 root 运行，而 virtiofsd 暂不支持，参考 https://github.com/kata-containers/kata-containers/issues/2542*
3. 执行 virtiofsd 可执行文件，启动 virtiofsd 进程
4. 启动 goroutine，如果 virtiofsd 程序退出，则调用 Hypervisor 的 **StopVM**，执行清理操作
5. 返回 virtiofsd 进程 PID

## Stop

**关停 virtiofsd 服务**

1. kill 掉 virtiofsd 服务进程
2. 移除 <storage.PersistDriver.RunVMStoragePath>/\<sandboxID\>/vhost-fs.sock 文件

## Mount

**将 rafs 格式文件挂载至 virtiofs 挂载点**

*virtiofsd 场景下暂未实现。*

## Umount

**移除 virtiofs 挂载点下的 rafs 挂载文件**

*virtiofsd 场景下暂未实现。*
