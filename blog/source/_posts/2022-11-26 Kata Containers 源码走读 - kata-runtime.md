---
title: "「 Kata Containers 」源码走读 — kata-runtime"
excerpt: "Kata Containers 命令行工具的流程梳理"
cover: https://picsum.photos/0?sig=20221126
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2022-11-26
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

kata-runtime 是一个可执行程序，用于运行基于 OCI（Open Container Initiative）构建打包的应用。

*<u>src/runtime/cmd/kata-runtime/main.go</u>*

kata-runtime 本身是基于 [urfave/cli](https://github.com/urfave/cli) 库构建。kata-runtime 包括 7 个子命令：check（kata-check）、env（kata-env）、exec、metrics、factory、direct-volume 和 iptables。

**beforeSubcommands**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/main.go#L221)

1. 如果指定了 --show-default-config-paths 参数，则展示配置文件默认的路径（/etc/kata-containers/configuration.toml 和 /opt/kata/share/defaults/kata-containers/configuration.toml）
2. 判断用户的输入是否需要展示用法（例如 kata-runtime、kata-runtime help、kata-runtime --help 和 kata-runtime -h 等），如果满足条件，则直接展示用法文本，不执行后续流程
3. 解析 --rootless 参数并设置
4. 如果子命令为 check（kata-check），则设置日志级别为 warn；否则，根据 --log 参数创建日志文件（默认为 /dev/null），根据 --log-format 设置日志格式（支持 text 和 json，默认为 text），日志中新增 command 字段标识子命令，提取 context 设置给 logger
5. 将配置文件内容解析并转为 OCI runtime 配置，设置在 context 中，后续的操作中不再解析配置文件

****

# check（kata-check）

**Kata Containers 的运行环境要求检查**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-check.go#L313)

1. 如果指定了 --verbose 参数，则设置日志级别为 Info
2. 如果没有指定 --no-network-checks 参数并且没有声明 KATA_CHECK_NO_NETWORK 环境变量，则借助网络尝试进行 release 版本检查：如果当前用户为 root 则输出 Not running network checks as super user，否则执行 release version 检查
   1. 校验当前版本号是否符合 SemVer 版本规范
   2. 根据版本解析中的主版本号获取 release URL：如果为 0，则不合法；如果为 1 则获取 1.x 的 [release URL](https://api.github.com/repos/kata-containers/runtime/releases)；如果为 2 则获取 2.x 的 [release URL](https://api.github.com/repos/kata-containers/kata-containers/releases)；如果环境变量中声明了 KATA_RELEASE_URL 则以此为准
   3. 检验 release URL 的合法性，除了默认的 1.x 和 2.x 版本之外，其余的均为不合法，因此通过环境变量 KATA_RELEASE_URL 声明的 release URL 也必须为官方默认的
   4. 如果当前用户为 root 则返回 No network checks allowed running as super user 错误，否则请求 release URL，根据是否指定了 --include-all-releases 参数，解析符合要求的 release 详情信息
   5. 如果指定了 --only-list-releases 参数，则展示所有的 release 详情，不执行后续流程
   6. 获取最新的 release，并展示其详情信息
3. 如果指定了 --check-version-only 参数或者 --only-list-releases 参数则不执行后续流程
4. 解析获得 OCI runtime 配置信息，根据使用的 hypervisor 的类别，设置 CPU 类别，获取运行所需的 CPU flags 和内核模块

   ***amd64***

   1. 根据 /proc/cpuinfo 文件中字符串匹配 GenuineIntel 或 AuthenticAMD 获得其 CPU 类型，x86 架构下支持  Intel 和 AMD 类型
   2. 如果 CPU 类型为 Intel 时：
      1. 根据 CPU flags 中是否含有 "hypervisor" 判断是否运行在 VM 环境中，如果没运行在 VM 中，则需要支持 [VMX Unrestricted](https://communities.vmware.com/t5/VMware-Workstation-Pro/What-is-VMX-Unrestricted-Guest/td-p/2748822) 模式（用于判断系统环境是否足够新，用以满足运行 Kata Containers，至少是 [Westmere](https://en.wikipedia.org/wiki/Westmere_(microarchitecture))）
      2. 如果 hypervisor 为 QEMU、Cloud hypervisor、Firecracker 和 Dragonball 时，则要求 CPU 具有 vmx、lm 和 sse4_1 的 flag 特性以及内核模块中 kvm、kvm_intel、vhost、vhost_net 和 vhost_vsock 应启动；如果 hypervisor 为 acrn 时，则要求 CPU 具有 lm 和 sse4_1 的 flag 特性以及内核模块中 vhm_dev、vhost 和 vhost_net 应启动；如果 hypervisor 为 mock 时，则要求 CPU 具有 vmx、lm 和 sse4_1 的 flag 特性
   3. 如果 CPU 类型为 AMD 时：
      1. 无论 hypervisor 的类型，要求 CPU 具有 svm、lm、sse4_1 的 flag 特性以及内核模块中 kvm、kvm_amd、vhost、vhost_net 和 vhost_vsock 应启动
      1. 记录以上依赖要求至全局变量中，后续会作为运行环境监测的依据

   ***arm64***

   1. arm64 架构下，setCPUtype 不做任何处理，而是采取了相关全局变量硬编码方式
   2. 要求内核模块中 kvm、vhost、vhost_net 和 vhost_vsock 应启动，CPU flag 特性无特殊要求
5. 判断当前环境是否满足运行 Kata Containers 要求，满足要求时输出 System is capable of running Kata Containers
6. 如果当前用户为 root，则通过系统调用创建一个最小化的 VM 之后并删除，用以检测当前环境是否能够满足创建 VM 的要求

   ***amd64***

   1. 如果 hypervisor 为 QEMU、Cloud Hypervisor、Firecracker 时，验证流程参考：[kvmIsUsable](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-check_amd64.go#L234)；hypervisor 为 ACRN 时，验证流程参考：[acrnIsUsable](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-check_amd64.go#L240)。满足要求时输出 System can currently create Kata Containers

   ***arm64***

   1. 不区分 hypervisor 类型，验证流程参考：[kvmIsUsable](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-check_arm64.go#L66)
   2. 验证是否支持 KVM Extension，验证流程参考：[checkKVMExtensions](#https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-check_arm64.go#L70)

****

# env（kata-env）

**展示 Kata Containers 的设置信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-env.go#L427)

1. 调用上述的 **setCPUtype**，根据 hypervisor 的类别，获取运行所需的 CPU flags 和内核模块
2. 生成 meta 配置项内容，其中 version 固定为 1.0.26
3. 通过配置文件和 OCI Runtime 的信息，生成 runtime、agent、 hypervisor、image、initrd、kernel  配置项内容
4. 通过解析 /proc/version 获取内核版本信息；通过解析 /etc/os-release 或者 /usr/lib/os-release 获取发行版名称和版本信息；通过解析 /proc/cpuinfo 获得 CPU 类别和型号；通过 /dev/vhost-vsock 的存在性，判断是否支持 vhost-sock。此外，汇合内存总量与使用量、CPU 是否满足运行要求等，生成 host 配置项内容
5. 汇总以上配置项内容，根据是否指定 --json 参数（默认为 TOML 格式），格式化展示内容

****

# exec

**借助 debug console 进入到 VM 中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-exec.go#L51)

1. 如果没有指定 --kata-debug-port 参数或者指定为 0，则 debug 端口设置为默认的 1026
2. 校验指定的 sandboxID 参数是否不为空，且正则匹配满足 ^\[a-zA-Z0-9][a-zA-Z0-9_.-]+$
3. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送 HTTP GET 请求至 shim server 的 `http://shim/agent-url`，解析内容获得 sandbox 的 console socket，示例如下

   ```SHELL
   $ curl --unix-socket /run/vc/sbs/dd2aa45873a9c0f5e1e93fc38cc0e1fe561e79e33aa85be49487162c1ebc7f43/shim-monitor.sock http://shim/agent-url
   vsock://4138340623:1024
   ```

4. 如果 sandbox 的 console socket 协议为 vsock，则构建成类似 vsock://4138340623:1026 的格式；如果协议为 hvsock，则构建成 hvsock:///run/vc/firecracker/340b412c97bf1375cdda56bfa8f18c8a/root/kata.hvsock:1026 的格式。仅支持此两种协议，建立 grpc 请求链接，用于 VM 内外的通信交互
5. 获取当前进程的 console，将 kata-runtime exec \<sandboxID\> 的输出流展示到当前 console 中

****

# metrics

**获取 VM 中暴露的指标信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-metrics.go#L16)

1. 校验指定的 sandboxID 参数是否不为空，且正则匹配满足 ^\[a-zA-Z0-9][a-zA-Z0-9_.-]+$
2. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送 HTTP GET 请求至 shim server 的 `http://shim/metrics`，展示请求返回内容

****

# factory

## init

**初始化 VM factory**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L148)

1. 如果启用 VM cache 特性（即 [factory].vm_cache_number 大于 0），则初始化一个新的 factory（即 fetchOnly 为 false）。启动 cache server，监听 [factory].vm_cache_endpoint（默认为 /var/run/kata-containers/cache.sock）
3. 如果启用 VM template 特性（即 [factory].enable_template 为 true），则初始化一个新的 factory（即 fetchOnly 为 false）；否则视为 VM cache 和 VM template 均未开启前提下调用 kata-runtime factory init，抛出相关报错

## destory

**销毁 VM factory**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L224)

1. 如果启用 VM cache 特性（即 [factory].vm_cache_number 大于 0），则通过 [factory].vm_cache_endpoint（默认为 /var/run/kata-containers/cache.sock）gRPC 调用 cache server 的 **Quit**，请求关闭 cache server
2. 如果启用 VM template 特性（即 [factory].enable_template 为 true），则获取现有的 factory （即 fetchOnly 为 true），调用 factory 的 <u>CloseFactory</u>，关闭 factory

## status

**查询 VM factory 的状态**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/factory.go#L274)

1. 如果启用 VM cache 特性（即 [factory].vm_cache_number 大于 0），则通过 [factory].vm_cache_endpoint（默认为 /var/run/kata-containers/cache.sock）gRPC 调用 cache server 的 **Status**，展示请求返回内容
2. 如果启用 VM template 特性（即 [factory].enable_template 为 true），则获取现有的 factory （即 fetchOnly 为 true），输出其是否存在

****

# direct-volume

```go
// MountInfo contains the information needed by Kata to consume a host block device and mount it as a filesystem inside the guest VM.
type MountInfo struct {
	// The type of the volume (ie. block)
	VolumeType string `json:"volume-type"`
	// The device backing the volume.
	Device string `json:"device"`
	// The filesystem type to be mounted on the volume.
	FsType string `json:"fstype"`
	// Additional metadata to pass to the agent regarding this volume.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Additional mount options.
	Options []string `json:"options,omitempty"`
}
```

直通卷的操作都是基于 MountInfo 结构，它描述了待直通至 VM 中的卷位于 host 侧的信息详情。

## add

**为指定的 VM 新增直通卷**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-volume.go#L42)

1. 对指定的 --volume-path 参数进行 URLEncoding 后，拼接成 /run/kata-containers/shared/direct-volumes/\<volumePath (base64)\> 路径目录
2. 如果该路径不存在，则创建目录层级；如果该路径存在，则判断其是否为目录
3. 将 --mount-info 参数内容持久化到该目录下的 mountInfo.json 文件中

## remove

**删除指定 VM 的直通卷**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-volume.go#L65)

1. 对指定的 --volume-path 参数进行 URLEncoding 后，拼接成 /run/kata-containers/shared/direct-volumes/\<volumePath (base64)\> 路径目录
2. 移除该目录

## stats

**获取 VM 中直通卷的文件系统信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-volume.go#L83)

1. 对指定的 --volume-path 参数进行 URLEncoding 后，拼接成 /run/kata-containers/shared/direct-volumes/\<volumePath (base64)\> 路径目录
2. 遍历目录，获取到 sandboxID（直通卷模式下，该目录中仅有一个 sandboxID 目录与 mountInfo.json 文件，因此名称不为 mountInfo.json 即为 sandboxID）
3. 获取并解析目录中的 mountInto.json 文件内容，得到 mountInfo.Device（即位于 host 上待直通至 VM 中的设备）
4. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送 HTTP GET 请求至 shim server 的 `http://shim/direct-volume/stats?path=<device>`，展示请求返回内容

## resize

**扩容 VM 的直通块设备的卷大小**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-volume.go#L104)

1. 对指定的 --volume-path 参数进行 URLEncoding 后，拼接成 /run/kata-containers/shared/direct-volumes/\<volumePath (base64)\> 路径目录
2. 遍历目录，获取到 sandboxID（直通卷模式下，该目录中仅有一个 sandboxID 目录与 mountInfo.json 文件，因此名称不为 mountInfo.json 的即为 sandboxID）
3. 获取并解析目录中的 mountInto.json 文件内容，得到 mountInfo.Device（即位于 host 上待直通至 VM 中的设备）
4. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送格式为 application/json 的 HTTP POST 请求至 shim server 的 `http://shim/direct-volume/resize`，其中请求体包含 mountInfo.Device 和卷扩容后的期望大小

****

# iptables

## get

**获取 VM 中的 iptables 规则**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-iptables.go#L36)

1. 校验指定的 sandboxID 参数是否不为空，且正则匹配满足 ^\[a-zA-Z0-9][a-zA-Z0-9_.-]+$
2. 如果额外指定了 --v6 参数，则 url 为 /ip6tables，否则为 /iptables
3. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送 HTTP GET 请求至 shim server 的 `http://shim/<url>`，展示请求返回内容

## set

**基于指定的文件内容，设置 VM 中的 iptables 规则**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-runtime/kata-iptables.go#L72)

1. 校验指定的 sandboxID 参数是否不为空，且正则匹配满足 ^\[a-zA-Z0-9][a-zA-Z0-9_.-]+$
2. 校验指定的 iptables 参数对应的文件是否存在，并读取 iptables 文件内容
3. 如果额外指定了 --v6 参数，则 url 为 /ip6tables，否则为 /iptables
4. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送格式为 application/octet-stream 的 HTTP PUT 请求至 shim server 的 `http://shim/<url>`，其中请求体包含 iptables 文件内容流
