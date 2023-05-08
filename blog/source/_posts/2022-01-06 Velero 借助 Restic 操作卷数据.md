---
title: "「 Velero 」操作卷数据（Restic）"
excerpt: "借助 Restic 实现对容器卷数据的快速、安全和高效备份与恢复"
cover: https://picsum.photos/0?sig=20220106
thumbnail: https://blogs.vmware.com/opensource/files/2022/03/velero.png
date: 2022-01-06
toc: true
categories:
- Disaster Recovery
tag:
- Velero
---

<div align=center><img width="170" style="border: 0px" src="https://velero.io/img/Velero.svg"></div>

------

> Based on **v1.6.3**

# 与 Restic 集成安装

## 命令行安装

```shell
velero install \
    --provider aws \
    --bucket velero \
    --plugins velero/velero-plugin-for-aws:v1.0.0 \
    --secret-file /root/credential \
    --use-restic \
    --default-volumes-to-restic \
    --use-volume-snapshots=false
```

**参数说明**

- `use-restic` 表示是否启用 Restic 组件操作 Pod 中的卷数据
- `default-volumes-to-restic` 表示是否默认备份 Pod 中所有的卷

## 部署文件

相比于之前的部署结果：

- 在指定 `--use-restic` 之后，额外部署了 DaemonSet 类型的 Restic 服务
- 在指定 `--default-volumes-to-restic` 之后，Velero 的启动参数中会新增 feature 特性

**default-volumes-to-restic**

```yaml
- args:
  - server
  - --features=
  - --default-volumes-to-restic=true
```

**use-restic**

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  creationTimestamp: null
  labels:
    component: velero
  name: restic
  namespace: velero
spec:
  selector:
    matchLabels:
      name: restic
  template:
    metadata:
      creationTimestamp: null
      labels:
        component: velero
        name: restic
    spec:
      containers:
      - args:
        - restic
        - server
        - --features=
        command:
        - /velero
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: VELERO_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: VELERO_SCRATCH_DIR
          value: /scratch
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /credentials/cloud
        - name: AWS_SHARED_CREDENTIALS_FILE
          value: /credentials/cloud
        - name: AZURE_CREDENTIALS_FILE
          value: /credentials/cloud
        - name: ALIBABA_CLOUD_CREDENTIALS_FILE
          value: /credentials/cloud
        image: velero:dev
        imagePullPolicy: IfNotPresent
        name: restic
        resources:
          limits:
            cpu: "1"
            memory: 1Gi
          requests:
            cpu: 500m
            memory: 512Mi
        volumeMounts:
        - mountPath: /host_pods
          mountPropagation: HostToContainer
          name: host-pods
        - mountPath: /scratch
          name: scratch
        - mountPath: /credentials
          name: cloud-credentials
      securityContext:
        runAsUser: 0
      serviceAccountName: velero
      volumes:
      - hostPath:
          path: /var/lib/kubelet/pods
        name: host-pods
      - emptyDir: {}
        name: scratch
      - name: cloud-credentials
        secret:
          secretName: cloud-credentials
  updateStrategy: {}
```

## 部署结果

```shell
$ kubectl get pod -n velero
NAME                      READY   STATUS      RESTARTS   AGE
minio-54b5867494-6plnl    1/1     Running     0          12m
minio-setup-54vx8         0/1     Completed   0          12m
restic-vqdmn              1/1     Running     0          14m
velero-598755d478-7l8gd   1/1     Running     0          14m
```

# 流程验证

以 local PV 作为 Pod 卷数据为例

**PV**

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: example-pv
spec:
  capacity:
    storage: 100Gi
  volumeMode: Filesystem
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: ""
  local:
    path: /tmp/example
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - sxh-1
```

**PVC**

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: example-pvc
spec:
  storageClassName: ""
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Mi
```

**Pod**

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: example-pod
spec:
  containers:
  - name: test-pod
    image: busybox
    command:
      - "/bin/sh"
    args:
      - "-c"
      - "sleep 1000000"
    volumeMounts:
      - name: local-pvc
        mountPath: "/mnt"
  volumes:
    - name: local-pvc
      persistentVolumeClaim:
        claimName: example-pvc
```

**操作验证**

```shell
# 写入测试数据
$ kubectl exec -it example-pod ls /mnt
hello-here

# 创建备份任务
$ velero backup create default

# 模拟故障
$ rm -rf /tmp/example/* && kubectl delete pod test-pod && kubectl delete pvc test-claim && kubectl delete pv test-pv 

# 重新部署 pv，后续在 troubleshooting 会说明原因
$ kubectl apply -f pv.yaml

# 创建恢复任务
$ velero restore create default --from-backup default

# 查看测试数据
$ kubectl exec -it example-pod ls /mnt
hello-here
```

# 流程走读

Velero 在对 Pod 卷数据做备份时，可以与开源项目 Restic 集成，集成使用时，Restic 本身的优势也会得到支持，如**加密传输**， **压缩备份**， **增量备份**， **断点续传**等。

## CRD

有关卷备份与恢复的 CRD 包含：`podvolumebackups.velero.io`、`podvolumerestores.velero.io` 和 `resticrepositories.velero.io`。

### ResticRepository

Restic 中 `repository` 的概念，表示备份用于存储的位置。

原生 Restic 中通过 `restic init --repo <repo>` 的方式初始化不同的 backend。

**local**

```shell
$ restic init --repo /srv/restic-repo
enter password for new repository:
enter password again:
created restic repository 085b3c76b9 at /srv/restic-repo
Please note that knowledge of your password is required to access the repository.
Losing your password means that your data is irrecoverably lost.
```

**sftp**

```shell
$ restic -r sftp:user@host:/srv/restic-repo init
enter password for new repository:
enter password again:
created restic repository f1c6108821 at sftp:user@host:/srv/restic-repo
Please note that knowledge of your password is required to access the repository.
Losing your password means that your data is irrecoverably lost.
```

**Amazon S3**

```shell
$ export AWS_ACCESS_KEY_ID=<MY_ACCESS_KEY>
$ export AWS_SECRET_ACCESS_KEY=<MY_SECRET_ACCESS_KEY>
$ restic -r s3:s3.amazonaws.com/bucket_name init
enter password for new repository:
enter password again:
created restic repository eefee03bbd at s3:s3.amazonaws.com/bucket_name
Please note that knowledge of your password is required to access the repository.
Losing your password means that your data is irrecoverably lost.
```

**Minio Server**

```shell
$ export AWS_ACCESS_KEY_ID=<YOUR-MINIO-ACCESS-KEY-ID>
$ export AWS_SECRET_ACCESS_KEY= <YOUR-MINIO-SECRET-ACCESS-KEY>
$ ./restic -r s3:http://localhost:9000/restic init
enter password for new repository:
enter password again:
created restic repository 6ad29560f5 at s3:http://localhost:9000/restic1
Please note that knowledge of your password is required to access
the repository. Losing your password means that your data is irrecoverably lost.
```

在 Velero 中封装了初始化 Restic Repository 的动作（具体包含 `restic init`、`restic check` 和 `restic prune`），但是仅支持 `velero.io/aws`（包含 aws 或者非 aws，但是兼容 s3 的存储，如 minio）、`velero.io/aure` 和 `velero.io/gcp`。

*pkg/restic/config.go*

```go
// getRepoPrefix returns the prefix of the value of the --repo flag for
// restic commands, i.e. everything except the "/<repo-name>".
func getRepoPrefix(location *velerov1api.BackupStorageLocation) (string, error) {
	var bucket, prefix string

	if location.Spec.ObjectStorage != nil {
		layout := persistence.NewObjectStoreLayout(location.Spec.ObjectStorage.Prefix)

		bucket = location.Spec.ObjectStorage.Bucket
		prefix = layout.GetResticDir()
	}

	backendType := getBackendType(location.Spec.Provider)

	if repoPrefix := location.Spec.Config["resticRepoPrefix"]; repoPrefix != "" {
		return repoPrefix, nil
	}

	switch backendType {
	case AWSBackend:
		var url string
		switch {
		// non-AWS, S3-compatible object store
		case location.Spec.Config["s3Url"] != "":
			url = location.Spec.Config["s3Url"]
		default:
			region, err := getAWSBucketRegion(bucket)
			if err != nil {
				url = "s3.amazonaws.com"
				break
			}

			url = fmt.Sprintf("s3-%s.amazonaws.com", region)
		}

		return fmt.Sprintf("s3:%s/%s", strings.TrimSuffix(url, "/"), path.Join(bucket, prefix)), nil
	case AzureBackend:
		return fmt.Sprintf("azure:%s:/%s", bucket, prefix), nil
	case GCPBackend:
		return fmt.Sprintf("gs:%s:/%s", bucket, prefix), nil
	}

	return "", errors.New("restic repository prefix (resticRepoPrefix) not specified in backup storage location's config")
}
```

针对每一个待备份卷数据的 Pod 所在的 namespace，velero 会创建一个和 namespace 对应的 ResticRepository，如果对应 namespace 的 ResticRepository 存在，则不会重复创建，命名方式为 `<namespace>-<backupstoragelocation>-<id>`。

```shell
$ velero restic repo get
NAME                    STATUS   LAST MAINTENANCE
default-default-n5mz4   Ready    2022-01-06 16:05:14 +0800 CST
velero-default-8q767    Ready    2022-01-06 15:46:31 +0800 CST

$ kubectl get resticrepositories -n velero
NAME                    AGE
default-default-n5mz4   2m
velero-default-8q767    2m28s

$ kubectl get resticrepositories -n velero default-default-n5mz4 -o yaml
<skip>
spec:
  backupStorageLocation: default
  maintenanceFrequency: 168h0m0s
  resticIdentifier: s3:http://minio.velero.svc:9000/velero/restic/default
  volumeNamespace: default
```

在 BackupStorageLocation 中的存储方式如下：

**velero.io/aws**

<div align=center><img width="600" style="border: 0px" src="/gallery/velero/resticrepositories-in-minio.png"></div>

### PodVolumeBackup

代表 Pod 卷备份任务，每有一个待备份的 Pod 卷，Velero 会创建一个和 backup 对应的 PodVolumeBackup，由于该 CR 是和 backup 对应且 backup 名称唯一，所以针对相同的 Pod 卷的多次备份，会创建多个 PodVolumeBackup，命名方式为 `<backup>-<id>`，各节点上的 restic daemonset  controller 会根据 PodVolumeBackup指定 `restic backup` 命令。

```shell
$ kubectl get podvolumebackups -n velero
NAME            AGE
default-l2dd6   105s
velero-j2xj5    2m5s
velero-llv9m    2m5s
velero-nxlkq    2m13s
velero-x6m72    2m7s
velero-xqdhk    2m7s
velero-z7rhq    2m12s

$ kubectl describe podvolumebackups -n velero default-l2dd6
<skip>
Spec:
  Backup Storage Location:  default
  Node:                     sxh-1
  Pod:
    Kind:           Pod
    Name:           example-pod
    Namespace:      default
    UID:            bb17e801-b595-4e96-8ced-e27e8686be23
  Repo Identifier:  s3:http://minio.velero.svc:9000/velero/restic/default
  Tags:
    Backup:        default
    Backup - UID:  2fa40af0-2dcc-43dc-9636-79e5f1c95045
    Ns:            default
    Pod:           example-pod
    Pod - UID:     bb17e801-b595-4e96-8ced-e27e8686be23
    Pvc - UID:     f89b5daf-cf5e-49f1-9ba8-26e62f55baf2
    Volume:        local-pvc
  Volume:          local-pvc
```

### PodVolumeRestore

代表 Pod volume 的恢复任务，每有一个待恢复的 Pod 卷，Velero 会创建一个和 restore 对应的 PodVolumeRestore，由于该 CR 是和 restore 对应且 restore 名称唯一，所以针对相同的 Pod 卷建多个 PodVolumeRestore，命名方式为 `<restore>-<id>`，各节点上的 restic daemonset  controller 会根据 PodVolumeRestore执行 `restic restore` 命令。

```shell
$ kubectl get podvolumerestores -n velero
NAME                        AGE
alls-20220106165015-p54kz   6m17s
alls-20220106165349-rl4sh   2m42s

$ kubectl describe podvolumerestores alls-20220106165349-rl4sh -n velero
Spec:
  Backup Storage Location:  default
  Pod:
    Kind:           Pod
    Name:           example-pod
    Namespace:      default
    UID:            d72f5c66-2f93-4e4b-b2f6-0e5ce0aa2042
  Repo Identifier:  s3:http://minio.velero.svc:9000/velero/restic/default
  Snapshot ID:      abdd9af5
  Volume:           local-pvc
```

## 备份

Velero 在开启 Restic 对 Pod volume 备份时，根据以下两种方式获取待备份卷的信息：

**Velero args**

在开启 `default-volumes-to-restic` 时，默认所有备份均使用 restic 备份所有的 Pod 卷。该参数即可以在 `velero install` 中全局生效，也可以在 `velero backup create` 时针对单次备份生效。

*pkg/cmd/cli/install/install.go*

**Pod annotation**

在未开启 `default-volumes-to-restic` 时，Velero 会根据 Pod annotation 的中声明信息，获取待备份的 Pod 卷，例如 `backup.velero.io/backup-volumes=nginx-logs`，也可以指定排除备份的卷，例如 `backup.velero.io/backup-volumes-excludes=nginx-logs`。

**注意**

1. 如果两种方式均开启时，仅 `backup-volumes-excludes` 生效

2. 并非所有的 in-tree volume 均会备份。例如，以下卷类型不会参与备份

   - **hostpath**，由于 hostpath 不会挂载到 `/var/lib/kubelet/pods` 中，因此无法被 Restic 获取
   - **secret**，会作为 K8s metadata 单独备份
   - **configMap**，会作为 K8s metadata 单独备份
   - **projected**，为运行时状态数据，不会备份
   - **"default-token"**，默认的 service account token，不需要备份

如上所述，Velero 为 Pod 中要备份的每个卷创建一个  PodVolumeBackup，并等待其状态返回 completed 或者 failed。

与此同时，每个节点的 restic controller 会有一个 `/var/lib/kubelet/pods` 的 hostPath 卷挂载用来访问 Pod 的卷数据，通过访问该 hostPath 卷，获取到待备份的 Pod 卷数据后，执行 `restic backup`，并根据实际情况将 odVolumeBackup 状态设置为 completed 或者 failed。

在每一个 PodVolumeBackup 完成时，Velero 会将信息添加到 `<backup-name>-podvolumebackups.json.gz` 文件中，备份完成时，汇总候上传到后端存储中，该文件中包含本次备份所有的 PodVolumeBackup  信息。

## 恢复

当对 Pod 卷数据进行恢复时，Velero 会根据 restore 的 `--from-backup` 的备份获取到 PodVolumeBackup。

对于获取到 PodVolumeBackup，Velero 会确保待恢复 Pod 的 namespace有与之对应的  ResticRepository，如果不存在，则创建一个，并执行 `restic init` 和 `restic check`。

Velero 向每一个待恢复卷数据的 Pod 注入一个 init 容器，这个程序会一直等待，直到在每一个恢复的卷中找到一个位于 .velero 下的文件，该文件的名称是恢复任务的 UID，即 Pod 完成所有卷数据的恢复。

Velero 创建出来的这个待恢复的 Pod，Kubernetes 调度器将这个 Pod 调度到一个可工作的节点，确保该 Pod 处于运行状态。如果 Pod 由于某种原因（即集群资源不足）启动失败，则不会进行 Restic 恢复。

Velero 为 Pod 中要恢复的每个卷创建一个 PodVolumeRestore，并等待其状态返回 completed 或者 failed。

与此同时，每个节点的 restic controller 会有一个 `/var/lib/kubelet/pods` 的 hostPath 卷挂载用来访问 Pod 的卷数据，等待 Pod 运行 init 容器，通过访问该 hostPath 卷，获取到待备份的 Pod 卷数据后，执行 `restic restore`，成功之后，将文件写入 Pod 卷中的 .velero 子目录中，名称是恢复任务的 UID，并根据实际情况将 PodVolumeRestore 状态设置为 completed 或者 failed。

```shell
$ ls -l /mnt/.velero
total 0
-rw-r--r--    1 root     root             0 Jan 17 06:28 b1f704be-7f60-475e-833f-4471544a2f87
```

当 init 容器在 .velero 下获取到所有的待恢复的卷信息后，便会成功退出，Pod 继续运行其他 init 容器/主进程。

## Troubleshooting

### 静态供应的 PV 的备份与恢复

默认情况下，Velero 会备份静态 PV，如 local pv、nfs pv 等，但是在恢复的时候，如果待恢复的对象中包含使用该 PV 的 Pod 时，Velero 并不会恢复 PV，而是默认由 StorageClass 动态供应创建 PV。此时， PVC 会处于 pending 状态（由于不存在 PV），Pod 也会处于 pending 状态（由于 PVC pending），restore 任务会等待直至超时。

目前来看 Velero 社区并没有将此视为 bug，建议在使用层面进行处理，方案的核心思路是如果恢复的 Pod 使用静态的 PV 时，需要确保恢复流程执行之前，存在一个可以满足 PVC 的 PV，例如以下两种方案：

**手动创建 PV**

也就是流程验证中的操作，通过手动创建 PV，来与 Velero 恢复的 PVC 进行绑定，完成后续恢复流程。

**单独备份 PV**

通过 `--include-resources pv` 单独备份静态 PV，在恢复之前，单独恢复静态 PV，具体流程为：

1. 备份静态 PV
2. 创建备份任务
3. 恢复静态 PV
4. 恢复备份任务

> https://github.com/vmware-tanzu/velero/issues/2520
