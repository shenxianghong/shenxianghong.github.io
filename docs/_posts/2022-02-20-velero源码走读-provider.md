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

StorageProvider 提供了一系列的封装了 ObjectStore Plugin 的接口，用于获取位于 BackupStorageLocation 上数据。

## IsValid

**调用接口**

ListCommonPrefixes，入参为 bucket、prefix + /、和 /

**流程逻辑**

针对获取到的公共前缀子目录，获取其子目录层级是否为 backups、restores、restic、metadata 和 plugins 之一，如果不是则返回 error，认为 BackupStorageLocation 不可用。

*例如，BackupStorageLocation 上的目录层级为 bucket/prefix/backups/xxx 和 bucket/prefix/invalid/xxx 调用 ListCommonPrefixes，获取到的公共前缀子目录层级有 prefix/backups 和 prefix/invalid，进一步处理后，获取到的子目录层级为 backups 和 invalid，其中，invalid 不满足 5 个固定的名称之一，因此认为 BackupStorageLocation 不可用。*

## ListBackups

## PutBackup

## GetBackupMetadata

## GetBackupVolumeSnapshots

## GetBackupVolumeBackups

## GetBackupContents

## GetCSIVolumeSnapshots

## GetCSIVolumeSnapshotContents

## BackupExists

## DeleteBackup

## PutRestoreLog

## PutRestoreResults

## DeleteRestore

## GetDownloadURL

# SnapshotProvider

