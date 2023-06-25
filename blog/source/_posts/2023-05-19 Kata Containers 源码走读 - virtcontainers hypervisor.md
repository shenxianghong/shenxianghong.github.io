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
	// -name 参数，例如 sandbox-4230a13dac935c3fef99f8b15d27d493ff1de957224043354374efd50bdfeeb7
	// sandbox-<qemuID>
	Name string

	// UUID is the qemu process UUID.
	// -uuid 参数，例如 -uuid 42f0c7b9-7aa9-4581-a26c-2d84b40f1190
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
	// -qmp 参数，例如 -qmp unix:/run/vc/vm/<qemuid>/qmp.sock,server=on,wait=off
	// Type: unix
	// Name: 
	// - root 权限: /run/vc/vm/<qemuID>/qmp.sock
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/qmp.sock（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
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
	//   例如 -device virtio-serial-pci,disable-modern=true,id=serial0 -device virtconsole,chardev=charconsole0,id=console0 -chardev socket,id=charconsole0,path=/run/vc/vm/<qemuID>/console.sock,server=on,wait=off
	//   CharDevice
	//     Driver: virtconsole
	//     Backend: socket
	//     DeviceID: console0
	//     ID: charconsole0
	//     Path: 
	//     - root 权限: /run/vc/vm/<qemuID>/console.sock
	//     - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/console.sock（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	//   SerialDevice
	//     Driver: virtio-serial
	//     ID: serial0
	//     DisableModern: 
	//     - amd64: 当未禁用 [hypervisor].disable_nesting_checks，且 CPU flags 中有 hypervisor，视为 true；否则，为 false
	//     - arm64: false
	//     MaxPorts: 2
	// - 启用 [hypervisor].use_legacy_serial
	//   例如 -serial chardev:charconsole0 -chardev socket,id=charconsole0,path=/run/vc/vm/<qemuID>/console.sock,server=on,wait=off
	//   CharDevice
	//     Driver: serial
	//     Backend: socket
	//     DeviceID: console0
	//     ID: charconsole0
	//     Path: 
	//     - root 权限: /run/vc/vm/<qemuID>/console.sock
	//     - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/console.sock（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
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
	// -vga 参数，例如 -vga none
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
	// -global 参数，例如 -global kvm-pit.lost_tick_policy=discard
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
	// -pidfile 参数，例如 -pidfile /run/vc/vm/<qemuID>/pid
	// - root 权限: /run/vc/vm/<qemuID>/pid
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/pid（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	PidFile string

	// LogFile is the -D parameter
	// -D 参数，例如 -D /run/vc/vm/<qemuID>/qemu.log
	// - root 权限: /run/vc/vm/<qemuID>/qemu.log
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/qemu.log（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	LogFile string

	qemuParams []string
}
```

```go
type cloudHypervisor struct {
	console         console.Console
	virtiofsDaemon  VirtiofsDaemon
	APIClient       clhClient
	ctx             context.Context
	id              string
	netDevices      *[]chclient.NetConfig
	devicesIds      map[string]string
	netDevicesFiles map[string][]*os.File
	vmconfig        chclient.VmConfig
	state           CloudHypervisorState
	config          HypervisorConfig
}
```

```go
// firecracker is an Hypervisor interface implementation for the firecracker VMM.
type firecracker struct {
	console console.Console
	ctx     context.Context

	pendingDevices []firecrackerDevice // Devices to be added before the FC VM ready

	firecrackerd *exec.Cmd              //Tracks the firecracker process itself
	fcConfig     *types.FcConfig        // Parameters configured before VM starts
	connection   *client.FirecrackerAPI //Tracks the current active connection

	id               string //Unique ID per pod. Normally maps to the sandbox id
	vmPath           string //All jailed VM assets need to be under this
	chrootBaseDir    string //chroot base for the jailer
	jailerRoot       string
	socketPath       string
	hybridSocketPath string
	netNSPath        string
	uid              string //UID and GID to be used for the VMM
	gid              string
	fcConfigPath     string

	info   FirecrackerInfo
	config HypervisorConfig
	state  firecrackerState

	jailed bool //Set to true if jailer is enabled
}
```

```go
// Acrn is an Hypervisor interface implementation for the Linux acrn hypervisor.
type Acrn struct {
	sandbox    *Sandbox
	ctx        context.Context
	arch       acrnArch
	store      persistapi.PersistDriver
	id         string
	state      AcrnState
	acrnConfig Config
	config     HypervisorConfig
	info       AcrnInfo
}
```

## CreateVM

**准备创建 VM 所需的配置信息**

### QEMU

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L490)

1. 根据 QEMU 实现的 hypervisor 配置项初始化对应架构下的 qemu，其中包含了 qemu-system（govmmQemu.Config）和 virtiofsd/nydusd（VirtiofsDaemon）进程的配置参数

# VirtiofsDaemon

*<u>src/runtime/virtcontainers/virtiofsd.go</u>*

VirtiofsDaemon 是用于 host 与 guest 的文件共享的进程服务，实现包括传统的 virtiofsd 以及针对蚂蚁社区提出的 nydusd。

```go
type virtiofsd struct {
	// Neded by tracing
	ctx context.Context
	// PID process ID of virtiosd process
	PID int

	// path to virtiofsd daemon
	// [hypervisor].shared_fs
	path string

	// socketPath where daemon will serve
	// - root 权限: /run/vc/vm/<qemuID>/vhost-fs.sock
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/vhost-fs.sock（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	socketPath string

	// cache size for virtiofsd
	// [hypervisor].virtio_fs_cache
	cache string

	// sourcePath path that daemon will help to share
	// - root 权限: /run/kata-containers/shared/sandboxes/<containerID>/shared
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/<containerID>/shared（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	sourcePath string

	// extraArgs list of extra args to append to virtiofsd command
	// [hypervisor].virtio_fs_extra_args
	extraArgs []string
}
```

```go
type nydusd struct {
	startFn         func(cmd *exec.Cmd) error // for mock testing
	waitFn          func() error              // for mock
	setupShareDirFn func() error              // for mock testing
  	pid             int
  
	// [hypervisor].shared_fs
	path string
  
	// - root 权限: /run/vc/vm/<qemuID>/vhost-fs.sock
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/vhost-fs.sock（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	sockPath string

	// - root 权限: /run/vc/vm/<qemuID>/nydusd-api.sock
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/vc/vm/<qemuID>/nydusd-api.sock（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	apiSockPath string

	// - root 权限: /run/kata-containers/shared/sandboxes/<containerID>/shared
	// - rootless 权限: <XDG_RUNTIME_DIR>/run/kata-containers/shared/sandboxes/<containerID>/shared（XDG_RUNTIME_DIR 默认为 /run/user/<UID>）
	sourcePath string
  
	// [hypervisor].virtio_fs_extra_args
	extraArgs []string

	// [hypervisor].debug
	debug bool
}
```

