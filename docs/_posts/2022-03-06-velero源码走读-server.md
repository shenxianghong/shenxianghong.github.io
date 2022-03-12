---
layout: post
title: "「 Velero 」 5.6 源码走读 — Server"
date: 2022-03-06
excerpt: "Velero 中与 Server 等控制面相关的源码走读"
tag:
- Cloud Native
- Kubernetes
- Velero
categories:
- Velero
---

![](https://velero.io/img/Velero.svg)

# Generic Controller

*<u>pkg/controller/generic_controller.go</u>*

顾名思义，Generic Controller 定义所有 Controller 的通用行为，本身负责周期性调用 Controller 注册的方法处理 Key，维护 Controller  Key 的生命周期。

每一个 Controller 都继承了 Generic Controller，主要包括注册 syncHandler 和 resyncFunc，以及 queue 和 cacheSyncWaiters 等。

Generic Controller 主要包含以下核心属性：

**queue**

默认使用 K8s 提供的 NewNamedRateLimitingQueue，队列中就是需要处理的 Key，格式为 \<namespace\>/\<name\> 或者 \<name\>（取决于对象是否是 namespaced scope）。

Generic Controller 提供了 enqueue 的方法，用于 Key 的入队（本质上就是 queue 的 Add 方法，只不过转换成了上述的格式）。

**syncHandler**

Generic Controller 会周期性的调用 Controller 注册的 syncHandler，处理 queue 中的 Key。

**resyncFunc**

Generic Controller 会根据 resyncPeriod 周期性的调用 Controller 注册的 resyncFunc，执行额外声明的逻辑。

**cacheSyncWaiters**

Generic Controller 在执行 syncHandler 和 resyncFunc 之前会等待注册在 cacheSyncWaiters 全部缓存完成（本质上，就是一组 func() bool 均返回 true 即可，只不过传入的均为 podInformer.HasSynced 函数）。

## Run

[Run 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/generic_controller.go#L54)

Generic Controller 的核心逻辑

1. 校验 syncHandler 和 resyncFunc 是否至少注册了一个
2. 如果注册了 cacheSyncWaiters，则等待其缓存同步完成<br>*PodVolumeBackup Controller 和 PodVolumeRestore Controller 均注册了 cacheSyncWaiters，用于同步 Pod、PVC 以及 PodVolumeBackup（PodVolumeRestore）* 
3. 启动指定 worker 数量的 Goroutine，每 1 秒钟处理一次以下逻辑<br>*该逻辑本身是死循环，只有在 queue 关闭时返回 false，因此隔 1 秒钟还会重新执行*
   1. 从 queue 中获取 Key（Get）
   2. 调用 syncHandler 注册的 Handler，处理 Key
      - 如果处理成功，则在 queue 中移除（Forget）
      - 如果处理失败，则限制速率重新加入 queue 中（AddRateLimited）
4. 每隔 resyncPeriod 执行一次 resyncFunc 逻辑<br>*resyncFunc 的处理不一定和 Key 相关，可以执行一些指标上报等操作，例如 Backup Controller 的 resyncFunc 实现*

# Velero Server

*pkg/cmd/server/server.go*

本质上是 velero cli 的 server 子命令，根据 install 以及更多的自定义参数，启动 Velero 服务。

## newServer

[newServer 源码](https://github.com/vmware-tanzu/velero/blob/3c49ec4fb4ff7f5aaa4ed56e8f7ff1a26f966d72/pkg/cmd/server/server.go#L246)

工厂函数

1. 设置 client 的 QPS 和 Burst，最终会作用在 KubeClient，VeleroClient 和 DynamicClient
2. 初始化 KubeClient，VeleroClient 和 DynamicClient
3. 初始化 PluginRegistry，发现注册在 /plugins 目录下的所有插件
4. 如果 Velero 开启了 CSI 特性，则初始化 CSISnapshotClient
5. 初始化 CredentialFileStore，用于操作 /tmp/credentials 目录下的认证文件信息
6. 根据以上内容，构建 server 对象

## run

[run 源码](https://github.com/vmware-tanzu/velero/blob/3c49ec4fb4ff7f5aaa4ed56e8f7ff1a26f966d72/pkg/cmd/server/server.go#L352)

server 运行的主体逻辑

1. 如果配置了 profile 地址，则启动 pprof 服务
2. 检查 Velero namespace 是否存在，如果不存在则报错
3. 初始化 DiscoveryHelper，每 5 分钟刷新一次，获取可以备份的对象信息
4. 检查 Velero 服务所需要的 CRD 是否存在，如果不存在则报错
5. 检查 Restic 是否存在，如果不存在则输出 warning 信息，确保 restic 所需要的 secret 存在（即 velero-restic-credentials），初始化 RepositoryManager
6. 调用 runControllers，启动所有的 Controller

## runControllers

[runControllers 源码](https://github.com/vmware-tanzu/velero/blob/3c49ec4fb4ff7f5aaa4ed56e8f7ff1a26f966d72/pkg/cmd/server/server.go#L566)



