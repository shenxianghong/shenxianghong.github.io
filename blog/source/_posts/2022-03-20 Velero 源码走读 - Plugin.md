---
title: "「 Velero 」源码走读 — Plugin"
excerpt: "Velero 中与 ObjectStore、VolumeSnapshotter 等插件相关的流程梳理"
cover: https://picsum.photos/0?sig=20220320
thumbnail: https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/stacked/199150-vmw-os-lgo-velero-final_stacked-gry.svg
date: 2022-03-20
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://raw.githubusercontent.com/vmware-tanzu/velero/main/assets/one-line/199150-vmw-os-lgo-velero-final_gry.svg"></div>

------

> based on **v1.6.3**

# ObjectStore

*<u>pkg/plugin/velero/object_store.go</u>*

*Velero 无该内置类型的 plugin，具体参考 [ velero-plugin-for-aws](https://github.com/vmware-tanzu/velero-plugin-for-aws)、[velero-plugin-for-microsoft-azure](https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure)、[ velero-plugin-for-gcp](https://github.com/vmware-tanzu/velero-plugin-for-gcp)。*

ObjectStore 包含以下接口

- Init

  初始化 Object<br>*如果接口返回 error，BackupStorageLocation Controller 则无法调用 IsValid 判断 Provider 的可用性*

- PutObject

  上传 key（io.Reader）到 Provider 中

- ObjectExists

  判断 key 在 Provider 中是否存在

- GetObject

  获取 key 的内容（io.ReadCloser）<br>*Velero 会在调用后负责 Close 该 io 操作，无需 plugin 操作*

- ListCommonPrefixes

  获取满足给定公共前缀的所有 key

- ListObjects

  获取满足给定前缀的所有 key

- DeleteObject

  删除 Provider 中的指定 key<br>*仅会删除 key 本身，如果对于文件存储类型的 Provider，需要在 plugin 中实现删除最后一个 key 时，同时删除目录的功能*

- CreateSignedURL

  生成具有过期时间的 pre-signed url，用于获取 Provider 中的文件<br>*对于文件存储类型等不具备提供对外服务的 Provider，该接口需要有其他服务支撑*

# VolumeSnapshotter

*<u>pkg/plugin/velero/volume_snapshotter.go</u>*

*Velero 无该内置类型的 plugin，具体参考 [ velero-plugin-for-aws](https://github.com/vmware-tanzu/velero-plugin-for-aws)、[velero-plugin-for-microsoft-azure](https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure)、[ velero-plugin-for-gcp](https://github.com/vmware-tanzu/velero-plugin-for-gcp)。*

VolumeSnapshotter 包含以下接口

- Init

  初始化 VolumeSnapshotshotter

- CreateVolumeFromSnapshot

  根据指定的 zone、IOPS 信息从快照恢复卷<br>

  *Velero 创建卷的动作为同步处理，会一直等待，直至完成或者失败*

- GetVolumeID

  根据指定的 PV 获取 VolumeID

- SetVolumeID

  对指定的 PV 设置 VolumeID

- GetVolumeInfo

  获取 Volume 信息

- CreateSnapshot

  对指定的卷创建快照<br>*Velero 创建 PV 快照的动作为同步处理，会一直等待，直至完成或者失败*

- DeleteSnapshot

  删除快照

# DeleteItemAction

*<u>pkg/plugin/velero/delete_item_action.go</u>*

*Velero 无该内置类型的 plugin。*

DeleteItemAction 包含以下接口

- AppliesTo

  返回应该对哪些资源执行额外操作（通过 Included/Excluded Namespaces/Resources 实现），结果会由 Execute 处理执行额外操作

- Execute

  根据自定义的逻辑判断是否要对函数入参 Item（即符合 AppliesTo 过滤后的资源对象）在删除 Backup 时，执行一些额外的操作

# BackupItemAction

*<u>pkg/plugin/velero/backup_item_action.go</u>*

BackupItemAction 包含以下接口

- AppliesTo

  返回应该对哪些资源执行额外操作（通过 Included/Excluded Namespaces/Resources 实现），该过滤结果会和 Backup 本身的过滤取交集，结果会由 Execute 处理执行额外操作

- Execute

  根据自定义的逻辑判断是否要对函数入参 Item（即符合 AppliesTo 过滤后的资源对象）做额外操作，函数会返回两个核心内容
  
  - 更新后的 item，此后的流程会以此 item 为基准
  - 需要额外操作的对象，会加入此后备份流程中执行备份

## velero.io/pv

*<u>pkg/backup/backup_pv_action.go</u>*

**AppliesTo**

IncludedResources: persistentvolumeclaims

**Execute**

1. PVC item 未更新
2. 获取 PVC 绑定的 PV 作为额外操作的对象返回

## velero.io/pod

*<u>pkg/backup/pod_action.go</u>*

**AppliesTo**

IncludedResources: pods

**Execute**

1. Pod item 未更新
2. 获取 Pod 中使用的 PVC 作为额外操作的对象返回

## velero.io/service-account

*<u>pkg/backup/service_account_action.go</u>*

**AppliesTo**

IncludedResources: serviceaccounts

**Execute**

1. Service Account item 未更新
2. 获取 Service Account 关联的 ClusterRole 和 ClusterRoleBinding 作为额外操作的对象返回

## velero.io/crd-remap-version

*<u>pkg/backup/remap_crd_version_action</u>*

**AppliesTo**

IncludedResources: customresourcedefinition.apiextensions.k8s.io

**Execute**

1. 如果 CRD 的 apiVersion 为 apiextensions.k8s.io/v1，并且如果是单一版本的 CRD，或者存在位解构的字段或者预留字段时，则将 v1 版本的 CRD item 转换为 v1beta1 版本，并返回<br>*参考：https://kubernetes.io/zh/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions*
2. 无额外操作的对象

# RestoreItemAction

*<u>pkg/plugin/velero/restore_item_action.go</u>*

RestoreItemAction 包含以下接口

- AppliesTo

  返回应该对哪些资源执行额外操作（通过 Included/Excluded Namespaces/Resources 实现），该过滤结果会和 Restore 本身的过滤取交集，结果会由 Execute 处理执行额外操作

- Execute

  根据自定义的逻辑判断是否要对 Item（即符合 AppliesTo 过滤后的资源对象）做额外操作，函数会返回三个核心内容
  
  - 更新后的 item，此后的流程会以此 item 为基准
  - 需要额外操作的对象，会加入此后恢复流程中执行恢复
  - 是否跳过恢复的标识 skipRestore

## velero.io/job

*<u>pkg/restore/job_action.go</u>*

**AppliesTo**

IncludedResources: jobs

**Execute**

1. 删除 Job item 中的 controller-uid 字段（job.Spec.Selector.MatchLabels 以及 job.Spec.Template.ObjectMeta.Labels），并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/pod

*<u>pkg/restore/pod_action.go</u>*

**AppliesTo**

IncludedResources: pods

**Execute**

1. 清空 Pod item 的 nodeName、priority 和无需保留卷、卷挂载信息，并返回<br>*如果 Pod 卷或卷挂载以 pod.Spec.ServiceAccountName + "-token-" 开头，则无需保留。*
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/restic

*<u>pkg/restore/restic_restore_action.go</u>*

**AppliesTo**

IncludedResources: pods

**Execute**

1. 获取到 Pod item 中的被 Restic 备份的卷信息，如果有的话，则会根据一系列的配置信息对 Pod item 注入一个 Init Container（restic-wait），并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/init-restore-hook

*<u>pkg/restore/init_restorehook_pod_action.go</u>*

**AppliesTo**

IncludedResources: pods

**Execute**

1. 获取 Pod 中有关 init.hook.restore.velero.io/xxx 的 annotation 信息，构建一个 Init Container，如果 Pod 没有相关的 annotation，则获取 Restore 中的 hook 信息，根据 selector 的匹配结果，构建一个 Init Container。以上构建的 Init Container 会追加在 Pod 中，顺序为：restic-wait（如果 Pod 已经存在），hook1，hook2...，追加作为更新后的 Pod item 返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/service

*<u>pkg/restore/service_action.go</u>*

**AppliesTo**

IncludedResources: services

**Execute**

1. 如果 ClusterIP 不为 None（也就是说分配了一个 Service IP），则删除 ClusterIP，如果 Restore 没有指定 --preserve-nodeports 参数，则删除 Service 的 NodePort 信息，并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/service-account

*<u>pkg/restore/service_account_action.go</u>*

**AppliesTo**

IncludedResources: serviceaccounts

**Execute**

1. 删除 Service Account 中的运行状态时生成的 \<serviceAccountName\>-token，并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/add-pvc-from-pod

*<u>pkg/restore/add_pvc_from_pod_action.go</u>*

**AppliesTo**

IncludedResources: pods

**Execute**

1. Pod item 未更新
2. 获取 Pod 中使用的 PVC 作为额外的操作对象返回
3. skipRestore 为 false

## velero.io/add-pv-from-pvc

*<u>pkg/restore/add_pv_from_pod_action.go</u>*

**AppliesTo**

IncludedResources: persistentvolumeclaims

**Execute**

1. PVC item 未更新
2. 获取 PVC 绑定的 PV 作为额外的操作对象返回
3. skipRestore 为 false

## velero.io/change-storage-class

*<u>pkg/restore/change_storageclass_action.go</u>*

**AppliesTo**

IncludedResources: persistentvolumeclaims, persistentvolumes

**Execute**

1. 获取集群中有关 StorageClass 映射关系的唯一的 ConfigMap 信息（即 ConfigMap label 中有 velero.io/plugin-config="" 和 velero.io/change-storage-class=RestoreItemAction，如果不存在则不做映射关系），根据 PV 或 PVC 中的 StorageClassName 获得映射后的新 StorageClassName，更新 PV 或 PVC，并返回

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: change-storage-class
     namespace: velero
     labels: 
       velero.io/plugin-config: ""
       velero.io/change-storage-class: RestoreItemAction
   data:
     ussvd-sc: local-sc
   ```

2. 无额外操作的对象

3. skipRestore 为 false

## velero.io/role-bindings

*<u>pkg/restore/rolebinding_action.go</u>*

**AppliesTo**

IncludedResources: rolebindings

**Execute**

1. 根据 Restore 中的 namespace 映射关系，修改 RoleBinding 的 namespace，并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/cluster-role-bindings

*<u>pkg/restore/clusterrolebinding_action.go</u>*

**AppliesTo**

IncludedResources: clusterrolebindings

**Execute**

1. 根据 Restore 中的 namespace 映射关系，修改 ClusterRoleBinding 的 namespace，并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/crd-preserve-fields

*<u>pkg/restore/crd_v1_preserve_unknown_fields_action.go</u>*

**AppliesTo**

IncludedResources: customresourcedefinition.apiextensions.k8s.io

**Execute**

1. 设置版本为 apiextensions.k8s.io/v1 的 CRD 的 PreserveUnknownFields 信息，并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/change-pvc-node-selector

*<u>pkg/restore/change_pvc_node_selector.go</u>*

**AppliesTo**

IncludedResources: persistentvolumeclaims

**Execute**

1. 获取集群中有关 PVC NodeSelector 映射关系的唯一 ConfigMap 信息，根据 PVC annotation 中的 volume.kubernetes.io/selected-node 获得映射后的新 Node，如果存在映射关系，则认为新的 Node 存在；如果不存在映射关系，则判断旧 Node 是否存在，不存在则删除 volume.kubernetes.io/selected-node 信息，存在则设置 PVC annotation nodeSelector 为旧 Node。无论怎样，均更新 PVC，并返回
2. 无额外操作的对象
3. skipRestore 为 false

## velero.io/apiservice

*<u>pkg/restore/apiservice_action.go</u>*

**AppliesTo**

IncludedResrouces: apiservices

LabelSelector: kube-aggregator.kubernetes.io/automanaged

**Execute**

1. apiservices item 未更新
2. 无额外操作的对象
3. skipRestore 为 true，因为这些 apiservices 由 kube-aggergator 负责维护，无需恢复

