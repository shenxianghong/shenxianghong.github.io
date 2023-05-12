---
title: "「 Velero 」源码走读 — Location"
excerpt: "Velero 中与 Location、Repository 等存储站点相关的流程梳理"
cover: https://picsum.photos/0?sig=20220117
thumbnail: https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/stacked/199150-vmw-os-lgo-velero-final_stacked-gry.svg
date: 2022-01-17
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/one-line/199150-vmw-os-lgo-velero-final_gry.svg"></div>

------

> based on **v1.6.3**

# BackupStorageLocation

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/backupstoragelocation_types.go)

## backup-location

*<u>pkg/cmd/cli/backuplocation/backup-location.go</u>*

velero backup-location 包括 4 个子命令：create、delete、get 和 set。

## create

*<u>pkg/cmd/cli/backup-location/create.go</u>*

**校验规则**

- --provider 和 --bucket 为必需参数
- 在指定 --backup-sync-period 参数时，其值必须要大于等于 0 
- 在指定 --credential 参数时，其值仅能包含一对 key-value

**主体流程**

1. 根据命令行参数，构建 BackupStorageLocation 对象，下发至集群中创建，后续由 BackupStorageLocation Controller 负责维护
2. 如果新创建的 BackupStorageLocation 为默认的，则将集群中其余的 BackupStorageLocation 设为非默认

## delete

*<u>pkg/cmd/cli/backup-location/delete.go</u>*

**校验规则**

- name、--all 和 --selector 参数有且仅能有一个
- 要删除的 BackupStorageLocation 在集群中必须存在

**主体流程**

1. 根据命令行参数，获取集群中的 BackupStorageLocation 资源并删除

## get

*<u>pkg/cmd/cli/backup-location/get.go</u>*

**校验规则**

- 如果显式指定要获取的 BackupStorageLocation，则其在集群中必须存在

**流程逻辑**

1. 获取到 BackupStorageLocation 资源，根据 --output 指定的样式格式化输出

## set

*<u>pkg/cmd/cli/backup-location/set.go</u>*

**校验规则**

- 在指定 --credential 参数时，其值仅能包含一对 key-value

**主体流程**

1. 根据命令行提供的参数信息，将其更新至集群中的 BackupStorageLocation 对象中
2. 其中，如果参数信息是将其设置为默认（--default），则会将集群中其余的 BackupStorageLocation 设为非默认

# ResticRepository

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/restic_repository.go)

ResticRepository 不支持通过命令行手动创建，而是在备份流程中，由 Backup Controller 调用 **ensureRepo**，针对每一个卷命名空间，创建一个该对象。<br>*注意是 Pod 所在的命名空间（即卷命名空间），而非 Velero 或者 Restic 所在的命名空间*

## repo

*<u>pkg/cmd/cli/restic/repo/repo.go</u>*

### get

*<u>pkg/cmd/cli/restic/repo/get.go</u>*

**校验规则**

- 如果显式指定要获取的 ResticRepository，则其在集群中必须存在

**主体流程**

1. 获取到 ResticRepository 资源，根据 --output 指定的样式格式化输出

## server

*<u>pkg/cmd/cli/restic/server.go</u>*

server 本身是 velero restic 的 hidden 类型的命令，是 Restic 服务的启动命令。

1. 由于 Pod 会将所在所在节点的 /var/lib/kubelet/pods 挂在到容器内 /host_pods 目录下，因此，会校验当前节点的所有 Pod 卷是否均已挂载到容器内
2. 启动 Restic 服务，内部会启动 1 个 PodVolumeBackupController 和 1 个 PodVolumeRestoreController，用于处理卷备份与恢复的流程

# BackupStorageLocation Controller

*<u>pkg/controller/backup_storage_location_controller.go</u>*

## Reconcile

[Reconcile 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_storage_location_controller.go#L55)

1. 针对集群中的每一个 BackupStorageLocation，如果不存在默认的时，会根据 Velero 服务启动命令中的信息设置一个默认 <br>*在 Velero 之前版本中，Velero 服务启动时可以设置默认的 BackupStorageLocation，在 2.0 之后废弃，改用 velero backup-location set --default 的方式，而在 Controller 中仍然保留了这段逻辑，用于向后兼容*
2. 计算 BackupStorageLocation 是否已经准备好进行验证（即是否到达上次验证时间 + 验证频率），规则大致为
   - 频率等于 0 时，不作验证
   - 频率小于 0 时，为不合法场景，将频率重置为默认的 1 分钟
   - 如果未做过验证（即第一次尝试验证时），无视其设置的验证频率，直接返回 true，表示立即开始验证流程
3. 构建请求 StorageProvider 中所需要 BackupStorageLocation  对象，其中如果 BackupStorageLocation 中设置了 Credential 信息，则会获取位于 Velero 命名空间下的 Secret，将 Secret 内容以 \<Secret Name\> - \<Secret Key\> 的形式持久化到磁盘中，并将 BackupStorageLocation 的 credentialsFile 字段指向该文件<br>
   *通常位于 /tmp/credentials/velero 目录中*
4. 通过 StorageProvider 的 IsValid 接口判断 BackupStorageLocation 是否可用
5. 更新集群中 BackupStorageLocation 状态和上次验证时间
6. 最终无论结果如何，都会重新入队

# ResticRepository Controller

*<u>pkg/controller/restic_repository_controller.go</u>*

## NewResticRepositoryController

[NewResticRepositoryController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restic_repository_controller.go#L57)

工厂函数

1. 注册 Generic Controller 中的 syncHandler 和 resyncFunc
2. 监听 ResticRepository 资源的 Add 事件，将 ResticRepository 以 key（namespace/name）的形式加入 Generic Controller 的 queue 中

## processQueueItem

[processQueueItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restic_repository_controller.go#L111)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 ResticRepository key，通过解析获取的 namespace 和 name 查询到集群中的 ResticRepository 对象
2. 如果 ResticRepository 对象状态为空或者是 New，则调用 **initializeRepo** 初始化一个 Restic 仓库
2. 否则会进一步判断 ResticRepository 对象状态
   - 如果状态为 Ready，则执行 restic prune 命令，判断是否可以建立连接，并更新 ResticRepository 上次维护时间信息
   - 如果状态为 NotReady，则调用 **ensureRepo** 尝试检查或初始化一个仓库，并根据返回结果更新 ResticRepository 的状态为 Ready 或者 NotReady<br>*restic prune 的执行失败并不会影响到主流程，只是会在 ResticRepository 对象中记录错误信息*

## initializeRepo

[initializeRepo 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restic_repository_controller.go#L157)

尝试初始化 Restic 仓库的主体流程（真正初始化的动作位于 **ensureRepo**）

1. ResticRepository 对象中有 BackupStorageLocation 的信息，根据这个信息获取集群中的 BackupStorageLocation 对象，如果获取失败，则更新 ResticRepository 对象的状态为 NotReady
2. 调用 **GetRepoIdentifier** 获取 Restic 仓库信息（即 --repo 所需的信息），并更新至 ResticRepository 对象中，如果获取失败，则将 ResticRepository 对象的状态设置为 NotReady
3. 调用 **ensureRepo** 尝试检查或初始化一个仓库，并根据返回结果更新 ResticRepository 的状态和上次维护时间信息

## ensureRepo

[ensureRepo 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restic_repository_controller.go#L205)

检查 Restic 仓库是否存在，如果不存在则尝试初始化一个

1. 通过执行 restic snapshots 的结果，来确保 Restic 仓库存在并且权限可达
2. 如果命令执行返回错误信息中包含 “Is there a repository at the following location?”  字符串，表示 Restic 仓库不存在，会通过 restic init 命令初始化一个新仓库

## GetRepoIdentifier

[GetRepoIdentifier 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restic/config.go#L101)

构建 Restic 命令行需要的 --repo 参数内容

1. 根据 BackupStorageLocation 的 Provider 信息获取到对应后端类型（velero.io/provider），Velero Restic 支持的类型有 velero.io/aws、velero.io/azure 和 velero.io/gcp，拼接后作为 RepoPrefix
2. RepoPrefix 拼接上 ResticRepository 中的 VolumeNamespace 信息作为 RepoIdentifier，也就是 Restic 原生命令中的 --repo 参数

## enqueueAllRepositories

[enqueueAllRepositories 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restic_repository_controller.go#L97)

注册在 Generic Controller 中 resyncFunc 的实现，周期为 5 分钟

1. 获取集群中所有的 ResticRepository 对象，全量加入到 Generic Controller 的 queue 中<br>*之所以是全量，是因为网络连接的不确定性，需要重新判断所有的 ResticRepository 可达状态*
