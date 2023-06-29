---
title: "「 Kubernetes 」CPU 精细化管理"
excerpt: "Kubernetes NUMA 感知调度方案与节点 CPU 编排的探索与优化"
cover: https://picsum.photos/0?sig=20230625
thumbnail: /gallery/kubernetes/thumbnail.svg
date: 2023-06-25
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

现代多核服务器大多采用非统一内存访问架构（Non-uniform memory access，简称 NUMA）来提高硬件的可伸缩性。NUMA 是一种为多处理器的电脑设计的内存架构，内存访问时间取决于内存相对于处理器的位置。在 NUMA 架构下，处理器访问它自己的本地内存的速度比非本地内存（内存位于另一个处理器，或者是处理器之间共享的内存）快一些。

在 Kubernetes 中，调度器的调度粒度为节点级别，并不感知和考虑节点硬件拓扑的存在。在某些延迟敏感的场景下，可能希望 Kubernetes 为 Pod 分配拓扑最优的节点和硬件，以提升硬件利用率和程序性能。CPU 敏感型应用有如下特点：

- 对 CPU throttling 敏感
- 对上下文切换敏感
- 对处理器缓存未命中敏感
- 对跨 socket 内存访问敏感

同时，在某些复杂场景下，部分的 Pod 属于 CPU 密集型工作负载，Pod 之间会争抢节点的 CPU 资源。当争抢剧烈的时候，Pod 会在不同的 CPU core 之间进行频繁的切换，更糟糕的是在 NUMA node 之间的切换。这种大量的上下文切换，会影响程序运行的性能。Kubernetes 的 CPU manager 一定程度可以解决以上问题，但是因为 CPU manager 特性是节点级别的 CPU 调度选择，所以无法在集群维度中选择最优的 CPU core 组合。同时 CPU manager 特性要求 Pod QoS 为 Guaranteed 时才能生效，且无法适用于所有 QoS 类型的 Pod。

Kubernetes 中虽然有 Topology Manager 来管理节点资源的拓扑对齐，但是没有与调度器联动，导致调度结果和设备资源分配结果可能不一致。此外，Topology Manager 在进行资源对齐时，仅仅停留在 NUMA 维度，并未考量到 CPU socket 和 core 拓扑等细粒度概念。

# 设计思考

## NUMA 拓扑感知调度

*[KEP 议题](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/119-node-resource-topology-aware-scheduling/README.md)*

引入 Topology Manager 后，支持 Pod 在存在不同的 NUMA 拓扑和不同数量的拓扑资源集群节点中启动。但是存在 Pod 可能被调度到总资源量足够的节点上，但资源分配却无法满足预期的拓扑策略，从而导致 Pod 启动失败（TopologyAffinityError）。对于 Kube-scheduler 来说，更好的行为方式应该是选择适当的节点，与 Kubelet Topology Manager 策略对齐，以便 Kubelet 可以允许 Pod 运行。

**需要做出的改动有**

- 当节点上有 NUMA 拓扑时，通过使用 scheduler-plugin 使调度过程更加精确
- 考虑 NUMA 拓扑，做出更优化的调度决策

需要一个在 Kubelet 外部运行的 agent（[社区参考实现](https://github.com/k8stopologyawareschedwg/resource-topology-exporter)），用于收集有关正在运行 Pod 的所有必要信息，根据节点的可分配资源和 Pod 消耗的资源，它将在 CRD 中提供可用资源，其中一个 CRD 实例代表一个节点。 CRD 实例的名称就是节点的名称。

Filter 插件实现了一个与原 Topology Manager 算法不同的简化版的 Topology Manager。该插件以 single-numa-node 策略的标准检查各节点是否具备运行 Pod 的能力。由于这是最严格的 Topology Manager 策略，如果该策略条件通过，则意味着也必然满足其他策略条件。Filter 插件将使用 CRD 来识别节点上启用的拓扑策略以及节点上可用资源的拓扑信息。另外，Score 插件将进一步考虑最适合运行 Pod 的节点。

**CRD 设计**

具有节点拓扑的可用资源应存储在 CRD 中，其格式应遵循 [Kubernetes Node Resource Topology Custom Resource Definition Standard](https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit?pli=1)。[社区参考设计](https://github.com/k8stopologyawareschedwg/noderesourcetopology-api)。

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopologyList is a list of NodeResourceTopology resources
type NodeResourceTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NodeResourceTopology `json:"items"`
}

// NodeResourceTopology is a specification for a NodeResourceTopology resource
type NodeResourceTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	TopologyPolicies []string `json:"topologyPolicies"`
	Zones            ZoneList `json:"zones"`
}

// Zone is the spec for a NodeResourceTopology resource
type Zone struct {
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Parent     string           `json:"parent,omitempty"`
	Costs      CostList         `json:"costs,omitempty"`
	Attributes AttributeList    `json:"attributes,omitempty"`
	Resources  ResourceInfoList `json:"resources,omitempty"`
}

type ZoneList []Zone

type ResourceInfo struct {
	Name        string             `json:"name"`
	Allocatable intstr.IntOrString `json:"allocatable"`
	Capacity    intstr.IntOrString `json:"capacity"`
}
type ResourceInfoList []ResourceInfo

type CostInfo struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}
type CostList []CostInfo

type AttributeInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type AttributeList []AttributeInfo
```

例如：

```yaml
apiVersion: topology.node.k8s.io/v1alpha1
kind: NodeResourceTopology
metadata:
  name: node1
topologyPolicies:
- SingleNUMANodeContainerLevel
zones:
- costs:
  - name: node-0
    value: 10
  - name: node-1
    value: 21
  name: node-0
  resources:
  - allocatable: "12"
    available: "12"
    capacity: "24"
    name: cpu
  - allocatable: "68590714880"
    available: "68590714880"
    capacity: "68590714880"
    name: memory
  type: Node
- costs:
  - name: node-0
    value: 21
  - name: node-1
    value: 10
  name: node-1
  resources:
  - allocatable: "24"
    available: "12"
    capacity: "24"
    name: cpu
  - allocatable: "68719476736"
    available: "68719476736"
    capacity: "68719476736"
    name: memory
  type: Node
```

**已知限制**

Kube-scheduler 在 NUMA 感知调度 Pod 流程之后，并不知道节点上 Topology Manager 实际为 Pod 分配的 NUMA 情况，节点上的 Topology Manager 也未必按照 scheduler-plugin 中的预选算法进行分配。 

因此，KEP 中建议 Kube-scheduler 可以将分配的 NUMA ID 作为 Pod 提示透传，节点的 Topology Manager 也可以根据 Pod 中的相关提示信息考虑实际的分配策略（这部分涉及到 Topology Manager 的改动，暂未实现）。

## 节点 CPU 编排

<div align=center><img width="600" style="border: 0px" src="/gallery/cpu-manage/cpu-assign.png"></div>

**分配优先级**

1. 为了多核共享 L1 和 L2 cache，优先分配位于同一物理核心的两个逻辑核心。即图中的 0 和 16 号 CPU 分配优先级高于 0 和 1 号 CPU
2. 为了多核共享 L3 cache ，优先分配位于同一 NUMA 的两个逻辑核心。即图中的 0 和 1 号 CPU 分配优先级高于 0 和 4 号 CPU

**扩展思考点**

- 考虑到超线程性能的发挥瓶颈，对于 CPU 满载服务而言，同一物理核心的两个逻辑核心未必比来自不同物理核心的性能强，因此可以针对应用本身的业务模型，是否分配自同一个物理核心有待考量
- CPU 的分配优先级可以不仅仅从静态拓扑结构角度思考设计，也可以结合 CPU 频率、flag 等属性信息以及 CPU 真实使用率等动态实时信息，多维度的考量
- 考虑到节点资源利用率，对于非 Guaranteed QoS 的 Pod 而言，往往也需要不同程度的 CPU 精细化管理
- 由于集群资源动态变化，最初未满足最佳分配策略的服务，可以借助适时重分配或重调度调整至最优分配效果
- 拓扑资源对齐不仅仅限制于 CPU 资源，往往一套完整的拓扑资源对齐方案会将 CPU、内存、GPU、网卡等硬件设备均考虑在内
- 现阶段，在不修改 CPU Manager、Topology Manager 等原有模块逻辑的前提下，往往需要一个旁路 agent 或者 hook CRI 调用的模式来接管资源管理的能力，并且往往需要禁用原生的管理策略
- 随着 NRI（Node Resource Interface）规范的完善，可以基于 NRI hook 扩展，实现资源编排

# 社区成果

## Crane

*https://github.com/gocrane/crane*

<div align=center><img width="800" style="border: 0px" src="/gallery/crane/overview.png"></div>

Crane 是一个基于 FinOps 的云资源分析与成本优化平台。它的愿景是在保证客户应用运行质量的前提下实现极致的降本。

**设计概述**

<div align=center><img width="600" style="border: 0px" src="/gallery/crane/topology-awareness-architecture.png"></div>

Crane-scheduler 和 Crane-agent 配合工作，完成拓扑感知调度与资源分配的工作：

1. Crane-agent 从节点采集资源拓扑，包括 NUMA、socket、设备等信息，汇总到 NodeResourceTopology CRD 中
2. Crane-scheduler 在调度时会参考节点的 NodeResourceTopology 对象获取到节点详细的资源拓扑结构，在调度到节点的同时还会为 Pod 分配拓扑资源，并将结果写到 Pod 的 annotations 中
3. Crane-agent 在节点上 watch 到 Pod 被调度后，从 Pod 的 annotations 中获取到拓扑分配结果，并按照用户给定的 CPU 绑定策略进行 CPUset 的细粒度分配

<div align=center><img width="1000" style="border: 0px" src="/gallery/crane/topology-awareness-details.png"></div>

**CPU 分配策略**

Crane 中提供了四种 CPU 分配策略，分别如下：

1. none：该策略不进行特别的 CPUset 分配，Pod 会使用节点 CPU 共享池
2. exclusive：该策略对应 Kubelet 的 static 策略，Pod 会独占 CPU 核心，其他任何 Pod 都无法使用
3. numa：该策略会指定 NUMA Node，Pod 会使用该 NUMA Node 上的 CPU 共享池
4. immovable：该策略会将 Pod 固定在某些 CPU 核心上，但这些核心属于共享池，其他 Pod 仍可使用

**为系统组件预留 CPU**

在某些场景下，希望能对 Kubelet 预留的 CPU 做一些保护，使用场景包括但不限于：

- 在混部场景下，不希望离线任务绑定系统预留的 CPU 核心，防止对 K8s 系统组件产生影响
- 0 号核心在 Linux 有独特用途，比如处理网络包、内核调用、处理中断等，因此不希望任务绑定 0 号核心

在 Crane 中，可以通过以下方式为系统组件预留 CPU：

1. Kubelet 设置预留 CPU：按照[官方指引](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/#explicitly-reserved-cpu-list)设置预留的 CPU 列表
2. 查看 NodeResourceTopology 对象，spec.attributes 中的 `go.crane.io/reserved-system-cpus` 存储了预留的 CPU 列表
3. 在 Pod 的 annotations 中添加 `topology.crane.io/exclude-reserved-cpus`，表明 Pod 不绑定预留的 CPU 核心

## Koordinator

<u>*https://github.com/koordinator-sh/koordinator*</u>

<div align=center><img width="700" style="border: 0px" src="/gallery/koordinator/overview.png"></div>

Koordinator 是一个基于 QoS 的 Kubernetes 混合工作负载调度系统，旨在提高对延迟敏感的工作负载和批处理作业的运行时效率和可靠性，简化与资源相关的配置调整的复杂性，并增加 Pod 部署密度以提高资源利用率。

**设计概述**

<div align=center><img width="1000" style="border: 0px" src="/gallery/koordinator/cpu-orchestration.svg"></div>

当 Koordlet 启动时，Koordlet 从 Kubelet 收集 NUMA 拓扑信息，包括 NUMA 拓扑、CPU 拓扑、Kubelet CPU 管理策略、Kubelet 为 Guaranteed Pod 分配的 CPU 等，并更新到节点资源拓扑 CRD。当延迟敏感的应用程序扩容时，可以为新 Pod 设置 Koordinator QoS LSE/LSR、CPU 绑定策略和 CPU 独占策略，要求 Koord-scheduler 分配最适合的 CPU 以获得最佳性能。当 Koord-scheduler 调度 Pod 时，Koord-scheduler 会过滤满足 NUMA 拓扑对齐策略的节点，并通过评分选择最佳节点，在 Reserve 阶段分配 CPU，并在 PreBinding 时将结果记录到 Pod annotations。Koordlet 通过 hook Kubelet CRI 请求，替换通过 Koord-scheduler 调度的 CPU 配置参数到运行时，例如配置 cgroup。

**QoS**

Koordinator 调度系统支持的 QoS 有五种类型:

| QoS                              | 特点                                                         | 说明                                                         |
| -------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| SYSTEM                           | 系统进程，资源受限                                           | 对于 DaemonSets 等系统服务，虽然需要保证系统服务的延迟，但也需要限制节点上这些系统服务容器的资源使用，以确保其不占用过多的资源 |
| LSE(Latency Sensitive Exclusive) | 保留资源并组织同 QoS 的 Pod 共享资源                         | 很少使用，常见于中间件类应用，一般在独立的资源池中使用       |
| LSR(Latency Sensitive Reserved)  | 预留资源以获得更好的确定性                                   | 类似于社区的 Guaranteed，CPU 核被绑定                        |
| LS(Latency Sensitive)            | 共享资源，对突发流量有更好的弹性                             | 微服务工作负载的典型QoS级别，实现更好的资源弹性和更灵活的资源调整能力 |
| BE(Best Effort)                  | 共享不包括 LSE 的资源，资源运行质量有限，甚至在极端情况下被杀死 | 批量作业的典型 QoS 水平，在一定时期内稳定的计算吞吐量，低成本资源 |

Koordinator 和 Kubernetes QoS 之间是有对应关系的:

| Koordinator QoS | Kubernetes QoS       |
| --------------- | -------------------- |
| SYSTEM          | ---                  |
| LSE             | Guaranteed           |
| LSR             | Guaranteed           |
| LS              | Guaranteed/Burstable |
| BE              | BestEffort           |

**CPU 编排基本原则**

1. 仅支持 Pod 维度的 CPU 分配机制
2. Koordinator 将机器上的 CPU 分为 CPU Shared Pool，statically exclusive CPUs 和 BE CPU Shared Pool：
   1. CPU Shared Pool 是一组共享 CPU 池，Burstable 和 LS Pod 中的任何容器都可以在其上运行。Guaranteed fractional CPU requests 的 Pod 也可以运行在 CPU Shared Pool 中。CPU Shared Pool 包含节点中所有未分配的 CPU，但不包括由 Guaranteed、LSE 和 LSR Pod 分配的 CPU。如果 Kubelet 保留 CPU，则 CPU Shared Pool 包括保留的 CPU
   2. statically exclusive CPUs 是指分配给 Guaranteed、LSE/LSR Pods 使用的一组独占 CPU。当 Guaranteed、LSE 和 LSR Pod 申请 CPU 时，Koord-scheduler 将从 CPU Shared Pool 中分配
   3. BE CPU Shared Pool 是一组 BestEffort 和 BE 的 Pod 都可运行的 CPU 池。BE CPU Shared Pool 包含节点中除 Guaranteed 和 LSE Pod 分配的之外的所有 CPU

**Koordinator QoS CPU 编排原则**

1. LSE/LSR Pod 的 requests 和 limits 必须相等，CPU 值必须是 1000 的整数倍
2. LSE Pod 分配的 CPU 是完全独占的，不得共享。如果节点是超线程架构，只保证逻辑核心维度是隔离的，但是可以通过 CPUBindPolicyFullPCPUs 策略获得更好的隔离
3. LSR Pod 分配的 CPU 只能与 BE Pod 共享
4. LS Pod 绑定了与 LSE/LSR Pod 独占之外的共享 CPU 池
5. BE Pod 绑定使用节点中除 LSE Pod 独占之外的所有 CPU 
6. 如果 Kubelet 的 CPU 管理器策略为 static 策略，则已经运行的 Guaranteed Pods 等价于 LSR
7. 如果 Kubelet 的 CPU 管理器策略为 none 策略，则已经运行的 Guaranteed Pods 等价于 LS
8. 新创建但未指定 Koordinator QoS 的 Guaranteed Pod 等价于 LS

<div align=center><img width="800" style="border: 0px" src="/gallery/koordinator/qos-cpu-orchestration.png"></div>

**Kubelet CPU Manager 策略兼容原则**

1. 如果 Kubelet 设置 CPU Manager 策略选项 `full-pcpus-only=true` 或者 `distribute-cpus-across-numa=true`，并且节点中没有 Koordinator 定义的新 CPU 绑定策略，则遵循 Kubelet 定义的这些参数的定义
2. 如果 Kubelet 设置了 Topology Manager 策略，并且节点中没有 Koordinator 定义的新的 NUMA Topology Alignment 策略，则遵循 Kubelet 定义的这些参数的定义

**接管 Kubelet CPU 管理策略**

Kubelet 预留的 CPU 主要服务于 BestEffort 和 Burstable Pods。但 Koordinator 不会遵守该策略。Burstable Pod 应该使用 CPU Shared Pool，而 BestEffort Pods 应该使用 BE CPU Shared Pool。LSE 和 LSR Pod 不会从被 Kubelet 预留的 CPU 中分配。

1. 对于 Burstable 和 LS Pod
   1. 当 Koordlet 启动时，计算 CPU Shared Pool 并将共享池应用到节点中的所有 Burstable 和 LS Pod，即更新它们的 CPU cgroups, 设置 CPUset。在创建或销毁 LSE/LSR Pod 时执行相同的逻辑
   2. Koordlet 会忽略 Kubelet 预留的 CPU，将其替换为 Koordinator 定义的 CPU Shared Pool
2. 对于 BestEffort 和 BE Pod
   1. 如果 Kubelet 预留了 CPU，BestEffort Pod 会首先使用预留的 CPU
   2. Koordlet 可以使用节点中的所有 CPU，但不包括由具有整数 CPU 的 Guaranteed 和 LSE Pod 分配的 CPU。这意味着如果 Koordlet 启用 CPU Suppress 功能，则应遵循约束以保证不会影响 LSE Pod。同样，如果 Kubelet 启用了 CPU Manager static 策略，则也应排除 Guaranteed Pod
3. 对于 Guaranteed Pod
   1. 如果 Pod 的 annotations 中有 Koord-scheduler 更新的 `scheduling.koordinator.sh/resource-status`，在 sandbox/container 创建阶段，则会替换 Kubelet CRI 请求中的 CPUset
   2. Kubelet 有时会调用 CRI 中定义的 Update 方法来更新容器 cgroup 以设置新的 CPU，因此 Koordlet 和 koord-runtime-proxy 需要 hook 该方法
4. 自动调整 CPU Shared Pool 大小
   1. Koordlet 会根据 Pod 创建/销毁等变化自动调整 CPU Shared Pool 的大小。如果 CPU Shared Pool 发生变化，Koordlet 应该更新所有使用共享池的 LS 或 Burstable Pod 的 cgroups
   2. 如果 Pod 的 annotations `scheduling.koordinator.sh/resource-status` 中指定了对应的 CPU Shared Pool，Koordlet 在配置 cgroup 时只需要绑定对应共享池的 CPU 即可

接管逻辑要求 koord-runtime-proxy 添加新的扩展点并且 Koordlet 实现新的运行时插件的 hook 。当没有安装 koord-runtime-proxy 时，这些接管逻辑也将能够实现。

**CPU 绑定策略**

标签 `node.koordinator.sh/cpu-bind-policy` 限制了调度时如何绑定 CPU：

- None 或空值 — 不执行任何策略
- FullPCPUsOnly — 要求调度器必须分配完整的物理核。等效于 Kubelet CPU Manager 策略选项 full-pcpus-only=true
- SpreadByPCPUs — 要求调度器必须按照物理核维度均匀的分配 CPU

**NUMA 分配策略**

标签 `node.koordinator.sh/numa-allocate-strategy` 表示在调度时如何选择满意的 NUMA 节点：

- MostAllocated — 表示从可用资源最少的 NUMA 节点分配
- LeastAllocated — 表示从可用资源最多的 NUMA 节点分配
- DistributeEvenly — 表示在 NUMA 节点上平均分配 CPU

**NUMA 拓扑对齐策略**

标签 `node.koordinator.sh/numa-topology-alignment-policy` 表示如何根据 NUMA 拓扑对齐资源分配。策略语义遵循 K8s 社区。相当于 NodeResourceTopology 中的 TopologyPolicies 字段，拓扑策略 SingleNUMANodePodLevel 和 SingleNUMANodeContainerLevel 映射到 SingleNUMANode 策略：

- None — 是默认策略，不执行任何拓扑对齐
- BestEffort — 表示优先选择拓扑对齐的 NUMA node，如果没有，则继续为 Pod 分配资源
- Restricted — 表示每个 Pod 在 NUMA 节点上请求的资源是拓扑对齐的，如果不是，Koord-scheduler 会在调度时跳过该节点
- SingleNUMANode — 表示一个 Pod 请求的所有资源都必须在同一个 NUMA 节点上，如果不是，Koord-scheduler 调度时会跳过该节点

**NodeResourceTopology 维护**

Koordinator 在社区提供的 NodeResourceTopology CRD 基础之上通过 annotations 和 label 扩展了更多的 CPU 管理策略与限制。

- Koordlet 负责创建/更新 NodeResourceTopology
- 建议 Koordlet 通过解析 `/var/lib/kubelet/cpu_manager_state` 文件来获取现有 Guaranteed Pod 的 CPU 分配信息。或者通过 Kubelet 提供的 CRI 接口和 gRPC 获取这些信息
- 当 Koord-scheduler 分配 Pod 的 CPU 时，替换 Kubelet 状态检查点文件中的 CPU
- 建议 Koordlet 从 [kubeletConfiguration](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/) 获取 CPU Manager 策略和选项

## CRI Resource Manager

*https://github.com/intel/cri-resource-manager*

<div align=center><img width="400" style="border: 0px" src="/gallery/cri-resource-manager/overview.png"></div>

CRI Resource Manager 是 CRI 代理，位于客户端和实际容器运行时实现（Containerd、CRI-O）之间，用于转发请求和响应。代理的主要目的是通过在转发请求之前修改请求或在处理和代理期间执行与请求相关的额外操作来应用策略以将硬件感知的资源分配策略应用于系统中运行的容器。

**架构概述**

<div align=center><img width="400" style="border: 0px" src="/gallery/cri-resource-manager/cri-resmgr.svg"></div>

CRI Resource Manager 可以通过加载节点静态配置文件，也可以通过 gRPC 请求 CRI Resource Manager Node Agent 组件动态配置。 Node Agent 组件的主要功能是维护节点级别或者全局级别的 ConfigMap，以响应 CRI Resource Manager 的 gRPC 请求，返回策略配置。

默认情况下，CRI Resource Manager 无法获取 Pod spec 中指定的原始容器资源需求。它尝试使用 CRI 容器创建请求中的相关参数来预估 CPU 和内存资源。但是，无法使用这些参数来预估其他扩展资源。如果想确保 CRI Resource Manager 使用原始 Pod spec 资源需求，CRI Resource Manager Webhook 组件负责将这部分声明复制到 Pod annotations 中，用于 CRI Resource Manager 感知扩展资源。

CRI Resource Manager 提供了极为丰富的硬件拓扑感知的能力，包括但不限于 CPU、内存、blockIO、RDT、SST 等；提供了 topology-aware、static-pools、balloons、podpools 等多种策略。

*CRI Resource Manager 聚焦在节点级别的拓扑资源管理，并未提供 NUMA 拓扑感知调度器。*

**topology-aware 策略**

topology-aware 策略根据检测到的硬件拓扑自动构建池树。每个池都有一组分配为其资源的 CPU 和内存区域。工作负载的资源分配首先选择最适合工作负载资源需求的池，然后从该池中分配 CPU 和内存：

- CPU 和内存拓扑对齐分配，以最严格的可用对齐方式将 CPU 和内存分配给工作负载
- 设备的对齐分配，根据已分配设备的位置选择工作负载池
- CPU 核心共享分配，将工作负载分配给池 CPU 的共享子集
- CPU 核心独占分配，从共享子集中动态分割 CPU 核心并分配给工作负载
- CPU 核心混合分配，将独占和共享 CPU 核心分配给工作负载
- 发现和使用内核隔离的 CPU 核心 ( [isolcpus](https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html#cpu-lists) )，将内核隔离的 CPU 核心用于专门分配的 CPU 核心
- 将分配的资源暴露给工作负载
- 通知工作负载有关资源分配的更改
- 动态放缓内存对齐以防止 OOM，动态加宽工作负载内存集以避免池/工作负载 OOM
- 多层内存分配：将工作负载分配到其首选类型的内存区域，该策略感知三种内存：DRAM 是常规系统主存储器；PMEM 是大容量内存，例如 [Intel® Optane™内存](https://www.intel.com/content/www/us/en/products/memory-storage/optane-dc-persistent-memory.html)；[HBM](https://en.wikipedia.org/wiki/High_Bandwidth_Memory) 是高速存储器，通常出现在一些专用计算系统上
- 冷启动，在初始预热期间将工作负载专门固定到 PMEM
- 动态页面降级，强制将只读和空闲容器内存页迁移到 PMEM

**static-pools 策略**

static-pools 策略是 [Intel CMK ](https://github.com/intel/CPU-Manager-for-Kubernetes)项目的功能移植。

**balloons 策略** 

balloons 策略是一种用于管理系统中容器 CPU 资源分配的方法。它涉及将可用的 CPU 划分为相互独立的池，称为 balloon，每个 balloon 可以根据容器的资源请求进行扩大或缩小，即可以增加或减少其中的 CPU 数量。

balloon 可以是静态的或动态的。静态 balloon 需要手动创建并保持固定的大小，而动态 balloon 则可以根据容器的资源需求自动创建和销毁。这可以实现更高效的资源利用，因为 balloon 可以实时调整以满足不断变化的需求。

除了控制每个 balloon 中 CPU 数量外，balloon 还可以配置特定的设置，例如 CPU 核心和非核心的最小和最大频率。这可以对 CPU 资源的分配进行精细控制，确保每个容器都分配了其运行所需的资源。

大致流程为：

1. 用户可以配置不同类型的 balloon，策略可以根据这些配置实例化 balloon
2. balloon 有一组 CPU 和一组在 CPU 上运行的容器
3. 每个容器都被分配给一个 balloon。容器可以使用其 balloon 的所有 CPU，而不能使用其他 CPU
4. 每个逻辑 CPU 最多属于一个 balloon，也可能存在不属于任何 balloon 的 CPU
5. balloon 中的 CPU 数量在 balloon 的生命周期内可能会发生变化。如果 balloon 膨胀，也就是增加了 CPU，那么 balloon 中的所有容器都可以使用更多的 CPU，反之亦然
6. 当在 Kubernetes 节点上创建新容器时，策略首先决定将运行该容器的 balloon 的类型。该决定基于 Pod annotations，或者如果未给出 annotations 则基于命名空间
7. 接下来，策略决定哪个 balloon 将运行容器。选项有：
   - 现有的 balloon 已经有足够的 CPU 来运行当前和新的容器
   - 现有的 balloon 可以扩大以适应其当前和新的容器
   - 新 balloon
8. 当向 balloon 添加或从其中移除 CPU 时，会根据 balloon 的 CPU 类属性或空闲 CPU 类属性重新配置 CPU

**podpools 策略**

podpools 策略实现 Pod 级别的工作负载放置。它将 Pod 的所有容器分配到同一个 CPU/内存池。池中的 CPU 数量可由用户配置。

**容器亲和与反亲和**

亲和与反亲和的提示是通过 Pod annotations 声明：

- 同一 NUMA 节点内的 CPU 视为彼此亲和
- 同一 socket 中不同 NUMA 节点内的 CPU，以及不同 socket 内的 CPU 视为彼此反亲和

**blockIO**

blockIO 提供以下控制：

- 块设备 IO 调度优先级（权重）
- 限制 IO 带宽
- 限制 IO 操作的数量

CRI Resource Manager 通过 cgroups blockIO 控制器将 blockIO 的相关参数应用于 Pod。

## Volcano

*https://github.com/volcano-sh/volcano*

<div align=center><img width="600" style="border: 0px" src="/gallery/volcano/overview.png"></div>

Volcano 是 CNCF 下首个也是唯一的基于 Kubernetes 的容器批量计算平台，主要用于高性能计算场景。它提供了 Kubernetes 目前缺少的一套机制，这些机制通常是机器学习大数据应用、科学计算、特效渲染等多种高性能工作负载所需的。作为一个通用批处理平台，Volcano 与几乎所有的主流计算框架无缝对接，如Spark、TensorFlow 、PyTorch、 Flink 、Argo 、MindSpore 、 PaddlePaddle 等。它还提供了包括基于各种主流架构的 CPU、GPU 在内的异构设备混合调度能力。Volcano 的设计理念建立在 15 年来多种系统和平台大规模运行各种高性能工作负载的使用经验之上，并结合来自开源社区的最佳思想和实践。

**感知调度流程**

<div align=center><img width="800" style="border: 0px" src="/gallery/volcano/numa-aware-process.png"></div>

| policy           | action                                                      |
| ---------------- | ----------------------------------------------------------- |
| none             | 无                                                          |
| best-effort      | 过滤出拓扑策略为 best-effort 的节点                         |
| restricted       | 过滤出拓扑策略为 restricted 且满足 CPU 拓扑要求的节点       |
| single-numa-node | 过滤出拓扑策略为 single-numa-node 且满足 CPU 拓扑要求的节点 |

Volcano 在的感知调度和其他项目类似，将 Kubernetes Topology Manager 的原生策略扩展至调度器层面，只不过 CRD 采用的是 Volcano 设计的 [Numatopology](https://github.com/volcano-sh/apis/blob/master/pkg/apis/nodeinfo/v1alpha1/numatopo_types.go)，而非社区提出的 NodeResourceTopology CRD，其他流程方面大同小异。

**节点 CPU 编排**

Volcano 并未提供节点 CPU 编排的能力，但是参考华为 CCE 产品文档中，CCE 基于社区原生的 CPU Manager 策略的基础上，提出了 enhanced-static 策略，是在兼容 static 策略的基础上，新增一种符合某些资源特征的 Burstable Pod（CPU 的 requests 和 limits 值都是正整数）优先使用某些 CPU 的能力，以减少应用在多个 CPU 间频繁切换带来的影响。

该特性是基于 Huawei Cloud EulerOS 2.0 内核中优化了 CPU 调度能力实现的。在 Pod 容器优先使用的 CPU 利用率超过 85% 时，会自动分配到其他利用率较低的 CPU 上，进而保障了应用的响应能力。

<div align=center><img width="500" style="border: 0px" src="/gallery/volcano/enhanced-static.png"></div>

- 开启 enhanced-static 策略时，应用性能优于 none 策略，但弱于 static 策略
- 应用分配的优先使用的 CPU 并不会被独占，仍处于共享的 CPU 池中。因此在该 Pod 处于业务波谷时，节点上其他 Pod 可使用该部分 CPU 资源

# 实践验证

*以 cri-resource-manager为例*

> based on **v0.8.3**

**服务安装**

```shell
# 安装 cri-resource-manager 服务
$ yum -y install https://github.com/intel/cri-resource-manager/releases/download/v0.8.3/cri-resource-manager-0.8.3-0.centos-7.x86_64.rpm

# 安装 cri-resmgr-agent 服务（需要手动编译并替换 IMAGE_PLACEHOLDER 占位符，这里不做详述）
$ kubectl apply -f https://raw.githubusercontent.com/intel/cri-resource-manager/master/cmd/cri-resmgr-agent/agent-deployment.yaml
```

**安装结果**

```shell
$ systemctl start cri-resource-manager
$ systemctl status cri-resource-manager
● cri-resource-manager.service - A CRI proxy with (hardware) resource aware container placement policies.
   Loaded: loaded (/usr/lib/systemd/system/cri-resource-manager.service; enabled; vendor preset: disabled)
   Active: active (running) since Mon 2023-06-28 16:26:04 CST; 29min ago
     Docs: https://github.com/intel/cri-resource-manager
 Main PID: 32130 (cri-resmgr)
    Tasks: 49
   Memory: 41.6M
   CGroup: /system.slice/cri-resource-manager.service
           └─32130 /usr/bin/cri-resmgr --fallback-config /etc/cri-resmgr/fallback.cfg
           
$ kubectl get ds -A
NAMESPACE            NAME                  DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE
kube-system          cri-resmgr-agent      1         1         1       1            1           <none>                   11m

# 采用 cri-resmgr-agent 维护的动态配置，采用 topology-aware 策略
# 节点全量的 CPU 为 0-47
# - 0-4 用于非 Kubernetes 平台使用，如节点系统服务等
# - AvailableResources 中的 5-47 号 CPU 用于 Kubernetes 平台使用
#   - ReservedResources 5-10 号 CPU 用于 Kubernetes 的预留命名空间下的服务使用
#   - 剩余的 10-47 号 CPU 用于 Kubernetes 的其他命名空间下的服务使用
$ kubectl get cm -n kube-system cri-resmgr-config.node.node1 -o yaml
apiVersion: v1
data:
  policy: |
    Active: topology-aware
    topology-aware:
      ReservedPoolNamespaces: [kube-system,arsdn,secboat]
    ReservedResources:
      cpu: cpuset:5-10
    AvailableResources:
      cpu: cpuset:5-47
kind: ConfigMap
```

**服务配置**

```shell
# 配置 Kubelet 的 CRI endpoint 为 cri-resmgr.sock
$ cat /var/lib/kubelet/kubeadm-flags.env
KUBELET_KUBEADM_ARGS="--container-runtime=remote --container-runtime-endpoint=unix:///var/run/cri-resmgr/cri-resmgr.sock"

$ cat /etc/kubernetes/kubelet.env
...
KUBELET_ARGS="--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
--config=/etc/kubernetes/kubelet-config.yaml \
--kubeconfig=/etc/kubernetes/kubelet.conf \
--log-dir=/var/log/kubelet \
--log-file=/var/log/kubelet/kubelet.log \
--logtostderr=false \
--alsologtostderr=false \
--feature-gates=CSIInlineVolume=true,CSIVolumeHealth=true,CPUManagerPolicyOptions=true \
--pod-infra-container-image=harbor.archeros.cn:443/library/ake/pause:3.5-amd64 \
--container-runtime=remote \
--runtime-request-timeout=15m \
--container-runtime-endpoint=unix:///var/run/cri-resmgr/cri-resmgr.sock \
--runtime-cgroups=/systemd/system.slice \
...

$ systemctl daemon-reload && systemctl restart kubelet
```

**节点 CPU 编排**

```shell
$ numactl -H
available: 2 nodes (0-1)
node 0 cpus: 0 1 2 3 4 5 6 7 8 9 10 11 24 25 26 27 28 29 30 31 32 33 34 35
node 0 size: 65413 MB
node 0 free: 15969 MB
node 1 cpus: 12 13 14 15 16 17 18 19 20 21 22 23 36 37 38 39 40 41 42 43 44 45 46 47
node 1 size: 65536 MB
node 1 free: 21933 MB
node distances:
node   0   1
  0:  10  21
  1:  21  10
```

```shell
# 部署共享 CPU 的 Pod
$ kubectl apply -f besteffort.yaml && kubectl apply -f busterable.yaml && kubectl apply -f guaranteed.yaml
# 查看 CPU 分配情况：共享一个合适的 NUMA node
$ crictl ps | grep besteffort | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "12-23,36-47",
$ crictl ps | grep busterable | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "12-23,36-47",
$ crictl ps | grep guaranteed | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "12-23,36-47",           

# 部署独占 CPU 的 Pod
$ kubectl apply -f guaranteed-exclusive.yaml
# 查看 CPU 分配情况：独占同一物理核心的两个逻辑核心
$ crictl ps | grep guaranteed-exclusive | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "23,47",

# 查看热更新，共享 CPU 中将独占的 CPU 扣除
$ crictl ps | grep besteffort | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "12-22,36-46",
$ crictl ps | grep busterable | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "12-22,36-46",
$ crictl ps | grep guaranteed | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "12-22,36-46",

# 预留 namespace CPU 分配
$ kubectl apply -f reserved.yaml
$ crictl ps | grep reserved | awk '{print $1}' | xargs crictl inspect | grep "\"cpus\":"
            "cpus": "5-10",
```

