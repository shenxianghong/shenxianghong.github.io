---
title: "「 Kubernetes 」节点资源超卖"
excerpt: "基于 Pod QoS 的混部方案实现 Kubernetes 节点资源超卖的理念探索与优化"
cover: https://picsum.photos/0?sig=20220711
thumbnail: https://github.com/cncf/artwork/raw/master/projects/kubernetes/stacked/color/kubernetes-stacked-color.svg
date: 2023-06-13
toc: true
categories:
- Scheduling & Orchestration
- Kubernetes
tag:
- Kubernetes
---

<div align=center><img width="200" style="border: 0px" src="https://github.com/cncf/artwork/raw/master/projects/kubernetes/horizontal/color/kubernetes-horizontal-color.svg"></div>

------

> based on **v1.24.10**

# 概述

Kubernetes 设计原语中，Pod 的 spec.resources.requests 用于描述服务容器所需的最小规格，Kube-scheduler 会根据资源请求量运行调度流程，并在全局资源视图扣除这部分资源，spec.resources.limits 用于限制服务容器的最大资源使用量，避免容器服务使用过多的资源导致节点性能下降或崩溃。Kubelet 通过参考 Pod 的 QoS 等级来管理容器的资源质量，例如 OOM（Out of Memory）优先级控制等。Pod 的 QoS 级别分为 Guaranteed、Burstable 和 BestEffort。QoS 级别并不是显式定义，而是取决于 Pod 配置的 spec.resources.requests 和 spec.resources.limits 中声明的 CPU 与内存。

而在实际使用过程中，为了提高稳定性，应用管理员在提交 Guaranteed 和 Burstable 这两类 QoS Pod 时会预留相当数量的资源缓冲来应对上下游链路的负载波动，在大部分时间段，服务的 spec.resources.requests 会远高于实际的资源利用率。为了提升集群资源利用率，应用管理员会提交一些 BestEffort QoS 的低优任务，来充分使用那些已分配但未使用的资源，实现对集群资源的超卖，即基于 Pod QoS 的服务混部，但是这种基础的混部方案存在一些弊端：

- 节点可容纳低优任务的资源量没有任何参考，即使节点实际负载已经很高，由于 BestEffort 任务在资源规格上缺少容量约束，仍然会被调度到节点上运行
- BestEffort 任务间缺乏公平性保证，任务资源规格存在区别，但无法在 Pod 描述上体现

# 设计理念

