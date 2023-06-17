---
title: "「 Velero 」源码走读 — Request"
excerpt: "Velero 中与 DownloadRequest、ServerStatusRequest 等资源请求相关的流程梳理"
cover: https://picsum.photos/0?sig=20220205
thumbnail: /gallery/velero/thumbnail.svg
date: 2022-02-05
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="/gallery/velero/logo.svg"></div>

------

> based on **v1.6.3**

# DownloadRequest

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/download_request_types.go)

## DownloadTargetKind

代表着要从 BackupStorageLocation 中下载的文件，映射关系如下

| DownloadTargetKind    | BackupStorageLocation 中的文件                        |
| --------------------- | ----------------------------------------------------- |
| BackupLog             | backups/\<backup\>/\<backup\>-logs.gz                 |
| BackupContents        | backups/\<backup\>/\<backup\>.tar.gz                  |
| BackupVolumeSnapshots | backups/\<backup\>/\<backup\>-volumesnapshots.json.gz |
| BackupResourceList    | backups/\<backup\>/\<backup\>-resource-list.json.gz   |
| RestoreLog            | restores/\<restore\>/\<restore\>-logs.gz              |
| RestoreResults        | restores/\<restore\>/restore-\<restore\>-results.gz   |

# ServerStatusRequest

[API](https://raw.githubusercontent.com/vmware-tanzu/velero/release-1.6/pkg/apis/velero/v1/server_status_request_types.go)

ServerStatusRequest 不支持通过命令行手动创建，而是在获取 Velero 组件状态时，会自动生成该对象，由 ServerStatusRequest Controller 维护。

# DownloadRequest Controller

*<u>pkg/controller/download_request_controller.go</u>*<br>*<u>pkg/cmd/util/downloadrequest/downloadrequest.go</u>*

## Reconcile

[Reconcile 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/download_request_controller.go#L52)

1. 如果 DownloadRequest 状态不为空并且有过期时间的，代表该对象不是已经处理过，则需要进一步判断其是否达到过期时间
   - 如果已过期，则从集群中删除
   - 如果未过期，并且状态为 Processed，则不做处理<br>*虽然已经处理过了，但是并没有直接删除是因为有可能 log 文件流还在使用*
3. 如果 DownloadRequest 状态是空或者 New 的，调用 StorageProvider 的 GetDownloadURL 接口获取 DownloadURL 信息并设置到 DownloadRequest 中，同时更新其状态为 Processed，重新设置过期时间为 10 分钟<br>*如果调用 GetDownloadURL 时出现异常，则需要终止流程，并且重新入队，交给下次流程处理*
4. 最终，以上步骤中如果存在流程异常或者对象被合理删除，则不再重新入队，此后流程中，不再处理；否则，

## Stream

[Stream 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/cmd/util/downloadrequest/downloadrequest.go#L43)

*上层应用方式，并非 DownloadRequest Controller 逻辑*

1. 根据函数入参信息构建 DownloadRequest 对象，下发至集群中创建，后续由 DownloadRequest Controller 负责维护
2. 每 25 毫秒检测一下，直至 DownloadRequest 的 DownloadURL 被设置<br>*该动作就是 DownloadRequest Controller 核心工作内容*
3. 根据 DownloadURL 信息，构建 HTTP GET 请求，获取到 StorageProvider 中的文件数据，写入签名提供的 io.Writer 中

应用的场景包括

- velero backup download
- velero backup/restore describe
- velero backup/restore logs

# ServerStatusRequest Controller

*<u>pkg/controller/server_status_request_controller.go</u>*

## Reconcile

[Reconcile 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/controller/server_status_request_controller.go#L63)

1. 如果 ServerStatusRequest 状态是空或者 New，则更新其状态为 Processed，设置处理时间戳以及 Velero 服务中安装的插件信息
2. 如果 ServerStatusRequest 状态是 Processed，则判断其是否达到过期时间
   - 如果已过期，则从集群中删除
   - 如果未过期，则设置下次入队时间为 5 分钟之后
3. 如果 ServerStatusRequest 状态不满足上述，则不再重新入队，此后流程中，不再处理

## GetServerStatus

[GetServerStatus 源码](https://github.com/vmware-tanzu/velero/blob/5fe3a50bfddc2becb4c0bd5e2d3d4053a23e95d2/pkg/cmd/cli/serverstatus/server_status.go#L41)

*上层应用方式，非 ServerStatusRequest Controller 逻辑*

1. 根据函数入参信息构建 ServerStatusRequest 对象，下发至集群中创建，后续由 ServerStatusRequest Controller 负责维护
2. 每 250 毫秒检测一下，直至 ServerStatusRequest 的状态为 Processed，返回 ServerStatusRequest 对象信息<br>*该动作就是 ServerStatusRequest Controller 核心工作内容*

应用的场景包括

- velero plugin get





