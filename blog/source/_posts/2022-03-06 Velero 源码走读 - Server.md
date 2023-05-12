---
title: "「 Velero 」源码走读 — Server"
excerpt: "Velero 中与 VeleroServer、ResticServer 等主体服务相关的流程梳理"
cover: https://picsum.photos/0?sig=20220306
thumbnail: https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/stacked/199150-vmw-os-lgo-velero-final_stacked-gry.svg
date: 2022-03-06
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/one-line/199150-vmw-os-lgo-velero-final_gry.svg"></div>

------

> based on **v1.6.3**

# Config

*<u>pkg/client/factory.go</u>*<br>*<u>pkg/client/config.go</u>*<br>*<u>pkg/cmd/velero/velero.go</u>*<br>

Velero 的全局配置参数，比如 namespaces，features，cacert 和 colorized 等信息，是在初始化 velero 根 command 时，解析位于 host 的 \<HOME\>/.config/velero/config.json 的配置文件获取配置对象，初始化 factory 对象，将其透传给下级子 command 实现，由于 velero 服务的启动也是 velero 的子命令（即 velero server），因此实现了全局配置透传功能。

# Generic Controller

*<u>pkg/controller/generic_controller.go</u>*<br>*<u>pkg/controller/interface.go</u>*<br>*<u>internal/util/managercontroller/managercontroller.go</u>*

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

## Runable

[Runable 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/internal/util/managercontroller/managercontroller.go#L31)

用于将调用 Run 函数启动 Controller 的方法封装成 manager.Runable 返回，供 manager.Add 以及 manager.Start 使用<br>*manager 为 controller-runtime 的 manager*

# Velero Server

*<u>pkg/cmd/server/server.go</u>*

本质上是 velero cli 的 server 子命令，根据 install 以及更多的自定义参数，启动 Velero 服务。

## newServer

[newServer 源码](https://github.com/vmware-tanzu/velero/blob/3c49ec4fb4ff7f5aaa4ed56e8f7ff1a26f966d72/pkg/cmd/server/server.go#L246)

工厂函数

1. 设置 client 的 QPS 和 Burst，最终会作用在 Kube Client，Velero Client 和 Dynamic Client
2. 初始化 Kube Client，Velero Client 和 Dynamic Client
3. 初始化 PluginRegistry，发现注册在 /plugins 目录下的所有插件，并调用 velero run-plugins 命令启动插件的 GRPC 服务<br>*插件包括 item-action、objectStore 以及 volumeSnapshotter 等*
4. 如果 Velero 开启了 CSI 特性，则初始化 CSI Snapshot Client
4. 构建 controller-runtime 的 manager 对象
5. 初始化 CredentialFileStore，用于操作认证文件信息
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

启动 controller 以及其他服务

1. 启动 promHttp 服务，对接 Prometheus
2. 初始化 [pluginManager](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/plugin/clientmgmt/manager.go#L30)，提供与 Velero Plugin 交互的原生接口
3. 初始化 backupStoreGetter，提供操作 Backup 和 Restore 的上层接口<br>*即 Provider 章节中的 StorageProvider*
4. 按需初始化 CSI Snapshot Lister 和 CSI SnapshotContent Lister
5. 根据以上内容，初始化 Backup Controller、BackupSync Controller、Schedule Controller、GC Controller、BackupDeletion Controller、Restore Controller 以及 ResticRepo Controller，并将这些初步设定为默认开启的 Controller
6. 此外，ServerStatusRequest Controller 和 DownloadRequest Controller 作为服务运行时的状态 Controller，也会作为默认开启<br>*该类型的 Controller 与步骤 5 中的 Controller 处理逻辑相同，但是是分开处理和启动的*
7. 如果 Velero 服务为 restoreOnly 模式，则禁用 Backup Controller、Schedule Controller、GC Controller 以及 BackupDeletion Controller
8. 将启用的 Controller 和禁用的 Controller 取差集后，即为最终 Velero 服务中启动的 Controller 信息
9. 等待 Velero Client 和 CSI Snapshot Client 同步缓存（waitForCacheSync）
10. 启动 BackupStorageLocation Controller（Reconciler 方式），按需启动 ServerStatusRequest Controller 和 DownloadRequest Controller
11. 启动剩余的 Controller（不包含 Request 类型的 Controller），所有的 Controller 默认均为 1 worker

# Restic Server

*<u>pkg/cmd/cli/restic/server.go</u>*

本质上是 velero restic cli 的 server 子命令，根据自定义参数，启动 Restic 服务。

## newResticServer

[newResticServer 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/cmd/cli/restic/server.go#L119)

1. 初始化 KubeClient 和 Velero Client
2. 初始化 PodInformer，仅获取调度在本节点上的 Pod
3. 构建 controller-runtime 的 manager 对象
4. 根据以上内容，构建 restic server 对象
5. 判断挂载在 restic 服务中 /hosts_pods 目录下所有的 Pod 信息和集群中的所有的 Pod 是否一一对应

## run

[run 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/cmd/cli/restic/server.go#L183)

1. 启动 promHttp 服务，对接 Prometheus
2. 初始化 CredentialFileStore，用于操作认证文件信息
3. 根据以上内容，初始化 PodVolumeBackup Controller 和 PodVolumeRestore Controller
4. 启动 PodVolumeBackup Controller 和 PodVolumeRestore Controller，默认均为 1 worker

# Velero Restic Restore Helper

*<u>cmd/velero-restic-restore-helper/main.go</u>*

本质是 Velero 项目的另一个 binary 执行文件（还有一个是 velero 本身），在恢复 Pod 卷数据时，会给该 Pod 注入一个 InitContainer，该 binary 就是 InitContainer 所用镜像（velero/velero-restic-restore-helper:\<velero-version\>）中的启动服务。

## main

[main 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/cmd/velero-restic-restore-helper/main.go#L27)

每一个待恢复卷数据的 Pod 的 InitContainer 中运行的服务

1. 命令行接受一个参数，即 Restore 的 UID
2. 启动死循环定时器，每一秒钟检查 InitContainer 的 /restores 目录下每一个子目录的 .velero 目录下是否有所提供的 Restore UID 文件，如果每一个子目录都有该 UID 文件，则认为恢复完成，退出死循环，该 InitContainer 生命周期结束，否则，继续等待，无超时时间<br>*/restores 下每一个子目录代表一个待恢复的卷，命名为 Pod 使用的 PVC volumeMount 名称*
