---
title: "「 Kubernetes 」节点资源超卖"
excerpt: "基于 Pod QoS 的混部方案实现 Kubernetes 节点资源超卖的理念探索与优化"
cover: https://picsum.photos/0?sig=20230613
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

# 背景

Kubernetes 设计原语中，Pod 的 spec.resources.requests 用于描述容器所需资源的最小规格，Kube-scheduler 会根据资源请求量执行调度流程，并在节点资源视图中扣除；spec.resources.limits 用于限制容器资源最大使用量，避免容器服务使用过多的资源导致节点性能下降或崩溃。Kubelet 通过参考 Pod 的 QoS 等级来管理容器的资源质量，例如 OOM 优先级控制等。Pod 的 QoS 级别分为 Guaranteed、Burstable 和 BestEffort，QoS 级别并不是显式定义，而是取决于 Pod 配置的 spec.resources.requests 和 spec.resources.limits 中声明的 CPU 与内存。

而在实际使用过程中，为了提高稳定性，应用管理员在提交 Guaranteed 和 Burstable 这两类 QoS Pod 时会预留相当数量的资源缓冲来应对上下游链路的负载波动，在大部分时间段，服务的资源请求量会远高于实际的资源使用率。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/twitter.png"></div>

为了提升集群资源利用率，应用管理员会提交一些 BestEffort QoS 的低优任务，来充分使用那些已分配但未使用的资源，也就是基于 Pod QoS 的服务混部（co-location）以实现 Kubernetes 节点资源的超卖（overcommitted）。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/overcommitted.png"></div>

这种策略常用于容器服务平台的在离线业务混部，但是这种基础的混部方案存在一些弊端：

- 混部会带来底层共享资源（CPU、内存、网络、磁盘等）的竞争，会导致在线业务性能下降，并且这种下降是不可预测的
- 节点可容纳低优任务的资源量没有任何参考，即使节点实际负载已经很高，由于 BestEffort 任务在资源规格上缺少容量约束，仍然会被调度到节点上运行
- BestEffort 任务间缺乏公平性保证，任务资源规格存在区别，但无法在 Pod 描述上体现

# 设计理念

在基于 Pod QoS 的混部实现的 Kubernetes 节点资源超卖方案中，所要解决的核心问题是如何科学合理的利用缓冲资源，即 request buffer 与 limit buffer。其中，limit buffer 在 Kubernetes 设计中天然支持超卖，区别于 request，Pod 在声明 spec.resources.limits 时，无需考虑集群资源剩余情况：

```shell
Allocated resources:
  (Total limits may be over 100 percent, i.e., overcommitted.)
  Resource                    Requests      Limits
  --------                    --------      ------
  cpu                         7100m (88%)   9500m (118%)
  memory                      2122Mi (18%)  7086Mi (60%)
  ephemeral-storage           0 (0%)        0 (0%)
  hugepages-1Gi               0 (0%)        0 (0%)
  hugepages-2Mi               0 (0%)        0 (0%)
```

因此，节点资源超卖设计理念更多的是针对 request 的 buffer 科学利用。

## 资源回收

资源回收是指回收业务应用已申请的，但是目前还空闲的资源，将其给低优业务使用。这部分资源是低质量的，没有太高的可用性保证。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/reclaim.png"></div>

reclaimed 资源代表可动态超卖的资源量，这部分需要根据节点真实负载情况动态更新，并以标准扩展资源的形式实时更新到 Kubernetes 的 Node 元信息中。低优任务可以通过在 spec.resources.requests 和 spec.resources.limits 中定义的 reclaimed 资源配置来使用这部分资源，这部分配置同时也会体现在节点侧的资源限制参数上，保证低优作业之间的公平性。

可回收资源的推导公式大致如下：

> reclaimed = nodeAllocatable * thresholdPercent - podUsage - systemUsage
>
> nodeAllocatable：节点可分配资源总量
> thresholdPercent ：预留水位比例
> podUsage：高优 Pod 的资源使用量
> systemUsage：系统资源真实使用量

## 负载感知调度

现阶段，原生 Kube-scheduler 主要基于资源的分配率情况进行调度，这种行为本质上是静态调度，也就是根据容器的资源请求（spec.resources.requests）执行调度算法，而非考虑容器的实际资源使用率，也就是节点的实际负载。所以，经常会发生节点负载较低，但是却无法满足 Pod 调度要求。

<div align=center><img width="400" style="border: 0px" src="/gallery/overcommitted/static-schedule-1.png"></div>

另外，静态调度会导致节点之间的负载不均衡，有的节点资源利用率很高，而有的节点资源利用率很低。Kubernetes 在调度时是有一个负载均衡优选调度算法（LeastRequested）的，但是它调度均衡的依据是资源请求量而不是节点实际的资源使用率。

<div align=center><img width="400" style="border: 0px" src="/gallery/overcommitted/static-schedule-2.png"></div>

因此，调度算法中的预选与优选阶段需要新增节点实际负载情况的考量，也就是基于节点实际负载实现动态调度机制。

## 资源限制

为了低优任务间公平性保证，Pod 描述中需要体现任务资源规格，即 spec.resources.limits。

由于 Kubelet cgroup manager 不支持接口扩展，往往需要借助 agent 类型的组件实现容器 cgroup 更新。

## 热点打散重调度

节点的利用率会随着时间、集群环境、工作负载的流量或请求等动态变化，导致集群内节点间原本负载均衡的情况被打破，甚至有可能出现极端负载不均衡的情况，影响到工作负载运行时质量。因此需要提供重调度能力，可以持续优化节点的负载情况，通过将负载感知调度和热点打散重调度结合使用，可以获得集群最佳的负载均衡效果。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/descheduler.png"></div>

# 社区成果

