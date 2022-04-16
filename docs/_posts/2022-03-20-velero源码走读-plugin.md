---
layout: post
title: "「 Velero 」 5.7 源码走读 — Plugin"
date: 2022-03-06
excerpt: "Velero 中与 Plugin 相关的源码走读"
tag:
- Cloud Native
- Kubernetes
- Velero
categories:
- Velero
---

![](https://velero.io/img/Velero.svg)

# ObjectStore

*<u>pkg/plugin/velero/object_store.go</u>*

Velero 无该内置类型的 plugin，具体参考 [ velero-plugin-for-aws](https://github.com/vmware-tanzu/velero-plugin-for-aws)、[velero-plugin-for-microsoft-azure](https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure)、[ velero-plugin-for-gcp](https://github.com/vmware-tanzu/velero-plugin-for-gcp)

# VolumeSnapshotter

*<u>pkg/plugin/velero/volume_snapshotter.go</u>*

Velero 无该内置类型的 plugin，具体参考 [ velero-plugin-for-aws](https://github.com/vmware-tanzu/velero-plugin-for-aws)、[velero-plugin-for-microsoft-azure](https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure)、[ velero-plugin-for-gcp](https://github.com/vmware-tanzu/velero-plugin-for-gcp)

# DeleteItemAction

*<u>pkg/plugin/velero/delete_item_action.go</u>*

Velero 无该内置类型的 plugin。

# BackupItemAction

*<u>pkg/plugin/velero/backup_item_action.go</u>*

BackupItemAction 包含两个函数：

- AppliesTo

  返回应该对哪些资源执行额外操作（通过 Included/Excluded Namespaces/Resources 实现），该过滤结果会和 Backup 本身的过滤取交集，结果会由 Execute 处理执行额外操作

- Execute

  Execute 会根据自定义的逻辑判断是否要对 Item（即符合 AppliesTo 过滤后的资源对象）做额外操作，函数会返回两个核心内容，一个是更新后的 item（此后的流程会以此 item 为基准），一个是要额外操作的对象（会加入此后备份流程中执行备份）。

## velero.io/pv

*<u>pkg/backup/backup_pv_action.go</u>*

**AppliesTo**

IncludedResources: persistentvolumeclaims

**Execute**

获取并返回绑定该 PVC 的 PV。

## velero.io/pod

*<u>pkg/backup/pod_action.go</u>*

**AppliesTo**

IncludedResources: pods

**Execute**

获取 Pod 中使用的 PVC，与 Pod 一并备份。

## velero.io/service-account

*<u>pkg/backup/service_account_action.go</u>*

**AppliesTo**

IncludedResources: serviceaccounts

**Execute**

获取 Service Account 关联的 ClusterRole 和 ClusterRoleBinding，与 Service Account 一并备份。

## velero.io/crd-remap-version

*<u>pkg/backup/remap_crd_version_action</u>*

**AppliesTo**

IncludedResources: customresourcedefinition.apiextensions.k8s.io

**Execute**

将 v1 版本的 CRD 转为 v1beta1 版本并备份。

# RestoreItemAction

*<u>pkg/plugin/velero/restore_item_action.go</u>*

RestoreItemAction 包含两个函数：

- AppliesTo

  返回应该对哪些资源执行额外操作（通过 Included/Excluded Namespaces/Resources 实现），该过滤结果会和 Restore 本身的过滤取交集，结果会由 Execute 处理执行额外操作

- Execute

  Execute 会根据自定义的逻辑判断是否要对 Item（即符合 AppliesTo 过滤后的资源对象）做额外操作，函数会返回一个更新后的 item（此后的流程会以此 item 为基准）。

## velero.io/job

*<u>pkg/restore/job_action.go</u>*

**AppliesTo**

IncludedResources: jobs

**Execute**

删除 Job 对象中的 controller-uid 字段（job.Spec.Selector.MatchLabels 以及 job.Spec.Template.ObjectMeta.Labels）。

## velero.io/pod

*<u>pkg/restore/pod_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/restic

*<u>pkg/restore/restic_restore_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/init-restore-hook

*<u>pkg/restore/init_restorehook_pod_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/service

*<u>pkg/restore/service_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/service-account

*<u>pkg/restore/service_account_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/add-pvc-from-pod

*<u>pkg/restore/add_pvc_from_pod_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/add-pv-from-pvc

*<u>pkg/restore/add_pv_from_pod_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/change-storage-class

*<u>pkg/restore/change_storageclass_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/role-bindings

*<u>pkg/restore/rolebinding_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/cluster-role-bindings

*<u>pkg/restore/clusterrolebinding_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/crd-preserve-fields

*<u>pkg/restore/crd_v1_preserve_unknown_fields_action.go</u>*

**AppliesTo**

**Execute**

## velero.io/change-pvc-node-selector

*<u>pkg/restore/change_pvc_node_selector.go</u>*

**AppliesTo**

**Execute**

## velero.io/apiservice

*<u>pkg/restore/apiservice_action.go</u>*

**AppliesTo**

**Execute**
