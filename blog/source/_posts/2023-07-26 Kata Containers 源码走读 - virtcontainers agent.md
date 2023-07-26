---
title: "「 Kata Containers 」源码走读 — virtcontainers/agent"
excerpt: "virtcontainers 中与 Kata-agent 组件相关的流程梳理"
cover: https://picsum.photos/0?sig=20230726
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2023-07-26
toc: true
categories:
- Container Runtime
tag:
- Kata Containers

---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

# agent

*<u>src/runtime/virtcontainers/agent.go</u>*

从代码结构来看，agent 是一个典型的 CS 架构，客户端部分位于 virtcontainers 库中，与位于 guest VM 中运行的服务端进程 gRPC 通信，管理 VM 中容器进程的生命周期。

目前 agent 的实现方式仅有一种：kataAgent。

```go
type kataAgent struct {
	ctx      context.Context
	vmSocket interface{}

	client *kataclient.AgentClient

	// lock protects the client pointer
	sync.Mutex

	state KataAgentState

	reqHandlers map[string]reqFunc
    
	// [agent].kernel_modules
	kmodules    []string

    // [agent].dial_timeout
	dialTimout uint32

	// 固定为 true
	keepConn bool
	dead     bool
}
```

## init

**初始化 agent 服务**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/kata_agent.go#L348)

1. disableVMShutdown 是否为 true 取决于是否启用了 [agent].enable_tracing<br>*当 disableVMShutdown 为 true，也就意味着在关停 VM 时，会向 QMP 服务发送 quit 命令，请求关闭 VM 实例；否则，直接 syscall kill 掉 QEMU 进程，不会等待 VM 关闭，因此在启用 [agent].enable_tracing 时，VM 的关停时间会有所增加*
2. 初始化 agent 实现，返回 disableVMShutdown，后续决定在调用 Hypervisor 的 **Stop** 操作时的 VM 关闭方式

## capabilities

## check

## longLiveConn

## disconnect

## getAgentURL

## setAgentURL

## reuseAgent

## createSandbox

## exec

## startSandbox

## stopSandbox

## createContainer

## startContainer

## stopContainer

## signalProcess

## winsizeProcess

## writeProcessStdin

## closeProcessStdin

## readProcessStdout

## readProcessStderr

## updateContainer

## waitProcess

## onlineCPUMem

## memHotplugByProbe

## statsContainer

## pauseContainer

## resumeContainer

## configure

## configureFromGrpc

## reseedRNG

## updateInterface

## listInterfaces

## updateRoutes

## listRoutes

## getGuestDetails

## setGuestDateTime

## copyFile

## addSwap

## markDead

## cleanup

## save

## load

## getOOMEvent

## getAgentMetrics

## getGuestVolumeStats

## resizeGuestVolume

## getIPTables

## setIPTables
