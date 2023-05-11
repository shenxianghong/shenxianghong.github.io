---
title: "「 Velero 」源码走读 — Restore"
excerpt: "Velero 中与 Restore 等恢复模块相关的流程梳理"
cover: https://picsum.photos/0?sig=20220204
thumbnail: https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/stacked/199150-vmw-os-lgo-velero-final_stacked-gry.svg
date: 2022-02-04
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/one-line/199150-vmw-os-lgo-velero-final_gry.svg"></div>

------

> based on **v1.6.3**

# Restore

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/restore.go)

## restore

*<u>pkg/cmd/cli/restore/restore</u>*

velero restore 包括 5 个子命令：create、delete、describe、get 和 logs。

## create

*<u>pkg/cmd/cli/restore/create.go</u>*

**校验规则**

- --from-backup 和 --from-schedule 参数有且仅能有一个
- 在指定 --from-backup 或者 --from-schedule 参数时，其在集群中必须存在
- 在指定 --from-schedule 参数时，则由该 Schedule 创建出的 Backup 至少有一个

**主体流程**

1. 如果指定了 --from-schedule 参数并且 --allow-partially-failed 参数为 true 时，获取集群中由该 Schedule 创建出的状态为 Completed 或者 PartiallyFailed 的最新 Backup，作为恢复的基准，并且将 --from-schedule 信息置空；否则，则直接透传 --from-schedule 信息用于后续构建 Restore 对象
2. 根据命令行参数，构建 Restore 对象，下发至集群中创建，后续由 Restore Controller 负责维护
3. 如果开启了 --wait，则启动 informer 监听 Restore 对象状态，阻塞直至状态不再是 New 或者 InProgress

## delete

*<u>pkg/cmd/cli/restore/delete.go</u>*

**校验规则**

- name、--all 和 --selector 参数有且仅能有一个
- 要删除的 Restore 在集群中必须存在

**主体流程**

1. 根据命令行参数，获取集群中的 Restore 资源并删除

## describe

*<u>pkg/cmd/cli/restore/describe.go</u>*

**校验规则**

- 要获取的 Restore 在集群中必须存在

**主体流程**

1. 获取到 PodVolumeRestore 信息
2. 将以上信息和 Restore 的元信息、规格、状态以及 Pod 卷数据恢复等汇总作为描述信息格式化输出
   - 如果 Restore 中有 Error 或者 Warning 的日志时，会构建 DownloadRequest 对象获取  RestoreResults 的信息，展示原因
   - 在开启 --details 时，会输出恢复的 Pod 卷信息

## get

*<u>pkg/cmd/cli/restore/get.go</u>*

**校验规则**

- 要获取的 Restore 在集群中必须存在

**主体流程**

1. 获取到 Restore 资源，根据 --output 指定的样式格式化输出

## logs

*<u>pkg/cmd/cli/restore/logs.go</u>*

**校验规则**

- 要获取的 Restore 在集群中必须存在
- Restore 的状态必须为 Completed、PartiallyFailed 或者 Failed

**主体流程**

1. 根据命令行参数，构建 DownloadRequest 对象，下发至集群中创建 RestoreLog，下发至集群中创建，后续由 DownloadRequest Controller 负责维护，获取 RestoreLog 的信息
2. 阻塞直至 DownloadRequest 的 DownloadURL 被设置，将内容写入 stdout 中

# PodVolumeRestore

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/pod_volume_restore.go) 

对象不支持手动创建，而是在恢复流程中，由 Restore Controller 调用 **restoreItem**，针对每一个 Pod 卷，创建一个该对象。

# Restore Controller

*<u>pkg/controller/restore_controller.go</u>*<br>*<u>pkg/restore/restore.go</u>*

## NewRestoreController

[NewRestoreController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restore_controller.go#L99)

工厂函数

1. 注册 Generic Controller 中的 syncHandler 和 resyncFunc
2. 监听 Restore 资源的 Add 事件，将状态是空或者 New 的 Restore 以 key（namespace/name）的形式加入 Generic Controller 的 queue 中

## processQueueItem

[processQueueItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restore_controller.go#L178)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 Restore key，通过解析获取的 namespace 和 name 查询到集群中的 Restore 对象
2. 仅处理状态为空或者 New 的 Restore 对象，调用 **processRestore**，执行恢复

## processRestore

[processRestore 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restore_controller.go#L215)

恢复的整体流程

1. 调用 **validateAndComplete** 做一些校验准备工作，并根据校验结果设置状态为 Restore 的状态为 FailedValidation 或者 InProgress
2. 过滤掉校验失败的 Restore，调用 **runValidatedRestore**，执行恢复和上传恢复信息的流程
3. 根据步骤 2 的恢复结果，设置 Restore 的状态
   - 如果有错误返回（函数返回 error），则认为恢复失败，将 Restore 状态设为 Failed
   - 如果 Restore 的 Errors 存在错误信息，则为部分失败，将 Restore 状态设为 PartiallyFailed
   - 否则认为 Restore 已完全恢复，将 Restore 状态设为 Completed

   *runValidatedRestore 执行恢复过程中会设置 Restore 对象的 status.Errors 和 status.Warning 信息*

## validateAndComplete

[validateAndComplete 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restore_controller.go#L288)

校验 Restore 对象并返回 backupInfo 信息

```go
type backupInfo struct {
	backup      *api.Backup
	location    *velerov1api.BackupStorageLocation
	backupStore persistence.BackupStore
}
```

1. 在 Restore 对象的 ExcludedResources 中追加以下资源（以下资源会被备份，但是不会被恢复）
   - nodes
   - events
   - events.events.k8s.io
   - backups.velero.io
   - restores.velero.io
   - resticrepositories.velero.io
2. 校验 Restore 对象的 IncludedResources 中是否包含上述资源
3. 校验 included/excluded resources/namespaces 信息以及 --from-backup 和 --from-schedule 是否有且仅有一个
4. 如果指定了 --from-schedule，则校验并获取 Schedule 下最新的一次 Backup，并设置在 Restore 中<br>*如果没有设置，则 --from-backup 必然设置了，因此就不需要设置 Restore 的 Backup 信息了*
5. 根据 Restore 的 Backup 信息，查询并返回 BackupStorageLocation、Backup 和 StorageProvider 信息

## runValidatedRestore

[runValidatedRestore 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/restore_controller.go#L430)

恢复的整体流程

1. 获取注册的 RestoreItemActions 插件<br>*后续在调用 **Restore** 函数恢复时会使用到*
2. 调用 StorageProvider 的 GetBackupContents 接口，获取到 Backup 内容文件信息，写入临时文件中
3. 根据 Restore 中的 Backup 名称信息，获取到集群中 PodVolumeBackup 信息
4. 调用 StorageProvider 的 GetBackupVolumeSnapshots 接口，获取到 VolumeSnapshot 信息
5. 用以上信息构建 Restore Request（原理类似 Backup 的 Request，不做赘述），其中 BackupReader 为步骤 2 中的临时文件句柄

   ```go
   type Request struct {
   	*velerov1api.Restore
   
   	Log              logrus.FieldLogger
   	Backup           *velerov1api.Backup
   	PodVolumeBackups []*velerov1api.PodVolumeBackup
   	VolumeSnapshots  []*volume.Snapshot
   	BackupReader     io.Reader
   }
   ```

6. 调用 **Restore** 进行恢复，返回恢复的结果
7. 重新获取 StorageProvider，避免长时间恢复中认证信息变动
8. 关闭日志文件，本次恢复任务日志记录完毕，调用 StorageProvider 的 PutRestoreLog 接口，将恢复任务的日志信息上传至 BackupStorageLocation 
9. 统计 Warning 和 Error 级别日志数量，更新至 Restore 对象中，并构建 Result 对象（即代码块中的 m 对象），记录过程中产生的日志信息

   ```go
   restore.Status.Warnings = len(restoreWarnings.Velero) + len(restoreWarnings.Cluster)
   for _, w := range restoreWarnings.Namespaces {
   	restore.Status.Warnings += len(w)
   }
   
   restore.Status.Errors = len(restoreErrors.Velero) + len(restoreErrors.Cluster)
   for _, e := range restoreErrors.Namespaces {
   	restore.Status.Errors += len(e)
   }
   
   m := map[string]pkgrestore.Result{
   	"warnings": restoreWarnings,
   	"errors":   restoreErrors,
   }
   ```

10. 调用 StorageProvider 的 PutRestoreResults 接口，将 Result 信息上传至 BackupStorageLocation 中

调用 StorageProvider 接口上传的具体文件对应关系如下：

| 名称    | BackupStorageLocation 中的文件 | 数据源                          |
| ------- | ------------------------------ | ------------------------------- |
| Log     | restore-\<restore\>-logs.gz    | 步骤 8 中最终生成的 log 文件    |
| Results | restore-\<restore\>-results.gz | 步骤 9 中最终生成的 Result 对象 |

## Restore

[Restore 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restore/restore.go#L159)

恢复动作本身的流程

1. 获取到 resource & namespace 的 included & excluded、resources hook 以及 resolvedActions（RestoreItemAction），构建 Restore Context，Context 是用于执行恢复所需要的上下文信息
2. 将 request 中的 BackupReader 解压，并将内容解析成 Backup 的资源信息
3. 如果 Velero 开启了 APIGroupVersions 特性，调用 [chooseAPIVersionsToRestore](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restore/prioritize_group_version.go#L46) 对多 API Group Versions 的资源选择一个要恢复的版本
4. 生成 update 队列，用于记录 Restore 状态信息，同时启动一个 Goroutine 监听队列，每秒钟获取一次 update 队列中的进度并更新至集群的 Restore 中
5. 首先恢复 CRD 资源，调用 **getOrderedResourceCollection** 先获取要恢复的 CRD 资源集合，调用 **processSelectedResource** 开始恢复
6. 接下来调用 **getOrderedResourceCollection** 获取剩余待恢复资源的有序集合，并调用 **processSelectedResource** 恢复<br>*内置的恢复顺序为：<br>1. customresourcedefinitions<br />2. namespaces<br />3. storageclasses<br />4. volumesnapshotclass.snapshot.storage.k8s.io<br />5. volumesnapshotcontents.snapshot.storage.k8s.io<br />6. volumesnapshots.snapshot.storage.k8s.io<br />7. persistentvolumes<br />8. persistentvolumeclaims<br />9. secrets<br />10. configmaps<br />11. serviceaccounts<br />12. limitranges<br />13. pods<br />14. replicasets.apps<br />15. clusters.cluster.x-k8s.io<br />16. clusterresourcesets.addons.cluster.x-k8s.io*
7. 元数据恢复已经完成，更新集群中 Restore 的进度信息
8. 等待 Restic 恢复所有的 Pod 卷数据<br>*PodVolumeRestore 的创建位于步骤 6 中*
9. 等待 post restore exec hook 执行完毕

## getOrderedResourceCollection

[getOrderedResourceCollection 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restore/restore.go#L1642)

构建要恢复的资源对象的有序集合

1. 构建需要恢复的资源列表，列表中元素为资源的名称，比如 customresourcedefinitions 等
2. 针对每一个资源名称，构建 GroupResource，跳过已经处理的、被 included 和 excluded 排除在外的、类型为 namespace 的 GroupResource
3. 针对该 GroupResource 的每一组资源实例（这组资源实例是以 namespace 做的划分），如果该资源实例的命名空间被排除，则忽略；否则，根据恢复时的命名空间映射关系，获取最终要创建恢复资源的命名空间
4. 如果最终要恢复到的命名空间为空，代表该资源是集群级别的，并且在未指定包含全部命名空间或者指定了特定恢复的命名空间的情况，则忽略
5. 针对每一组资源实例中的具体资源，根据 Backup 的 Content 目录读取该资源文件信息，反序列化成对象结构，最终针对该资源组实例，构建出一个可恢复、待恢复的资源集合

## processSelectedResource

[processSelectedResource 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restore/restore.go#L582)

单个 item 资源的恢复流程

1. 针对每一个要恢复的 item 资源，在要恢复到的命名空间不存在的情况下，提前创建
2. 根据 Backup 的 Content 目录读取该资源文件信息，反序列化成对象结构，调用 **restoreItem** 进行恢复
3. 每恢复完一个对象，则更新一下 update 队列，上报恢复进度

## restoreItem

[restoreItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restore/restore.go#L907)

单个 item 资源的恢复流程

1. 传入的 item 是解构状态的资源信息（runtime.Unstructured）
2. 如果最终要恢复到的命名空间不为空，需要根据实际情况提前创建；如果为空，代表该资源是集群级别的，在不恢复集群级别资源时，会忽略掉该资源
3. 以下资源视为“完成”，不会进行恢复
   - 相位为 Succeed 或者 Failed 的 Pod
   - 已经有完成时间的 Job
   - 已经恢复的资源
   - mirror Pod
4. 如果恢复的资源是 PV 类型，需要考虑以下场景
   - 如果要恢复的 PV 中有快照信息，代表着备份时 SnapshotProvider 执行过快照操作
     先判断是否需要对其重命名，以下情况不需要重命名
     - 恢复时未指定命名空间映射关系
     - 要恢复的 PV 未被认领
     - 认领该 PV 的命名空间不需要被映射
     - 当前集群中不存在重名 PV
     如果恢复时，指定了命名空间映射关系，需要根据映射关系，将认领 PV 的命名空间（即 spec.ClaimRef.Namespace）进行更新，如果判断后不需要对 PV 重命名时，则说明该 PV 应该被恢复，反之需要判断该 PV 对象是否应该被恢复，具体规则如下
     - 当前集群中不存在重名 PV，则应该被恢复
     - 重名 PV 处于 Release 状态，等待直至超时
     - 重名 PV 没有被认领，则不应该被恢复
     - 重名 PV 被认领，但是没找到相关 PVC，则等待 PVC 创建直至超时
     - 认领重名 PV 的 PVC 处于删除状态，等待直至超时
     - 认领重名 PV 的 PVC 所在命名空间不存在或者处于删除状态，等待直至超时
     - *默认超时时间为 10 分钟，未超时期间内会周期性判断，一旦超时则表示不应该被恢复*
     如果 PV 应该被恢复，则重置之前的绑定信息，调用 SnapshotProvider 的一系列接口恢复卷数据信息，参考 executePVAction，并根据需要重命名 PV
   - 如果要恢复的 PV 的卷数据是由 Restic 负责备份的（即 PodVolumeBackup annotation 信息中记录了 pv.Spec.ClaimRef.Name），则不会恢复，而是交给 StorageClass 重新动态供应
   - 如果要恢复的 PV 的回收策略为 Delete，则不会恢复，而是交给 StorageClass 重新动态供应
   - 如果并非以上任意场景，则不需要额外的特殊操作，重置绑定信息，进行后续流程直接恢复即可
5. 删除掉不关键的元信息，仅保留 name、namespace、labels 和 annotation，并删除对象 status 信息，执行 RestoreItemAction 的动作
6. 如果恢复的资源是 PVC 类型，则重置之前的和 PV 的绑定信息以及 K8s 设置的 annotation，如果认领的 PV 重命名了，则同步更新 PVC 信息
7. 根据命名空间映射关系，设置 item 的新命名空间，并设置 velero.io/backup-name 和 velero.io/restore-name 标签信息
8. 在当前集群中创建 item 资源，如果 item 资源已经存在（但是却又不深度一致）并且是 ServiceAccount 类型，则会以集群当前的 ServiceAccount 资源为准合并 item 资源并更新，而对于其他类型的资源则认为恢复失败，原因是备份的版本和恢复的版本不一致；而如果已存在的 item 和待恢复的 item 深度一致，也就是两者版本一致，不会恢复，也不会视为错误
9. 如果恢复的资源是 Pod 类型，会针对每一个被 Restic 备份的卷，创建一个 PodVolumeRestore 对象，后续由 PodVolumeRestore Controller 负责维护，同时，主进程会阻塞直到 PodVolumeRestore 有状态返回，表示卷数据恢复完成
10. 如果恢复的资源是 Pod 类型，执行 post restore hook 操作
11. 如果恢复的资源是 CRD 类型，则等待直至 CRD 资源变得可用才继续执行后续恢复流程

# PodVolumeRestore Controller

*<u>pkg/controller/pod_volume_restore_controller.go</u>*<br>*<u>pkg/restic/exec_commands.go</u>*

## NewPodVolumeRestoreController

[NewPodVolumeRestoreController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_restore_controller.go#L72)

工厂函数

1. 注册 Generic Controller 中的 syncHandler，并将 PodVolumeRestore、Pod 和 PVC 添加到 cacheSyncWaiters，等待同步完成
2. 监听 PodVolumeRestore 资源的 Add 和 Update 事件，根据状态是 New 的 PodVolumeRestore 资源获取到相关联的 Pod，如果 Pod 运行在本节点，并且 Restic Init Container 处于 Running，则将 PodVolumeRestore 以 key（namespace/name） 的形式加入 Generic Controller 的 queue 中<br>*restic-wait 处于运行状态表示，该 Pod 正在等待 Restic 为其恢复卷数据*
3. 监听 Pod 资源的 Add 和 Update 事件，针对运行在本节点，并且 Restic Init Container 处于 Running，获取到相关联的 PodVolumeRestore 资源，将状态是 New 的 PodVolumeRestore 以 key（namespace/name） 的形式加入 Generic Controller 的 queue 中

## processQueueItem

[processQueueItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_restore_controller.go#L240)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 PodVolumeRestore key，通过解析获取的 namespace 和 name 查询到集群中的 PodVolumeRestore 对象
2. 调用 **processRestore**，执行卷数据的备份

## processRestore

[processRestore 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_restore_controller.go#L277)

卷数据恢复的整体流程

1. 更新 PodVolumeRestore 状态为 InProgress
2. 获取到 PodVolumeRestore 相关联的 Pod，进一步获取到 Pod 内部的挂载卷目录信息
3. 调用 **restorePodVolume** 执行卷数据的恢复，如果调用结果有错误返回，则设置 PodVolumeRestore 状态为 Failed
4. 更新 PodVolumeRestore 状态为 Completed

## restorePodVolume

[restorePodVolume 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_restore_controller.go#L326)

借助 Restic 能力恢复卷数据的流程

1. 根据 Pod 内部的挂载卷信息和 Restic Pod 内部的 /host_pods 目录，拼接出到 Pod 挂载数据卷目录<br>*最终数据会从 Restic Repo 将卷数据恢复至这个目录，例如 /host_pods/new-pod-uid/volumes/volume-plugin-name/volume-dir*
2. 生成用于连接 Restic Repo 所需要的临时密码文件，文件固定为 /tmp/credentials/velero/velero-restic-credentials-repository-password，内容为 static-passw0rd，用于 restic 原生命令中的 --password 参数<br>*密码会以 Secret 的形式存储在集群中，名为 velero-restic-credentials，位于 velero 命名空间内*
3. 构建 restic restore 命令
4. 如果 BackupStorageLocation 有 caCert 证书信息，会将其临时写入到磁盘中，供 Restic 认证使用，并设置在 restic restore 命令中
5. 给 restic restore 命令设置 Restic 原生所需的环境变量信息
6. 调用 **RunRestore**，执行 Restic 原生的卷数据恢复流程
7. 为了保险起见，先移除卷目录下的 .velero 目录，然后重新创建，并写入 Restore 的 UID，Restic 的 Init Container 会一致等待直到读取到这个文件的生成

## RunRestore

[RunRestore 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restic/exec_commands.go#L186)

调用 Restic 原生备份命令 restic restore

1. 通过 restic stats 命令获取到总快照的大小，并更新至 PodVolumeRestore 中
2. 将入参的 Command 对象构建成可执行命令并执行
3. 启动一个 Goroutine，每 10 秒钟获取一次已经恢复的数据目录总大小，并更新 PodVolumeRestore 总恢复进度（待恢复总文件大小和当前恢复文件大小）
4. 恢复完成后，更新 PodVolumeRestore 进度至 100%<br>*此处未判断恢复成功与否，只是将进度设为 100%，并返回 restic restore 的 stdout，stderr 和 err 信息，由上层调用方判断*
