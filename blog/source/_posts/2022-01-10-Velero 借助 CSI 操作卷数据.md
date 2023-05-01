---
title: "「 Velero 」 1.3 操作卷数据（CSI）"
excerpt: "借助 CSI 实现对容器卷数据的备份与恢复"
cover: https://picsum.photos/0?sig=20220110
thumbnail: https://blogs.vmware.com/opensource/files/2022/03/velero.png
date: 2022-01-10
toc: true
categories:
- Getting Started
tag:
- Velero
---

<div align=center><img width="200" style="border: 0px" src="https://velero.io/img/Velero.svg"></div>

------

> Based on **v1.6.3**

可以将 CSI 快照支持集成到 Velero 中，使 Velero 能够使用 Kubernetes CSI 快照 API 备份和恢复 CSI 支持的卷。通过 CSI 快照 API，Velero 可以支持任何具有 CSI 快照的卷，而无需特定的 Velero 插件。

*在 Velero v1.6 中，此特性为 beta 版本，目前 1.9+ 为稳定版本*

# 前置依赖

- Kubernetes 版本至少为 1.17
- 集群中的 CSI 具备快照能力，兼容 v1beta1 版本 API
- 跨集群 CSI 卷快照恢复时，CSI Driver 快照类名需要保持一致

# 部署 Velero，开启 CSI 特性

```shell
$ velero install \
     --provider aws \
     --features EnableCSI \
     --plugins velero/velero-plugin-for-aws:v1.0.0,velero/velero-plugin-for-csi:v0.1.0 \
     --bucket velero \
     --secret-file ./credentials-velero \
     --use-volume-snapshots=false \
     --backup-location-config region=minio,s3ForcePathStyle="true",s3Url=http://minio.velero.svc:9000
```

**参数说明**

- `--features` 开启 feature 特性

# 流程验证

**StorageClass**

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: openebs-lvmsc
allowVolumeExpansion: true
parameters:
  volgroup: "lvm_im"
  fstype: "ext4"
  maxVolumeSize: "5" 
provisioner: local.csi.openebs.io  
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

**PersistentVolumeClaim**

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: local-pvc
spec:
  storageClassName: openebs-lvmsc
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 30Mi
```

**Pod**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-local-im
spec:
  tolerations:
  - effect: NoSchedule
  	key: node-role.kubernetes.io/master
  restartPolicy: Never
  containers:
  - name: perfrunner
    image: busybox:1.27
    command: ["sh"]
    args: ["-c", "while true ;do sleep 50; done"]
    volumeMounts:
    - mountPath: /datadir
      name: fio-vol
    tty: true
  volumes:
  - name: fio-vol
    persistentVolumeClaim:
      claimName: local-pvc
```

**操作验证**

```shell
# 写入测试数据
$ kubectl exec -it pod-local-im ls /datadir
data        lost+found

# 创建备份任务
$ velero backup create default --include-namespaces default

# 查看快照信息，发现此时 Velero 已经调用 CSI 创建了 volumesnapshot，并生成了 volumesnapshotcontent，并且查看 volumesnapshot 的状态 readyToUse 为 True
$ kubectl get volumesnapshot
NAMESPACE   NAME                     READYTOUSE   SOURCEPVC   SOURCESNAPSHOTCONTENT   RESTORESIZE   SNAPSHOTCLASS         SNAPSHOTCONTENT                                    CREATIONTIME   AGE
default     velero-local-pvc-bdbw8   true         local-pvc                           30Mi          csi-local-snapclass   snapcontent-93903201-f07a-405c-92d9-2c0b6bafd6a1   117s           118s

$ kubectl get volumesnapshotcontent
NAME                                               READYTOUSE   RESTORESIZE   DELETIONPOLICY   DRIVER                 VOLUMESNAPSHOTCLASS   VOLUMESNAPSHOT           VOLUMESNAPSHOTNAMESPACE   AGE
snapcontent-93903201-f07a-405c-92d9-2c0b6bafd6a1   true         31457280      Delete           local.csi.openebs.io   csi-local-snapclass   velero-local-pvc-bdbw8   default                   2m38s

# 模拟故障
$ kubectl delete -f pod.yaml && kubectl delete -f pvc.yaml

# 创建恢复任务
$ velero restore create --from-backup default

$ kubectl get volumesnapshotcontent 
NAME                                               READYTOUSE   RESTORESIZE   DELETIONPOLICY   DRIVER                 VOLUMESNAPSHOTCLASS   VOLUMESNAPSHOT           VOLUMESNAPSHOTNAMESPACE   AGE
snapcontent-93903201-f07a-405c-92d9-2c0b6bafd6a1   true         31457280      Delete           local.csi.openebs.io   csi-local-snapclass   velero-local-pvc-bdbw8   default                   3m31s
velero-velero-local-pvc-bdbw8-z9t2f                true         0             Delete           local.csi.openebs.io                         velero-local-pvc-bdbw8   default                   2m3s

# 可以看到，Velero 调用 CSI Driver 基于之前的 volumesnapshot 恢复出来了一个 PVC
$ kubectl get pvc
NAME        STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS    AGE
local-pvc   Bound    pvc-f034e865-340a-40cf-9381-3b95b1bd2f1f   30Mi       RWO            openebs-lvmsc   4m16s

$ kubectl get pvc -o yaml
<skip>
spec:
    accessModes:
    - ReadWriteOnce
    dataSource:
      apiGroup: snapshot.storage.k8s.io
      kind: VolumeSnapshot
      name: velero-local-pvc-bdbw8
    resources:
      requests:
        storage: 30Mi
    storageClassName: openebs-lvmsc
    volumeMode: Filesystem
    volumeName: pvc-f034e865-340a-40cf-9381-3b95b1bd2f1f

# 数据已经从快照恢复
$ kubectl exec -it pod-local-im ls /datadir
data        lost+found
```

# 流程走读

Velero 的 CSI 支持不依赖 Velero VolumeSnapshotter 插件。相反，Velero 采用一组 BackupItemAction 插件用于在操作 PersistentVolumeClaims 之前进行一些额外的动作。

备份时，当 BackupItemAction 发现有一个 PersistentVolumeClaims 指向由 CSI Driver 创建的 PersistentVolume 时，它将获取具有相同 Driver 名称的 VolumeSnapshotClass 来创建以 PersistentVolumeClaim 为源的 CSI VolumeSnapshot 对象，VolumeSnapshot 和 PersistentVolumeClaim 位于同一命名空间中。

接着，CSI external-snapshotter watch 到 VolumeSnapshot 之后创建一个 VolumeSnapshotContent 对象，它将指向存储系统中实际的、基于磁盘的快照。 external-snapshotter 会调用 CSI Driver 的 snapshot 方法，Driver 会调用存储系统的 API 生成快照。一旦生成 ID 并且存储系统将快照标记为可用于恢复，VolumeSnapshotContent 对象将使用 status.snapshotHandle 进行更新，并且设置status.readyToUse 字段为 true。

Velero 将在备份 tarball 中包含生成的 VolumeSnapshot 和 VolumeSnapshotContent 对象，并将 JSON 文件中的所有 VolumeSnapshots 和 VolumeSnapshotContents 对象上传到对象存储系统。当 Velero 将备份同步到新集群时，VolumeSnapshotContent 对象也将同步到集群中，以便 Velero 可以适当地管理备份过期。

VolumeSnapshotClass 的 DeletionPolicy 设置为 Retain 时，Velero 备份的生命周期内保留存储系统中的卷快照，并防止在发生灾难时删除存储系统中的卷快照，其中命名空间与VolumeSnapshot 对象可能会丢失。

当 Velero 备份到期时，VolumeSnapshot 对象将被删除，VolumeSnapshotContent 对象的 DeletionPolicy 将更新 Delete，以释放存储系统上的空间。
