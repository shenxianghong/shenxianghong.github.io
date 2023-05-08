---
title: "「 Virtual Kubelet 」Virtual Kubelet 概述"
excerpt: "Virtual Kubelet 的架构概述、Provider 实现以及实践操作"
cover: https://picsum.photos/0?sig=20221122
thumbnail: https://landscape.cncf.io/logos/virtual-kubelet.svg
date: 2022-11-22
toc: true
categories:
- Serverless
tag:
- Virtual Kubelet
---

<div align=center><img width="150" style="border: 0px" src="https://virtual-kubelet.io/img/color-logo.png"></div>

------

> based on **v1.7.0**

# 简介

> Kubernetes API on top, programmable back.

Virtual Kubelet（VK） 是 Kubernetes 中 Kubelet 的典型特性实现，向上伪装成 Kubelet，从而模拟出 Node 对象，对接 Kubernetes 的原生资源对象。向下提供 API，可对接其他资源管理平台提供的 Provider，不同的平台通过实现 Virtual Kubelet 定义的方法，允许节点由其对应的 Provider 提供，如 ACI、AWS Fargate、IoT Edge、Tensile Kube 等。Virtual Kubelet 的主要场景是将 Kubernetes API 扩展到 Serverless 容器平台（如 ACI 和 Fargate）或者扩展到如 Docker Swarm、Openstack ZUN 等容器平台中，也可以通过 Provider 纳管其他 Kubernetes 集群，甚至是原生的 IAAS 平台（如 VMware、Openstack 等。在社区宗旨中，Virtual Kubelet 不是用来实现集群联邦的手段。

Virtual Kubelet 具有可插拔架构和直接使用 Kubernetes 原语的特点，更易于构建。

# 架构

<div align=center><img width="800" style="border: 0px" src="https://raw.githubusercontent.com/virtual-kubelet/virtual-kubelet/master/website/static/img/diagram.svg"></div>

**与传统 Kubelet 的区别**

- 传统的 Kubelet 实现了所在节点的 Pod 和容器的操作行为
- Virtual Kubelet 以节点的形式注册，允许开发者以自定义的行为部署 Pod 和容器

**当前支持的 Kubernetes 特性**

- 创建、删除和更新 Pod
- 容器日志、管理和指标
- 获取 Pod 以及状态
- 节点地址、节点容量、节点守护进程端点
- 管理操作系统
- 携带私有虚拟网络

# Providers

Virtual Kubelet 专注于提供一个库，用户可以在项目中使用该库来构建自定义 Kubernetes 节点 agent。

该项目具有一个可插拔的 Provider 接口，开发者可以实现该接口来定义 Kubelet 的操作。

支持按需和与 Kubernetes 近乎即时的编排容器计算，无需管理 VM 基础设施，同时仍然利用可移植的 Kubernetes API。

每个 Provider 可能有自己的配置文件和所需的环境变量。

Provider 必须具备以下功能才能与 Virtual Kubelet 集成支持：

- 提供必要的后端服务，用于支持 Kubernetes 的 Pod、容器和支持资源的生命周期管理

- 符合 Virtual Kubelet 的 API 规范

- 无法访问 Kubernetes API 服务器，并且具有定义明确的回调机制来获取 Secret 或 ConfigMap 等数据

*参考：[virtual-kubelet godoc](https://pkg.go.dev/github.com/virtual-kubelet/virtual-kubelet#section-readme)*

## Admiralty Multi-Cluster Scheduler

Admiralty Multi-Cluster Scheduler 将特定的 Pod 运行在 Virtual Kubelet 节点上，作为“代理 Pod”，并在远程集群（实际运行容器）中创建相应的"委托 Pod”。通过控制循环机制，更新代理 Pod 的信息用来反映委托 Pod 的状态。

*参考：[Admiralty Multi-Cluster Scheduler documentation](https://github.com/admiraltyio/multicluster-scheduler)*

## Alibaba Cloud Elastic Container Instance (**ECI**)

阿里云 ECI（弹性容器实例）是一种无需管理服务器或集群即可运行容器的服务。阿里云 ECI Provider 是连接 K8s和 ECI 服务之间的桥梁。

*参考： [Alibaba Cloud ECI documentation](https://github.com/virtual-kubelet/alibabacloud-eci/blob/master/README.md)*

## Azure Container Instances (**ACI**)

Azure 容器实例 Provider 允许在同一个 Kubernetes 集群中同时使用 VM 上的 Pod 和 Azure 容器实例。

<div align=center><img width="600" style="border: 0px" src="/gallery/virtual-kubelet/aci.png"></div>

*参考：[Azure Container Instances documentation](https://github.com/virtual-kubelet/azure-aci/blob/master/README.md)*

## AWS Fargate

AWS Fargate 是一种允许运行容器而无需管理服务器或集群的技术。

AWS Fargate 提供商允许将 Pod 部署到 AWS Fargate。在 AWS Fargate 上的 Pod 可以访问具有子网中专用 ENI 的 VPC 网络、连接到互联网的公共 IP 地址、连接到 Kubernetes 集群的私有 IP 地址、安全组、IAM 角色、CloudWatch Logs 和许多其他 AWS 服务。 Fargate 上的 Pod 可以与同一 Kubernetes 集群中常规工作节点上的 Pod 共存。

*参考：[AWS Fargate documentation](https://github.com/virtual-kubelet/aws-fargate)*

## Elotl Kip

Kip 是一个在云实例中运行 Pod 的提供商，允许 Kubernetes 集群透明地将工作负载扩展到云中。当一个 Pod 被调度到虚拟节点上时，Kip 会为 Pod 的工作负载启动一个大小合适的云实例，并将 Pod 调度到该实例上。当 Pod 完成运行时，云实例将终止。

当工作负载在 Kip 上运行时，集群大小自然会随着集群工作负载而扩展，Pod 彼此高度隔离，并且用户无需管理工作节点并将 Pod 战略性地调度到节点上。

*参考：[Elotl Kip documentation](https://github.com/elotl/kip)*

## Kubernetes Container Runtime Interface (**CRI**)

CRI Provider 是基于 CRI 的容器运行时管理真实的 Pod 和容器。

CRI Provider 的目的仅用于测试和原型。不得用于任何其他目的！

Virtual Kubelet 项目的重点是为不符合标准节点模型的容器运行时提供接口。 而 Kubelet 代码库是全面的标准 CRI 节点代理，并且此 Provider 不会尝试重新创建它。

这个 Provider 实现是一个最基本的最小实现，它可以更容易地针对真实的 Pod 和容器测试 Virtual Kubelet 项目的核心功能 —— 换句话说，它比 MockProvider 更全面。

**已知限制**

- CRI Provider 实现了 Provider 接口全部操作，主要是管理 Pod 的生命周期、返回日志和其他内容
- 具备创建 emptyDir、configmap 和 secret volumes 的能力，但如果当发生变化时不会更新
- 不支持任何类型的持久卷
- 会在启动时尝试运行 kube-proxy，并且可以成功运行。但是，当将 Virtual Kubelet 转换为抽象地处理 Service 和路由的模型时，此功能将被重构为测试该功能的一种方式
- 网络目前是非功能性的

## Huawei Cloud Container Instance (**CCI**)

华为 CCI Virtual Kubelet Provider 将 CCI 项目配置成任何 Kubernetes 集群中的节点，例如华为 CCE（云容器引擎）。CCE 支持原生 Kubernetes 应用和工具作为私有集群，便于轻松搭建容器运行环境。被调度到 Virtual Kubelet Provider 的 Pod 将运行在 CCI 中，便于更好的利用 CCI 的高性能。

<div align=center><img width="600" style="border: 0px" src="https://raw.githubusercontent.com/virtual-kubelet/huawei-cci/master/cci-provider.svg"></div>

*参考：[Huawei CCI documentation](https://github.com/virtual-kubelet/huawei-cci/blob/master/README.md#readme)*

## HashiCorp Nomad

Virtual Kubelet 的 HashiCorp Nomad Provider 通过将 Nomad 集群公开为 Kubernetes 中的一个节点，将Kubernetes 集群与 Nomad 集群连接起来。借助 Provider，在 Kubernetes 上注册的虚拟 Nomad 节点上调度的 Pod 将作为 Job 在 Nomad 客户端上运行，就像在 Kubernetes 节点上一样。

*参考：[HashiCorp Nomad documentation](https://github.com/virtual-kubelet/nomad/blob/master/README.md)*

## Liqo

Liqo 为 Virtual Kubelet 实现了一个 Provider，旨在透明地将 Pod 和服务卸载到“对等”Kubernetes 远程集群。 Liqo 能够发现邻居集群（使用 DNS、mDNS）并与其“对等”，或者说建立关系以共享集群的部分资源。当集群建立对等连接时，会生成一个新的 Liqo Virtual Kubelet 实例，通过提供远程集群资源的抽象来无缝扩展集群的容量。该提供商与 Liqo 网络结构相结合，通过启用 Pod 到 Pod 流量和多集群东西向服务扩展集群网络，支持两个集群上的端点。

*参考：[Liqo documentation](https://github.com/liqotech/liqo/blob/master/README.md)*

## OpenStack Zun

Virtual Kubelet 的 OpenStack Zun Provider 用于将 Kubernetes 集群与 OpenStack 集群打通，从而可以在 OpenStack 上运行 Kubernetes 的 Pod。借助子网中的 Neutron 端口，在 OpenStack 上的 Pod 可以访问 OpenStack 租户网络，每个 Pod 都有私有 IP 地址，可以连接到租户内的其他 OpenStack 资源（例如 VM），也可以借助浮动 IP 地址连接互联网，或者将 Cinder 卷绑定给 Pod 容器使用。

*参考：[OpenStack Zun documentation](https://github.com/virtual-kubelet/openstack-zun/blob/master/README.md)*

## Tencent Games Tensile Kube

Tensile Kube Provider 由腾讯游戏提供，可将 Kubernetes 集群与其他 Kubernetes 集群连接起来。该 Provider 能够将 Kubernetes 集群规模无限扩展。底层集群以虚拟节点的形态注册到上层集群中，借助 Provider，调度在虚拟节点上的 Pod 将在其他 Kubernetes 集群的节点上运行。

### 架构设计

<div align=center><img width="600" style="border: 0px" src="https://raw.githubusercontent.com/virtual-kubelet/tensile-kube/master/docs/tensile-kube.png"></div>

- virtual-node

  基于 virtual-kubelet 实现的 Kubernetes Provider。在上层集群中创建的 Pod 将同步到底层集群。如果 Pod 依赖于 ConfigMaps 或 Secret，那么依赖关系也会在集群中创建

- multi-cluster scheduler

  基于 K8s schedule framework 实现。它会在调度 Pod 时根据所有底层集群的容量，并调用 filter 过滤器。如果可用节点数大于或等于 1，则 Pod 可以被调度。因此，这可能会消耗更多资源，因此添加了另一个实现（descheduler）

- descheduler

  descheduler 是基于 K8s descheduler 的二次优化，改变了一些逻辑。它会通过注入一些 nodeAffinity 重新创建一些不可调度的 Pod。可以选择部署上层集群中的 multi-scheduler 和 descheduler 之一，也可以同时选择两者

  - 大规模集群不建议使用 multi-scheduler，当节点总数超过 10000 时，descheduler 开销相对更小
  - 当集群中的节点较少时，multi-scheduler 效果会更好，例如有 10 个集群，每个集群只有 100 个节点

- webhook

  Webhook 是基于 K8s mutation webhook 设计的。用于转换一些可能影响上层集群中调度 Pod（不在 kube-system 中）的字段，例如 nodeSelector、nodeAffinity 和 tolerations。但是只有标签为 virtual-pod:true 的 Pod 才会被转换

  强烈建议 Pod 运行在底层集群并添加标签 virtual-pod:true，除非那些 Pod 必须部署在上层集群的 kube-system 中

  - 对于 K8s < 1.16，没有 label 的 pod 不会被转换。但仍会发送请求到 webhook
  - 对于 K8s >= 1.16，可以使用 label selector 为一些指定的 pod 启用 webhook
  - 总的来说，最初的想法是只在底层集群中运行 Pod

### 限制

- 如果要使用 Server，必须保持 Pod 间通信正常。集群 A 中的 Pod A 可以被集群 B 中的 Pod B 通过 ip 访问。default 命名空间中的服务 Kubernetes 和 kube-system 中的其他服务将同步到较底层的集群
- 在 repo 中开发的 multi-scheduler 可能会花费更多资源，因为它会同步所有较底层集群中调度程序需要的所有对象
- descheduler 不能绝对避免资源碎片化
- PV/PVC 只支持本地 PV 的 WaitForFirstConsumer，上层集群的调度器应该忽略 VolumeBindCheck

### 用例

<div align=center><img width="800" style="border: 0px" src="https://raw.githubusercontent.com/virtual-kubelet/tensile-kube/master/docs/multi.png"></div>

*参考：[Tensile Kube documentation](https://github.com/virtual-kubelet/tensile-kube/blob/master/README.md)*

# Provider 接口实现

Provider 实现了 Kubernetes 节点代理 （即 Kubelet）的核心逻辑。

## PodLifecylceHandler

用于处理在 Kubernetes 中创建、更新或删除 Pod 的请求。

[godoc#PodLifecylceHandler](https://godoc.org/github.com/virtual-kubelet/virtual-kubelet/node#PodLifecycleHandler)

```go
type PodLifecycleHandler interface {
    // CreatePod takes a Kubernetes Pod and deploys it within the provider.
    CreatePod(ctx context.Context, pod *corev1.Pod) error

    // UpdatePod takes a Kubernetes Pod and updates it within the provider.
    UpdatePod(ctx context.Context, pod *corev1.Pod) error

    // DeletePod takes a Kubernetes Pod and deletes it from the provider.
    DeletePod(ctx context.Context, pod *corev1.Pod) error

    // GetPod retrieves a pod by name from the provider (can be cached).
    GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)

    // GetPodStatus retrieves the status of a pod by name from the provider.
    GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error)

    // GetPods retrieves a list of all pods running on the provider (can be cached).
    GetPods(context.Context) ([]*corev1.Pod, error)
}
```

还有一个可选实现的接口 PodNotifiery，用于 Provider 异步通知 virtual-kubelet 有关 Pod 状态更改的信息。如果没有实现这个接口，virtual-kubelet 会周期性的检查所有 Pod 的状态。

*强烈建议实现 PodNotifier，尤其是当运行大量 Pod 时。*

[godoc#PodNotifier](https://godoc.org/github.com/virtual-kubelet/virtual-kubelet/node#PodNotifier)

```go
type PodNotifier interface {
    // NotifyPods instructs the notifier to call the passed in function when
    // the pod status changes.
    //
    // NotifyPods should not block callers.
    NotifyPods(context.Context, func(*corev1.Pod))
}
```

PodLifecycleHandler 由 PodController 维护，PodController 是管理分配给节点 Pod 的核心逻辑。

```go
pc, _ := node.NewPodController(podControllerConfig) // <-- instatiates the pod controller
pc.Run(ctx) // <-- starts watching for pods to be scheduled on the node
```

## NodeProvider

NodeProvider 负责通知 virtual-kubelet 节点状态更新。 virtual-kubelet 会定期检查节点的状态并相应地更新至 Kubernetes。

[godoc#NodeProvider](https://godoc.org/github.com/virtual-kubelet/virtual-kubelet/node#NodeProvider)

```go
type NodeProvider interface {
    // Ping checks if the node is still active.
    // This is intended to be lightweight as it will be called periodically as a
    // heartbeat to keep the node marked as ready in Kubernetes.
    Ping(context.Context) error

    // NotifyNodeStatus is used to asynchronously monitor the node.
    // The passed in callback should be called any time there is a change to the
    // node's status.
    // This will generally trigger a call to the Kubernetes API server to update
    // the status.
    //
    // NotifyNodeStatus should not block callers.
    NotifyNodeStatus(ctx context.Context, cb func(*corev1.Node))
}
```

NodeProvider 由 NodeController 维护，这是 Kubernetes 管理节点对象的核心逻辑。

```go
nc, _ := node.NewNodeController(nodeProvider, nodeSpec) // <-- instantiate a node controller from a node provider and a kubernetes node spec
nc.Run(ctx) // <-- creates the node in kubernetes and starts up he controller
```

[godoc#NaiveNodeProvider](https://godoc.org/github.com/virtual-kubelet/virtual-kubelet/node#NaiveNodeProvider)

Virtual Kubelet 提供了一个 NaiveNodeProvider，用于不打算自定义节点行为时。

## API endpoints

Kubelet 的工作之一是接受来自 API Server 的请求，比如 kubectl logs 和 kubectl exec。

如果想在集群中使用 HPA（Horizontal Pod Autoscaler），Provider 应该实现 GetStatsSummary 函数。然后 metrics-server 将能够获取 virtual-kubelet 上的 Pod 的指标。否则，可能会在 metrics-server 上看到 No metrics for pod，这意味着不会收集 virtual-kubelet 上的 Pod 的指标。

# 已知问题和解决方案

## Service 缺少 Load Balancer IP

**Provider 不支持服务发现**

Kubernetes 1.9 为控制平面的 Controller Manager 引入了一个新标识 ServiceNodeExclusion。在 Controller Manager 的配置文件中启用此标志允许 Kubernetes 将 Virtual Kubelet 节点排除在添加到负载均衡器池之外，创建具有外部 IP 的 Service。

**解决方案**

集群要求：Kubernetes 1.9或以上

Controller Manager 配置文件中新增 --feature-gates=ServiceNodeExclusion=true 参数启用 ServiceNodeExclusion 标识。

# 实践操作

以 Tensile Kube Provider 为例。

**准备工作**

Tensile Kube Provider 是将其他的 Kubernetes 集群以虚拟节点的形式加入到主集群，因此需要准备两个集群。

```shell
# 上层集群
$ kubectl get node
NAME           STATUS   ROLES                  AGE   VERSION
desktop-ca83   Ready    control-plane,master   16d   v1.22.9
 
# 底层集群
$ kubectl get node
NAME              STATUS   ROLES    AGE    VERSION
archcnstcm67370   Ready    master   2d5h   v1.18.15
archcnstcm67371   Ready    master   2d5h   v1.18.15
archcnstcm67372   Ready    master   2d5h   v1.18.15
```

**编译 virtual-node**

```shell
$ git clone https://github.com/virtual-kubelet/tensile-kube.git
$ cd tensile-kube && make
```

**Provider 部署**

Tensile Kube Provider 运行在底层集群中，将其三节点集群以虚拟节点 vk-node 加入到上层集群中。

```shell
# 底层集群启动 Tensile Kube Provider，其中 kubeconfig 为上层集群的配置文件，client-kubeconfig 为底层集群的配置文件
$ ./virtual-node --nodename vk-node --kubeconfig ./config --client-kubeconfig /root/.kube/config
I1124 16:47:19.501216 3727733 provider.go:158] Informer started
I1124 16:47:19.902257 3727733 service_controller.go:114] Starting controller
I1124 16:47:19.902298 3727733 common_controller.go:115] Starting controller
I1124 16:47:19.902331 3727733 pv_controller.go:129] Starting controller
ERRO[0000] TLS certificates not provided, not setting up pod http server  caPath= certPath= keyPath= node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
INFO[0000] Initialized                                   node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
I1124 16:47:19.902447 3727733 node.go:98] Called NotifyNodeStatus
I1124 16:47:19.902455 3727733 pod.go:321] Called NotifyPods
INFO[0000] Pod cache in-sync                             node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
INFO[0000] starting workers                              node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
INFO[0000] started workers                               node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
I1124 16:47:20.607794 3727733 service_controller.go:144] enqueue service addkey istio-system/istio-ingressgateway
I1124 16:47:20.607839 3727733 service_controller.go:144] enqueue service addkey istio-system/istiod
I1124 16:47:20.607855 3727733 service_controller.go:144] enqueue service addkey velero/minio
I1124 16:47:20.607803 3727733 service_controller.go:176] enqueue endpoint add key velero/minio
I1124 16:47:20.607878 3727733 service_controller.go:176] enqueue endpoint add key istio-system/istio-ingressgateway
I1124 16:47:20.607887 3727733 service_controller.go:176] enqueue endpoint add key istio-system/istiod
I1124 16:47:20.702816 3727733 pv_controller.go:135] Sync caches from master successfully
I1124 16:47:20.702859 3727733 pv_controller.go:140] Sync caches from client successfully
I1124 16:47:20.702816 3727733 service_controller.go:120] Sync caches from master successfully
I1124 16:47:20.702903 3727733 service_controller.go:125] Sync caches from client successfully

# 上层集群
$ kubectl get node
NAME           STATUS   ROLES                  AGE   VERSION
desktop-ca83   Ready    control-plane,master   16d   v1.22.9
vk-node        Ready    agent                  9s    v1.18.15
```

Provider 可以部署在任意集群中，通过 --kubeconfig 和 --client-kubeconfig 指定上层底层集群即可，也可以组成网状集群或者公用虚拟节点的集群。

**调度**

```yaml
# 上层集群部署的负载信息
apiVersion: v1
kind: ConfigMap
metadata:
  name: myconfigmap
data:
  username: vk-demo
---
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
type: Opaque
data:
  USER_NAME: YWRtaW4=
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: vk-demo
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
          - vk-node
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: vk-demo
  labels:
    app: nginx
spec:
  # Optional:
  # storageClassName: <YOUR_STORAGE_CLASS_NAME>
  storageClassName: ""
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Mi
---
apiVersion: v1
kind: Pod
metadata:
  name: vk-demo
spec:
  automountServiceAccountToken: false
  containers:
  - name: busybox
    image: busybox:1.27
    imagePullPolicy: IfNotPresent
    command: ["/bin/sh", "-c", "tail -f /dev/null"]
    volumeMounts:
    - name: foo
      mountPath: "/etc/foo"
      readOnly: true
    - name: bar
      mountPath: "/etc/bar"
      readOnly: true
    - mountPath: /datadir
      name: local
  nodeSelector:
    type: virtual-kubelet
  tolerations:
  - key: "virtual-kubelet.io/provider"
    operator: "Exists"
    effect: "NoSchedule"
  volumes:
  - name: foo
    configMap:
      name: myconfigmap
  - name: bar
    secret:
      secretName: mysecret
      optional: false
  - name: local
    persistentVolumeClaim:
      claimName: vk-demo
```

```yaml
# 底层集群预先准备可用的卷
apiVersion: v1
kind: PersistentVolume
metadata:
  name: vk-demo
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
          - archcnstcm67370
```

创建负载之后可以看到，原本在上层的 configmap、secret 和 pvc，透传到对应的底层集群创建了一份

```shell
# 底层集群
$ kubectl get configmap
NAME          DATA   AGE
myconfigmap   1      15m

$ kubectl get secret
NAME                  TYPE                                  DATA   AGE
default-token-jxhcg   kubernetes.io/service-account-token   3      2d7h
mysecret              Opaque                                1      15m

$ kubectl get pvc
NAME                 STATUS   VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS             AGE
vk-demo              Bound    vk-demo  100Gi      RWO                                     16m
```

负载在上层集群和底层集群均可以查询到，但由于两个集群 Pod 网络未打通，因此无法通过上层集群 exec 或者 log 查看

```shell
I1124 17:52:59.931178 2460220 helper.go:57] pod vk-demo depends on secrets [mysecret]
I1124 17:52:59.931202 2460220 helper.go:70] pod vk-demo depends on configMap [myconfigmap]
I1124 17:52:59.931210 2460220 helper.go:83] pod vk-demo depends on pvc [vk-demo]
I1124 17:52:59.936998 2460220 pod.go:457] Create myconfigmap in default success
I1124 17:52:59.937019 2460220 pod.go:79] Create configmaps [myconfigmap] of default/vk-demo success
INFO[0028] Created pod in provider                      
INFO[0028] Event(v1.ObjectReference{Kind:"Pod", Namespace:"default", Name:"vk-demo", UID:"1c52b390-29ab-40de-b892-860db9fe3418", APIVersion:"v1", ResourceVersion:"3395567", FieldPath:""}): type: 'Normal' reason: 'ProviderCreateSuccess' Create pod in provider successfully  node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
I1124 17:53:00.023488 2460220 pod.go:84] Create pvc [vk-demo] of default/vk-demo success
INFO[0029] Updated pod in provider                      
INFO[0029] Event(v1.ObjectReference{Kind:"Pod", Namespace:"default", Name:"vk-demo", UID:"1c52b390-29ab-40de-b892-860db9fe3418", APIVersion:"v1", ResourceVersion:"3395570", FieldPath:""}): type: 'Normal' reason: 'ProviderUpdateSuccess' Update pod in provider successfully  node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=
INFO[0042] Updated pod in provider                      
INFO[0042] Event(v1.ObjectReference{Kind:"Pod", Namespace:"default", Name:"vk-demo", UID:"1c52b390-29ab-40de-b892-860db9fe3418", APIVersion:"v1", ResourceVersion:"3395617", FieldPath:""}): type: 'Normal' reason: 'ProviderUpdateSuccess' Update pod in provider successfully  node=vk-node operatingSystem=Linux provider=k8s watchedNamespace=

$ kubectl get pod -o wide
NAME      READY   STATUS    RESTARTS   AGE   IP           NODE      NOMINATED NODE   READINESS GATES
vk-demo   1/1     Running   0          16m   10.244.0.1   vk-node   <none>           <none>

$ kubectl get pod -o wide
NAME      READY   STATUS    RESTARTS   AGE   IP            NODE              NOMINATED NODE   READINESS GATES
vk-demo   1/1     Running   0          17m   10.244.0.1    archcnstcm67370   <none>           <none>
```

**总结**

Tensile Kube Provider

- 底层多节点集群是以一个单独的虚拟节点加入的上层集群中
- 在上层集群创建的负载，会通过 virtual-node 组件下发到底层集群对应的 api-server 中（可以支持 HA），会根据底层集群的情况进行实际调度
- 由于工作负载真实运行在底层集群中，其依赖的 Secret、Configmap 和 PVC 等资源同样的会透传给底层集群创建，其运行时的资源由底层集群分配并维护，例如 Pod 的网络、存储等
