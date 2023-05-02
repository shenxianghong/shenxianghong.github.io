---
title: "「 Velero 」 2.2 源码走读 — Backup"
excerpt: "Velero 中与 Backup 相关的源码走读"
cover: https://picsum.photos/0?sig=20220127
thumbnail: https://blogs.vmware.com/opensource/files/2022/03/velero.png
date: 2022-01-27
toc: true
categories:
- Code Walkthrough
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://velero.io/img/Velero.svg"></div>

------

> Based on **v1.6.3**

# Backup

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/backup.go)

## backup

*<u>pkg/cmd/cli/backup/backup.go</u>*

velero backup 包括 6 个子命令：create、delete、describe、download、get 和 logs。

## create

*<u>pkg/cmd/cli/backup/create.go</u>*

**校验规则**

- 必须指定 Backup 名称，除非指定了 --from-schedule 参数
- 在指定 --storage-location 参数时，其在集群中必须存在
- 在指定 --volume-snapshot-locations 参数时，其在集群中必须存在

**主体流程**

1. 根据命令行参数，构建 Backup 对象，下发至集群中创建，后续由 Backup Controller 负责维护
2. 如果开启了 --wait，则启动 informer 监听 Backup 对象状态，阻塞直至状态不再是 New 或者 InProgress

*如果 Backup 是基于定时任务（Schedule）创建的，则忽略其他所有的 filter 信息，以 Schedule 规格为准，除此之外，Backup 的名称也可以不指定，默认格式为 schedule-timestamp。*

## delete

*<u>pkg/cmd/cli/backup/delete.go</u>*

**校验规则**

- name、--all 和 --selector 参数有且仅能有一个
- 要删除的 Backup 在集群中必须存在

**主体流程**

1. 根据命令行参数，构建 DeleteBackupRequest 对象，下发至集群中创建，后续由 BackupDeletion Controller 负责维护

## describe

*<u>pkg/cmd/cli/backup/describe.go</u>*

**校验规则**

- 要获取的 Backup 在集群中必须存在

**主体流程**

1. 获取到 Backup 的删除事件
2. 获取到卷备份的信息和 CSI 快照信息（如果 Velero 服务开启了 CSI 特性）
3. 将以上信息和 Backup 的元信息、规格、状态等汇总作为描述信息格式化输出<br>*在开启 --details 时，会构建 DownloadRequest 对象获取  BackupResourceList 的信息*

## download

*<u>pkg/cmd/cli/backup/download.go</u>*

**校验规则**

- 要获取的 Backup 在集群中必须存在

**主体流程**

1. 根据命令行参数，构建 DownloadRequest 对象，下发至集群中创建，后续由 DownloadRequest Controller 负责维护，获取 BackupContents 的信息 
2. 阻塞直至 DownloadRequest 的 DownloadURL 被设置，将内容写入 --output 指定的位置

## get

*<u>pkg/cmd/cli/backup/get.go</u>*

**校验规则**

- 要获取的 Backup 在集群中必须存在

**主体流程**

1. 获取到 Backup 资源，根据 --output 指定的样式格式化输出

## logs

*<u>pkg/cmd/cli/backup/logs.go</u>*

**校验规则**

- 要获取的 Backup 在集群中必须存在
- Backup 的状态必须为 Completed、PartiallyFailed 或者 Failed

**主体流程**

1. 根据命令行参数，构建 DownloadRequest 对象，下发至集群中创建，后续由 DownloadRequest Controller 负责维护，获取 BackupLog 的信息 
2. 阻塞直至 DownloadRequest 的 DownloadURL 被设置，将内容写入 stdout 中

# PodVolumeBackup

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/pod_volume_backup.go)

对象不支持手动创建，而是在备份流程中，由 Backup Controller 调用 **backupPodVolumes**，针对每一个 Pod 卷，创建一个该对象。

# Schedule

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/schedule.go)

## schedule

*<u>pkg/cmd/cli/schedule/schedule.go</u>*

velero schedule 包括 4 个子命令：create、delete、describe 和 get。

## create

*<u>pkg/cmd/cli/schedule/create.go</u>*

**校验规则**

- --schedule 为必需参数
- 在指定 --storage-location 参数时，其在集群中必须存在
- 在指定 --volume-snapshot-locations 参数时，其在集群中必须存在

*创建定时备份任务时并不会校验 schedule 表达式的合法性，而是交给 Schedule Controller 作后续处理。*

**主体流程**

1. 根据命令行参数，构建 Schedule 对象，下发至集群中创建，后续由 Scheduler Controller 负责维护

## delete

*<u>pkg/cmd/cli/schedule/delete.go</u>*

**校验规则**

- name、--all 和 --selector 参数有且仅能有一个
- 要删除的 Schedule 在集群中必须存在

**主体流程**

1. 根据命令行参数，获取集群中的 Schedule 资源并删除

## describe

*<u>pkg/cmd/cli/schedule/describe.go</u>*

**校验规则**

- 要获取的 Schedule 在集群中必须存在

**主体流程**

1. 获取到 Schedule 资源，格式化输出

## get

*<u>pkg/cmd/cli/schedule/get.go</u>*

**校验规则**

- 要获取的 Schedule 在集群中必须存在

**主体流程**

1. 获取到 Schedule 资源，根据 --output 指定的样式格式化输出

# Backup Controller

*<u>pkg/controller/backup_controller.go</u>*<br>*<u>pkg/backup/backup.go</u>*

## NewBackupController

[NewBackupController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_controller.go#L89)

工厂函数

1. 注册 Generic Controller 中的 syncHandler 和 resyncFunc
2. 监听 Backup 资源的 Add 事件，将状态是空或者 New 的 Backup 以 key（namespace/name）的形式加入 Generic Controller 的 queue 中

## processBackup

[processBackup 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_controller.go#L207)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 Backup key，通过解析获取的 namespace 和 name 查询到集群中的 Backup 对象
2. 仅处理状态是空或者 New 的 Backup 对象
3. 调用 **prepareBackupRequest** 做一些校验准备工作，并根据校验的结果设置 Backup 的状态为 FailedValidation 或者 InProgress
4. 过滤掉校验失败的 Backup，调用 **runBackup**，执行备份和上传备份信息的流程，执行结果决定了备份是否顺利完成，如果有错误返回，则记录 Backup 状态为 Failed

## prepareBackupRequest

[prepareBackupRequest 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_controller.go#L328)

在不破坏集群中原 Backup 对象（以下称为 original）的情况下，构建了一个 BackupRequest 对象（以下称为 request），这个对象包含了 original 的详细规格（即 original 的深拷贝），并且包含一些丰富处理流程的中间态信息，并且在整个备份流程中的操作都是基于 request，在备份完成后，会将 request 信息同步更新至集群中的 Backup 对象。

```go
// Request is a request for a backup, with all references to other objects
// materialized (e.g. backup/snapshot locations, includes/excludes, etc.)
type Request struct {
	*velerov1api.Backup

	StorageLocation           *velerov1api.BackupStorageLocation
	SnapshotLocations         []*velerov1api.VolumeSnapshotLocation
	NamespaceIncludesExcludes *collections.IncludesExcludes
	ResourceIncludesExcludes  *collections.IncludesExcludes
	ResourceHooks             []hook.ResourceHook
	ResolvedActions           []resolvedAction

	VolumeSnapshots  []*volume.Snapshot
	PodVolumeBackups []*velerov1api.PodVolumeBackup
	BackedUpItems    map[itemKey]struct{}
}
```

对 request 赋值和校验的操作

1. 将 original 深拷贝至 request 中的 Backup 中，并设置 request  版本、过期时间、是否将卷数据备份至 Restic、StorageLocation 等信息
3. 校验 BackupStorageLocation 的合法性，以及 access mode 是否为预期的 ReadWrite
4. 检验 volume snapshot location 的合法性<br>*backup.Spec.VolumeSnapshotLocation 为 []string 类型，支持多个 location，但是要求 location 和 VolumeSnapshotter 必须是一对一的关系（也就是说不允许多个 location 对应同一个 VolumeSnapshotter）。默认情况下，--snapshotvolume 为 true，所以只要存在一个合法的 default vsl，则最终的 backup.Spec.VolumeSnapshotLocation 均会包含这个 default vsl，详细逻辑参考 [TestValidateAndGetSnapshotLocations](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_controller_test.go#L861)*
5. 设置 request 的注解和 resources & namespaces 的 included & excluded 检验信息

## runBackup

[runBackup 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_controller.go#L533)

备份的整体流程

1. 基于临时文件句柄生成 gzip writer ，并指定 stdout 和 gzip writer 为 logger 输出，初始化用于统计日志级别数量的 counter<br>*因此，日志不仅会输出在 Velero Pod 中，并且会生成 BackupLog，后续会上传至 BackupStorageLocation 中*
2. 生成一个用于存放 Backup 版本和内容的临时文件<br>*后续在调用 **Backup** 时会传入该临时文件*
3. 获取注册的 BackupItemAction 插件<br>*后续在调用 **Backup** 时会传入该 action 信息*
4. 通过 StorageProvider 的 BackupExists 接口判断远端存储中是否有同名备份
   - 如果存在，则设置 Backup 状态为 Failed，本次备份失败
   - 如果不存在，则调用 **Backup** 进行准备与备份
5. 如果 Velero 开启了 CSI 特性，则获取集群中与该 Backup 相关的 VolumeSnapshots 和 VolumeSnapshotContents 信息<br>*后续会上传至 BackupStorageLocation 中*
6. 设置 request 的卷快照总量与成功量、完成时间戳、Warnings 和 Errors 的个数以及备份的状态等等，关闭日志文件，本次备份任务日志记录完毕<br>*备份的状态（Failed、PartiallyFailed 或者 Completed）是根据日志输出级别的统计决定的，fatalErrs 记录了调用 backupper.Backup 产生的错误日志信息，不仅如此，在后续上传备份文件的时候，如果发生异常，也会记录，所以 request 的状态是否为 Failed 是 fatalErrs 决定的，而状态是否为 Completed 和 PartiallyFailed 是根据日志中 Error 级别的输出数量决定的，**该日志也包括调用 StorageProvider 中产生的日志级别信息***
   
   ```go
   backup.Status.Warnings = logCounter.GetCount(logrus.WarnLevel)
   backup.Status.Errors = logCounter.GetCount(logrus.ErrorLevel)
   
   // Assign finalize phase as close to end as possible so that any errors
   // logged to backupLog are captured. This is done before uploading the
   // artifacts to object storage so that the JSON representation of the
   // backup in object storage has the terminal phase set.
   switch {
   case len(fatalErrs) > 0:
   	backup.Status.Phase = velerov1api.BackupPhaseFailed
   case logCounter.GetCount(logrus.ErrorLevel) > 0:
   	backup.Status.Phase = velerov1api.BackupPhasePartiallyFailed
   default:
   	backup.Status.Phase = velerov1api.BackupPhaseCompleted
   }
   ```
   
7. 重新获取 StorageProvider，避免长时间备份中认证信息变动， 调用 StorageProvider 的 PutBackup 接口将备份信息上传至 BackupStorageLocation 中，具体文件的对应关系如下：

   | 名称                      | BackupStorageLocation 中的文件                | 数据源                             |
   | ------------------------- | --------------------------------------------- | ---------------------------------- |
   | Metadata                  | velero-backup.json                            | backup.Backup 对象                 |
   | Content                   | \<backup\>.tar.gz                             | 步骤 2 中的临时文件内容            |
   | Log                       | \<backup\>-logs.gz                            | 步骤 6 中最终生成的 log 文件       |
   | PodVolumeBackups          | \<backup\>-podvolumebackups.json              | backup.PodVolumeBackups            |
   | VolumeSnapshots           | \<backup\>-volumesnapshots.json.gz            | backup.VolumeSnapshots             |
   | BackupResourceList        | \<backup\>-resource-list.json.gz              | backup.BackedUpItems               |
   | CSIVolumeSnapshots        | \<backup\>-csi-volumesnapshots.json.gz        | 步骤 5 中 volume snapshots         |
   | CSIVolumeSnapshotContents | \<backup\>-csi-volumesnapshotcontents.json.gz | 步骤 5 中 volume snapshot contents |

   这里需要区分 PodVolumeBackups 和 VolumeSnapshots
   - PodVolumeBackups 是描述了备份的 Pod 中的卷数据信息，与之关联的是 Restic 相关概念，数据最终会写入 ResticRepository 中
   - VolumeSnapshots 是相比于 CSI 而言属于 Velero 原生的卷快照，用于描述一个 PV 快照的信息，本身作为 Backup 的一部分，数据最终会由对应的 SnapshotProvider 处理
   - 即使两者最终的数据均不会存放在 BackupStorageLocation 中，但是仍然会在 BackupStorageLocation 中记录其基础信息

## Backup

[Backup 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/backup/backup.go#L205)

备份动作本身的流程

1. 基于传入的临时文件，生成 gzip writer，最终会以 *backup*.tar.gz 格式存放在 BackupStorageLocation 中作为 Content，该 tar.gz 包括两个主要内容：metadata 和 resources，前者用于存放版本，后者用于存放被备份资源的详细规格信息<br>*区别于记录 Backup 本身的 Metadata，此处的 metdata 仅为一个目录层级；在后续 **backupItem** 流程中，会将资源信息写入 resources 文件中*
2. 在 metadata 目录下写入版本信息 version，固定值为 1.1.0
3. 设置 request 的 resource & namespace 的 included & excluded、resources hook 以及 resolve action（BackupItemAction）
   - namespace 和 resource 的处理方式并不一样，是因为 namespace 只要做匹配即可，存在与不存在很容易定性，但是 resource 有很多种表示方式，例如 pv 和 persistentvolumes，对于复杂的资源来讲，还有 ApiGroup 的概念，因此，需要 RESTmapping 的 ResourceFor 匹配出最合适且规范的一个 GVR，这也就是为什么使用时可以在合理范围任意指定资源名称均会匹配正确的原因
   - BackupItemAction 有两个接口，一个是 AppliesTo，一个是 Execute，前者用于返回 labelSelector，用于备份阶段的过滤筛选，后者用于执行 BackupItemAction 中定义的额外动作
4. 生成临时空文件，供后续 itemCollector 使用，itemCollector 根据 Backup 规格信息通过 K8s API 收集待备份的资源详细信息并写入空文件中<br>*该空文件用于 **getAllItems** 时，作为每一个 item 的解构文件的根目录*
5. [getAllItems](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/backup/item_collector.go#L57) 通过 discovery 获取到所有的 API group（例如 batch/v1beta1，networking.k8s.io/v1beta1 等等），然后根据每个 group 获取到 resource（例如 cronjobs，networkpolicies 等等），根据 namespace 和 resource 的 include 和 exclude 以及标签选择器规则进行过滤资源，过滤后的结果就是待备份的对象（item）<br>*item 的内容（unstructured.Unstructured 对象）已经写入了步骤 4 的临时目录里，同时，这个文件的路径会记录在 item 的 path 中，后续在 **backupItem** 时，会解析作为解构状态的 item 传入*
5. 更新 Backup 的待备份资源的总量信息
7. 生成 update 队列，用于记录 Backup 状态信息，同时启动一个 Goroutine 监听队列，每秒钟获取一次 update 队列中的进度并更新至 Backup 中
8. 遍历每一个待备份的 item，调用 **backupItem** 函数，进行备份，并将进度信息写入 update 队列中，完成进度上报
9. 如果备份规格中指定了备份集群级别资源（IncludeClusterResources），则额外备份 CRD 资源<br>*这里的 CRD 资源其实是有限制条件的，就是仅处理已经备份了与 CRD 相关的 CR 资源时，才会备份对应的 CRD*
10. 更新集群中 Backup 对象的备份进度信息至 100%

## backupItem

[backupItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/backup/backup.go#L424)

单个 item 资源的备份流程

1. 传入的 item 是解构状态的资源信息（runtime.Unstructured），解构数据来源于 item 的 path 字段，需要重新构建成 metav1.Object，提取信息
2. 对于以下情况跳过备份，并且返回 false 标识，表示资源未参与备份<br>*未参与备份的资源，即使在指定了备份集群级别资源时也不会备份相关的 CRD*
   - 标签中有 velero.io/exclude-from-backup 字段的资源
   - 资源属于 Backup 规格中指定的 namespace 和 groupResource 的
   - 非 namespace 的资源即是集群级别的，但是备份规格中的 IncludeClusterResources 为 false
   - 资源处于删除状态
3. 对于已经备份的资源，自然也会跳过备份，但是返回 true 标识，表示资源已经备份
4. 执行 pre hook 动作（只有类型是 Pod 的 资源才会真正的处理 pre hook），hook 作用的对象需要满足 BackupItemAction 中定义的 applyTo 接口的标签选择，hook command 的执行（即 execute 接口）是通过 K8s rest API exec 实现
5. 针对类型是 Pod 的资源，在 spec.Volumes 中获取需要借助 Restic 备份的卷<br>*由于 PV 和 PVC 是 1 对 1，PVC 和 Pod 是 1 对 n 的关系，所以在这里通过跟踪 PVC 的备份情况即可判断卷是否被备份过，备份过的卷，会在相关的 tracker 中以 PVC 为 key 作记录*
6. 调用 BackupItemAction 的 execute 接口，进行额外的操作，如更新资源等，此后的流程基于接口返回的对象继续操作，执行失败时，会执行 post hook 动作
7. 针对类型是 PV 的资源，调用 takePVSnapshot，忽略已经被 Restic 备份的 PV，初始化 SnapshotProvider，通过一系列的接口，完成卷快照的操作，并将快照信息记录在 request 的 VolumeSnapshots 中<br>*在 Velero 开启 CSI 特性时，需要额外加载一个 plugin（https://github.com/vmware-tanzu/velero-plugin-for-csi），该 plugin 就是 SnapshotProvider 类型。因此在这步时，便会对 PV 做快照操作*
8. 针对类型是 Pod 的资源，调用 **BackupPodVolumes**，借助 Restic 能力实现对 Pod 卷数据的备份
9. 执行 post hook 动作
10. 在 resources 目录下写入备份的资源信息，资源会根据 kind、 namepace 等信息归类，文件内容为 item 的 runtime.Unstructured 形式，以 json 格式存储<br>*文件是上层调用传入的 Content 文件*
12. 至此，针对单一的 item 备份流程结束，其中包含了存储在 kube-apiserver 中的结构化元信息和卷数据的备份

## BackupPodVolumes

[backupPodVolumes 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restic/backupper.go#L99)

借助 Restic 能力备份卷数据的流程

1. 每一个 Pod 卷所属的 namespace 必须有且仅有一个对应的 ResticRepository 对象，如果不存在，则会创建一个，由 ResticRepository Controller 会负责维护状态，而 Backup Controller  会阻塞直至 ResticRepository 超时或者 ready<br>*创建对象时会指定基于的 BackupStorageLocation，并将 BackupStorageLocation 转换成 RepoIdentifier 信息，也就是 Restic 原生命令中的 -r 参数*
2. 过滤掉 hostPath 类型的卷，对其余的合法卷会创建对应的 PodVolumeBackup 对象，其中 PodVolumeBackup 中会设置 velero.io/backup-name 标签以及 PVC 的 UID 信息<br>*此时，该对象的 spec.RepoIdentifier 已被设置，例如 s3:http://minio.velero.svc:9000/velero/restic/velero。此外，由于在 velero 1.6.3 中不会判断 Pod 的状态，因此依旧会对 pending 状态的 Pod 创建 PodVolumeBackup 对象，但是此 PodVolumeBackup 对象没有 nodeName 属性，导致 Restic 不作处理，从而阻塞 Velero 直至超时。除此之外，restic 状态异常也会导致类似问题，参考：https://github.com/vmware-tanzu/velero/issues/4874*
3. PodVolumeBackup Controller 会负责卷的备份，而 Backup Controller 会阻塞直至卷备份返回完成或者失败

## resync

[resync 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_controller.go#L166)

注册在 Generic Controller 中 resyncFunc 的实现，周期为 1 分钟

1. 获取集群中所有的 Backup 对象，更新 backup_total 指标，value 为集群中所有 Backup 总数
2. 针对每一个状态已经完成且归属于某一个 Schedule 的 Backup，设置 backup_last_successful_timestamp 指标，key 为 Schedule 名称，value 为最近的一次备份时间戳

# PodVolumeBackup Controller

*<u>pkg/controller/pod_volume_backup_controller.go</u>*<br>*<u>pkg/restic/exec_commands.go</u>*

## NewPodVolumeBackupController

[NewPodVolumeBackupController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_backup_controller.go#L72)

工厂函数

1. 注册 Generic Controller 中的 syncHandler，并将 PodVolumeBackup、Pod 和 PVC 添加到 cacheSyncWaiters，等待同步完成
2. 监听 PodVolumeBackup 资源的 Add 和 Update 事件，将状态是空或者 New 并且位于当前节点的 PodVolumeBackup 资源以 key （namespace/name） 的形式加入 Generic Controller 的 queue 中<br>*PodVolumeBackup Controller 运行在 DaemonSet 形式的 Restic 服务中，挂载所在节点的 Pod 卷，因此 PodVolumeBackup 具有节点属性，PodVolumeBackup Controller 仅处理当前节点的 PodVolumeBackup 对象*

## processQueueItem

[processQueueItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_backup_controller.go#L140)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 PodVolumeBackup key，通过解析获取的 namespace 和 name 查询到集群中的 PodVolumeBackup 对象
2. 仅处理状态为空或者 New 的 PodVolumeBackup 对象，调用 **processBackup**，执行卷数据的备份

## ProcessBackup

[ProcessBackup 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_backup_controller.go#L188)

卷数据备份的整体流程

1. 更新 PodVolumeBackup 状态为 InProgress
2. 校验 PodVolumeBackup 对象规格中声明的卷源 Pod 是否存在，获取到 Pod 卷在 host 上子目录信息<br>*例如  /host_pods/e4ccf918-76d7-4972-a54a-39b39f15b53b/volumes/kubernetes.io~empty-dir/plugins，/host_pods 为 Restic Pod 中的目录，挂载了 host 的 /var/lib/kubelet/pods/， 详细逻辑参考 [TestGetVolumeDirectorySuccess](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/util/kube/utils_test.go#L159)*
2. 生成用于连接 Restic Repo 所需要的临时密码文件，文件固定为 /tmp/credentials/velero/velero-restic-credentials-repository-password，内容为 static-passw0rd，用于 restic 原生命令中的 --password 参数<br>*密码会以 Secret 的形式存储在集群中，名为 velero-restic-credentials，位于 velero 命名空间内*
3. 构建 restic backup 命令
4. 如果 BackupStorageLocation 有 caCert 证书信息，会将其临时写入到磁盘中，供 Restic 认证使用，并设置在 restic backup 命令中<br>*因为本质上，Restic 的 repo 和 Velero 的 BackupStorageLocation 为同一个*
5. 给 restic backup 命令设置 Restic 原生所需的环境变量信息
6. 在备份流程中生成 PodVolumeBackup 对象时，如果 Pod 卷源于 PVC，则会对 PodVolumeBackup 加一个 velero.io/pvc-uid 的 label，值为 PVC 的 uid。因此，在这里会通过这个 label 判断卷是否源于 PVC，如果是，则会获取集群中所有带有此 label、状态已经完成并且 BackupStorageLocation 相同的 PodVolumeBackup 对象，如果存在则表示这个卷已经备份过，后续的备份会基于最近的一次备份点进行增量备份，反之则全量备份，这里的增量备份借助了 Restic 原生功能（--parent）<br>*详细逻辑参考 [getParentSnapshot](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/pod_volume_backup_controller.go#L338)*
7. 调用 **RunBackup**，执行 Restic 原生的卷数据备份流程
8. 更新 PodVolumeBackup 对象的状态、完成时间、快照 ID 和卷数据路径等信息

## RunBackup

[RunBackup 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/restic/exec_commands.go#L73)

调用 Restic 原生备份命令 restic backup

1. 将入参的 Command 对象构建成可执行命令并执行
2. 启动一个 Goroutine，每 10 秒钟解析一次 restic backup 命令的标准输出，更新 PodVolumeBackup 的备份进度信息（待备份总文件大小和当前备份文件大小）
3. 通过解析标准输出判断如果 Restic 备份成功，则更新 PodVolumeBackup 进度至 100%

# Schedule Controller

*<u>pkg/controller/schedule_controller.go</u>*

## NewScheduleController

[NewScheduleController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/schedule_controller.go#L60)

工厂函数

1. 注册 Generic Controller 中的 syncHandler 和 resyncFunc
2. 监听 Schedule 资源的 Add 事件，将状态是空、New 或者 Enabled 的 Schedule 以 key（namespace/name）的形式加入 Generic Controller 的 queue 中

## processSchedule

[processSchedule 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/schedule_controller.go#L136)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 Schedule key，通过解析获取的 namespace 和 name 查询到集群中的 Schedule 对象
2. 仅处理状态为空、New 或者 Enabled 的 Schedule 对象
3. 检验 cron 表达式的合法性，根据校验结果更新 Schedule 的状态为 FailedValidation 或者 Enabled
4. 判断是否到达定时任务的下次执行时间，如果达到则立即创建一个备份。另外，如果定时任务从未执行，也会立即创建一个备份任务

## enqueueAllEnabledSchedules

[enqueueAllEnabledSchedules 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/schedule_controller.go#L115)

注册在 Generic Controller 中 resyncFunc 的实现，周期为 1 分钟

1. 获取集群中所有的 Schedule 对象
2. 将状态是 Enabled 的 Schedule 对象加入到 Generic Controller 的 queue 中

# BackupDeletion Controller

*<u>pkg/controller/backup_deletion_controller.go</u>*<br>*<u>internal/delete/delete_item_action_handler.go</u>*

删除 Backup 时，与 Backup 相关联的各种资源基本上都是通过 velero.io/backup-name 标签获取到。因此，在备份的时候，创建的相关资源也都会打上该标签。

## NewBackupDeletionController

[NewBackupDeletionController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_deletion_controller.go#L86)

工厂函数

1. 注册 Generic Controller 中的 syncHandler 和 resyncFunc
2. 监听 DeleteBackupRequest 资源的 Add 事件，将 DeleteBackupRequest 以 key（namespace/name）的形式加入 Generic Controller 的 queue 中

## processQueueItem

[processQueueItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_deletion_controller.go#L146)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 DeleteBackupRequest key，通过解析获取的 namespace 和 name 查询到集群中的 DeleteBackupRequest 对象
2. 仅处理状态不为 Processed 的 DeleteBackupRequest 对象，调用 **processRequest**，执行 Backup 的删除

## processRequest

[processRequest 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_deletion_controller.go#L176)

备份删除的整体流程

1. 如果 DeleteBackupRequest 对象所属的 Backup 信息不存在，则认为 DeleteBackupRequest 处理完成，即将其状态设为 Processed，并设置错误信息
2. 删除集群中针对该 Backup 的其余 DeleteBackupRequest 对象，仅处理当前的
3. 如果要删除的 Backup 处于 InProgress 状态或者不存在，则将 DeleteBackupRequest 状态设为 Processed，并设置错误信息
4. 如果 Backup 所属的 BackupStorageLocation 不存在或者模式为 ReadOnly 时，则将 DeleteBackupRequest 状态设为 Processed，并设置错误信息
5. 至此，校验工作已经完成，将 DeleteBackupRequest 状态设置为 InProgress，并设置 velero.io/backup-name 和 velero.io/backup-uid 标签
6. 设置 Backup 的状态为 Deleting
7. 获取注册的 DeleteItemAction 插件，如果获取到了，则下载 BackupStorageLocation 中的 Content 文件，构建运行 DeleteItemAction 所需要环境变量信息，调用 **InvokeDeleteActions** 处理 DeleteItemAction 中定义的逻辑
8. 调用 StorageProvider 的 GetBackupVolumeSnapshots 方法获取 BackupVolumeSnapshotsKey（也就是 backup-volumesnapshots.json.gz） 的内容，调用 SnapshotProvider 的 DeleteSnapshot 接口删除 PV 快照信息
9. 获取到与 Backup 相关联的 PodVolumeBackup 信息，进而获取到 Restic 的 snapshots，执行 restic forget 删除快照<br>*创建备份的时候，会根据 Pod 卷创建 PodVolumeBackup 对象，并会设置 velero.io/backup-name 标签*
10. 调用 StorageProvider 的 DeleteBackup 接口删除 BackupStorageLocation 中 Backup 所在的目录
11. 如果 Velero 开启了 CSI 特性，那么也会删除与 Backup 相关联的 VolumeSnapshot 和 VolumeSnapshotContent 对象<br>*删除之前会将 VolumeSnapshotContent 回收策略置为 Delete*
12. 调用 StorageProvider 的 DeleteRestore 方法，删除 BackupStorageLocation 上基于该 Backup 创建的 Restore 文件，删除基于该 Backup 创建的 Restore 对象
13. 如果以上步骤均无错误返回，则删除集群中相关的 Backup 对象
14. 更新 DeleteBackupRequest 状态设为 Processed，并设置错误信息
15. 如果以上步骤均无报错返回，则删除与该 Backup 相关的所有 DeleteBackupRequest 对象

## InvokeDeleteActions

[InvokeDeleteActions 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/internal/delete/delete_item_action_handler.go#L48)

执行 DeleteItemAction 中的动作

1. 如果未定义 action 信息，则直接返回，继续处理删除流程
2. 将传入的 BackupStorageLocation 中的 Content 文件解压至临时文件下，通过 discovery API 将文件内容转换成 GroupResource
3. 遍历 GroupResource 中的所有资源，执行 DeleteItemAction 中声明的动作

## deleteExpiredRequests

[deleteExpiredRequests 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_deletion_controller.go#L595)

注册在 Generic Controller 中 resyncFunc 的实现，周期为 1 小时

1. 获取集群中所有的 DeleteBackupRequest 对象，将状态已经处于 Processed，并且 age 超过 24 小时的 DeleteBackupRequest 对象删除<br>*因为并非所有的 DeleteBackupRequest 对象在走完 syncHandler 流程后均会被删除*

# BackupSync Controller

*<u>pkg/controller/backup_sync_controller.go</u>*

## NewBackupSyncController

[NewBackupSyncController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_sync_controller.go#L60)

工厂函数

1. 注册 Generic Controller 中的 resyncFunc

## run

[run 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/backup_sync_controller.go#L124)

注册在 Generic Controller 中 resyncFunc 的实现，周期为 30 秒

1. 获取集群中所有的 BackupStorageLocation 对象，构建一个默认的 BackupStorageLocation 位于第一位的列表
2. 遍历步骤 1 中的 BackupStorageLocation 列表，如果同步周期等于 0 代表该 BackupStorageLocation 不作同步操作，直接跳过即可，否则判断其是否到达下次同步时间
3. 调用 StorageProvider 的 ListBackups 方法获取所有的 Backup；同时获取集群中所有的 Backup
5. 获取在 BackupStorageLocation 中但是不在集群中的 Backup，这些就是待同步的 Backup
6. 针对每一个待同步的 Backup，调用 StorageProvider 的 GetBackupMetadata 方法，获取 Metadata 文件（即 velero-backup.json），解析内容得到 Backup 对象
7. 设置 Backup 的 namespace、resourceVersion、storageLocation 和 label 等信息，并在集群中创建<br>*既然已经存在于 BackupStorage Location，状态必然为完成状态（即不是空或者 New），因此不会被 Backup Controller 重复处理*
8. 调用 StorageProvider 的 GetPodVolumeBackups 方法获取与该 Backup 相关的 PodVolumeBackup
9. 设置 PodVolumeBackup 的 namespace、resourceVersion、ownerReferences 和 label 等信息，并在集群中创建<br>*同理，不会被 PodVolumeBackup Controller 重复处理*
10. 如果 Velero 开启了 CSI 特性，通过 StorageProvider 的 GetCSIVolumeSnapshotContents 方法获取与该 Backup 相关的 VolumeSnapshotContents
11. 设置 VolumeSnapshotContent 的 resourceVersion 等信息，并在集群中创建
12. 删除孤儿 Backup，也就是在集群中存在，状态为 Completed，但是在 BackupStorageLocation 中不存在的 Backup
13. 更新集群中该 BackupStorageLocation 的上次同步时间<br>*实际上步骤 4 获取 BackupStorageLocation 的 Backup 时可以作为同步操作*

# GC Controller

*<u>pkg/controller/gc_controller.go</u>*

## NewGCController

[NewGCController 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/gc_controller.go#L58)

工厂函数

1. 注册 Generic Controller 中的 syncHandler 和 resyncFunc
2. 监听 Backup 资源的 Add 和 Update 事件，将 Backup 以 key（namespace/name）的形式加入 Generic Controller 的 queue 中

## processQueueItem

[processQueueItem 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/gc_controller.go#L104)

注册在 Generic Controller 中 syncHandler 的实现

1. 函数入参就是 Generic Controller 的 queue 中待处理的 Backup key，通过解析获取的 namespace 和 name 查询到集群中的 Backup 对象
2. 仅处理已经过期的 Backup 对象
3. 获取 Backup 所属的 BackupStorageLocation，判断其模式是否为 ReadWrite
4. 获取集群中和该 Backup 相关的 DeleteBackupRequest 对象，如果其中存在状态为空、New 和 InProgress 的，则认为正在删除，本次不做处理；否则，构建一个 DeleteBackupRequest 对象，下发至集群中创建，后续由 BackupDeletion Controller 负责维护 

## enqueueAllBackups

[enqueueAllBackups 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/gc_controller.go#L90)

注册在 Generic Controller 中 resyncFunc 的实现，周期为 1 小时

1. 获取集群中所有的 Backup 对象，全量加入到 Generic Controller 的 queue 中
