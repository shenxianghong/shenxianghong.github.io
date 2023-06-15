---
title: "「 Kubernetes 」节点资源超卖"
excerpt: "基于 Pod QoS 混部实现 Kubernetes 节点资源超卖方案的探索与优化"
cover: https://picsum.photos/0?sig=20230613
thumbnail: /gallery/kubernetes/thumbnail.svg
date: 2023-06-13
toc: true
categories:
- Scheduling & Orchestration
- Kubernetes
tag:
- Kubernetes
---

<div align=center><img width="200" style="border: 0px" src="/gallery/kubernetes/logo.svg"></div>

------

> based on **v1.24.10**

# 背景

Kubernetes 设计原语中，Pod 声明的 spec.resources.requests 用于描述容器所需资源的最小规格，Kube-scheduler 会根据资源请求量执行调度流程，并在节点资源视图中扣除；spec.resources.limits 用于限制容器资源最大使用量，避免容器服务使用过多的资源导致节点性能下降或崩溃。Kubelet 通过参考 Pod 的 QoS 等级来管理容器的资源质量，例如 OOM 优先级控制等。Pod 的 QoS 级别分为 Guaranteed、Burstable 和 BestEffort，QoS 级别并不是显式定义，而是取决于 Pod 声明的 spec.resources.requests 和 spec.resources.limits 中 CPU 与内存。

而在实际使用过程中，为了提高稳定性，应用管理员在提交 Guaranteed 和 Burstable 这两类 QoS Pod 时会预留相当数量的资源缓冲来应对上下游链路的负载波动，在大部分时间段，服务的资源请求量会远高于实际的资源使用率。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/twitter.png"></div>

为了提升集群资源利用率，应用管理员会提交一些 BestEffort QoS 的低优任务，来充分使用那些已分配但未使用的资源。即基于 Pod QoS 的服务混部（co-location）以实现 Kubernetes 节点资源的超卖（overcommitted）。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/overcommitted.png"></div>

这种策略常用于容器服务平台的在离线业务混部，但是这种基础的混部方案存在一些弊端：

- 混部会带来底层共享资源（CPU、内存、网络、磁盘等）的竞争，会导致在线业务性能下降，并且这种下降是不可预测的
- 节点可容纳低优任务的资源量没有任何参考，即使节点实际负载已经很高，由于 BestEffort 任务在资源规格上缺少容量约束，仍然会被调度到节点上运行
- BestEffort 任务间缺乏公平性保证，任务资源规格存在区别，但无法在 Pod 描述上体现

# 设计思考

在基于 Pod QoS 混部实现的 Kubernetes 节点资源超卖方案中，所要解决的核心问题是如何充分合理的利用缓冲资源，即 request buffer 与 limit buffer。

其中，limit buffer 在 Kubernetes 设计中天然支持超卖，Pod 在声明 spec.resources.limits 时，不受集群剩余资源的影响，集群中 Pod limits 之和也存在超出节点资源容量的情况，limit buffer 部分的资源是共享抢占的；而 request buffer 部分的资源是逻辑独占的，也就是说 spec.resources.requests 的大小会决定 Pod 能否调度，进而直接影响到节点资源的使用率。

因此，节点资源超卖理念更多的是对 request buffer 如何充分利用的思考。

## 资源回收与超卖

资源回收是指回收业务应用已申请的，目前还处于空闲的资源，将其给低优业务使用。但是这部分资源是低质量的，不具备太高的可用性保证。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/reclaim.png"></div>

如图所示，reclaimed 资源代表可动态超卖的资源量，这部分需要根据节点真实负载情况动态更新，并以标准扩展资源的形式实时更新到 Kubernetes 的 Node 元信息中。低优任务可以通过在 spec.resources.requests 和 spec.resources.limits 中定义的 reclaimed 资源配置来使用这部分资源，这部分配置同时也会体现在节点侧的资源限制参数上，保证低优作业之间的公平性。

可回收资源的推导公式大致如下：

> reclaimed = nodeAllocatable * thresholdPercent - podUsage - systemUsage

- *nodeAllocatable — 节点可分配资源总量*
- *thresholdPercent — 预留水位比例*
- *podUsage — 高优任务 Pod 的资源使用量*
- *systemUsage — 系统资源真实使用量*

## 弹性资源限制

原生的 BestEffort 应用缺乏资源用量的公平保证，而使用动态资源的 BestEffort 应用需要保证其 CPU 使用量被限制在其允许使用的合理范围内，避免在不同 QoS 混部的场景下对高优 Pod 的干扰，确保整机的资源使用率控制在安全水位之下。

考虑到 Kubelet cgroup manager 不支持接口扩展，所以需要借助 agent 类型的组件维护容器的 cgroup，同时在 CPU 竞争时也能按照各自声明量公平竞争。

## 负载感知调度

现阶段，原生 Kube-scheduler 主要基于资源的分配率情况进行调度，这种行为本质上是静态调度，也就是根据容器的资源请求（spec.resources.requests）执行调度算法，而非考虑节点的实际资源使用率与负载。所以，经常会发生节点负载较低，但是却无法满足 Pod 调度要求。

<div align=center><img width="400" style="border: 0px" src="/gallery/overcommitted/static-schedule-1.png"></div>

另外，静态调度会导致节点之间的负载不均衡，有的节点资源利用率很高，而有的节点资源利用率很低。Kubernetes 在调度时是有一个负载均衡优选调度算法（LeastRequested）的，但是它调度均衡的依据是资源请求量而不是节点实际的资源使用率。

<div align=center><img width="400" style="border: 0px" src="/gallery/overcommitted/static-schedule-2.png"></div>

因此，调度算法中的预选与优选阶段需要新增节点实际负载情况的考量，也就是需要引入基于节点实际负载实现动态调度机制。

## 热点打散重调度

节点的利用率会随着时间、集群环境、工作负载的流量或请求等动态变化，导致集群内节点间原本负载均衡的情况被打破，甚至有可能出现极端负载不均衡的情况，影响到工作负载运行时质量。因此需要提供重调度能力，可以持续优化节点的负载情况，通过将负载感知调度和热点打散重调度结合使用，可以获得集群最佳的负载均衡效果。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/descheduler.png"></div>

# 社区成果

## Crane

<div align=center><img width="800" style="border: 0px" src="/gallery/overcommitted/crane-overview.png"></div>

Crane 是一个基于 FinOps 的云资源分析与成本优化平台。它的愿景是在保证客户应用运行质量的前提下实现极致的降本。

**负载感知调度**

<div align=center><img width="800" style="border: 0px" src="/gallery/overcommitted/crane-scheduler.png"></div>

Crane-scheduler 是一组基于 [scheduler framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/) 的调度插件，依赖于 Prometheus 和 Node-exporter 收集和汇总指标数据，它由两个组件组成：

- Node-annotator 定期从 Prometheus 拉取数据，并以注释的形式在节点上用时间戳标记它们
- Dynamic plugin 直接从节点的注释中读取负载数据，过滤并基于简单的算法对候选节点进行评分

动态调度器提供了一个默认值调度策略并支持用户自定义策略。默认策略依赖于以下指标：

- cpu_usage_avg_5m
- cpu_usage_max_avg_1h
- cpu_usage_max_avg_1d
- mem_usage_avg_5m
- mem_usage_max_avg_1h
- mem_usage_max_avg_1d

在调度的 Filter 阶段，如果该节点的实际使用率大于上述任一指标的阈值，则该节点将被过滤。而在 Score 阶段，最终得分是这些指标值的加权和。

在生产集群中，可能会频繁出现调度热点，因为创建 Pod 后节点的负载不能立即增加。因此，Crane 定义了一个额外的指标，名为 Hot Value，表示节点最近几次的调度频率。并且节点的最终优先级是最终得分减去 Hot Value。

**弹性资源超卖**

Crane 通过如下两种方式收集了节点的空闲资源量，综合后作为节点的空闲资源量，增强了资源评估的准确性：<br>*这里以 CPU 为例，同时也支持内存的空闲资源回收和计算。*

1. 通过本地收集的 CPU 用量信息

   > nodeCpuCannotBeReclaimed = nodeCpuUsageTotal + exclusiveCPUIdle - extResContainerCpuUsageTotal

   exclusiveCPUIdle 是指被 CPU manager 策略 为 exclusive 的 Pod 占用的 CPU 的空闲量，虽然这部分资源是空闲的，但是因为独占的原因，是无法被复用的，因此加上被算作已使用量<br>*exclusive  策略是 Crane 在精细化管理 CPU 中提出的概念，该策略对应 Kubelet 的 static 策略，Pod 会独占 CPU 核心，其他任何 Pod 都无法使用*

   extResContainerCpuUsageTotal 是指被作为动态资源使用的 CPU 用量，需要减去以免被二次计算

2. 创建节点 CPU 使用量的 TSP，默认情况下自动创建，会根据历史预测节点 CPU 用量

   ```yaml
   apiVersion: v1
   data:
     spec: |
       predictionMetrics:
       - algorithm:
           algorithmType: dsp
           dsp:
             estimators:
               fft:
               - highFrequencyThreshold: "0.05"
                 lowAmplitudeThreshold: "1.0"
                 marginFraction: "0.2"
                 maxNumOfSpectrumItems: 20
                 minNumOfSpectrumItems: 10
             historyLength: 3d
             sampleInterval: 60s
         resourceIdentifier: cpu
         type: ExpressionQuery
         expressionQuery:
           expression: 'sum(count(node_cpu_seconds_total{mode="idle",instance=~"({{.metadata.name}})(:\\d+)?"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"({{.metadata.name}})(:\\d+)?"}[5m]))'
       predictionWindowSeconds: 3600    
   kind: ConfigMap
   metadata:
     name: noderesource-tsp-template
     namespace: default
   ```

时间序列预测是指使用过去的时间序列数据来预测未来的值。时间序列数据通常包括时间和相应的数值，例如资源用量、股票价格或气温。时间序列预测算法 DSP（Digital Signal Processing）是一种数字信号处理技术，可以用于分析和处理时间序列数据。

离散傅里叶变换（DFT）就是 DSP 领域常用的一种算法。DFT 是一种将时域信号转换为频域信号的技术。通过将时域信号分解成不同的频率成分，可以更好地理解和分析信号的特征和结构。在时间序列预测中，DFT 可以用于分析和预测信号的周期性和趋势性，从而提高预测的准确性。

Crane 使用在数字信号处理（Digital Signal Processing）领域中常用的的离散傅里叶变换、自相关函数等手段，识别、预测周期性的时间序列。更多参考：https://gocrane.io/docs/core-concept/timeseriees-forecasting-by-dsp/。

结合预测算法和当前实际用量推算节点的剩余可用资源，并将其作为拓展资源赋予节点，Pod 可标明使用该扩展资源作为离线作业将空闲资源利用起来，以提升节点的资源利用率，部署 Pod 时 limit 和 request 使用 gocrane.io/\<resourceName\>: \<value\> 即可，如下：

```yaml
spec: 
   containers:
   - image: nginx
     imagePullPolicy: Always
     name: extended-resource-demo-ctr
     resources:
       limits:
         gocrane.io/cpu: "2"
         gocrane.io/memory: "2000Mi"
       requests:
         gocrane.io/cpu: "2"
         gocrane.io/memory: "2000Mi"
```

**弹性资源限制**

原生的 BestEffort 应用缺乏资源用量的公平保证，Crane 保证使用动态资源的 BestEffort Pod 其 CPU 使用量被限制在其允许使用的合理范围内，agent 保证使用扩展资源的 Pod 实际用量也不会超过其声明限制，同时在 CPU 竞争时也能按照各自声明量公平竞争；同时使用弹性资源的 Pod 也会受到水位线功能的管理。部署 Pod 时 limit 和 request 使用  gocrane.io/\<resourceName\>: \<value\> 即可。

## Koordinator

<div align=center><img width="400" style="border: 0px" src="/gallery/overcommitted/koordinator.png"></div>

ack-koordinator 与开源项目 [Koordinator](https://koordinator.sh/) 紧密相关。Koordinator 是一个基于 QoS 的 Kubernetes 混合工作负载调度系统，源自阿里巴巴在差异化 SLO 调度领域多年累积的经验，旨在提高对延迟敏感的工作负载和批处理作业的运行时效率和可靠性，简化与资源相关的配置调整的复杂性，并增加 Pod 部署密度以提高资源利用率。

ack-koordinator 组件的前身是 ack-slo-manager，一方面 ack-slo-manager 为 Koordinator 开源社区的孵化提供了宝贵的经验，另一方面随着 Koordinator 逐渐成熟稳定，技术上对 ack-slo-manager 实现了反哺。因此，ack-koordinator 提供两类功能，一是 Koordinator 开源版本已经支持的功能，二是原 ack-slo-manager 提供的一系列差异化 SLO 能力。

ack-koordinator 由中心侧组件和单机侧组件两大部分组成，各模块功能描述如下

- Koordinator Manager：以 Deployment 的形式部署的中心组件，由主备两个实例组成，以保证组件的高可用
  - SLO Controller：用于资源超卖管理，根据节点混部时的运行状态，动态调整集群的超卖资源量，同时为管理各节点的差异化 SLO 策略
  - Recommender：提供资源画像功能，预估工作负载的峰值资源需求，简化您的配置容器资源规格的复杂度
- Koordinator Descheduler：以 Deployment 的形式部署的中心组件，提供重调度功能
- Koordlet：以 DaemonSet 的形式部署的单机组件，用于支持混部场景下的资源超卖、单机精细化调度，以及容器 QoS 保证等

**负载感知调度**

负载感知调度是 ACK 调度器 Kube Scheduler 基于 Kubernetes Scheduling Framework 实现的插件。与 K8s 原生调度策略不同的是，原生调度器主要基于资源的分配率情况进行调度，而 ACK 调度器可以感知节点实际的资源负载情况。通过参考节点负载的历史统计并对新调度 Pod 进行预估，调度器会将 Pod 优先调度到负载较低的节点，实现节点负载均衡的目标，避免出现因单个节点负载过高而导致的应用程序或节点故障。

如下图所示，已分配资源量（Requested）代表已申请的资源量，已使用资源量（Usage）代表真实使用的资源量，只有真实使用的资源才会被算作真实负载。面对相同的节点情况，ACK 调度器会采用更优的策略，将新创建的 Pod 分配到负载更低的节点 B。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/ack-kube-scheduler.png"></div>

负载感知调度功能由 ACK 调度器和 ack-koordinator 组件配合完成。其中，ack-koordinator 负责节点资源利用率的采集和上报，ACK 调度器会根据利用率数据对节点进行打分排序，优先选取负载更低的节点参与调度。

ACK的差异化SLO（Service Level Objectives）提供将这部分资源量化的能力。将上图中的红线定义为Usage，蓝线到红线预留部分资源定义为Buffered，绿色覆盖部分定义为Reclaimed。

**动态资源超卖**

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/ack-slo.png"></div>

ACK 的差异化 SLO（Service Level Objectives）提供将这部分资源量化的能力。将上图中的红线定义为 Usage，蓝线到红线预留部分资源定义为 Buffered，绿色覆盖部分定义为 Reclaimed。为体现与原生资源类型的差异性，ack-koordinator 使用 Batch 优先级的概念描述该部分超卖资源，也就是 batch-cpu 和 batch-memory。

同理，超卖资源的声明也为标准扩展资源：

```yaml
metadata:
  labels:
    # 必填，标记为低优先级 Pod
    koordinator.sh/qosClass: "BE"
spec:
  containers:
  - resources:
      requests:
        # 单位为千分之一核，如下表示 1 核
        kubernetes.io/batch-cpu: "1k"
        # 单位为字节，如下表示 1 GB
        kubernetes.io/batch-memory: "1Gi"
      limits:
        kubernetes.io/batch-cpu: "1k"
        kubernetes.io/batch-memory: "1Gi"
```

节点中可超卖资源的计算公式为：

> nodeBatchAllocatable = nodeAllocatable * thresholdPercent - podRequest(non-BE) - systemUsage

公式中的 thresholdPercent 为可配置参数，通过修改 ConfigMap 中的配置项可以实现对资源的灵活管理，例如：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ack-slo-config
  namespace: kube-system
data:
  colocation-config: |
    {
      "enable": true,
      "metricAggregateDurationSeconds": 60,
      "cpuReclaimThresholdPercent": 60,
      "memoryReclaimThresholdPercent": 70,
      "memoryCalculatePolicy": "usage"
    }
```

**弹性资源限制**

ack-koordinator 在宿主机节点提供了弹性资源限制能力，确保低优先级 BE（BestEffort）类型 Pod 的 CPU 资源使用在合理范围内，保障节点内容器稳定运行。

在 ack-koordinator 提供的动态资源超卖模型中，Reclaimed 资源总量根据高优先级LS（Latency Sensitive）类型 Pod 的实际资源用量而动态变化，这部分资源可以供低优先级 BE（BestEffort）类型 Pod 使用。通过动态资源超卖能力，可以将 LS 与 BE 类型容器混合部署，以此提升集群资源利用率。为了确保 BE 类型Pod 的 CPU 资源使用在合理范围内，避免 LS 类型应用的运行质量受到干扰，ack-koordinator 在节点侧提供了 CPU 资源弹性限制的能力。弹性资源限制功能可以在整机资源用量安全水位下，控制 BE 类型 Pod 可使用的 CPU 资源量，保障节点内容器稳定运行。

如下图所示，在整机安全水位下（CPU Threshold），随着 LS 类型 Pod 资源使用量的变化（Pod（LS）.Usage），BE 类型 Pod 可用的 CPU 资源被限制在合理的范围内（CPU Restriction for BE）。限制水位的配置与动态资源超卖模型中的预留水位基本一致，以此保证 CPU 资源使用的一致性。

<div align=center><img width="600" style="border: 0px" src="/gallery/overcommitted/ack-restriction.png"></div>

ack-koordinator 支持通过 ConfigMap 配置弹性限制参数：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ack-slo-config
  namespace: kube-system
data:
  resource-threshold-config: |
    {
      "clusterStrategy": {
        # 集群是否开启弹性资源限制能力
        "enable": true,
        # 单位为百分比，表示弹性资源限制对应的节点安全水位阈值，默认为 65
        "cpuSuppressThresholdPercent": 65
      }
    }
```

**负载热点重调度**

ack-koordinator 组件提供 koord-descheduler 模块，其中 LowNodeLoad 插件负责感知负载水位并完成热点打散重调度工作。与 Kubernetes 原生的Descheduler 的插件 LowNodeUtilization 不同，LowNodeLoad 是根据节点真实利用率决策重调度，而 LowNodeUtilization 是根据资源分配率决策重调度。

<div align=center><img width="800" style="border: 0px" src="/gallery/overcommitted/koord-descheduler.jpg"></div>

koord-descheduler 模块周期性运行，每个周期内的执行过程分为以下三个阶段：

1. 数据收集：获取集群内的节点和工作负载信息，以及相关的资源利用率数据

2. 策略插件执行

   以 LowNodeLoad 为例

   1. 筛选负载热点节点
   2. 遍历热点节点，从中筛选可以迁移的 Pod，并进行排序
   3. 遍历每个待迁移的 Pod，检查其是否满足迁移条件，综合考虑集群容量、资源利用率水位、副本数比例等约束
   4. 若满足条件则将 Pod 归类为待迁移副本，若不满足则继续遍历其他 Pod 和热点节点

3. 容器驱逐迁移：针对待迁移的 Pod 发起 Evict 驱逐操作

## Katalyst
