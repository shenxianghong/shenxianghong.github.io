---
layout: post
title: "「 Kata Containers 」 4.4.3 源码走读 — virtcontainers device"
date: 2023-03-18
excerpt: "virtcontainers 库中 device 模块源码走读"
tag:
- Cloud Native
- Container Runtime
- Kata Containers
categories:
- Kata Containers
---

<div align=center><img width="300" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> Based on **v3.0.0**

DeviceReceiver 是一组相对而言较底层实现方式，其直接调用 hypervisor，执行设备热插拔等操作；而 Device 是一组设备实现的方式，内部会调用 DeviceReceiver 的接口实现各自的热插拔的能力；对外暴露设备管理入口的则是 DeviceManager，其内部屏蔽了设备的具体类型，而是直接调用 Device 的接口管理设备。

# DeviceReceiver

*<u>src/runtime/pkg/device/api/interface.go</u>*

DeviceReceiver 声明了一组接受设备对象的接口，用于调用 hypervisor 的设备管理接口，执行设备的热插拔等操作。

DeviceReceiver 的实现由 Sandbox 实现。

```go
// Sandbox is composed of a set of containers and a runtime environment.
// A Sandbox can be created, deleted, started, paused, stopped, listed, entered, and restored.
type Sandbox struct {
	ctx        context.Context
	devManager api.DeviceManager
	factory    Factory
	hypervisor Hypervisor
	agent      agent
	store      persistapi.PersistDriver
	fsShare    FilesystemSharer

	swapDevices []*config.BlockDrive
	volumes     []types.Volume

	monitor         *monitor
	config          *SandboxConfig
	annotationsLock *sync.RWMutex
	wg              *sync.WaitGroup
	cw              *consoleWatcher

	sandboxController  resCtrl.ResourceController
	overheadController resCtrl.ResourceController

	containers map[string]*Container

	id string

	network Network

	state types.SandboxState

	sync.Mutex

	swapSizeBytes int64
	shmSize       uint64
	swapDeviceNum uint

	sharePidNs        bool
	seccompSupported  bool
	disableVMShutdown bool
}
```

*工厂函数参考 virtcontainers 部分。*

DeviceReceiver 中声明的 **GetHypervisorType** 为参数获取，无复杂逻辑，不作详述。

## HotplugAddDevice

**热添加设备到 sandbox 中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1789)

1. 调用 sandboxController 的 **AddDevice**，将 device 的 **GetHostPath** 添加到 cgroup 管理中
2. 如果设备类型为 vfio
   1. 调用 device 的 **GetDeviceInfo**，获取 iommu group 中所有设备
   2. 调用 hypervisor 的 **HotplugAddDevice**，热添加所有 vfio 设备<br>*group 是 IOMMU 能够进行 DMA 隔离的最小硬件单元，一个 group 内可能只有一个 device，也可能有多个 device，这取决于物理平台上硬件的 IOMMU 拓扑结构。 设备直通的时候一个 group 里面的设备必须都直通给一个虚拟机。 不能够让一个group 里的多个 device 分别从属于 2 个不同的 VM，也不允许部分 device 在 host 上而另一部分被分配到 guest 里， 因为就这样一个 guest 中的 device 可以利用 DMA 攻击获取另外一个 guest 里的数据，就无法做到物理上的 DMA 隔离。*
3. 如果设备类型为 block 或 vhost-user-blk-pci，直接调用 hypervisor 的 **HotplugAddDevice**，热添加设备
4. 如果设备类型为 generic（即非 vfio、block 或者 vhost-user 设备），则不做操作<br>*根据注释的 TODO，猜测后续版本会有操作，截至 3.0.0 暂无逻辑*
5. 如果为其他设备类型，则不做操作

## HotplugRemoveDevice

**热移除 sandbox 中的设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1843)

1. 如果设备类型为 vfio
   1. 调用 device 的 **GetDeviceInfo**，获取 iommu group 中所有设备
   2. 调用 hypervisor 的 **HotplugRemoveDevice**，热移除所有 vfio 设备
2. 如果设备类型为 block（非 PMEM 设备，因为持久内存设备无法热移除）或 vhost-user-blk-pci
   1. 调用 device 的 **GetDeviceInfo**，获取设备详情
   2. 调用 hypervisor 的 **HotplugRemoveDevice**，热移除设备
3. 如果设备类型为 generic（即非 vfio、block 或者 vhost-user 设备），则不做操作<br>*根据注释的 TODO，猜测后续版本会有操作，截至 3.0.0 暂无逻辑*
4. 如果为其他设备类型，则不做操作
5. 调用 sandboxController 的 **RemoveDevice**，将 device 的 **GetHostPath** 从 cgroup 管理中移除

## GetAndSetSandboxBlockIndex

**获取并设置 virtio-block 索引，仅支持 virtio-blk 和 virtio-scsi 类型设备**

*用于记录分配给 sandbox 中容器的块设备索引（通过 BlockIndexMap（map[int]struct{}））*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1901)

1. 获取维护在 sandbox.state.BlockIndexMap 中，从 0 到 65534 范围内没有被使用的索引 ID

## UnsetSandboxBlockIndex

**释放记录的 virtio-block 索引，仅支持 virtio-blk 和 virtio-scsi 类型设备**

*用于记录分配给 sandbox 中容器的块设备索引（通过 BlockIndexMap（map[int]struct{}））*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1907)

1. 移除维护在 sandbox.state.BlockIndexMap（map[int]struct{}）中的索引

## AppendDevice

**向 sandbox 中添加一个 vhost-user 类型的设备，用于向 hypervisor 传递启动参数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/sandbox.go#L1914)

1. 如果设备类型为 vhost-user-scsi-pci、virtio-net-pci、vhost-user-blk-pci 和 vhost-user-fs-pci
   1. 调用 device 的 **GetDeviceInfo**，获取设备信息
   2. 调用 hypervisor 的 **AddDevice**，添加设备
2. 如果设备类型为 vfio
   1. 调用 device 的 **GetDeviceInfo**，获取 vfio group 中所有设备
   2. 调用 hypervisor 的 **AddDevice**，添加所有 vfio 设备
3. 其余设备类型均不支持

****

# Device

*<u>src/runtime/pkg/device/api/interface.go</u>*

Device 有以下实现方式：GenericDevice、VFIODevice、BlockDevice、VhostUserBlkDevice、VhostUserFSDevice、VhostUserNetDevice 和 VhostUserSCSIDevice，其中均以 GenericDevice 为基础，扩展部分方法。

```go
// GenericDevice refers to a device that is neither a VFIO device, block device or VhostUserDevice.
type GenericDevice struct {
	DeviceInfo *config.DeviceInfo

	ID string

	RefCount    uint
	AttachCount uint
}
```

```go
// VFIODevice is a vfio device meant to be passed to the hypervisor
// to be used by the Virtual Machine.
type VFIODevice struct {
	*GenericDevice
	VfioDevs []*config.VFIODev
}
```

一个 VFIO 设备也就是一组 iommu 设备。

```go
// BlockDevice refers to a block storage device implementation.
type BlockDevice struct {
	*GenericDevice
	BlockDrive *config.BlockDrive
}
```

```go
// VhostUserBlkDevice is a block vhost-user based device
type VhostUserBlkDevice struct {
	*GenericDevice
	VhostUserDeviceAttrs *config.VhostUserDeviceAttrs
}
```

```go
// VhostUserFSDevice is a virtio-fs vhost-user device
type VhostUserFSDevice struct {
	*GenericDevice
	config.VhostUserDeviceAttrs
}
```

```go
// VhostUserNetDevice is a network vhost-user based device
type VhostUserNetDevice struct {
	*GenericDevice
	*config.VhostUserDeviceAttrs
}
```

```go
// VhostUserSCSIDevice is a SCSI vhost-user based device
type VhostUserSCSIDevice struct {
	*GenericDevice
	*config.VhostUserDeviceAttrs
}
```

```go
// VFIODev represents a VFIO drive used for hotplugging
type VFIODev struct {
	// ID is used to identify this drive in the hypervisor options.
	ID string

	// BDF (Bus:Device.Function) of the PCI address
	// 从设备文件名称和路径中解析，例如 04:00.0
	BDF string

	// sysfsdev of VFIO mediated device
	// 常规类型下，例如 /sys/bus/pci/devices/0000:04:00.0
	// mediated 类型下，例如 /sys/devices/pci0000:00/0000:04:00.0/f79944e4-5a3d-11e8-99ce-479cbab002e4（软链接为 /sys/kernel/iommu_groups/0/devices/f79944e4-5a3d-11e8-99ce-479cbab002e4）
	SysfsDev string

	// VendorID specifies vendor id
	VendorID string

	// DeviceID specifies device id
	DeviceID string

	// PCI Class Code
	// /sys/bus/pci/devices/0000:<BDF>/class 文件内容
	Class string

	// Bus of VFIO PCIe device
	// rp0,rp1...，取决于当前所有的 PCIe 设备数量
	Bus string

	// Guest PCI path of device
	GuestPciPath vcTypes.PciPath

	// Type of VFIO device
	// VFIO 设备进一步分为两种类型，可以通过名称区别：一种为常规类型（例如 0000:04:00.0），一种为 mediated 类型（例如 f79944e4-5a3d-11e8-99ce-479cbab002e4）
	Type VFIODeviceType

	// IsPCIe specifies device is PCIe or PCI
	// /sys/bus/pci/devices/0000:<BDF>/config 文件大小如果为 256 则为 PCI 设备，如果为 4096 则为 PCIe 设备
	IsPCIe bool
}
```

VFIODev 为 vfio 设备属性。

```go
// BlockDrive represents a block storage drive which may be used in case the storage
// driver has an underlying block storage device.
type BlockDrive struct {
	// File is the path to the disk-image/device which will be used with this drive
	File string

	// Format of the drive
	Format string

	// ID is used to identify this drive in the hypervisor options.
	ID string

	// MmioAddr is used to identify the slot at which the drive is attached (order?).
	MmioAddr string

	// SCSI Address of the block device, in case the device is attached using SCSI driver
	// SCSI address is in the format SCSI-Id:LUN
	SCSIAddr string

	// NvdimmID is the nvdimm id inside the VM
	NvdimmID string

	// VirtPath at which the device appears inside the VM, outside of the container mount namespace
	VirtPath string

	// DevNo identifies the css bus id for virtio-blk-ccw
	DevNo string

	// PCIPath is the PCI path used to identify the slot at which the drive is attached.
	PCIPath vcTypes.PciPath

	// Index assigned to the drive. In case of virtio-scsi, this is used as SCSI LUN index
	Index int

	// ShareRW enables multiple qemu instances to share the File
	ShareRW bool

	// ReadOnly sets the device file readonly
	ReadOnly bool

	// Pmem enables persistent memory. Use File as backing file
	// for a nvdimm device in the guest
	Pmem bool

	// This block device is for swap
	Swap bool
}
```

BlockDrive 为 block 设备属性。

```go
// VhostUserDeviceAttrs represents data shared by most vhost-user devices
type VhostUserDeviceAttrs struct {
	DevID      string
	// vhost-user-blk 类型下为 device.DeviceInfo.HostPath
	SocketPath string
	// MacAddress is only meaningful for vhost user net device
	MacAddress string

	// These are only meaningful for vhost user fs devices
	Tag string

	Cache string
	// vhost-user-blk 类型下为 vhost-user-blk-pci
	Type DeviceType

	// PCIPath is the PCI path used to identify the slot at which
	// the drive is attached.  It is only meaningful for vhost
	// user block devices
	PCIPath vcTypes.PciPath

	// Block index of the device if assigned
	Index int

	CacheSize uint32
}
```

VhostUserDeviceAttrs 为 VhostUserBlkDevice、VhostUserFSDevice、VhostUserNetDevice 和 VhostUserSCSIDevice 的 vhost-user 设备属性。

```go
// DeviceState is a structure which represents host devices
// plugged to a hypervisor, one Device can be shared among containers in POD
// Refs: pkg/device/drivers/generic.go:GenericDevice
type DeviceState struct {
	// DriverOptions is specific options for each device driver
	// for example, for BlockDevice, we can set DriverOptions["block-driver"]="virtio-blk"
	DriverOptions map[string]string

	// VhostUserDeviceAttrs is specific for vhost-user device driver
	VhostUserDev *VhostUserDeviceAttrs `json:",omitempty"`

	// BlockDrive is specific for block device driver
	BlockDrive *BlockDrive `json:",omitempty"`

	ID string

	// Type is used to specify driver type
	// Refs: pkg/device/config/config.go:DeviceType
	Type string

	// Type of device: c, b, u or p
	// c , u - character(unbuffered)
	// p - FIFO
	// b - block(buffered) special file
	// More info in mknod(1).
	DevType string

	// VFIODev is specific VFIO device driver
	VFIODevs []*VFIODev `json:",omitempty"`

	RefCount    uint
	AttachCount uint

	// Major, minor numbers for device.
	Major int64
	Minor int64

	// ColdPlug specifies whether the device must be cold plugged (true)
	// or hot plugged (false).
	ColdPlug bool
}
```

*工厂函数为简单的赋值操作，具体参考 DeviceManager.NewDevice。*

Device 中声明的 **DeviceID**、**GetAttachCount**、**GetHostPath** 和 **GetMajorMinor** 均为参数获取与赋值，无复杂逻辑，不作详述。<br>此外，**DeviceType** 返回各自 Device 实现的类型（如 generic、vfio、vhost-user-blk-pci、vhost-user-fs-pci、virtio-net-pci 和 vhost-user-scsi-pci）；**GetDeviceInfo** 返回各自 Device 实现的属性信息；**Reference** 和 **Dereference** 用于维护设备的引用计数，未达到最多（^uint(0)，即 2 的 64 次方减一）和最少引用时，则计数加一或减一并返回；**Save** 和 **Load** 用于 Device 和 DeviceState 之间转换，不同的实现额外赋值其各自的属性信息。

## bumpAttachCount

**记录设备的 attach 次数**

*bumpAttachCount 并非 Device 声明的接口，而是 GenericDevice 的一个常用方法，用于判断是否需要执行实际 attach 或 detach 操作，函数入参中的 bool 用于表明是否为 attach 操作，出参中的 bool 用于表明是否为单纯的计数。*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/generic.go#L101)

1. 如果为 attach 操作
   1. 如果当前 attach 计数为 0，则计数加一，并返回 false，即需要执行实际的 attach 操作
   2. 如果当前 attach 计数为 ^uint(0)（即 2 的 64 次方减一），则返回 true 和设备 attach 次数过多的错误
   3. 除此之外，默认计数加一，并返回 true，即不需要执行实际的 attach 操作
2. 如果为 detach 操作
   1. 如果当前 attach 计数为 0，则返回 true 和设备并未 attach 的错误
   2. 如果当前 attach 次数为 1，则计数减一，并返回 false，即需要执行实际的 detach 操作
   3. 除此之外，默认计数减一，并返回 true，即不需要执行实际的 detach 操作

## Attach

**attach 设备**

*根据不同的实现，可能是冷启动添加或者热添加*

### GenericDevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/generic.go#L36)

1. 调用 **bumpAttachCount**，维护 attach 计数，不执行实际操作

### VFIODevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/vfio#L58)

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 遍历 /sys/kernel/iommu_groups/\<device.DeviceInfo.HostPath\>/devices，获取 VFIO 设备的 BDF（PCIe 总线中的每一个功能都有一个唯一的标识符与之对应。这个标识符就是 BDF，即 Bus，Device，Function）、sysfsDev 和设备类型，判断是否为 PCIe 设备，获取 PCI class 等信息，如果为 PCIe 设备，生成 Bus 信息<br>*具体参考 VFIODev 结构体注释*
3. 如果设备必须冷启动，则调用 devReceiver 的 **AppendDevice**，添加设备；否则调用 devReceiver 的 **HotplugAddDevice**，热添加设备

### BlockDevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/block.go#L38)

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 调用 devReceiver 的 **GetAndSetSandboxBlockIndex**，设置并返回可用的索引 ID
3. 根据 device.DeviceInfo.DriverOptions 中 block-driver key，回写对应的字段（SCSIAddr 和 VirtPath）
   1. 如果未指定则视为 virtio-scsi，根据索引 ID 计算出 SCSIAddr，格式为 \<index / 256\>:\<index % 256\><br>*qemu 代码建议 scsi-id 可以取值从 0 到 255（含），而 lun 可以取值从 0 到 16383（含）。 但是超过 255 的 lun 值似乎不遵循一致的 SCSI 寻址。 因此限制为 255*
   2. 如果指定不为 nvdimm，则根据索引 ID 计算出 VirtPath，例如 /dev/vda<br>*其中，索引 0 对应 vda，25 对应 vdz，27 对应 vdab，704 对应 vdaac，18277 对应 vdzzz*
4. 调用 devReceiver 的 **HotplugAddDevice**，热添加设备

### VhostUserBlkDevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/vhost_user_blk.go#L40)

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 根据 device.DeviceInfo.DriverOptions 中 block-driver key，判断 block-driver 是否是 virtio-blk<br>*如果未指定则视为 virtio-scsi；如果指定为 virtio-blk、virtio-blk-ccw 或 virtio-mmio 则视为 virtio-blk*
3. 如果是 virtio-blk，则调用 devReceiver 的 **GetAndSetSandboxBlockIndex**，获取未被使用的块索引；否则，索引默认为 -1
4. 调用 devReceiver 的 **HotplugAddDevice**，热添加设备

### VhostUserFSDevice、VhostUserNetDevice、VhostUserSCSIDevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/vhost_user_fs.go#L25)

*VhostUserFSDevice、VhostUserNetDevice 和 VhostUserSCSIDevice 实现方式一致，以 GenericDevice 为例*

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 调用 devReceiver 的 **AppendDevice**，添加设备

## Detach

**detach 设备**

*不同的实现下未必支持 detach 操作*

### GenericDevice、VhostUserFSDevice、VhostUserNetDevice、VhostUserSCSIDevice

*GenericDevice、VhostUserFSDevice、VhostUserNetDevice 和 VhostUserSCSIDevice 实现方式一致，以 GenericDevice 为例*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/generic.go#L42)

1. 调用 **bumpAttachCount**，维护 attach 计数，不执行实际操作

### VFIODevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/vfio#L128)

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 如果设备是冷启动的，说明没有运行后的 attach 动作，因此则无需 detach；否则，调用 devReceiver 的 **HotplugRemoveDevice**，热移除设备

### BlockDevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/block.go#L124)

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 调用 devReceiver 的 **HotplugRemoveDevice**，热移除设备

### VhostUserBlkDevice

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/drivers/vhost_user_blk.go#L118)

1. 调用 **bumpAttachCount**，维护 attach 计数，判断是否执行后续实际操作
2. 调用 devReceiver 的 **HotplugRemoveDevice**，热移除设备
3. 根据 device.DeviceInfo.DriverOptions 中 block-driver key，判断 block-driver 是否是 virtio-blk。如果是 virtio-blk，则调用 devReceiver 的 **UnsetSandboxBlockIndex**，释放记录的 virtio-block 索引<br>*如果未指定则视为 virtio-scsi；如果指定为 virtio-blk、virtio-blk-ccw 或 virtio-mmio 则视为 virtio-blk*

****

# DeviceManager

*<u>src/runtime/pkg/device/api/interface.go</u>*

DeviceManager 用于创建新设备，可用作单个设备管理对象。

```go
type deviceManager struct {
	devices map[string]api.Device
	
	// [hypervisor].block_device_driver
	blockDriver        string
	// [hypervisor].vhost_user_store_path
	// Its sub-path "block" is used for block devices; "block/sockets" is
	// where we expect vhost-user sockets to live; "block/devices" is where
	// simulated block device nodes for vhost-user devices to live.
	// 默认为：/var/run/kata-containers/vhost-user
	vhostUserStorePath string

	sync.RWMutex
	
	// [hypervisor].enable_vhost_user_store
	// Enabling this will result in some Linux reserved block type
	// major range 240-254 being chosen to represent vhost-user devices.
	vhostUserStoreEnabled bool
}
```

*工厂函数是简单的根据配置文件中 hypervisor 配置项参数的初始化。*

Device 中声明的 **IsDeviceAttached**、**GetDeviceByID** 和 **GetAllDevices** 为参数获取，无复杂逻辑，不作详述。<br>

```go
// DeviceInfo is an embedded type that contains device data common to all types of devices.
type DeviceInfo struct {
	// DriverOptions is specific options for each device driver
	// for example, for BlockDevice, we can set DriverOptions["block-driver"]="virtio-blk"
	DriverOptions map[string]string

	// Hostpath is device path on host
	HostPath string

	// ContainerPath is device path inside container
	ContainerPath string `json:"-"`

	// Type of device: c, b, u or p
	// c , u - character(unbuffered)
	// p - FIFO
	// b - block(buffered) special file
	// More info in mknod(1).
	DevType string

	// ID for the device that is passed to the hypervisor.
	ID string

	// Major, minor numbers for device.
	Major int64
	Minor int64

	// FileMode permission bits for the device.
	FileMode os.FileMode

	// id of the device owner.
	UID uint32

	// id of the device group.
	GID uint32

	// Pmem enabled persistent memory. Use HostPath as backing file
	// for a nvdimm device in the guest.
	Pmem bool

	// If applicable, should this device be considered RO
	ReadOnly bool

	// ColdPlug specifies whether the device must be cold plugged (true)
	// or hot plugged (false).
	ColdPlug bool
}
```

## NewDevice

**初始化设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/manager/manager.go#L136)

1. 如果设备不是 pmem 类型（即 devInfo.Pmem 为 false）
   1. 如果启用了 [hypervisor].enable_vhost_user_store、devInfo.DevType 为 b 并且设备 devInfo.Major 是 242（即 vhost-user-scsi）或者 241（即 vhost-user-blk），则获取 \<vhostUserStorePath\>/block/devices 目录下，格式为 major:minor 的文件名，作为 socket 文件，返回 \<vhostUserStorePath\>/block/sockets/\<socket\> 文件路径<br>*用于获取 vhost-user 设备的主机路径。 对于 vhost-user 块设备，如 vhost-user-blk 或 vhost-user-scsi，其 socket 应位于目录 \<vhostUserStorePath>/block/sockets/ 下，它对应的设备节点应该在目录 \<vhostUserStorePath>/block/devices/ 下*
   2. 如果 devInfo.DevType 为 c 或者 u，则 uevent 路径为 /sys/dev/char/\<major:minor\>/uevent；如果 devInfo.DevType 为 b，则 uevent 路径为  /sys/dev/block/\<major:minor\>/uevent。如果 uevent 文件不存在，则返回 devInfo.ContainerPath，否则读取文件内容（文件为 ini 格式），解析 DEVNAME 项，返回 /dev/\<DEVNAME \> 文件路径<br>*某些设备（例如 /dev/fuse、/dev/cuse）并不总是在 /sys/dev 下实现 sysfs 接口，这些设备默认由 docker 传递。 只需返回在设备配置中传递的路径，这确实意味着这些设备不支持设备重命名*
   3. 设置 devInfo.HostPath 为上述返回的路径
2. 根据 devInfo.Major 和 devInfo.Minor，判断设备是否已经存在 deviceManager 的 devices 中，存在则直接返回即可
3. 为了避免 deviceID 冲突，重新生成 devInfo.ID
4. 根据设备类别，初始化对应的设备
   1. 如果 devInfo.HostPath 为 /dev/vfio/xxx（排除 /dev/vfio/vfio 字符设备），则视为 vfio 设备类型
   2. 如果 devInfo.DevType 为 b，并且 devInfo.Major 为 241，则视为 vhost-user-blk 设备类型
   3. 如果 devInfo.DevType 为 b，则视为 block 设备类型（也就是 devInfo.Major 不为 241）
   4. 除此之外，均视为 generic 设备类型（也就是 vhost-user-fs、vhost-user-net 和 vhost-user-scsi 设备均为此类型）
5. 调用 device 的 **Reference**，维护设备的引用计数
6. 维护 deviceManager 中的设备信息，其中 key 为调用 device 的 **DeviceID** 获得，后续用于判断设备是否已经创建

## RemoveDevice

**移除维护的设备信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/manager/manager.go#L147)

1. 校验设备是否已经创建
2. 调用 device 的 **Dereference**，移除引用
3. 如果移除后引用为 0，则并调用 device 的 **GetAttachCount**，校验当前设备 attach 次数是否为 0，移除维护在 deviceManager 的设备信息

## AttachDevice

**attach 设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/manager/manager.go#L181)

1. 校验设备是否已经创建
2. 调用 device 的 **Attach**，attach 设备

## DetachDevice

**detach 设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/manager/manager.go#L196)

1. 校验设备是否已经创建
2. 调用 device 的 **GetAttachCount**，校验当前设备 attach 次数是否不为 0
3. 调用 device 的 **Detach**，detach 设备

## LoadDevices

**加载设备信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/device/manager/manager.go#L244)

1. 遍历入参 []config.DeviceState 中每一个设备信息，根据其类型初始化对应的 device 对象
2. 调用 device 的 **Load**，加载设备
3. 维护 deviceManager 中的设备信息，其中 key 为调用 device 的 **DeviceID** 获得，后续用于判断设备是否已经创建
