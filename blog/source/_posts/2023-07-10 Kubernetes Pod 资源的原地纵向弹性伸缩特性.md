---
title: "「 Kubernetes 」InPlacePodVerticalScaling 特性"
excerpt: "Kubernetes v1.27 版本中的 Pod 资源原地纵向弹性伸缩特性的实践验证"
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

# InPlacePodVerticalScaling 

在 Kubernetes v1.27 中，添加了一个新的 alpha 功能 — InPlacePodVerticalScaling，允许用户在不重启容器的情况下调整分配给 Pod 的 CPU 或内存资源的大小。为了实现这一点，Pod resources 字段现在允许对 CPU 和内存资源进行更改，可以通过 patch 修改正在运行的 Pod spec 来实现。

这也意味着 Pod spec 中 resources 字段不能再作为 Pod 实际资源的指标。监控工具和其他此类应用程序现在必须查看 Pod status 中的新字段。Kubernetes 通过 CRI API 调用运行时来查询实际的 CPU 和内存的 request 与 limit，来自容器运行时的响应反映在 Pod 的 status 中。

对于原地调整 Pod 资源而言：

- 针对 CPU 和内存资源的容器的 requests 和 limits 是可变更的

- Pod 状态中 containerStatuses 的 `allocatedResources` 字段反映了分配给 Pod 容器的资源

- Pod 状态中 containerStatuses 的 resources 字段反映了如同容器运行时所报告的、针对正运行的容器配置的实际资源 requests 和 limits

- Pod 状态中 `resize` 字段显示上次请求待处理的调整状态：

  - `Proposed`：此值表示请求调整已被确认，并且请求已被验证和记录

  - `InProgress`：此值表示节点已接受调整请求，并正在将其应用于 Pod 的容器

  - `Deferred`：此值意味着在此时无法批准请求的调整，节点将继续重试。 当其他 Pod 退出并释放节点资源时，调整可能会被真正实施

  - `Infeasible`：此值是一种信号，表示节点无法承接所请求的调整值。 如果所请求的调整超过节点可分配给 Pod 的最大资源，则可能会发生这种情况

**容器调整策略**

调整策略允许更精细地控制 Pod 中的容器如何针对 CPU 和内存资源进行调整。 例如，容器的应用程序可以处理 CPU 资源的调整而不必重启， 但是调整内存可能需要应用程序重启，因此容器也必须重启。

为了实现这一点，容器规约允许用户指定 `resizePolicy`。 针对调整 CPU 和内存可以设置以下重启策略：

- `NotRequired`：在运行时调整容器的资源，默认值<br>*如果 Pod 的 restartPolicy 为 Never，则 Pod 中所有容器的调整重启策略必须被设置为 NotRequired*
- `RestartContainer`：重启容器并在重启后应用新资源

