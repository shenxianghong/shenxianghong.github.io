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

*工厂函数是根据 [hypervisor.\<type\>] 中的类型，初始化对应的 hypervisor 空结构体。*

