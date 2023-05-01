---
title: "「 Kata Containers 」 3.4.5 源码走读 — virtcontainers network"
excerpt: "virtcontainers 库中 Endpoint 和 Network 模块源码走读"
cover: https://picsum.photos/0?sig=20230415
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-04-15
toc: true
categories:
- Code Walkthrough
tag:
- Kata Containers
---

<div align=center><img width="300" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> Based on **v3.0.0**

# Endpoint

*<u>src/runtime/virtcontainers/endpoint.go</u>*

Endpoint 代表了一组物理或虚拟网卡接口，具体包括：veth、ipvlan、macvlan、physical、vhostuser、tap 和 tuntap 7 种实现方式。

```go
// VethEndpoint gathers a network pair and its properties.
type VethEndpoint struct {
	// 固定为 virtual
	EndpointType       EndpointType
	PCIPath            vcTypes.PciPath
	EndpointProperties NetworkInfo
	// NetPair.TapInterface.Name 为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap<idx>_kata
	// NetPair.VirtIface.Name 为初始化入参指定，缺省为 eth<idx>
	// NetPair.NetInterworkingModel 为初始化入参指定
	NetPair            NetworkInterfacePair
	RxRateLimiter      bool
	TxRateLimiter      bool
}
```

```go
// IPVlanEndpoint represents a ipvlan endpoint that is bridged to the VM
type IPVlanEndpoint struct {
	// 固定为 ipvlan
	EndpointType       EndpointType
	PCIPath            vcTypes.PciPath
	EndpointProperties NetworkInfo
	// NetPair.TapInterface.Name 为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap<idx>_kata
    // NetPair.VirtIface.Name 为初始化入参指定，缺省为 eth<idx>
	// NetPair.VirtIface.HardAddr 为随机生成
	// NetPair.NetInterworkingModel 为 NetXConnectTCFilterModel
	NetPair            NetworkInterfacePair
	RxRateLimiter      bool
	TxRateLimiter      bool
}
```

```go
// MacvlanEndpoint represents a macvlan endpoint that is bridged to the VM
type MacvlanEndpoint struct {
	// 固定为 macvlan
	EndpointType       EndpointType
	PCIPath            vcTypes.PciPath
	EndpointProperties NetworkInfo
	// NetPair.TapInterface.Name 为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap<idx>_kata
	// NetPair.VirtIface.Name 为初始化入参指定，缺省为 eth<idx>
	// NetPair.VirtIface.HardAddr 为随机生成
	// NetPair.NetInterworkingModel 为初始化入参指定
	NetPair            NetworkInterfacePair
	RxRateLimiter      bool
	TxRateLimiter      bool
}
```

```go
// MacvtapEndpoint represents a macvtap endpoint
type MacvtapEndpoint struct {
	// 初始化入参
	EndpointProperties NetworkInfo
	// 固定为 macvtap
	EndpointType       EndpointType
	VMFds              []*os.File
	VhostFds           []*os.File
	PCIPath            vcTypes.PciPath
	RxRateLimiter      bool
	TxRateLimiter      bool
}
```

```go
// PhysicalEndpoint gathers a physical network interface and its properties
type PhysicalEndpoint struct {
	// 初始化入参 netInfo.Iface.Name
	IfaceName          string
	// 初始化入参 netInfo.Iface.HardwareAddr
	HardAddr           string
	EndpointProperties NetworkInfo
	// 固定为 physical
	EndpointType       EndpointType
	// 根据初始化入参 netInfo.Iface.Name 获取
	BDF                string
	// 软链接 /sys/bus/pci/devices/<BDF>/driver 指向实体文件路径的基础
	Driver             string
	// 由 /sys/bus/pci/devices/<BDF>/vendor 和 /sys/bus/pci/devices/<BDF>/device 文件内容拼接而成
	VendorDeviceID     string
	PCIPath            vcTypes.PciPath
}
```

```go
// VhostUserEndpoint represents a vhost-user socket based network interface
type VhostUserEndpoint struct {
	// Path to the vhost-user socket on the host system
	// 初始化入惨
	SocketPath string
	// MAC address of the interface
	// 初始化入参 netInfo.Iface.HardwareAddr
	HardAddr           string
	// 初始化入参 netInfo.Iface.Name
	IfaceName          string
	EndpointProperties NetworkInfo
	// 固定为 vhost-user
	EndpointType       EndpointType
	PCIPath            vcTypes.PciPath
}
```

```go
// TapEndpoint represents just a tap endpoint
type TapEndpoint struct {
  // TapInterface.Name 为初始化入参指定，缺省为 eth<idx>
	// TapInterface.TAPIface.Name 为 tap<idx>_kata
	TapInterface       TapInterface
	EndpointProperties NetworkInfo
	// 固定为 tap
	EndpointType       EndpointType
	PCIPath            vcTypes.PciPath
	RxRateLimiter      bool
	TxRateLimiter      bool
}
```

```go
// TuntapEndpoint represents just a tap endpoint
type TuntapEndpoint struct {
	// 固定为 tuntap
	EndpointType       EndpointType
	PCIPath            vcTypes.PciPath
	// TuntapInterface.Name 为初始化入参指定，缺省为 eth<idx>
	// TuntapInterface.TAPIface.Name 为 tap<idx>_kata
	// TuntapInterface.TAPIface.HardAddr 为初始化入参指定
	TuntapInterface    TuntapInterface
	EndpointProperties NetworkInfo
	// NetPair.TapInterface.Name 为 br<idx>_kata
	// NetPair.TapInterface.TAPIface.Name 为 tap<idx>_kata
	// NetPair.VirtIface.Name 为初始化入参指定，缺省为 eth<idx>
	// NetPair.NetInterworkingModel 为初始化入参指定
	NetPair            NetworkInterfacePair
	RxRateLimiter      bool
	TxRateLimiter      bool
}
```

```go
// NetworkInterfacePair defines a pair between VM and virtual network interfaces.
type NetworkInterfacePair struct {
	TapInterface
	VirtIface NetworkInterface
	NetInterworkingModel
}
```

NetworkInterfacePair 即 netpair（例如 br0_kata），描述了 tap 设备（TapInterface）和 veth 设备（VirtIface，即位于容器命名空间内部的 veth-pair 设备，如 eth0）的数据结构。

*工厂函数为简单的赋值操作，具体参考 Network。*

Endpoint 中声明的 **Properties**、**Type**、**PciPath**、**SetProperties**、**SetPciPath**、**GetRxRateLimiter**、**SetRxRateLimiter**、**GetTxRateLimiter** 和 **GetTxRateLimiter** 均为参数获取与赋值，无复杂逻辑，不作详述。

其中，**Name**、**HardwareAddr** 和 **NetworkPair** 视不同的 Endpoint 实现，取值有所不同，具体为：

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

### VethEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L99)

1. 调用 Network 的 **xConnectVMNetwork**
1. 调用 hypervisor 的 **AddDevice**，添加 VethEndpoint 设备到 VM 中

## Detach

## HotAttach

## HotDetach

# Network

## xConnectVMNetwork

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L518)

1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，进一步获取其属性信息以及网络模型（即[runtime].internetworking_model，默认为 tcfilter）
2. 调用 hypervisor 的 **Capabilities**，判断 hypervisor 是否支持多队列特性。如果支持，则队列数设为 [hypervisor].default_vcpus；否则为 0
3. 如果网络模型为 macvtap
   1. 将 endpoint 类型（VethEndpoint、MacvlanEndpoint、IPVlanEndpoint 和 TuntapEndpoint）转换成 netlink 对应的类型
   2. 目前 macvtap 需要 workaround 处理索引，初始化 macvtap 类型的 netlink.Link 对象，创建名为 tap0_kata（示例名称，其中 0 为递增生成的索引）、属性（例如 parentIndex 和 txQLen）继承自 veth 设备的 macvtap 设备（类比调用 ip link add \<netlink\>）<br>*Linux 内核中存在一个限制，它会阻止 macvtap/macvlan link 在网络名称空间中创建时获得正确的 link 索引
      https://github.com/clearcontainers/runtime/issues/708
      在修复该错误之前，需要选择一个随机的非冲突索引并尝试创建一个 link。 如果失败，需要尝试另一个。
      所有内核都不会检查链接 ID 是否与主机上的 link ID 冲突，因此需要偏移 link ID 以防止与主机索引发生任何重叠，内核将确保没有竞争条件*
   3. 设置 macvtap 设备的 mtu 值为 veth 设备的 mtu 值（类比调用 ip link set \<netlink\> mtu \<mtu\>）
   4. 设置 veth 设备的 mac 地址为随机生成的 mac 地址（类比调用 ip link set \<netlink\> address \<hwaddr\>），并设置 macvtap 设备的 mac 地址为 veth 设备的 mac 地址<br>*以上操作的最终目的，是将 CNI 给 veth 设备分配的 mtc、mac 地址等信息设置给 tap 设备，后续用于调用 hypervisor 传递。其中初始化 tap 设备时，随机生成了一个 mac 地址，用于设置给 veth 设备*
   5. 启用 tap 设备（类比调用 ip link set \<netlink\> up）
