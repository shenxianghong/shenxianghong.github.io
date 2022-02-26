---
layout: post
title: "「 Velero 」 5.4 源码走读 — Provider"
date: 2022-02-20
excerpt: "Velero 中与 StorageProvider 和 SnapshotProvider 相关的源码走读"
tag:
- Cloud Native
- Kubernetes
- Velero
categories:
- Velero
---

![](https://velero.io/img/Velero.svg)

# StorageProvider

*<u>pkg/persistence/object_store.go</u>*

StorageProvider 提供了一系列的封装了 ObjectStore Plugin 的接口，用于获取位于 BackupStorageLocation 上数据。

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

**主体逻辑**

**应用场景**

## GetBackupMetadata

**调用接口**

- GetObject(\<bucket\>, \<prefix\>/backups/\<backup\>velero-backup.json)

**主体逻辑**

获取到 velero-backup.json 文件，读取内容，解析成 Backup 对象格式。

**应用场景**

- BackupSync Controller 在同步 Backup 时，用于构建待同步的 Backup 对象

## GetBackupVolumeSnapshots

**调用接口**

**主体逻辑**

**应用场景**

## GetBackupVolumeBackups

**调用接口**

**主体逻辑**

**应用场景**

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

调用 ObjectExists 判断 VolumeSnapshots 是否存在，如果存在，则解析返回

**应用场景**

- 无

## GetCSIVolumeSnapshotContents

**调用接口**

- ObjectExists(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-csi-volumesnapshotscontents.json.gz)
- GetObject(\<bucket\>, \<prefix\>/\<backups\>/\<backup\>/\<backup\>-csi-volumesnapshotscontents.json.gz)

**主体逻辑**

调用 ObjectExists 判断 VolumeSnapshotsContents 是否存在，如果存在，则解析返回

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

根据 DownloadRequest 的对象类别，构建不同的目标文件路径，调用接口，获取 DownloadURL。

**应用场景**

- DownloadRequest Controller 在处理 DownloadRequest 对象时，会通过该接口构建 DownloadURL，并回写至 DownloadRequest 对象中

# SnapshotProvider

