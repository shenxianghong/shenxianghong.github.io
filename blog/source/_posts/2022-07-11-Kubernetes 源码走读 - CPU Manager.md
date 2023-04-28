---
layout: post
title: "「 Kubernetes」 源码走读 - CPU Manager"
date: 2022-07-11
excerpt: "Kubelet CM 模块中与 CPU Manager 相关的源码走读"
tag:
- Cloud Native
- Kubernetes
categories:
- Kubernetes
---

<div align=center><img width="75" style="border: 0px" src="https://github.com/kubernetes/kubernetes/raw/master/logo/logo.png"></div>

------

> Base on **v1.20.12**

# state

*<u>pkg/kubelet/cm/cpumanager/state/state_checkpoint.go</u>*

基于内存，用于记录 CPU Manager 的状态，后续 Policy 获取 CPU 以及分配情况等均从 state 对象中获得

```go
// map[pod]map[container]CPU Set
type ContainerCPUAssignments map[string]map[string]cpuset.CPUSet

type stateMemory struct {
	sync.RWMutex
    // 记录 CPU 分配情况
	assignments   ContainerCPUAssignments
    // 记录 CPU 信息
	defaultCPUSet cpuset.CPUSet
}
```

## storeState

当有 CPU 分配或者回收时，会调用该函数，将 state 对象持久化为 checkpoint 文件

## restoreState

当 CPU Manager 启动时，会调用该函数，将节点上的 checkpoint 文件恢复为 state 对象

# checkpoint

基于文件（位于 host 上的 /var/lib/kubelet/cpu_manager_state），用于记录 CPU Manager 的状态，为了避免 state 对象在 Kubelet 重启后内存丢失的问题

# Policy

*<u>pkg/kubelet/cm/cpumanager/policy.go</u>*

```go
type Policy interface {
    // 返回策略名称
	Name() string
    // 针对 State 的校验流程
	Start(s state.State) error
	// 容器 CPU 的分配流程
	Allocate(s state.State, pod *v1.Pod, container *v1.Container) error
	// 容器移除后的回收流程
	RemoveContainer(s state.State, podUID string, containerName string) error
	// 调用 Policy 中的 topologymanager.HintProvider 的实现
	GetTopologyHints(s state.State, pod *v1.Pod, container *v1.Container) map[string][]topologymanager.TopologyHint
	// 调用 Policy 中的 topologymanager.HintProvider 的实现
	GetPodTopologyHints(s state.State, pod *v1.Pod) map[string][]topologymanager.TopologyHint
}
```

## none

*<u>pkg/kubelet/cm/cpumanager/policy_none.go</u>*

默认策略。

CPU Manager 的 none 策略并未做任何实际的逻辑处理，不提供任何系统调度器默认行为之外的亲和性策略。通过 CFS 配额来实现 Guaranteed Pods 和 Burstable Pods 的 CPU 使用限制。因此，共享池的 CPU 也会包含 Kubelet 预留的部分。

## static

*<u>pkg/kubelet/cm/cpumanager/policy_static.go</u>*

仅针对 QoS 为 Guaranteed 且 CPU 申请量为正整数的 Pod 赋予增强的 CPU 亲和性和独占性。

### Start

1. 初始化可分配的 CPU 信息，即获取所有可分配的 CPU（如果开启了 strictReserved，则取全量 CPU 和预留 CPU 的差集）
2. 如果开启了 strictReserved，则校验全量 CPU 和预留 CPU 是否没有重叠；如果未开启，则校验预留 CPU 是否全在全量 CPU 中
3. 校验已分配的 CPU 和可分配的 CPU 是否不重叠
4. 校验可分配的 CPU + 已分配的 CPU 是否等于所有的 CPU - 预留的 CPU（如果开启了 strictReserved，否则忽视预留的 CPU）

### Allocate

1. 判断 Pod QoS 是否是 Guaranteed 级别，并且 Container 的 CPU request 为整数，如果不满足条件，直接返回，不做处理

2. 从已分配的 CPU 信息中判断该 Pod 是否已经分配过，如果分配过，则本地更新

3. 调用 Topology Manager 获取所有的 hint providers 返回的 hint

4. 获取可申领的 CPU（即可分配 CPU + 步骤二中可复用的 CPU）

5. 如果开启了 NUMA 亲和特性，则获取到涉及到的  NUMA 中的所有 CPU，取 NUMA CPU 之和和申请 CPU 中的最小值作为待对齐分配的 CPU 数量，校验申请的 CPU 数量是否大于 1 且小于所有可用的 CPU，

6. 执行拓扑感知 best-fit 算法，优先对齐能满足 NUMA 的部分

   *参考 pkg/kubelet/cm/cpumanager/cpu_assignment_test.go 单元测试示例*

   1. 如果请求的 CPU 数量不小于单块 CPU Socket 中 Thread 数量，那么会优先将整块 CPU Socket 中的Thread 分配 

      *acc.freeSockets()，返回单 Socket 中所有 Thread 均可用的 Socket 列表*

   2. 如果剩余请求的 CPU 数量不小于单块物理 CPU Core 提供的 Thread 数量，那么会优先将整块物理 CPU Core 上的 Thread 分配

      *acc.freeCores()，返回单 Core 中所有 Thread 均可用的 Core 列表，按照 SocketID 做升序排列*

   3. 剩余请求的 CPU 数量则从按照如下规则排好序的 Thread 列表中选择

      *acc.freeCPUs()，返回所有可用的 Thread 列表，按照 SocketID 和 CoreID 做升序排列*

      1. 相同 Socket 上可用的 Thread
      2. 相同 Core 上可用的 Thread
      3. CPU ID 升序排列

7. 对于剩余的 CPU，进行如上拓扑感知 best-fit 算法，合并以上两部分，作为最终的 CPU 绑定结果

8. 从共享池 CPU 中去除待分配的 CPU

### RemoveContainer

1. 获取到容器的 CPU 分配信息，删除掉分配的记录信息，共享池 CPU 中添加 CPU

### GetTopologyHints

1. 获取容器申请的 CPU 数量
2. 如果容器已经分配了申请 CPU，那么判断申请的和已分配的是否相等，不相等则不给出 hint，直接返回；相等则用已分配的生成 hint
3. 获取可用的 CPU 和可复用的 CPU 信息，两者的合集作为可复用的 CPU，生成 hint

### GetPodTopologyHints

1. 获取 Pod 申请的 CPU 数量（获取 Init Container 最大值和 Container 之和，取两者最大为 CPU 数量）
2. 遍历 Pod 的每个容器，如果容器已经分配了申请 CPU，那么判断申请的和已分配的是否相等，不相等则不给出 hint，直接返回；如果所有容器的已分配的 CPU 之和等于 Pod 的申请 CPU 数量，则用已分配的生成 hint
3. 获取可用的 CPU 和可复用的 CPU 信息，两者的合集作为可复用的 CPU，生成 hint

### generateCPUTopologyHints

生成 CPU TopologyHint 信息，假设有两个 NUMA 节点（编号为 0 和 1），NUMA0 上有 CPU1 和 CPU2，NUMA1上有 CPU3 和 CPU4，某个 Pod 请求两个 CPU。那么 CPU Manager 这个 HintProvider 会调用 generateCPUTopologyHints 产生如下的 TopologyHint：

- {01: True} 代表从 NUMA0 取 2 个 CPU，并且是“优先考虑的”
- {10: True} 代表从 NUMA1 取 2 个 CPU，并且是“优先考虑的”
- {11: False} 代表从 NUMA0 和 NUMA1 各取一个 CPU，不是“优先考虑的”

1. 获取集群中的所有 NUMA 节点
2. 获取 NUMA 节点组合中涉及到的 CPU
3. 如果 NUMA 节点组合中所涉及到的 CPU 个数比请求的 CPU 数大，并且这个组合所涉及的 NUMA 节点个数是目前为止所有组合中最小的，那么就更新步骤 1 的获取结果
4. 循环统计当前节点可用的 CPU 中，有哪些是属于当前正在处理的 NUMA 节点组合
5. 如果当前 NUMA 组合中可用的 CPU 数比请求的 CPU 小，那么就直接返回，否则就创建一个 TopologyHint，并把它加入到 hints 中
6. 遍历每一个 hint，涉及到的 NUMA 节点个数最少（即步骤 3 中获取的结果）的组合，会标注 preferred 为 true

# CPU Manager

*<u>pkg/kubelet/cm/cpumanager/cpu_manager.go</u>*

```go
type Manager interface {
	// Kubelet 初始化时调用，启动 CPU Manager
	Start(activePods ActivePodsFunc, sourcesReady config.SourcesReady, podStatusProvider status.PodStatusProvider, containerRuntime runtimeService, initialContainers containermap.ContainerMap) error

	// 将 CPU 分配给容器，必须在 AddContainer() 之前的某个时间点调用，例如在 Pod Admission 时
	Allocate(pod *v1.Pod, container *v1.Container) error

	// 在容器创建和容器启动之间调用，以便可以将初始 CPU 亲和性设置写入第一个进程开始执行之前的容器运行时中
	AddContainer(p *v1.Pod, c *v1.Container, containerID string) error

	// 在 Kubelet 决定杀死或删除一个对象后调用，在此调用之后，CPU Manager 停止尝试协调该容器并且释放绑定于该容器的任何 CPU
    // 目前未发现调用处
	RemoveContainer(containerID string) error

	// 返回内部 CPU Manager 的状态
	State() state.Reader

	// 调用 topologymanager.HintProvider 的实现，处理 NUMA 资源对齐等逻辑
	GetTopologyHints(*v1.Pod, *v1.Container) map[string][]topologymanager.TopologyHint

	// 获取分配给 Pod 容器的 CPU 信息
	GetCPUs(podUID, containerName string) []int64

	// 调用 topologymanager.HintProvider 的实现，处理 NUMA 资源对齐等逻辑
	GetPodTopologyHints(pod *v1.Pod) map[string][]topologymanager.TopologyHint
}
```

## Start

1. 初始化 checkpoint 文件，并基于该文件初始化 state 对象
2. 调用 Policy 的 Start 接口，传入 state
3. 如果策略是 none，则直接返回，否则启动 goroutine 定时调和 state

## Allocate

1. 清理搁浅的资源，也就是获取 state 中记录的 CPU 信息，但是实际上使用的容器已经不是 active 状态
2. 调用 Policy 的 Allocate 接口

## AddContainer

1. 从 state 中获取 Pod 容器的 CPU 信息，如果为空直接返回
2. 调用 CRI 的 UpdateContainerResources 接口，更新容器的 CPU Set 信息，如果更新失败调用 Policy 的 RemoveContainer 接口回滚状态，从 containerMap 中移除容器信息

## RemoveContainer

1. 调用 Policy 的 RemoveContainer 接口
2. 从 containerMap 中移除 Container 信息

## State

1. 返回 state 对象

## GetTopologyHints

1. 清理搁浅的资源，也就是获取 state 中记录的 CPU 信息，但是实际上使用的容器已经不是 active 状态
2. 调用 Policy 的 GetTopologyHints 接口

## GetCPUs

1. 获取 state 中记录有给定 Pod 和容器的 CPU 分配情况，并返回

## GetPodTopologyHints

1. 清理搁浅的资源，也就是获取 state 中记录的 CPU 信息，但是实际上使用的容器已经不是 active 状态
2. 调用 Policy 的 GetPodTopologyHints 接口

## reconcileState

针对非 none 类型的 Policy 周期性调和

1. 清理搁浅的资源，也就是获取 state 中记录的 CPU 信息，但是实际上使用的容器已经不是 active 状态
2. 遍历所有的 active 状态的 Pod 的容器
3. 检查该 ContainerID 是否在 CPU Manager 维护的 state 中，然后检查对应的 Pod.Status.Phase 是否为 Running 且 DeletionTimestamp 为 nil，如果是，则调用 CPU Manager 的 AddContainer 对该 Container/Pod 进行 QoS 和 CPU request 检查，如果满足 static Policy 的条件，则调用 takeByTopology 为该 Container 分配最佳的 CPU Set，并写入到 state 和 checkpoint 文件中
4. 然后从 State 中获取该 ContainerID 对应的 CPU Set，调用 CRI UpdateContainerResources 接口更新容器的 CPU Set 信息
