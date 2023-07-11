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
- CRI 机制中缺少 Kubelet 查询并发现容器运行时中配置的 CPU 和内存限制
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
- Pod status 中 containerStatuses 的 `allocatedResources` 字段反映了分配给 Pod 容器的资源
- Pod status 中 containerStatuses 的 resources 字段反映了如同容器运行时所报告的、针对正运行的容器配置的实际资源 requests 和 limits
- Pod status 中 `resize` 解释容器上给定资源发生的情况


新增的 allocatedResources 字段代表正在进行的调整大小操作，由节点 checkpoint 中保留的状态驱动。在考虑节点上可用资源空间时，Kube-scheduler 应使用 Spec.Containers[i].Resources 和 Status.ContainerStatuses[i].AllocatedResources 中较大的值作为标准。

## 容器调整策略

`resizePolicy` 调整策略允许更精细地控制 Pod 中的容器如何针对 CPU 和内存资源进行调整。针对调整 CPU 和内存可以设置以下重启策略：

- `NotRequired`：默认值，如果可能的话，在不重新启动的情况下调整容器的大小
- `RestartContainer`：重启容器并在重启后应用新资源

NotRequired 调整大小的重新启动策略并不能保证容器不会重新启动。如果容器无法在不重新启动的情况下应用新资源，则运行时可能会选择停止容器；此外，容器的应用程序可以处理 CPU 资源的调整而不必重启， 但是调整内存可能需要应用程序重启，因此容器也必须重启。

如果同时更新具有不同策略的多种资源类型，则 `RestartContainer` 策略优先于 `NotRequired` 策略。

如果 Pod 的 restartPolicy 为 `Never`，则 Pod 中所有容器的调整重启策略必须被设置为 `NotRequired`，也就是说，如果无法就地调整大小，则任何就地调整大小的动作都可能导致容器停止，且无法重新启动。

## 调整状态大小

Pod status 中新增一个 resize 字段 ，用于表明 Kubelet 是否已接受或拒绝针对给定资源的建议调整大小操作。当  `spec.Containers[i].Resources.Requests` 与实际 `status.ContainerStatuses[i].Resources` 不同时给出具体的原因：

- `Proposed`：表示请求调整已被确认，并且请求已被验证和记录

- `InProgress`：表示节点已接受调整请求，并正在将其应用于 Pod 的容器

- `Deferred`：意味着在此时无法批准请求的调整，节点将继续重试。当其他 Pod 退出并释放节点资源时，调整可能会被真正实施

- `Infeasible`：表示节点无法承接所请求的调整值。 如果所请求的调整超过节点可分配给 Pod 的最大资源，则可能会发生这种情况

每当 Kube-apiserver 收到调整资源的请求时，它都会自动将该字段设为 Proposed。

## CRI 变化

Kubelet 调用 UpdateContainerResources API，该 API 目前采用 runtimeapi.LinuxContainerResources 参数，但不适用于 Windows。因此，此参数更改为 runtimeapi.ContainerResources，该参数与平台无关，并将包含特定于平台的信息，通过使 API 中传递的资源参数特定于目标运行时，使 UpdateContainerResources API 适用于 Windows 以及除 Linux 之外的任何其他未来运行时。

此外，ContainerStatus API 新增 runtimeapi.ContainerResources 信息，以便允许 Kubelet 从运行时查询容器的 CPU 和内存限制配置，需要运行时返回当前应用于容器的 CPU 和内存资源值。

为了实现上述理念，涉及到如下的改动：

- 新的 protobuf 消息对象 ContainerResources，它封装了 LinuxContainerResources 和 WindowsContainerResources。只需将新的特定于运行时的资源结构添加到 ContainerResources 消息中，即可轻松扩展并适应未来的运行时

  ```go
  // ContainerResources holds resource configuration for a container.
  message ContainerResources {
      // Resource configuration specific to Linux container.
      LinuxContainerResources linux = 1;
      // Resource configuration specific to Windows container.
      WindowsContainerResources windows = 2;
  }
  ```

- ContainerStatus 消息扩展为返回 ContainerResources，如下所示。这使得 Kubelet 能够使用 ContainerStatus API 查询运行时并发现当前应用于容器的资源

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
