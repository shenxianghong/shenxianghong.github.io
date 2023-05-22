---
title: "「 Kata Containers 」源码走读 — virtcontainers/hypervisor"
excerpt: "virtcontainers 中与 Hypervisor 等虚拟化相关的流程梳理"
cover: https://picsum.photos/0?sig=20230519
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-05-19
toc: true
categories:
- Container Runtime
tag:
- Kata Containers

---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

# Hypervisor

*<u>src/runtime/virtcontainers/hypervisor.go</u>*

Kata Containers 支持的 hypervisor 有 QEMU、Cloud Hypervisor、Firecracker、ACRN 以及 DragonBall，其中 DragonBall 是 Kata Containers 3.0 为新增的 runtime-rs 组件引入的内置 hypervisor，而 runtime-rs 的整体架构区别于当前的 runtime，不在此详读 DragonBall 实现。

```go
// qemu is an Hypervisor interface implementation for the Linux qemu hypervisor.
// nolint: govet
type qemu struct {
	arch qemuArch

	virtiofsDaemon VirtiofsDaemon

	ctx context.Context

	// fds is a list of file descriptors inherited by QEMU process
	// they'll be closed once QEMU process is running
	fds []*os.File

	id string

	state QemuState

	qmpMonitorCh qmpChannel

	qemuConfig govmmQemu.Config

	config HypervisorConfig

	// if in memory dump progress
	memoryDumpFlag sync.Mutex

	nvdimmCount int

	stopped bool

	mu sync.Mutex
}
```

```go
type qemuArchBase struct {
	// 固定为 /usr/bin/qemu-system-x86_64
	qemuExePath          string

    // type: [hypervisor].machine_type，默认为 q35
    // options: 默认为 accel=kvm,kernel_irqchip=on
    // 如果 sgxEPCSize 大于 0，则追加 sgx-epc.0.memdev=epc0,sgx-epc.0.node=0
	// 如果镜像类型为 [hypervisor].image 且 disableNvdimm 为 false，则追加 nvdimm=on
	// 如果启用 [hypervisor].confidential_guest，则覆盖 options 为 accel=kvm,kernel_irqchip=split
	// - 如果 protection 为 tdxProtection，则追加 kvm-type=tdx,confidential-guest-support=tdx
	// - 如果 protection 为 sevProtection，则追加 confidential-guest-support=sev
	qemuMachine          govmmQemu.Machine

	PFlash               []string

    // 默认为 quiet
    // 如果镜像类型为 [hypervisor].image，则追加 systemd.show_status=false
	kernelParamsNonDebug []Param

    // 默认为 debug
	// 如果镜像类型为 [hypervisor].image，则追加 systemd.show_status=true，systemd.log_level=debug
	kernelParamsDebug    []Param

    // 默认为 tsc=reliable,no_timer_check,rcupdate.rcu_expedited=1,i8042.direct=1,i8042.dumbkbd=1,i8042.nopnp=1,i8042.noaux=1 noreplace-smp,reboot=k,cryptomgr.notests,net.ifnames=0,pci=lastbus=0
	// 如果启用 [hypervisor].enable_iommu，则追加 intel_iommu=on,iommu=pt
	// 如果镜像类型为 [hypervisor].image：
	// - 如果 disableNvdimm 为 true，则追加 root=/dev/vda1,rootflags=data=ordered,errors=remount-ro ro,rootfstype=ext4
	// - 如果 disableNvdimm 为 false：
	// -- 如果 dax 为 false，则追加 root=/dev/pmem0p1,rootflags=data=ordered,errors=remount-ro ro,rootfstype=ext4
	// -- 如果 dax 为 true，则追加 root=/dev/pmem0p1,rootflags=dax,data=ordered,errors=remount-ro ro,rootfstype=ext4
	kernelParams         []Param

	Bridges              []types.Bridge

	// [hypervisor].memory_offset，默认为 0
	// 内存偏移量会追加到 hypervisor 最大内存，用于描述 NVDIMM 设备的内存空间大小
	// 如果 [hypervisor].block_device_driver 为 nvdimm，则需要设置 [hypervisor].memory_offset 为 NVDIMM 设备的内存空间大小
	memoryOffset         uint64

	networkIndex         int

    // 默认为 noneProtection，可选的有：
	// - tdxProtection (Intel Trust Domain Extensions)
	// - sevProtection (AMD Secure Encrypted Virtualization)
	// - pefProtection (IBM POWER 9 Protected Execution Facility)
	// - seProtection  (IBM Secure Execution (IBM Z & LinuxONE))
	// 如果启用 [hypervisor].confidential_guest，则进一步判断：如果 host 上 /sys/firmware/tdx_seam/ 文件夹存在或者 CPU flags 中包含 tdx，则为 tdxProtection；如果 host 上 /sys/module/kvm_amd/parameters/sev 文件存在且内容为 1 或者 Y 则为 sevProtection；否则，均为 noneProtection（表示在 host 不支持机密容器场景下，却启用 [hypervisor].confidential_guest，则报错返回）
	protection    guestProtection

	nestedRun     bool

	vhost         bool

	// [hypervisor].disable_image_nvdimm，默认为 false
	// 如果未禁用且支持 nvdimm，则使用 nvdimm 设备加载 guest 镜像，否则使用 virtio-block 设备
	// 在机器容器场景下不支持此特性，如果启用 [hypervisor].confidential_guest，则该参数会被强制设置为 false
	disableNvdimm bool

	// 固定为 true
	dax           bool

	// [hypervisor].use_legacy_serial，默认为 false
	// 是否为 guest console 启用传统的串行终端，否则使用 virtio-console
	legacySerial  bool
}

type qemuAmd64 struct {
	// inherit from qemuArchBase, overwrite methods if needed
	qemuArchBase

	// 是否为 factory 场景，包含两种：BootToBeTemplate 和 BootFromTemplate，两者均为视为 factory 场景
	vmFactory bool

	devLoadersCount uint32

    // 通过上层 annotation 传递 sgx.intel.com/epc，默认为 0
	sgxEPCSize int64
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

*工厂函数根据 [hypervisor.\<type\>] 中的类型，返回对应的 hypervisor 空结构体，后续会在流程中初始化。*

## CreateVM

**创建 VM**

### QEMU

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/qemu.go#L490)

1. 

