---
title: "「 Kata Containers 」源码走读 — virtcontainers network"
excerpt: "virtcontainers 库中 Endpoint 和 Network 模块源码走读"
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

Endpoint 代表了一组物理或虚拟网卡接口的基础结构，具体包括：veth、ipvlan、macvlan、physical、vhostuser、tap 和 tuntap 7 种实现方式。借助 `github.com/vishvananda/netlink` 将抽象 endpoint 类型转变成具体的 netlink 类型，配置后回写到 endpoint 的具体属性（例如 NetPair 等）后，交由 hypervisor 创建或配置该设备信息。

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
	// NetPair.VirtIface.HardAddr 为随机生成
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

NetworkInterfacePair 即 netpair（例如 br0_kata），描述了 tap 设备（TapInterface）和 veth 设备（VirtIface，即位于容器命名空间内部的 veth-pair 设备，如 eth0）的数据结构（非真实设备）。

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

### VethEndpoint、IPVlanEndpoint、MacvlanEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L99)

1. 调用 Network 的 **xConnectVMNetwork**，配置网络信息
1. 调用 hypervisor 的 **AddDevice**，添加 endpoint 中相关设备到 VM 中

### MacvtapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/macvtap_endpoint.go#L68)

1. 

## Detach

### VethEndpoint、IPVlanEndpoint、MacvlanEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L114)

1. 如果 netns 不是由 Kata Containers 创建的，则直接跳过后续<br>*根据创建 pod_sandbox 或者 single_container 时，spec.Linux.Namespace 中的 network 是否指定判断，如果未指定，表示需要由 Kata Containers 创建，反之表示 netns 已经提前创建好*
1. 进入到该 netns 中，调用 **xDisconnectVMNetwork**，移除网络信息

### MacvtapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/macvtap_endpoint.go#L92)

1. 无任何操作，直接返回

## HotAttach

### VethEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L130)

1. 调用 Network 的 **xConnectVMNetwork**，配置网络信息
2. 调用 hypervisor 的 **HotplugAddDevice**，热添加 endpoint 中相关设备到 VM 中

### IPVlanEndpoint、MacvlanEndpoint、MacvtapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/ipvlan_endpoint.go#L130)

1. 暂不支持热添加此类设备，返回错误

## HotDetach

### VethEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/veth_endpoint.go#L147)

1. 如果 netns 不是由 Kata Containers 创建的，则直接跳过后续<br>*根据创建 pod_sandbox 或者 single_container 时，spec.Linux.Namespace 中的 network 是否指定判断，如果未指定，表示需要由 Kata Containers 创建，反之表示 netns 已经提前创建好*
1. 进入到该 netns 中，调用 **xDisconnectVMNetwork**，移除网络信息
1. 调用 hypervisor 的 **HotplugRemoveDevice**，热移除 endpoint 中 VM 的相关设备

### IPVlanEndpoint、MacvlanEndpoint、MacvtapEndpoint

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/ipvlan_endpoint.go#L135)

1. 暂不支持热移除此类设备，返回错误

# Network

实际操作均借助 `github.com/vishvananda/netlink` 实现，该库提供了等价于 ip addr、ip link、tc qdisc、tc filter 命令行的功能。

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
     2. 创建名为 tap0_kata（示例名称，其中 0 为递增生成的索引）的 tuntap 设备，并返回空的 fds，回写到 netPair.VMFds 中
     3. 如果 [hypervisor].disable_vhost_net 未开启，则创建 /dev/vhost-net，构建 fds（[]*os.File，元素为队列长度数量的 /dev/vhost-net 文件句柄），回写到 netPair.VhostFds 中
     4. 设置 netPair.TAPIface.HardAddr 为 veth 设备的 MAC 地址<br>*将 veth MAC 地址保存到 tap 中，以便稍后用于构建 hypervisor 命令行。 此 MAC 地址必须是 VM 内部的地址，以避免任何防火墙问题。 host 上的网络插件预期流量源自这个 MAC 地址*
     5. 设置 tuntap 设备的 mtu 值为 veth 设备的 mtu 值
     6. 启用 tuntap 设备
     7. 为 tuntap 设备和 veth 设备创建 ingress 类型的 qdisc
     8. 为 tuntap 设备和 veth 设备创建 ingress 类型的 tc 规则分别指向对方，使得所有流量在两者之间可以被重定向
     
     综上所述，tcfilter 网络模式下，仅仅是在 veth 和 tap 设备之间配置 tc 规则，实现容器网络流量和 VM 网络流量的互通。

## xDisconnectVMNetwork

**根据不同的网络模型，移除容器和 VM 之间的网络配置**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/network_linux.go#L552)

1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象的网络模型（即[runtime].internetworking_model，默认为 tcfilter）
2. 根据网络模型，移除对应的 tap 设备
   - 如果网络模型为 macvtap
     1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，并进一步获取 macvtap 设备与 veth 设备
     2. 移除 macvtap 设备
     3. 将 veth 设备的 MAC 地址设置为 **xConnextVMNetwork** 流程中保存在 netPair.TAPIface.HardAddr 中的信息
     4. 关停 veth 设备
     5. 将 veth 设备的 IP 地址设置为 **xConnextVMNetwork** 流程中保存在 netPair.VirtIface.Addrs 中的信息
   - 如果网络模型为 tcfilter
     1. 调用 endpoint 的 **NetworkPair**，获取 netPair 对象，并进一步获取 tuntap 设备与 veth 设备
     2. 关停 tuntap 设备，并移除
     3. 获取 veth 设备所有的 ingress 类型的 tc 规则，并移除
     4. 获取 veth 设备所有的 ingress 类型的 qdisc，并移除
     5. 关停 veth 设备，并移除
