---
title: "「 Kubernetes 」InPlacePodVerticalScaling 特性"
excerpt: "Kubernetes v1.27 版本中的 Pod 资源原地纵向弹性伸缩特性与实践验证"
cover: https://picsum.photos/0?sig=20230710
thumbnail: /gallery/kubernetes/thumbnail.svg
date: 2023-07-10
toc: true
categories:
- Scheduling & Orchestration
- Kubernetes
tag:
- Kubernetes
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kubernetes/logo.svg"></div>

------

> based on **v1.27.3**

Kubernetes v1.27 版本中，添加了一个 alpha 版本的 feature gates — InPlacePodVerticalScaling，该特性允许用户在不重启容器的情况下调整分配给 Pod 的 CPU 或内存资源的大小。为了实现这一点，现在允许通过 patch 修改正在运行的 Pod resources 中 CPU 和内存资源。这也意味着 Pod spec 中 resources 字段不能再作为 Pod 实际资源的指标，监控工具类服务现在必须查看 Pod status 中的新字段。

[KEP-1287 提案](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1287-in-place-update-pod-resources)旨在改进 CRI，用于管理运行时容器的 CPU 和内存资源配置。扩展 UpdateContainerResources CRI，使其适用于 Windows 以及除 Linux 之外的其他未来运行时。此外，扩展 CRI 的 ContainerStatus API，允许 Kubelet 发现容器上配置的当前资源。

# 初衷

由于多种原因，分配给 Pod 容器的资源可能需要动态更改：

- Pod 负载大幅增加，当前资源不足
- Pod 负载显着下降，分配的资源未使用
- Pod 资源设置不正确

目前，由于 Pod 容器资源是不可变的，更改资源分配需要重新创建 Pod。虽然许多无状态工作负载可以接受此类中断，但有些工作负载更为敏感，尤其是在使用少量 Pod 副本时。此外，对于有状态或批量工作负载，Pod 重启会造成严重中断，导致可用性降低或运行成本升高。

因此，需要一种不重新创建 Pod 或重新启动容器的情况下更改资源的方案，即 InPlacePodVerticalScaling 特性，该特性依赖于 CRI 接口来更新 Pod 容器的 CPU 和内存的 requests 与 limits。

当前的 CRI 接口有一些需要解决的缺点：

- UpdateContainerResources API 接受的更改 Linux 容器资源的参数无法适用于 Windows 容器或未来可能出现的其他非 Linux 运行时
- CRI 中缺少 Kubelet 查询并发现容器运行时中配置的 CPU 和内存限制的机制
- 处理 UpdateContainerResources API 的预期行为并没有非常明确定义或记录

## 目标

- 允许更改容器资源 requests 和 limits，而无需重新启动容器
- 允许用户、VPA、StatefulSet、JobController 等角色决定在 Pod 无法就地调整资源大小时如何继续
- 允许用户指定哪些容器可以在不重新启动的情况下调整大小

此外，该提案为 CRI 设定了两个目标：

- 修改 UpdateContainerResources API，使其适用于 Windows 容器以及除 Linux 之外的其他运行时管理的容器
- CRI 提供查询容器运行时机制，用于获取当前应用于容器的 CPU 和内存资源配置

该提案的另一个目标是更好地定义和记录处理资源更新时容器运行时的预期行为。

## 非目标

提案明确非目标是避免介入未能就地资源调整大小的 Pod 的整个生命周期。这应该由发起调整大小的参与者来处理。其他确定的非目标是：

- 允许在不重新启动的情况下更改 Pod QoS 等级
- 无需重新启动即可更改 Init 容器的资源
- 驱逐优先级较低的 Pod 以方便调整 Pod 大小
- 更新扩展资源或除 CPU、内存之外的任何其他资源类型
- 支持除 None 策略之外的 CPU/内存管理器策略

该提案的目标并非是定义实现这些功能的详细或具体方式，实现细节留给运行时来确定，在预期行为的限制范围内即可。

# API 变更

API 的核心思想是让 Pod spec 中的容器资源 requests 和 limits 可变，对 Pod status 进行了扩展，以显示为 Pod 及其容器分配和应用的资源。

- Pod spec 中 resources 变成纯粹的声明，表示 Pod 资源的所需状态
- Pod status 中 containerStatuses 的 allocatedResources 字段反映了分配给 Pod 容器的资源
- Pod status 中 containerStatuses 的 resources 字段反映了如同容器运行时所报告的、针对正运行的容器配置的实际资源 requests 和 limits
- Pod status 中 resize 字段解释容器上给定资源发生的情况


新增的 allocatedResources 字段代表正在进行的调整大小操作。在考虑节点上可用资源空间时，Kube-scheduler 应使用 Pod spec 中的容器资源 requests 和 allocatedResources 中较大的值作为标准。

## 容器调整策略

resizePolicy 调整策略允许更精细地控制 Pod 中的容器如何针对 CPU 和内存资源进行调整。针对调整 CPU 和内存可以设置以下重启策略：

- `NotRequired`：默认值，如果可能的话，在不重新启动的情况下调整容器的大小
- `RestartContainer`：重启容器并在重启后应用新资源

NotRequired 调整大小的重新启动策略并不能保证容器不会重新启动。如果容器无法在不重新启动的情况下应用新资源，则运行时可能会选择停止容器；此外，容器的应用程序可以处理 CPU 资源的调整而不必重启， 但是调整内存可能需要应用程序重启，因此容器也必须重启。

如果同时更新具有不同策略的多种资源类型，则 RestartContainer 策略优先于 NotRequired 策略。

如果 Pod 的 restartPolicy 为 Never，则 Pod 中所有容器的调整重启策略必须被设置为 NotRequired，也就是说，如果无法就地调整大小，则任何就地调整大小的动作都可能导致容器停止，且无法重新启动。

## 调整状态大小

Pod status 中新增一个 resize 字段 ，用于表明 Kubelet 是否已接受或拒绝针对给定资源的建议调整大小操作。当 Pod spec 和 status 中 resources 不同时给出具体的原因：

- `Proposed`：表示请求调整已被确认，并且请求已被验证和记录

- `InProgress`：表示节点已接受调整请求，并正在将其应用于 Pod 的容器

- `Deferred`：意味着在此时无法批准请求的调整，节点将继续重试。当其他 Pod 退出并释放节点资源时，调整可能会被真正实施

- `Infeasible`：表示节点无法承接所请求的调整值。 如果所请求的调整超过节点可分配给 Pod 的最大资源，则可能会发生这种情况

每当 Kube-apiserver 收到调整资源的请求时，它都会自动将该字段设为 Proposed。

## CRI 变化

Kubelet 会调用 UpdateContainerResources API，该 API 目前采用 LinuxContainerResources 参数，但不适用于 Windows。因此，此参数更改为 ContainerResources，该参数与平台无关，并将包含特定于平台的信息，通过使 API 中传递的资源参数特定于目标运行时，使 UpdateContainerResources API 适用于 Windows 以及除 Linux 之外的任何其他未来运行时。

此外，ContainerStatus API 新增 ContainerResources 信息，以便允许 Kubelet 从运行时查询容器的 CPU 和内存限制配置，需要运行时返回当前应用于容器的 CPU 和内存资源值。

为了实现上述理念，涉及到如下的改动：

- 新的 protobuf 消息对象 ContainerResources 封装了 LinuxContainerResources 和 WindowsContainerResources。后续只需追加新运行时的资源结构，即可轻松扩展并适应未来的运行时

  ```go
  // ContainerResources holds resource configuration for a container.
  message ContainerResources {
      // Resource configuration specific to Linux container.
      LinuxContainerResources linux = 1;
      // Resource configuration specific to Windows container.
      WindowsContainerResources windows = 2;
  }
  ```

- ContainerStatus 消息对象新增 ContainerResources 字段，用于 Kubelet 使用 ContainerStatus API 查询运行时并发现当前应用于容器的资源

  ```go
  @@ -914,6 +912,8 @@ message ContainerStatus {
       repeated Mount mounts = 14;
       // Log path of container.
       string log_path = 15;
  +    // Resource configuration of the container.
  +    ContainerResources resources = 16;
   }
  ```

- UpdateContainerResources API 采用 ContainerResources 参数而不是 LinuxContainerResources

  ```go
  --- a/staging/src/k8s.io/cri-api/pkg/apis/services.go
  +++ b/staging/src/k8s.io/cri-api/pkg/apis/services.go
  @@ -43,8 +43,10 @@ type ContainerManager interface {
          ListContainers(filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error)
          // ContainerStatus returns the status of the container.
          ContainerStatus(containerID string) (*runtimeapi.ContainerStatus, error)
  -       // UpdateContainerResources updates the cgroup resources for the container.
  -       UpdateContainerResources(containerID string, resources *runtimeapi.LinuxContainerResources) error
  +       // UpdateContainerResources updates ContainerConfig of the container synchronously.
  +       // If runtime fails to transactionally update the requested resources, an error is returned.
  +       UpdateContainerResources(containerID string, resources *runtimeapi.ContainerResources) error
          // ExecSync executes a command in the container, and returns the stdout output.
          // If command exits with a non-zero exit code, an error is returned.
          ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
  ```

- Kubelet 代码对此也做了相应更改

# 设计细节

## Kubelet 与 Kube-apiserver 交互

对于新创建的 Pod，Kube-apiserver 将设置 allocatedResources 字段以匹配每个容器的资源请求量。当 Kubelet 接纳 Pod 时，allocatedResources 中的值用于确定是否有足够的空间接纳该 Pod，Kubelet 在接纳 Pod 时不会设置 allocatedResources。

当请求调整 Pod 大小时，Kubelet 会尝试更新分配给 Pod 及其容器的资源。Kubelet 首先通过计算节点中所有 Pod（正在调整大小的 Pod 除外）分配的资源总和（即 allocatedResources）来检查新的所需资源是否适合节点可分配资源。对于调整大小的 Pod，它将新的所需资源（即 Spec.Containers[i].Resources.Requests）添加到总和中。

- 如果新的所需资源适合，Kubelet 通过更新 allocatedResources 字段并将 `Status.Resize` 设置为 InProgress 来接受调整大小。然后，Kubelet 调用 UpdateContainerResources API 来更新容器资源限制。成功更新所有容器后，它会更新 Pod status 中的 resources 字段，以反映新的资源值并取消设置 resize 字段
- 如果新的所需资源不适合，Kubelet 会将 resize 字段更新为 Infeasible，并且不会对调整大小进行操作
- 如果新的所需资源适合但目前正在使用，Kubelet 会将 resize 字段更新为 Deferred

除了上述内容之外，每当接受或拒绝调整大小时，以及如果可能的话，在调整大小过程中的关键步骤上，Kubelet 都会在 Pod 上生成事件。

如果多个 Pod 需要调整大小，则会按照 Kubelet 定义的顺序（例如，按到达顺序）处理它们；Kube-scheduler 可以并行地将新的 Pod 分配给节点，如果发生竞争情况，也就是 Pod 调整大小后节点没有空间，Kubelet 将通过拒绝新 Pod 来解决该问题。

**Kubelet 重启容忍度**

如果 Kubelet 在处理 Pod 大小调整过程中发生重启，则在重新启动时，所有 Pod 都会以其当前的 allocatedResources 值被接纳，并在添加所有现有 Pod 后处理调整大小。这可确保调整大小不会影响之前的 Pod。

## Kube-scheduler 和 Kube-apiserver 交互

Kube-scheduler 使用 Pod spec 中 resources 的资源 request 来调度新的 Pod，并继续 watch Pod 更新并更新其缓存。为了计算分配给 Pod 的节点资源，它必须考虑待处理的调整大小，如 resize 所述：

- 对于 resize 为 InProgress 或 Infeasible 的容器，可以简单地使用 allocatedResources 
- 对于 resize 为 Proposed 的容器，假设调整大小被接受。因此，必须使用 Pod spec 中 resources 的资源 request 和 allocatedResources 值中较大的那个

## 工作流

```shell
T=0: A new pod is created
    - `spec.containers[0].resources.requests[cpu]` = 1
    - all status is unset

T=1: apiserver defaults are applied
    - `spec.containers[0].resources.requests[cpu]` = 1
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1
    - `status.resize[cpu]` = unset

T=2: kubelet runs the pod and updates the API
    - `spec.containers[0].resources.requests[cpu]` = 1
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1
    - `status.resize[cpu]` = unset
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=3: Resize #1: cpu = 1.5 (via PUT or PATCH or /resize)
    - apiserver validates the request (e.g. `limits` are not below
      `requests`, ResourceQuota not exceeded, etc) and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 1.5
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=4: Kubelet watching the pod sees resize #1 and accepts it
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
        `status.resize[cpu]` = "InProgress"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.5
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "InProgress"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=5: Resize #2: cpu = 2
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 2
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=6: Container runtime applied cpu=1.5
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.5
        `status.resize[cpu]` = unset
      }
    - apiserver fails the operation with a "conflict" error

T=7: kubelet refreshes and sees resize #2 (cpu = 2)
    - kubelet decides this is possible, but not right now
    - kubelet sends patch {
        `resourceVersion` = `<updated value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.5
        `status.resize[cpu]` = "Deferred"
      }
    - `spec.containers[0].resources.requests[cpu]` = 2
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "Deferred"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=8: Resize #3: cpu = 1.6
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=9: Kubelet watching the pod sees resize #3 and accepts it
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
        `status.resize[cpu]` = "InProgress"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = "InProgress"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=10: Container runtime applied cpu=1.6
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.6
        `status.resize[cpu]` = unset
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = unset
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6

T=11: Resize #4: cpu = 100
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 100
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6

T=12: Kubelet watching the pod sees resize #4
    - this node does not have 100 CPUs, so kubelet cannot accept
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.resize[cpu]` = "Infeasible"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 100
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = "Infeasible"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6
```

**CRI 工作流**

下图概述了 Kubelet 使用 UpdateContainerResources 和 ContainerStatus CRI API 设置新的容器资源限制，并更新 Pod status 以响应用户更改 Pod spec 中所需资源的情况。

```shell
   +-----------+                   +-----------+                  +-----------+
   |           |                   |           |                  |           |
   | apiserver |                   |  kubelet  |                  |  runtime  |
   |           |                   |           |                  |           |
   +-----+-----+                   +-----+-----+                  +-----+-----+
         |                               |                              |
         |       watch (pod update)      |                              |
         |------------------------------>|                              |
         |     [Containers.Resources]    |                              |
         |                               |                              |
         |                            (admit)                           |
         |                               |                              |
         |                               |  UpdateContainerResources()  |
         |                               |----------------------------->|
         |                               |                         (set limits)
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |                               |      ContainerStatus()       |
         |                               |----------------------------->|
         |                               |                              |
         |                               |     [ContainerResources]     |
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |      update (pod status)      |                              |
         |<------------------------------|                              |
         | [ContainerStatuses.Resources] |                              |
         |                               |                              |
```

- Kubelet 在 ContainerManager 接口中调用 UpdateContainerResources() CRI API，通过在 API 的 ContainerResources 参数中指定这些值来为容器配置新的 CPU 和内存限制。 Kubelet 在调用此 CRI API 时设置特定于目标运行时平台的 ContainerResources 参数
- Kubelet 在 ContainerManager 接口中调用 ContainerStatus() CRI API 来获取应用于 Container 的 CPU 和内存限制。它使用 ContainerStatus.Resources 返回的值来更新 Pod 状态中该容器的 ContainerStatuses[i].Resources.Limits

## 注意事项

- 如果节点 CPU Manager 策略为 static，则只允许整数值的 CPU 调整大小。如果请求非整数 CPU 调整大小，则将被拒绝，并在事件流中记录错误消息
- 所有组件在计算 Pod 使用的资源时都将使用 allocatedResources
- 如果在调整 Pod 大小时收到其他调整大小请求，这些请求将在当前完成后处理，并且调整大小会朝着最新的期望状态进行
- 如果应用正在占用内存页，降低内存限制可能并不能很快生效。 Kubelet 将使用控制循环来设置接近使用的内存限制，以强制回收，并仅在限制达到所需值时更新 Pod status 中的 resources
- Pod Overhead 的影响：Kubelet 将 Pod Overhead 添加到调整大小请求中，以确定是否可以就地调整大小
- 目前，VPA 不应该与 CPU、内存上的 HPA 一起使用。此 KEP 不会改变该限制

**受影响的组件**

- Pod v1 core API
- Admission Controllers：LimitRanger 和 ResourceQuota
- Kubelet
- Kube-scheduler
- 其他使用相关语义的 Kubernetes 组件

# 实践验证

InPlacePodVerticalScaling 特性需要开启相应的 feature gates：<br>*不开启时，仍然视为 Pod spec 中容器资源 requests 和 limits 不可变更。*

```shell
# Kube-apiserver 服务其中参数中新增 --feature-gates=InPlacePodVerticalScaling=true
$ cat /etc/kubernetes/manifests/kube-apiserver.yaml
    - --tls-cert-file=/etc/kubernetes/pki/apiserver.crt
    - --tls-private-key-file=/etc/kubernetes/pki/apiserver.key
    - --feature-gates=InPlacePodVerticalScaling=true
    
# Kubelet 参数中新增 featureGates
$ cat /var/lib/kubelet/config.yaml
syncFrequency: 0s
volumeStatsAggPeriod: 0s
featureGates: 
  InPlacePodVerticalScaling: true
```

测试服务如下：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: demo
spec:
  containers:
    - name: demo
      image: ubuntu:18.04
      command: ["/bin/bash", "-c", "tail -f /dev/null"]
      resizePolicy:
        - resourceName: cpu
          restartPolicy: NotRequired
        - resourceName: memory
          restartPolicy: RestartContainer
      resources:
        limits:
          memory: "200Mi"
          cpu: "1000m"
        requests:
          memory: "200Mi"
          cpu: "1000m"
```

Pod 运行后此时的状态信息为：

```yaml
spec:
  containers:
  - resizePolicy:
    - resourceName: cpu
      restartPolicy: NotRequired
    - resourceName: memory
      restartPolicy: RestartContainer
    resources:
      limits:
        cpu: 1
        memory: 200Mi
      requests:
        cpu: 1
        memory: 200Mi
status:
  containerStatuses:
  - allocatedResources:
      cpu: 1000m
      memory: 200Mi
    resources:
      limits:
        cpu: 1
        memory: 200Mi
      requests:
        cpu: 1
        memory: 200Mi
```

此时，将 Pod CPU 由 1C 调整为 2C：

```shell
$ kubectl patch pod demo --patch '{"spec":{"containers":[{"name":"demo", "resources":{"requests":{"cpu":"2000m"}, "limits":{"cpu":"2000m"}}}]}}'
```

可以看到，Pod resize 处于 InProgress 状态，allocatedResources 已经调整为预期规格，status 的 resources 暂未变化。在调整结束后，status 的 resources 会跟进更新，resize 字段重置为空。

```yaml
spec:
  containers:
  - resizePolicy:
    - resourceName: cpu
      restartPolicy: NotRequired
    - resourceName: memory
      restartPolicy: RestartContainer
    resources:
      limits:
        cpu: 2
        memory: 200Mi
      requests:
        cpu: 2
        memory: 200Mi
status:
  containerStatuses:
  - allocatedResources:
      cpu: "2"
      memory: 200Mi
    resources:
      limits:
        cpu: "1"
        memory: 200Mi
      requests:
        cpu: "1"
        memory: 200Mi
  resize: InProgress
```

调整之后，Pod cgroup 信息也跟着发生变化：

```shell
$ cat /sys/fs/cgroup/cpu/kubepods.slice/kubepods-podab959cd5_f9e3_4b34_8051_861f7caca04c.slice/cpu.cfs_period_us
100000

$ cat /sys/fs/cgroup/cpu/kubepods.slice/kubepods-podab959cd5_f9e3_4b34_8051_861f7caca04c.slice/cpu.cfs_quota_us
200000
```

InPlacePodVerticalScaling 特性不允许修改 Pod QoS：

```shell
$ kubectl patch pod demo --patch '{"spec":{"containers":[{"name":"demo", "resources":{"requests":{"cpu":"2000m"}, "limits":{"cpu":"3000m"}}}]}}'
The Pod "demo" is invalid: metadata: Invalid value: "Burstable": Pod QoS is immutable
```

当修改请求中，资源无法满足时，除 Pod spec 的 resources 变化外，allocatedResources、status 的 resources 以及 cgroup 等信息均未变化，resize 状态为 Infeasible，服务仍在运行，当集群资源满足时会自动调整。

当修改的资源 restartPolicy 为 RestartContainer 时，会触发一次重启操作：

```shell
$ kubectl get pod 
NAME   READY   STATUS    RESTARTS       AGE
demo   1/1     Running   1 (11s ago)    9m8s
```

当请求缩小内存资源时，通过容器内部 free 看到的内存仍未变化，但是 cgroup 中已经更新：

```shell
$ kubectl exec -it demo free 
kubectl exec [POD] [COMMAND] is DEPRECATED and will be removed in a future version. Use kubectl exec [POD] -- [COMMAND] instead.
              total        used        free      shared  buff/cache   available
Mem:       12057632     2150040     6937804      167128     2969788     9441392
Swap:             0           0           0

$ cat /sys/fs/cgroup/memory/kubepods.slice/kubepods-poddf0fdfe6_59d6_4ffb_9a8d_0c014c182cd0.slice/memory.limit_in_bytes
524288000
```

**troubleshooting**

当 Kubelet CPU Manager 策略为 static 时，调整 CPU 之后，发现并未生效，并且设置非整数的 CPU 时也未报错

```shell
$ cat /sys/fs/cgroup/cpuset/kubepods-pod243ca361_5bde_4b8d_b5f6_522961c3ae11.slice:cri-containerd:9bfaa6d8f87fdcbe22851c1fdcbcb662e0fbf025e1299f2cf4507f09da95de4b/cpuset.cpus
3

$ kubectl patch pod demo --patch '{"spec":{"containers":[{"name":"demo", "resources":{"requests":{"cpu":"3000m"}, "limits":{"cpu":"3000m"}}}]}}'

# 调整 CPU 后，CPUset 未发生变化，但是 CPU 限制配额却更新了
$ cat /sys/fs/cgroup/cpuset/kubepods-pod243ca361_5bde_4b8d_b5f6_522961c3ae11.slice:cri-containerd:9bfaa6d8f87fdcbe22851c1fdcbcb662e0fbf025e1299f2cf4507f09da95de4b/cpuset.cpus
3
$ cat /sys/fs/cgroup/cpu/kubepods.slice/kubepods-pod243ca361_5bde_4b8d_b5f6_522961c3ae11.slice/cpu.cfs_quota_us
300000
$ cat /sys/fs/cgroup/cpu/kubepods.slice/kubepods-pod243ca361_5bde_4b8d_b5f6_522961c3ae11.slice/cpu.cfs_period_us
100000

# Pod 可用 CPU 也仍然为 1，通过 stress 模拟也是一样
$ kubectl exec -it demo nproc
kubectl exec [POD] [COMMAND] is DEPRECATED and will be removed in a future version. Use kubectl exec [POD] -- [COMMAND] instead.
1
```

