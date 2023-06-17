---
title: "「 Kata Containers 」源码走读 — kata-monitor"
excerpt: "Kata Containers 指标采集、聚合与上报的流程梳理"
cover: https://picsum.photos/0?sig=20230129
thumbnail: /gallery/kata-containers/thumbnail.svg
date: 2023-01-29
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kata-containers/logo.svg"></div>

------

> based on **3.0.0**

Kata monitor 是一个守护进程，能够收集和暴露在同一 host 上运行的所有 Kata 容器工作负载相关的指标。一旦启动，它会检测 containerd-shim-kata-v2 系统中所有正在运行的 Kata Containers 运行时，并暴露一些 HTTP endpoints。主要 endpoint 是 /metrics（用于聚合来自所有 Kata 工作负载的指标）。

可用指标包括：

- Kata 运行时指标
- Kata agent 指标
- Kata guestOS 指标
- hypervisor 指标
- Firecracker 指标
- Kata monitor 指标

Kata monitor 提供的指标均采用 Prometheus 格式。虽然 Kata monitor 可以在任何运行 Kata Containers 工作负载的主机上用作独立守护进程，并且可以用于从正在运行的 Kata 运行时检索分析数据，但它的主要预期用途是作为 DaemonSet 部署在 Kubernetes 集群上。

*<u>src/runtime/cmd/kata-monitor/main.go</u>*

```go
type KataMonitor struct {
	// 维护 /run/vc/sbs 目录下的 sandbox 的基础信息，包括 uid、name 和 namespace
	sandboxCache *sandboxCache
    
	// --runtime-endpoint 参数指定，默认为 /run/containerd/containerd.sock
	runtimeEndpoint string
}
```

**main 函数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/cmd/kata-monitor/main.go#L69)

1. 如果指定的参数为 version 或者 --version，则展示其版本、架构、commit 等信息
2. 解构命令行参数与初始化日志
3. 初始化 monitor server

   1. 校验 --runtime-endpoint 参数是否不为空，默认为 /run/containerd/containerd.sock
   2. 注册指标至 Prometheus
   3. 启动 goroutine，实时处理 sandbox cache
      1. 启动 fsnotify watcher，监听 /run/vc/sbs 目录下的文件（文件名即为 sandboxID）
      2. 遍历目录内容，将 sandboxID 维护在 km.sandboxCache
      3. 根据 fsnotify watcher 监听到的创建或者删除事件，同步更新维护在 km.sandboxCache 中的 sandboxID
      4. 同时，默认每隔 5 秒，根据 --runtime-endpoint 参数构建 gRPC Client 调用 CRI 的 ListPodSandbox 获取到 CRI 中所有的 PodSandbox，将详细内容（即 sandboxCRIMetadata 对象，其中包含 UID、Name 和 Namespace 属性）更新至 km.sandboxCache 中
4. 注册 /metrics、/sandboxes、/agent-url 和一系列 Golang pprof 的 HTTP 端点；根服务请求（/）会展示所有可用的 HTTP 端点
5. 启动 monitor server 服务，监听地址通过 --listen-address 指定，默认为 127.0.0.1:8090

# ProcessMetricsRequest

**处理 /metrics 请求，获取 shim、hypervisor、vm 和 agent 指标**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/kata-monitor/metrics.go#L74)

1. 获取请求中的 sandbox 参数
2. 如果指定了 sandbox 参数，则通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送 HTTP GET 请求至 shim server 的 `http://shim/metrics`，获取指定 sandbox 的指标信息并返回（等价于 kata-runtime metrics \<sandboxID\>）
3. 如果没有指定 sandbox 参数，则通过 Prometheus 聚合所有 sandbox 的指标处理并返回

# ListSandboxes

**处理 /sandboxes 请求，获取所有运行的 sandbox**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/kata-monitor/monitor.go#L196)

1. 获取维护的所有 sandboxID
2. 根据实际请求，具体展示 HTML 或者 Text 格式的内容

# GetAgentURL

**处理 /agent-url 请求，获取指定 sandboxID 的 agent 地址**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/kata-monitor/monitor.go#L179)

1. 检验请求中的 sandbox 参数是否不为空
2. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 发送 HTTP GET 请求至 shim server 的 `http://shim/agent-url`，解析内容获得 sandbox socket 地址

# ExpvarHandler、PprofIndex、PprofCmdline、PprofProfile、PprofSymbol、PprofTrace

**处理 pprof 类请求，转发至 shim server**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/pkg/kata-monitor/pprof.go#L38)

1. 不同的 endpoint 在转发前会设置特定的请求头，例如 Content-Type 和 Content-Disposition
2. 代理请求
   1. 检验请求中的 sandbox 参数是否不为空
   2. 通过 /run/vc/sbs/\<sandboxID\>/shim-monitor.sock 转发 HTTP GET 请求至 shim server 的 `http://shim/<URL>`
   3. 根据调用传参中的请求处理方式加工数据并返回

