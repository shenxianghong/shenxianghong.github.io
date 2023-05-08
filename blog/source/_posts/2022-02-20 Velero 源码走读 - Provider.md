---
title: "「 Velero 」源码走读 — Provider"
excerpt: "Velero 中与 StorageProvider 和 SnapshotProvider 相关的源码走读"
cover: https://picsum.photos/0?sig=20220220
thumbnail: https://blogs.vmware.com/opensource/files/2022/03/velero.png
date: 2022-02-20
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://velero.io/img/Velero.svg"></div>

------

> Based on **v1.6.3**

# Storage Provider

*<u>pkg/persistence/object_store.go</u>*

StorageProvider 提供了一系列的封装了 ObjectStore Plugin 的接口，用于操作位于 BackupStorageLocation 上数据。

## IsValid

**调用接口**

- ListCommonPrefixes(\<bucket\>, \<prefix\>/, /)

**主体逻辑**

针对获取到的公共前缀子目录，获取其子目录层级是否为 backups、restores、restic、metadata 和 plugins 之一，如果不是则返回 error，认为 BackupStorageLocation 不可用。

*例如，BackupStorageLocation 上的目录层级为 \<bucket\>/\<prefix\>/backups/xxx 和 \<bucket\>/\<prefix\>/invalid/xxx，调用 ListCommonPrefixes，获取到的公共前缀子目录层级有 prefix/backups 和 prefix/invalid，进一步处理后，获取到的子目录层级为 backups 和 invalid，其中，invalid 不满足 5 个固定的名称之一，因此认为 BackupStorageLocation 不可用。*

**应用场景**

- BackupStorageLocation Controller 会通过该接口周期性检查 BackupStorageLocation 是否可用

## ListBackups

**调用接口**

- ListCommonPrefixes(\<bucket\>, \<prefix\>/backups/, /)

**主体逻辑**

针对获取到的公共前缀子目录，其子目录层级为 Backup 的名称，聚合并返回。

*例如，BackupStorageLocation 上的目录层级为 \<bucket\>/\<prefix\>/backups/backupA 和 \<bucket\>/\<prefix\>/backups/backupB，调用 ListCommonPrefixes，获取到的公共前缀子目录层级有 \<prefix\>/backups/BackupA 和 \<prefix\>/backups/BackupB，进一步处理后，获取到的子目录层级为 BackupA 和 BackupB，则返回包含两者的列表。*

**应用场景**

- BackupSync Controller 在同步 Backup 时，用于获取 BackupStorageLocation 中的 Backup

## PutBackup

**调用接口**

- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>.logs.gz, \<log\>)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/velero-backup.json, \<metadata\>)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>.tar.gz, \<content\>)
- DeleteObject(\<bucket\>, \<prefix\>/backups/\<backup\>/velero-backup.json)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>-podvolumebackups.json.gz, \<podvolumebackups\>)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>-volumesnapshots.json.gz, \<volumesnapshots\>)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>-resource-list.json.gz, \<resource-list\>)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>-csi-volumesnapshots.json.gz, \<csi-volumesnapshots\>)
- PutObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>-csi-volumesnapshotcontents.json.gz, \<csi-volumesnapshotcontents>)

**主体逻辑**

按照以下顺序上传文件

1. Backup Logs<br>*备份过程中，Velero 服务产生的日志*
2. Backup Metadata<br>*描述 Backup 本身的信息*
3. Backup Content<br>*被备份的资源内容*
4. PodVolumeBackups<br>*被备份的 Pod 卷信息，由 Restic 创建并维护*
5. PodVolumeSnapshots<br>*被备份的 Pod 卷快照信息，由 SnapshotProvider 创建并维护*
6. Backup ResourcesList<br>*被备份的资源清单*
7. CSI VolumeSnapshot<br>*被备份的 Pod 卷快照信息，由 CSI 创建并维护*
8. CSI VolumeSnapshotContents<br>*被备份的 Pod 卷快照内容信息，由 CSI 创建并维护*

*第一步日志上传失败时，仅会打印错误日志，而不会中断上传流程，是因为该步骤不影响备份的主体逻辑。此后如果有步骤失败，会影响到 Backup 的状态，并会清空之前的操作，即调用 DeleteObject 删除已上传的数据。*

**应用场景**

- Backup Controller 在备份完成时，备份的资源信息上传至 BackupStorageLocation 时

## GetBackupMetadata

**调用接口**

- GetObject(\<bucket\>, \<prefix\>/backups/\<backup\>velero-backup.json)

**主体逻辑**

获取到 velero-backup.json 文件，读取内容，解析成 Backup 对象格式。

**应用场景**

- BackupSync Controller 在同步 Backup 时，用于构建待同步的 Backup 对象

## GetBackupVolumeSnapshots

**调用接口**

- ObjectExists(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-volumesnapshots.json.gz)
- GetObject(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-volumesnapshots.json.gz)

**主体逻辑**

判断 BackupVolumeSnapshots 是否存在，如果存在，则获取并解析返回。

**应用场景**

- Restore Controller 在恢复时，会获取 Backup 相关联的 VolumeSnapshot 并恢复
- BackupDeletion Controller 在删除备份时，会一并删除 VolumeSnapshot 信息

## GetBackupVolumeBackups

**调用接口**

- ObjectExists(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-podvolumebackups.json.gz)
- GetObject(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-podvolumebackups.json.gz)

**主体逻辑**

判断 PodVolumeBackup 是否存在，如果存在，则获取并解析返回。

**应用场景**

- BackupSync Controller 在处理需要同步的 Backup 时，也会判断是否有相关联的 PodVolumeBackup 需要被同步

## GetBackupContents

**调用接口**

GetObject(\<bucket\>, \<prefix\>/backups/\<backup\>/\<backup\>.tar.gz)

**主体逻辑**

调用接口，判断指定的备份内容。

**应用场景**

- Restore Controller 在恢复时，会将备份的内容下载到临时文件中，恢复创建资源
- BackupDeletion Controller 在删除备份时，如果定义了 action，会将备份的内容下载到临时文件中，获取到 action 定义的动作并执行

## GetCSIVolumeSnapshots

**调用接口**

- ObjectExists(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-csi-volumesnapshots.json.gz)
- GetObject(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-csi-volumesnapshots.json.gz)

**主体逻辑**

调用 ObjectExists 判断 VolumeSnapshots 是否存在，如果存在，则获取并解析返回

**应用场景**

- 无

## GetCSIVolumeSnapshotContents

**调用接口**

- ObjectExists(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-csi-volumesnapshotscontents.json.gz)
- GetObject(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-csi-volumesnapshotscontents.json.gz)

**主体逻辑**

判断 VolumeSnapshotsContents 是否存在，如果存在，则获取并解析返回

**应用场景**

- BackupSync Controller 在获取到需要同步 Backup 时，如果 Velero 开启了 CSI 特性，用于获取 VolumeSnapshotContents 对象，后续也会同步该对象

## BackupExists

**调用接口**

- ObjectExists(\<bucket\>, \<prefix\>/backups/\<backup\>/velero-backup.json)

**主体逻辑**

调用接口，判断给定的备份是否存在。

**应用场景**

- Backup Controller 在备份时，会判断本次创建的备份在 BackupStorageLocation 中是否已经存在，如果存在则将备份状态设为 Failed

## DeleteBackup

**调用接口**

- ListObjects(\<bucket\>, \<prefix\>/backups/\<backup\>/)
- DeleteObject(\<bucket\>, \<key\>)

**主体逻辑**

调用 ListObjects，获取指定备份名称目录下的所有文件，即 key，遍历所有文件，调用 DeleteObject 删除 key。<br>*可以看到，Velero 在删除备份时仅会调用 DeleteBackup 删除所有的子文件，也就是所有的 key，但是不会删除这个备份目录，因此如果先要实现删除最后的空目录，需要在 StorageProvider 的 DeleteObject 接口实现。*

**应用场景**

- BackupDeletion Controller 删除指定备份时，用于删除 BackupStorageLocation 中的 Backup 数据

## PutRestoreLog

**调用接口**

- PutObject(\<bucket\>, \<prefix\>/restores/\<restore\>/restore-\<restore\>-logs.gz, \<log\>)

**主体逻辑**

调用接口，上传恢复日志文件。

**应用场景**

- Restore Controller 在恢复完成时，用于上传恢复过程中 Velero 产生的日志

## PutRestoreResults

**调用接口**

- PutObject(\<bucket\>, \<prefix\>/restores/\<restore\>/restore-\<restore\>-logs.gz, \<result\>)

**主体逻辑**

调用接口，上传恢复结果信息。

**应用场景**

- Restore Controller 在恢复完成时，用于将本次恢复过程中产生的 warning 和 error 信息上传

## DeleteRestore

**调用接口**

- ListObject(\<bucket\>, \<prefix\>/restores/\<restore\>/)
- DeleteObject(\<bucket\>, \<key\>)

**主体逻辑**

调用 ListObjects，获取指定恢复名称目录下的所有文件，即 key，遍历所有文件，调用 DeleteObject 删除 key。<br>*可以看到，Velero 在删除恢复时仅会调用 DeleteBackup 删除所有的子文件，也就是所有的 key，但是不会删除这个恢复目录，因此如果先要实现删除最后的空目录，需要在 StorageProvider 的 DeleteObject 接口实现。*

**应用场景**

- BackupDeletion Controller 在删除 Backup 时，会同步删除相关联的 Restore，并删除 BackupStorageLocation 中的 Restore 信息<br>*BackupStorageLocation 中 Restore 的删除仅在该场景下，手动删除 Restore，不会触发 DeleteRestore 流程，也就是通过 velero restore delete xxx 是无法删除 BackupStorageLocation 中 Restore 文件的。*

## GetDownloadURL

**调用接口**

- CreateSignedURL(\<bucket\>, \<target\>, 10min)<br>*target 表示要根据 DownloadRequest 对象要获取的目标文件，具体参考 DownloadRequest 章节*

**主体逻辑**

根据 DownloadRequest 的对象类别，即 target，构建不同的目标文件路径，调用接口，获取 DownloadURL。

**应用场景**

- DownloadRequest Controller 在处理 DownloadRequest 对象时，会通过该接口构建 DownloadURL，并回写至 DownloadRequest 对象中

# Snapshot Provider

*<u>pkg/backup/item_backupper.go</u>*<br>*<u>pkg/restore/pv_restorer.go</u>*

Velero 并未像 Storage Provider 封装一些上层接口，而是将底层接口的调用简单封装了以下两个函数，用于备份和恢复时，对于 PV 类型的资源做的快照和恢复的操作。

## takePVSnapshot

**调用接口**

- init(snapshotLocation.Spec.Config)
- GetVolumeID(\<Unstructured PV\>)
- GetVolumeInfo(\<volumeID\>, \<volumeZone\>)
- CreateSnapshot(\<volumeID\>, \<volumeZone\>, \<tags\>)

**主体逻辑**

1. 判断 Backup 的 SnapshotVolumes 是否开启，如果开启则表示需要对卷做快照，继续执行以下逻辑
2. 如果 PV 已经被认领，则需要判断是否被 Restic 已经备份（没有被认领的 PV 肯定不会被 Restic 备份），如果已经备份，则不会再次创建卷快照
3. 通过 PV 的 label 获取 PV 的 zone 信息（topology.kubernetes.io/zone 或者 failure-domain.beta.kubernetes.io/zone）
4. 针对 backup 中每一个 Snapshot Location，初始化一个 volumeSnapshotter，调用 GetVolumeID 尝试获取 VolumeID<br>*PV 卷肯定是由 Snapshot Provider 创建出来的，所以 GetVolumeID 肯定会有记录保存*
5. 根据 Backup 的 label 创建 Tag，并追加 velero.io/backup=\<backupName\> 和 velero.io/pv=\<pvName\>
6. 调用 GetVolumeInfo 接口，获取到卷信息，包括卷类型和 IOPS
7. 根据以上信息构建 volumeSnapshot 对象，调用 CreateSnapshot 接口，创建快照
8. 更新 volumeSnapshot 状态等信息

## executePVAction

**调用接口**

- init(snapshotLocation.Spec.Config)
- CreateVolumeFromSnapshot(\<snapshotID\>, \<volumeType\>, \<volumeZone\>,  \<volumeIOPS\>)
- SetVolumeID(\<pv\>, \<VolumeID\>)

**主体逻辑**

1. 校验 PV 的合法性，判断名称是否存在
2. 判断是否需要通过快照恢复卷，即 Backup 中是否指定了备份卷（backup.snapshotVolumes）以及 Restore 中是否指定了恢复卷（restore.restorePVs）
3. 根据 PV 名称以及 Restore 相关信息获取 snapshot 对象
4. 如果获取到 Snapshot 对象，则初始化 volumeSnapshotter
5. 根据 snapshot 对象中的卷信息，如卷类型，zone，IOPS 等信息，调用 CreateVolumeFromSnapshot 创建卷，并获取 VolumeID 信息
6. 调用 SetVolumeID，给 PV 设置 VolumeID，并返回 PV 的解构类型
