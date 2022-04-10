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

Velero 无该内置类型的 plugin

# BackupItemAction

*<u>pkg/plugin/velero/backup_item_action.go</u>*

## velero.io/pv

*<u>pkg/backup/backup_pv_action.go</u>*

## velero.io/pod

*<u>pkg/backup/pod_action.go</u>*

## velero.io/service-account

*<u>pkg/backup/service_account_action.go</u>*

## velero.io/crd-remap-version

*<u>pkg/backup/remap_crd_version_action</u>*

# RestoreItemAction

*<u>pkg/plugin/velero/restore_item_action.go</u>*

## velero.io/job

*<u>pkg/restore/job_action.go</u>*

## velero.io/pod

*<u>pkg/restore/pod_action.go</u>*

## velero.io/restic

*<u>pkg/restore/restic_restore_action.go</u>*

## velero.io/init-restore-hook

*<u>pkg/restore/init_restorehook_pod_action.go</u>*

## velero.io/service

*<u>pkg/restore/service_action.go</u>*

## velero.io/service-account

*<u>pkg/restore/service_account_action.go</u>*

## velero.io/add-pvc-from-pod

*<u>pkg/restore/add_pvc_from_pod_action.go</u>*

## velero.io/add-pv-from-pvc

*<u>pkg/restore/add_pv_from_pod_action.go</u>*

## velero.io/change-storage-class

*<u>pkg/restore/change_storageclass_action.go</u>*

## velero.io/role-bindings

*<u>pkg/restore/rolebinding_action.go</u>*

## velero.io/cluster-role-bindings

*<u>pkg/restore/clusterrolebinding_action.go</u>*

## velero.io/crd-preserve-fields

*<u>pkg/restore/crd_v1_preserve_unknown_fields_action.go</u>*

## velero.io/change-pvc-node-selector

*<u>pkg/restore/change_pvc_node_selector.go</u>*

## velero.io/apiservice

*<u>pkg/restore/apiservice_action.go</u>*
