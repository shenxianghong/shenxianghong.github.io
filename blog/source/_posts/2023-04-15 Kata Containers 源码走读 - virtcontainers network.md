---
title: "「 Kata Containers 」源码走读 — virtcontainers/network"
excerpt: "virtcontainers 中与 Endpoint、Network 等网络管理相关的流程梳理"
cover: https://picsum.photos/0?sig=20230415
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-04-15
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

# Endpoint

*<u>src/runtime/virtcontainers/endpoint.go</u>*

Endpoint 代表了某一个物理或虚拟网络设备的基础结构，具体包括：veth、ipvlan、macvlan、macvtap、physical、vhostuser、tap 和 tuntap 8 种实现方式。借助 `github.com/vishvananda/netlink` 将抽象 endpoint 类型转变成具体的 netlink 类型，配置后回写到 endpoint 的具体属性（例如 netPair 等）后，交由 hypervisor 创建或配置该设备信息。

```go
// VethEndpoint gathers a network pair and its properties.
type VethEndpoint struct {
	// 固定为 virtual
	EndpointType EndpointType

	// idx 为 VM 中 endpoint 设备的递增序号
	// NetPair.TapInterface.Name 为逻辑网桥名称，固定为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap 设备名称，固定为 tap<idx>_kata
	// NetPair.VirtIface.Name 为 endpoint 设备名称，默认为 eth<idx>
	// NetPair.VirtIface.HardAddr 为随机生成的 MAC 地址
	// NetPair.NetInterworkingModel 为 [runtime].internetworking_model，可选有 macvtap 和 tcfilter（默认）
	NetPair NetworkInterfacePair
	
	PCIPath vcTypes.PciPath

	// endpoint 设备属性信息
	EndpointProperties NetworkInfo

	// endpoint 设备 inbound/outbound 限速标识
	RxRateLimiter bool
	TxRateLimiter bool
}
```

```go
// IPVlanEndpoint represents a ipvlan endpoint that is bridged to the VM
type IPVlanEndpoint struct {
	// 固定为 ipvlan
	EndpointType EndpointType

	// idx 为 VM 中 endpoint 设备的递增序号
	// NetPair.TapInterface.Name 为逻辑网桥名称，固定为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap 设备名称，固定为 tap<idx>_kata
	// NetPair.VirtIface.Name 为 endpoint 设备名称，默认为 eth<idx>
	// NetPair.VirtIface.HardAddr 为随机生成的 MAC 地址
	// NetPair.NetInterworkingModel 为 tcfilter
	NetPair NetworkInterfacePair
	
	PCIPath vcTypes.PciPath
	
	// endpoint 设备属性信息
	EndpointProperties NetworkInfo

	// endpoint 设备 inbound/outbound 限速标识
	RxRateLimiter bool
	TxRateLimiter bool
}
```

```go
// MacvlanEndpoint represents a macvlan endpoint that is bridged to the VM
type MacvlanEndpoint struct {
	// 固定为 macvlan
	EndpointType EndpointType

	// idx 为 VM 中 endpoint 设备的递增序号
	// NetPair.TapInterface.Name 为逻辑网桥名称，固定为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap 设备名称，固定为 tap<idx>_kata
	// NetPair.VirtIface.Name 为 endpoint 设备名称，默认为 eth<idx>
	// NetPair.VirtIface.HardAddr 为随机生成的 MAC 地址
	// NetPair.NetInterworkingModel 为 [runtime].internetworking_model，可选有 macvtap 和 tcfilter（默认）
	NetPair NetworkInterfacePair

	PCIPath vcTypes.PciPath

	// endpoint 设备属性信息
	EndpointProperties NetworkInfo

	// endpoint 设备 inbound/outbound 限速标识
	RxRateLimiter bool
	TxRateLimiter bool
}
```

```go
// MacvtapEndpoint represents a macvtap endpoint
type MacvtapEndpoint struct {
	// 固定为 macvtap
	EndpointType EndpointType

	// 元素数量等于 [hypervisor].default_vcpus 的 /dev/tap<EndpointProperties.Iface.Index> 文件句柄
	VMFds    []*os.File
	// 元素数量等于 [hypervisor].default_vcpus 的 /dev/vhost-net 文件句柄
	VhostFds []*os.File

	PCIPath vcTypes.PciPath

	// endpoint 设备属性信息
    EndpointProperties NetworkInfo

	// endpoint 设备 inbound/outbound 限速标识
	RxRateLimiter bool
	TxRateLimiter bool
}
```

```go
// PhysicalEndpoint gathers a physical network interface and its properties
type PhysicalEndpoint struct {
	// 固定为 physical
	EndpointType EndpointType
    
	// 根据 IfaceName 解析获得，类比于 ethtool -i <IfaceName> 结果中的 bus-info
	BDF string

	// 软链接 /sys/bus/pci/devices/<BDF>/driver 指向实体文件路径的基础
	Driver string

	// 由 /sys/bus/pci/devices/<BDF>/vendor 和 /sys/bus/pci/devices/<BDF>/device 文件内容拼接而成
	VendorDeviceID string

	PCIPath vcTypes.PciPath

	// endpoint 设备属性信息
	IfaceName          string
	HardAddr           string
    EndpointProperties NetworkInfo
}
```

```go
// VhostUserEndpoint represents a vhost-user socket based network interface
type VhostUserEndpoint struct {
	// 固定为 vhost-user
	EndpointType EndpointType

	// Path to the vhost-user socket on the host system
	// 根据 endpoint 设备的所有 IP，获得一个存在的 /tmp/vhostuser_<IP>/vhu.sock 路径
	SocketPath string
	
	PCIPath vcTypes.PciPath

	// endpoint 设备属性信息
	HardAddr           string
	IfaceName          string
	EndpointProperties NetworkInfo
}
```

```go
// TapEndpoint represents just a tap endpoint
type TapEndpoint struct {
	// 固定为 tap
	EndpointType EndpointType
    
	// TapInterface.Name 为 endpoint 设备名称，默认为 eth<idx>
	// TapInterface.TAPIface.Name 为 tap 设备名称，固定为 tap<idx>_kata
	TapInterface TapInterface
	
	PCIPath vcTypes.PciPath
    
	// endpoint 设备属性信息
	EndpointProperties NetworkInfo
    
	// endpoint 设备 inbound/outbound 限速标识
	RxRateLimiter bool
	TxRateLimiter bool
}
```

```go
// TuntapEndpoint represents just a tap endpoint
type TuntapEndpoint struct {
	// 固定为 tuntap
	EndpointType EndpointType
	
	// idx 为 VM 中设备的递增序号
	// TuntapInterface.Name 为 endpoint 设备名称，默认为 eth<idx>
	// TuntapInterface.TAPIface.Name 为 tap 设备名称，固定为 tap<idx>_kata
	// TuntapInterface.TAPIface.HardAddr 为 tap 设备 MAC 地址
	TuntapInterface TuntapInterface

	// idx 为 VM 中设备的递增序号
	// NetPair.TapInterface.Name 为逻辑网桥名称，固定为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap 设备名称，固定为 tap<idx>_kata
	// NetPair.VirtIface.Name 为 endpoint 设备名称，默认为 eth<idx>
	// NetPair.VirtIface.HardAddr 为随机生成的 MAC 地址
	// NetPair.NetInterworkingModel 为 [runtime].internetworking_model，可选有 macvtap 和 tcfilter（默认）
	NetPair NetworkInterfacePair

    PCIPath vcTypes.PciPath

	// endpoint 设备属性信息
	EndpointProperties NetworkInfo
    
	// endpoint 设备 inbound/outbound 限速标识
	RxRateLimiter bool
	TxRateLimiter bool
}
```

```go
// NetworkInterfacePair defines a pair between VM and virtual network interfaces.
type NetworkInterfacePair struct {
	// 取决于具体 endpoint 实现，内容有所不同
	TapInterface
	VirtIface NetworkInterface

	// [runtime].internetworking_model，可选的有 macvtap 和 tcfilter（默认）
	NetInterworkingModel
}
```

NetworkInterfacePair 即 netpair（例如 br0_kata），描述了 tap 设备（TapInterface）和 veth 设备（VirtIface，即位于容器命名空间内部的 veth-pair 设备，如 eth0）的数据结构（netPair 并非真实设备，而是一个用于描述如何连通容器网络和 VM 网络的逻辑网桥）。

```go
// NetworkInfo gathers all information related to a network interface.
// It can be used to store the description of the underlying network.
type NetworkInfo struct {
	Iface     NetlinkIface
	DNS       DNSInfo
	Link      netlink.Link
	Addrs     []netlink.Addr
	Routes    []netlink.Route
	Neighbors []netlink.Neigh
}
```

NetworkInfo 描述 endpoint 设备的通用属性信息，通过相关 Golang 系统调用库获得。

Endpoint 中声明的 **Properties**、**Type**、**PciPath**、**SetProperties**、**SetPciPath**、**GetRxRateLimiter**、**SetRxRateLimiter**、**GetTxRateLimiter** 和 **GetTxRateLimiter** 均为参数获取与赋值，无复杂逻辑，不作详述。<br>其中，**Name**、**HardwareAddr** 和 **NetworkPair** 视不同的 endpoint 实现，取值字段有所不同，具体为：

| Endpoint          | Name                          | HardwareAddr                          | NetworkPair |
| ----------------- | ----------------------------- | ------------------------------------- | ----------- |
| VethEndpoint      | NetPair.VirtIface.Name        | NetPair.TAPIface.HardAddr             | NetPair     |
| IPVlanEndpoint    | NetPair.VirtIface.Name        | NetPair.TAPIface.HardAddr             | NetPair     |
| MacvlanEndpoint   | NetPair.VirtIface.Name        | NetPair.TAPIface.HardAddr             | NetPair     |
| MacvtapEndpoint   | EndpointProperties.Iface.Name | EndpointProperties.Iface.HardwareAddr | ---         |
| PhysicalEndpoint  | IfaceName                     | HardAddr                              | ---         |
| VhostUserEndpoint | IfaceName                     | HardAddr                              | ---         |
| TapEndpoint       | TapInterface.Name             | TapInterface.TAPIface.HardAddr        | ---         |
| TuntapEndpoint    | TuntapInterface.Name          | TapInterface.TAPIface.HardAddr        | NetPair     |

## Attach

**添加 endpoint 设备到 VM 中**

### VethEndpoint、IPVlanEndpoint、MacvlanEndpoint、TuntapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L99)

1. 调用 network 的 **xConnectVMNetwork**，配置网络信息
2. 调用 hypervisor 的 **AddDevice**，以 NetDev 类型添加 endpoint 设备到 VM 中

### MacvtapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/macvtap_endpoint.go#L68)

1. 创建 /dev/tap\<endpoint.EndpointProperties.Iface.Index\>，构建 fds（[]*os.File，元素为数量等于 [hypervisor].default_vcpus 的 /dev/tap\<endpoint.EndpointProperties.Iface.Index\> 文件句柄），回写到 endpoint.VMFds 中
2. 如果 [hypervisor].disable_vhost_net 未开启，则创建 /dev/vhost-net，构建 fds（[]*os.File，元素为数量等于 [hypervisor].default_vcpus 的 /dev/vhost-net 文件句柄），回写到 endpoint.VhostFds 中
3. 调用 hypervisor 的 **AddDevice**，以 NetDev 类型添加 endpoint 设备到 VM 中

### PhysicalEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/physical_endpoint.go#L82)

1. 将 endpoint.BDF 写入 /sys/bus/pci/devices/\<endpoint.BDF\>/driver/unbind 文件中<br>*用于解除该设备在 host driver 上的绑定*
2. 将 endpoint.VendorDeviceID 写入 /sys/bus/pci/drivers/vfio-pci/new_id 文件中；并将 endpoint.BDF 写入 /sys/bus/pci/drivers/vfio-pci/bind 文件中<br>*用于将该设备绑定到 vfio-pci driver 上，后续以 vfio-passthrough 传递给 hypervisor*
3. 获取 /sys/bus/pci/devices/\<endpoint.BDF\>/iommu_group 软链接的指向路径，得到其 base 路径（即路径最后一个元素），构建 vfio 设备路径，即 /dev/vfio/\<base\> 
4. 根据 vfio 设备路径，获取设备信息，构建 DeviceInfo，并调用 devManager 的 **NewDevice**，初始化 vfio 类型设备
5. 调用 devManager 的 **AttachDevice**，冷添加此设备到 VM 中

### VhostUserEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/vhostuser_endpoint.go#L84)

1. 调用 hypervisor 的 **AddDevice**，以 VhostuserDev 类型添加 virtio-net-pci 设备（socketPath、MacAddress 等信息从 endpoint 中赋值）到 VM 中

### TapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/tap_endpoint.go#L76)

1. 暂不支持添加此类设备，返回错误

## Detach

**移除 VM 中的 endpoint 设备**

### VethEndpoint、IPVlanEndpoint、MacvlanEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L114)

1. 如果 netns 不是由 Kata Containers 创建的，则直接跳过后续<br>*根据创建 pod_sandbox 或者 single_container 时，spec.Linux.Namespace 中的 network 是否指定判断，如果未指定，表示需要由 Kata Containers 创建，反之表示 netns 已经提前创建好*
1. 进入到该 netns 中，调用 network 的 **xDisconnectVMNetwork**，移除网络信息

### MacvtapEndpoint、VhostUserEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/macvtap_endpoint.go#L92)

1. 无任何操作，直接返回

### PhysicalEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/physical_endpoint.go#L112)

1. 将 endpoint.BDF 写入 /sys/bus/pci/devices/\<endpoint.BDF\>/driver/unbind 文件中<br>*用于解除该设备在 vfio-pci driver 上的绑定*
2. 将 endpoint.VendorDeviceID 写入 /sys/bus/pci/drivers/vfio-pci/remove_id 文件中；并将 endpoint.BDF 写入 /sys/bus/pci/drivers/\<endpoint.Driver\>/bind 文件中<br>*用于将该设备绑定到 host driver 上*

### TapEndpoint、TuntapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/tap_endpoint.go#L81)

1. 如果 netns 不是由 Kata Containers 创建的，并且 netns 路径存在，则直接跳过后续
2. 进入到该 netns 中，获取名为 tap0_kata（示例名称，其中 0 为递增生成的索引）的设备，关停并移除

## HotAttach

**热添加 endpoint 设备到 VM 中**

### VethEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L130)

1. 调用 Network 的 **xConnectVMNetwork**，配置网络信息
2. 调用 hypervisor 的 **HotplugAddDevice**，以 NetDev 类型热添加 endpoint 设备到 VM 中

### IPVlanEndpoint、MacvlanEndpoint、MacvtapEndpoint、PhysicalEndpoint、VhostUserEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/ipvlan_endpoint.go#L130)

1. 暂不支持热添加此类设备，返回错误

### TapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/tap_endpoint.go#L96)

1. 创建名为 tap0_kata（示例名称，其中 0 为递增生成的索引）的 tuntap 设备（mode 为 tap；队列长度取自 [hypervisor].default_vcpus 最大为 1，即如果队列长度大于 1，为了避免不支持多队列，需要重置为 1，参考 [tuntap 实现](https://github.com/kata-containers/kata-containers/blob/e6e5d2593ac319329269d7b58c30f99ba7b2bf5a/src/runtime/vendor/github.com/vishvananda/netlink/link_linux.go#L1164-L1316)），并返回空的 fds，回写到 endpoint.TapInterface.VMFds 中
2. 如果 [hypervisor].disable_vhost_net 未开启，则创建 /dev/vhost-net，构建 fds（[]*os.File，元素为队列长度数量的 /dev/vhost-net 文件句柄），回写到 endpoint.TapInterface.VhostFds 中
3. 设置 endpoint.TapInterface.TAPIface.HardAddr 为 veth 设备的 MAC 地址<br>*将 veth MAC 地址保存到 tap 中，以便稍后用于构建 hypervisor 命令行。 此 MAC 地址必须是 VM 内部的地址，以避免任何防火墙问题。 host 上的网络插件预期流量源自这个 MAC 地址*
4. 设置 tuntap 设备的 mtu 值为 veth 设备的 mtu 值
5. 启用 tuntap 设备
6. 调用 hypervisor 的 **HotplugAddDevice**，以 NetDev 类型热添加 endpoint 设备到 VM 中

### TuntapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/tuntap_endpoint.go#L107)

1. 创建名为 tap0_kata（示例名称，其中 0 为递增生成的索引）的 tuntap 设备（mode 为 tap；队列长度取自 [hypervisor].default_vcpus 最大为 1，即如果队列长度大于 1，为了避免不支持多队列，需要重置为 1，参考 [tuntap 实现](https://github.com/kata-containers/kata-containers/blob/e6e5d2593ac319329269d7b58c30f99ba7b2bf5a/src/runtime/vendor/github.com/vishvananda/netlink/link_linux.go#L1164-L1316)）
2. 设置 endpoint.TuntapInterface.TAPIface.HardAddr 为 veth 设备的 MAC 地址<br>*将 veth MAC 地址保存到 tap 中，以便稍后用于构建 hypervisor 命令行。 此 MAC 地址必须是 VM 内部的地址，以避免任何防火墙问题。 host 上的网络插件预期流量源自这个 MAC 地址*
3. 设置 tuntap 设备的 mtu 值为 veth 设备的 mtu 值
4. 启用 tuntap 设备
5. 调用 hypervisor 的 **HotplugAddDevice**，以 NetDev 类型热添加 endpoint 设备到 VM 中

## HotDetach

**热移除 VM 中的 endpoin 设备**

### VethEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L147)

1. 如果 netns 不是由 Kata Containers 创建的，则直接跳过后续<br>*根据创建 pod_sandbox 或者 single_container 时，spec.Linux.Namespace 中的 network 是否指定判断，如果未指定，表示需要由 Kata Containers 创建，反之表示 netns 已经提前创建好*
2. 进入到该 netns 中，调用 **xDisconnectVMNetwork**，移除网络信息
3. 调用 hypervisor 的 **HotplugRemoveDevice**，以 NetDev 热移除 endpoint 中 VM 的设备

### IPVlanEndpoint、MacvlanEndpoint、MacvtapEndpoint、PhysicalEndpoint、VhostUserEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/ipvlan_endpoint.go#L135)

1. 暂不支持热移除此类设备，返回错误

### TapEndpoint、TuntapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/tap_endpoint.go#L115)

1. 进入到该 netns 中，获取名为 tap0_kata（示例名称，其中 0 为递增生成的索引）的设备，关停并移除
2. 调用 hypervisor 的 **HotplugRemoveDevice**，以 NetDev 热移除 VM 中的 endpoint 设备

***

# Network

*<u>src/runtime/virtcontainers/network.go</u>*

实际操作均借助 `github.com/vishvananda` 实现，该库提供了等价于 ip addr、ip link、tc qdisc、tc filter 等命令行的功能。

```go
// LinuxNetwork represents a sandbox networking setup.
type LinuxNetwork struct {
	// OCI spec 中类型为 network 的 linux.namespace.path
	netNSPath string

	// netns 中的 endpoint 设备
	eps []Endpoint

	// [runtime].internetworking_model，可选有 macvtap 和 tcfilter（默认）
	interworkingModel NetInterworkingModel

	// 表示当前 netns 是否为 Kata Containers 创建
	// - false：netns 为事先准备好，创建 Kata 容器时，在 OCI spec 中传递该 netns（network 类型的 linux.namespace）。例如 Kubernetes 场景下，netns 由 CNI 创建
	// - true：Kata Containers 发现 OCI spec 中不存在 network 类型的 linux.namespace，则会手动创建一个 netns（以 cnitest 开头）。例如 Containerd 场景下，运行 single_container
	netNSCreated bool
}
```

Network 中声明的 **NetworkID**、**NetworkCreated**、**Endpoints** 和 **SetEndpoints** 均为参数获取与赋值，无复杂逻辑，不作详述。其中，**Run** 是封装了进入 netns 中执行回调函数的流程。

## xConnectVMNetwork

**根据不同的网络模型，打通容器和 VM 之间的网络**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L518)

1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象的网络模型（即[runtime].internetworking_model，默认为 tcfilter）
2. 调用 hypervisor 的 **Capabilities**，判断 hypervisor 是否支持多队列特性。如果支持，则队列数设为 [hypervisor].default_vcpus；否则为 0
3. 根据网络模型，创建对应的 tap 设备，连通容器和 VM 之间的网络

   *无论哪种网络模式，VM 中的 eth0 都是 hypervisor 基于 tap 设备虚拟化出来，并 attach 到 VM 中建立两者的关联关系。区别在于 tap 设备和 veth 设备（即 CNI 为容器内分配的 eth0）的网络打通方式*

   - 如果网络模型为 macvtap

     1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，并进一步获取 veth 设备
     2. 创建 macvtap 设备，其中名称为 tap0_kata（示例名称，其中 0 为递增生成的索引）、txQLen 属性继承自 veth 设备且 parentIndex 指向容器 eth0 设备（下称 veth 设备）<br>*目前 macvtap 场景下需要特殊处理索引（该索引后续用作命名 /dev/tap\<idx\>），是由于 Linux 内核中存在一个限制，会导致 macvtap/macvlan link 在网络 namespace 中创建时无法获得正确的 link 索引
        https://github.com/clearcontainers/runtime/issues/708
        在修复该错误之前，需要随机一个非冲突索引（即 8192 + 随机一个数字）并尝试创建一个 link。 如果失败，则继续重试，上限为 128 次。所有内核都不会检查链接 ID 是否与主机上的 link ID 冲突，因此需要偏移 link ID 以防止与主机索引发生任何重叠，内核将确保没有竞争条件*
     3. 设置 netPair.TAPIface.HardAddr 为 veth 设备的 MAC 地址<br>*将 veth MAC 地址保存到 tap 中，以便稍后用于构建 hypervisor 命令行。 此 MAC 地址必须是 VM 内部的地址，以避免任何防火墙问题。 host 上的网络插件预期流量源自这个 MAC 地址*
     4. 设置 macvtap 设备的 mtu 值为 veth 设备的 mtu 值
     5. 设置 veth 设备的 MAC 地址为随机生成的 MAC 地址（即 netPair.VirtIface.HardAddr，该字段初始化时为随机生成的 MAC  地址），并设置 macvtap 设备的 MAC 地址为 veth 设备的 MAC 地址
     6. 启用 macvtap 设备
     7. 获取 veth 设备的全部 IP 地址，保存至 netPair.VirtIface.Addrs，并从 veth 设备中移除这些 IP 地址<br>*清理掉 veth 设备中由 CNI 分配的 IP 地址，避免 ARP 冲突*
     8. 根据步骤 2 中生成随机索引，创建 /dev/tap\<idx\>，构建 fds（[]*os.File，元素为队列长度数量的 /dev/tap\<idx\> 文件句柄），回写到 netPair.VMFds 中
     9. 如果 [hypervisor].disable_vhost_net 未开启，则创建 /dev/vhost-net，构建 fds（[]*os.File，元素为队列长度数量的 /dev/vhost-net 文件句柄），回写到 netPair.VhostFds 中
     
     综上所述，macvtap 网络模式下，是将 veth 设备和 macvtap 设备的 mac 地址等信息互换，并将 veth 设备的网络信息转移到 VM 中 eth0 设备（实质上是清理 veth 设备网络信息，同时借助 VM dhcp 获取 CNI 分配的 IP 地址），结合 macvtap 设备的 parentIndex 指向 veth 设备，实现容器网络流量和 VM 网络流量的互通。
     
   - 如果网络模型为 tcfilter

     1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，并进一步获取 veth 设备
     2. 创建名为 tap0_kata（示例名称，其中 0 为递增生成的索引）的 tuntap 设备（mode 为 tap；队列长度最大为 1，即如果队列长度大于 1，为了避免不支持多队列，需要重置为 1，参考 [tuntap 实现](https://github.com/kata-containers/kata-containers/blob/e6e5d2593ac319329269d7b58c30f99ba7b2bf5a/src/runtime/vendor/github.com/vishvananda/netlink/link_linux.go#L1164-L1316)），并返回空的 fds，回写到 netPair.VMFds 中
     3. 如果 [hypervisor].disable_vhost_net 未开启，则创建 /dev/vhost-net，构建 fds（[]*os.File，元素为队列长度数量的 /dev/vhost-net 文件句柄），回写到 netPair.VhostFds 中
     4. 设置 netPair.TAPIface.HardAddr 为 veth 设备的 MAC 地址<br>*将 veth MAC 地址保存到 tap 中，以便稍后用于构建 hypervisor 命令行。 此 MAC 地址必须是 VM 内部的地址，以避免任何防火墙问题。 host 上的网络插件预期流量源自这个 MAC 地址*
     5. 设置 tuntap 设备的 mtu 值为 veth 设备的 mtu 值
     6. 启用 tuntap 设备
     7. 为 tuntap 设备和 veth 设备分别创建 ingress 类型的网络队列规则与 tc 规则，将一方的入站流量重定向到另一方进行出站处理，使得所有流量在两者之间可以被重定向
     
     综上所述，tcfilter 网络模式下，仅仅是在 veth 和 tap 设备之间配置 tc 规则，实现容器网络流量和 VM 网络流量的互通。
   
   ***效果示例***
   
   ```shell
   # 网络模型为 macvtap 时
   $ ip netns exec cni-97333755-9052-db96-37fe-37d4e39bf046 ethtool -i tap0_kata
   driver: macvlan
   version: 0.1
   firmware-version: 
   expansion-rom-version: 
   bus-info: 
   supports-statistics: no
   supports-test: no
   supports-eeprom-access: no
   supports-register-dump: no
   supports-priv-flags: no
   $ ip netns exec cni-fb0bd424-5621-3672-62d9-9233708dc54d ip a
   1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
       link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
       inet 127.0.0.1/8 scope host lo
          valid_lft forever preferred_lft forever
       inet6 ::1/128 scope host 
          valid_lft forever preferred_lft forever
   2: tunl0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN group default qlen 1000
       link/ipip 0.0.0.0 brd 0.0.0.0
   4: eth0@if18: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc noqueue state UP group default 
       link/ether 46:ba:a7:d6:85:ec brd ff:ff:ff:ff:ff:ff link-netnsid 0
   55446: tap0_kata@eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc fq_codel state UP group default qlen 1500
       link/ether c6:f1:06:ac:46:53 brd ff:ff:ff:ff:ff:ff
       inet6 fe80::c4f1:6ff:feac:4653/64 scope link 
          valid_lft forever preferred_lft forever
   $ kata-runtime exec 7af17cb96ddaa59a4e370c0de584ea6df5759278ce6c203a188a3ab18b461216 
   root@localhost:/# ip a
   1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
       link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
       inet 127.0.0.1/8 scope host lo
          valid_lft forever preferred_lft forever
       inet6 ::1/128 scope host 
          valid_lft forever preferred_lft forever
   2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc fq_codel state UP group default qlen 1000
       link/ether c6:f1:06:ac:46:53 brd ff:ff:ff:ff:ff:ff
       inet 10.244.69.173/32 brd 10.244.69.173 scope global eth0
          valid_lft forever preferred_lft forever
       inet6 fe80::c4f1:6ff:feac:4653/64 scope link 
          valid_lft forever preferred_lft forever
   
   # 网络模型为 tcfilter 时
   $ ip netns exec cni-d7e932c4-51a6-53e0-e73c-662aa84b4653 ethtool -i tap0_kata
   driver: tun
   version: 1.6
   firmware-version: 
   expansion-rom-version: 
   bus-info: tap
   supports-statistics: no
   supports-test: no
   supports-eeprom-access: no
   supports-register-dump: no
   supports-priv-flags: no
   $ ip netns exec cni-d7e932c4-51a6-53e0-e73c-662aa84b4653 ip a
   1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
       link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
       inet 127.0.0.1/8 scope host lo
          valid_lft forever preferred_lft forever
       inet6 ::1/128 scope host 
          valid_lft forever preferred_lft forever
   2: tunl0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN group default qlen 1000
       link/ipip 0.0.0.0 brd 0.0.0.0
   4: eth0@if17: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc noqueue state UP group default qlen 1000
       link/ether 1e:03:66:df:ad:5e brd ff:ff:ff:ff:ff:ff link-netnsid 0
       inet 10.244.69.163/32 scope global eth0
          valid_lft forever preferred_lft forever
       inet6 fe80::1c03:66ff:fedf:ad5e/64 scope link 
          valid_lft forever preferred_lft forever
   5: tap0_kata: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc mq state UNKNOWN group default qlen 1000
       link/ether ee:b0:99:52:54:ef brd ff:ff:ff:ff:ff:ff
       inet6 fe80::ecb0:99ff:fe52:54ef/64 scope link 
          valid_lft forever preferred_lft forever
   $ kata-runtime exec 8a390592512f2f27a35accd0fa5c2c82d29dea2f3d1eb982c6225be7856e78a6 
   root@localhost:/# ip a
   1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
       link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
       inet 127.0.0.1/8 scope host lo
          valid_lft forever preferred_lft forever
       inet6 ::1/128 scope host 
          valid_lft forever preferred_lft forever
   2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1430 qdisc fq_codel state UP group default qlen 1000
       link/ether 1e:03:66:df:ad:5e brd ff:ff:ff:ff:ff:ff
       inet 10.244.69.163/32 brd 10.244.69.163 scope global eth0
          valid_lft forever preferred_lft forever
       inet6 fe80::1c03:66ff:fedf:ad5e/64 scope link 
          valid_lft forever preferred_lft forever
   ```

## xDisconnectVMNetwork

**根据不同的网络模型，移除容器和 VM 之间的网络配置**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L552)

1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象的网络模型（即[runtime].internetworking_model，默认为 tcfilter）
2. 根据网络模型，移除对应的 tap 设备
   - 如果网络模型为 macvtap
     1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，并进一步获取 macvtap 设备与 veth 设备
     2. 移除 macvtap 设备
     3. 将 veth 设备的 MAC 地址还原（在 **xConnextVMNetwork** 流程中保存在 netPair.TAPIface.HardAddr）
     4. 关停 veth 设备
     5. 将 veth 设备的 IP 地址还原（在 **xConnextVMNetwork** 流程中保存在 netPair.VirtIface.Addrs）
   - 如果网络模型为 tcfilter
     1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，并进一步获取 tuntap 设备与 veth 设备
     2. 关停 tuntap 设备，并移除
     3. 删除 veth 设备所有的 tc 规则与 ingress 类型的网络队列规则
     4. 关停 veth 设备

## addSingleEndpoint

**添加 endpoint 设备到 VM 中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L110)

1. 根据网口的类型，初始化对应的 endpoint<br>*物理设备是根据 ethtools 获取指定网口名称的 bus 信息判断，如果 bus 格式为 0000:00:03.0（即以冒号切分后长度为 3），则表示为物理设备；<br>vhost-user 设备是根据 /tmp/vhostuser_\<addr\>/vhu.sock（其中 addr 为网卡的每一个地址）文件是否存在，如果存在，则表示为 vhost-user 设备；<br>tuntap 设备仅支持 tap mode*
2. 调用 endpoint 的 **SetProperties**，设置 endpoint 属性信息
3. 根据是否为 hotplug，则调用 endpoint 的 **HotAttach** 或 **Attach**，热（添加）endpoint 设备到 VM 中
4. 调用 hypervisor 的 **IsRateLimiterBuiltin**，判断是否内置支持限速特性。如果本身不支持限速（例如 QEMU），则需要额外配置：

   - 网络 I/O inbound 带宽限速（即 [hypervisor].rx_rate_limiter_max_rate 大于 0）

     *veth、ipvlan、tuntap 和 macvlan 类型的 endpoint，待限速设备为 endpoint.NetPair 的 tap 设备；<br>macvtap 和 tap 类型的 endpoint，待限速设备为其本身，即 endpoint.Name()*

     1. 调用 endpoint 的 **SetRxRateLimiter**，设置 inbound 限速标识
     2. 获取待限速设备的索引，使用 HTB（Hierarchical Token Bucket）qdisc traffic shaping 方案来控制网口流量，设置 class 的 rate 和 ceil 均为 [hypervisor].rx_rate_limiter_max_rate<br>*class 1:2 是基于 class 1:1 创建，两者的 rate 和 ceil 流控指标保持一致，class 1:2 最终作为默认的 class，class 1:n 用于限制特定流量（截至 Kata 3.0，暂未实现）<br>之所以创建了 class 1:2 作为默认的 class，是一种常规做法，一般 class 1:1 承担限制整体的最大速率，class 1:2 用于控制非特权流量。如果统一由 class 1:1 负责，可能会导致非特权流量无法得到适当的控制和优先级管理。没有专门的子类别来定义规则和限制非特权流量，可能会导致这些流量占用过多的带宽，从而影响网络的性能和服务质量；难以灵活地调整限制策略。如果需要根据具体情况对非特权流量进行不同的限制和优先级分配，使用单一的1:1类别会显得不够灵活。而有一个专门的子类别，可以根据需要定义更具体的规则和策略，更好地控制非特权流量。所以，通过设置专门的 class 1:2，可以更好地组织和管理流量，确保网络的资源分配和性能满足特定的需求和优先级*
   
        ```shell
         +-----+     +---------+     +-----------+      +-----------+
         |     |     |  qdisc  |     | class 1:1 |      | class 1:2 |
         | NIC |     |   htb   |     |   rate    |      |   rate    |
         |     | --> | def 1:2 | --> |   ceil    | -+-> |   ceil    |
         +-----+     +---------+     +-----------+  |   +-----------+
                                                    |
                                                    |   +-----------+
                                                    |   | class 1:n |
                                                    |   |   rate    |
                                                    +-> |   ceil    |
                                                    |   +-----------+
        ```

   - 网络 I/O outbound 带宽限速（即 [hypervisor].tx_rate_limiter_max_rate 大于 0）
   
     *veth、ipvlan、tuntap 和 macvlan 类型的 endpoint 且当网络模型为 tcfilter 时，待限速设备为 endpoint.NetPair 的 veth 设备，当网络模型为 macvtap 或 none 时，待限速设备为 endpoint.NetPair 的 tap 设备；<br>macvtap 和 tap 类型的 endpoint，待限速设备为设备本身，即 endpoint.Name()*
   
     1. 对于 veth、ipvlan、tuntap 和 macvlan 类型的 endpoint 且当网络模型为 tcfilter 时，则获取 endpoint.NetPair 中 veth 设备的索引，同样的使用 HTB（Hierarchical Token Bucket）qdisc traffic shaping 方案来控制 veth 网口流量，设置 class 的 rate 和 ceil 均为 [hypervisor].tx_rate_limiter_max_rate<br>*对于 tcfilter，只需将 htb qdisc 应用于 veth pair。 对于其他网络模型，例如 macvtap，借助 ifb，通过将 endpoint 设备入口流量重定向到 ifb 出口，然后将 htb 应用于 ifb 出口，实现限速*
     2. 其他场景时，调用 endpoint 的 **SetTxRateLimiter**，设置 outbound 限速标识
     3. 尝试加载 host 的 ifb 模块，创建名为 ifb0 的 ifb 设备并启用，返回 ifb 设备索引号
     4. 为待限速的设备创建 ingress 类型的网络队列规则
     5. 为待限速设备添加过滤器规则，将其入站流量重定向到 ifb 设备进行出站处理
     6. 使用 HTB（Hierarchical Token Bucket）qdisc traffic shaping 方案来控制 ifb 网口流量，设置 class 的 rate 和 ceil 均为 [hypervisor].tx_rate_limiter_max_rate
   
   ***限速示例（veth endpoint）***
   
   ```shell
   # inbound 限速为 1024，outbound 限速为 2048
   $ cat /etc/kata-containers/configuration.toml | grep rate_limiter_max_rate
   rx_rate_limiter_max_rate = 1024
   tx_rate_limiter_max_rate = 2048
   
   # 网络模型为 macvtap 时
   $ ip netns exec cni-593e147b-3839-2615-f57f-39dc53181ef5 tc qdisc show
   qdisc noqueue 0: dev lo root refcnt 2 
   qdisc noqueue 0: dev eth0 root refcnt 2 
   qdisc htb 1: dev tap0_kata root refcnt 2 r2q 10 default 2 direct_packets_stat 0 direct_qlen 1500
   qdisc ingress ffff: dev tap0_kata parent ffff:fff1 ---------------- 
   qdisc htb 1: dev ifb0 root refcnt 2 r2q 10 default 2 direct_packets_stat 0 direct_qlen 32
   ## inbound 限速作用在 tap0_kata 设备上
   $ ip netns exec cni-593e147b-3839-2615-f57f-39dc53181ef5 tc class show dev tap0_kata
   class htb 1:1 root rate 1024bit ceil 1024bit burst 1600b cburst 1600b 
   class htb 1:2 parent 1:1 prio 0 rate 1024bit ceil 1024bit burst 1600b cburst 1600b 
   ## outbound 限速作用在 ifb0 设备上
   $ ip netns exec cni-593e147b-3839-2615-f57f-39dc53181ef5 tc class show dev eth0
   $ ip netns exec cni-593e147b-3839-2615-f57f-39dc53181ef5 tc class show dev ifb0
   class htb 1:1 root rate 2048bit ceil 2048bit burst 1600b cburst 1600b 
   class htb 1:2 parent 1:1 prio 0 rate 2048bit ceil 2048bit burst 1600b cburst 1600b
   
   # 网络模型为 tcfilter 时
   $ ip netns exec cni-58d2c6b0-b9e5-797d-4c9f-291769802ac1 tc qdisc show
   qdisc noqueue 0: dev lo root refcnt 2 
   qdisc htb 1: dev eth0 root refcnt 2 r2q 10 default 2 direct_packets_stat 0 direct_qlen 1000
   qdisc ingress ffff: dev eth0 parent ffff:fff1 ---------------- 
   qdisc htb 1: dev tap0_kata root refcnt 257 r2q 10 default 2 direct_packets_stat 0 direct_qlen 1000
   qdisc ingress ffff: dev tap0_kata parent ffff:fff1 ---------------- 
   ## inbound 限速作用在 tap0_kata 设备上
   $ ip netns exec cni-58d2c6b0-b9e5-797d-4c9f-291769802ac1 tc class show dev tap0_kata
   class htb 1:1 root rate 1024bit ceil 1024bit burst 1600b cburst 1600b 
   class htb 1:2 parent 1:1 prio 0 rate 1024bit ceil 1024bit burst 1600b cburst 1600b 
   ## outbound 限速作用在容器 veth pair 的 eth0 设备上
   $ ip netns exec cni-58d2c6b0-b9e5-797d-4c9f-291769802ac1 tc class show dev eth0
   class htb 1:1 root rate 2048bit ceil 2048bit burst 1600b cburst 1600b 
   class htb 1:2 parent 1:1 prio 0 rate 2048bit ceil 2048bit burst 1600b cburst 1600b
   ```

## AddEndpoints

**添加 endpoint 设备到 VM 中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L324)

1. 如果未指定 endpoint，则默认添加 netns 中所有的 enpoint 设备
   1. 针对 netns 中每一个网络设备接口信息（即 NetworkInfo），获得其名称、类型、IP 地址、路由、ARP neighbor 等信息（后续会设置在 endpoint.EndpointProperties 中，用于描述 endpoint 的属性）
   2. 忽略缺少 IP 地址的网络接口，以及本地回环接口<br>*缺少 IP 地址意味着要么是没有命名空间的基本隧道设备，如 gre0、gretap0、sit0、ipip0、tunl0，要么是错误设置的接口*
   3. 进入到该 netns 中，调用 **addSingleEndpoint**，向 VM 中添加 endpoint 设备
2. 否则，针对每一个 endpoint，进入到该 netns 中，调用 **addSingleEndpoint**，向 VM 中添加 endpoint 设备

## RemoveEndpoints

**移除 VM 中的 endpoint 设备**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L356)

1. 如果未指定 endpoint，则默认为 netns 中所有的 endpoint 设备（也就是 AddEndpoints 中添加的 endpoint 设备），针对每一个待移除的 endpoint
   1. 调用 endpoint 的 **GetRxRateLimiter**，如果设置了 inbound 限速，则进入到该 netns 中，移除限速设备 htb 类型的网络队列规则<br>*本质上就是对 addSingleEndpoint 中 inbound 限速处理的逆操作*
   2. 调用 endpoint 的 **GetTxRateLimiter**，如果设置了 outbound 限速，则进入到该 netns 中，移除限速设备 htb 类型的网络队列规则、删除限速设备所有的 tc 规则与 ingress 类型的网络队列规则以及关停并移除 ifb0 设备<br>*本质上就是对 addSingleEndpoint 中 outbound 限速处理的逆操作*
   3. 根据是否为 hotplug，则调用 endpoint 的 **HotDetach** 或 **Detach**，（热）移除 VM 中的 endpoint 设备
2. 如果 netns 是由 Kata Containers 创建，并且未指定 endpoint（即删除了 netns 中所有的 endpoint），则移除该 netns 的挂载点，并删除该 netns

