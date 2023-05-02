---
title: "「 Istio 」流量管理 — WorkloadEntry"
excerpt: "Istio 中流量管理组件 WorkloadEntry 对象介绍"
cover: https://picsum.photos/0?sig=20220823
thumbnail: https://istio.io/v1.8/img/istio-bluelogo-whitebackground-framed.svg
date: 2022-08-23
toc: true
categories:
- Overview
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="https://www.vectorlogo.zone/logos/istioio/istioio-ar21.svg"></div>

------

> Based on **v1.15.0**

WorkloadEntry 描述单个非 Kubernetes 工作负载的属性，例如 VM 或裸机服务器，因为它被载入到网格中。 WorkloadEntry 必须伴随着 Istio ServiceEntry，它通过适当的标签选择工作负载并为 MESH_INTERNAL 服务（主机名、端口属性等）提供服务定义。 ServiceEntry 对象可以根据服务条目中指定的标签选择器选择多个工作负载条目以及 Kubernetes pod。

当工作负载连接到 istiod 时，自定义资源中的 Status 字段将更新，表示工作负载的健康状况以及其他详细信息，类似于 Kubernetes 更新 Pod 状态的方式。

以下示例为 details.bookinfo.com 服务声明一个表示 VM 的 Workload Entry。此 VM 已使用 details-legacy Service Account 安装和引导 Sidecar。该服务在端口 80 上暴露给网格中的应用程序。到该服务的 HTTP 流量被封装在 Istio 双向 TLS 中，并发送到目标端口 8080 上 VM 上的 sidecar，然后将其转发到同一端口上 localhost 上的应用程序。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: WorkloadEntry
metadata:
  name: details-svc
spec:
  # use of the service account indicates that the workload has a
  # sidecar proxy bootstrapped with this service account. Pods with
  # sidecars will automatically communicate with the workload using
  # istio mutual TLS.
  serviceAccount: details-legacy
  address: 2.2.2.2
  labels:
    app: details-legacy
    instance-id: vm1
```

相关联的 Service Entry：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: details-svc
spec:
  hosts:
  - details.bookinfo.com
  location: MESH_INTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
    targetPort: 8080
  resolution: STATIC
  workloadSelector:
    labels:
      app: details-legacy
```

以下示例使用其完全限定的 DNS 名称声明相同的 VM 工作负载。服务条目的解析模式应更改为 DNS，表示客户端 Sidecar 应在运行时动态解析 DNS 名称，然后再转发请求。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: WorkloadEntry
metadata:
  name: details-svc
spec:
  # use of the service account indicates that the workload has a
  # sidecar proxy bootstrapped with this service account. Pods with
  # sidecars will automatically communicate with the workload using
  # istio mutual TLS.
  serviceAccount: details-legacy
  address: vm1.vpc01.corp.net
  labels:
    app: details-legacy
    instance-id: vm1
```

相关联的 Service Entry：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: details-svc
spec:
  hosts:
  - details.bookinfo.com
  location: MESH_INTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
    targetPort: 8080
  resolution: DNS
  workloadSelector:
    labels:
      app: details-legacy
```

# WorkloadEntry

| Field          | Description                                                  |
| -------------- | ------------------------------------------------------------ |
| address        | 不带端口的网络 endpoint 地址。当且仅当 resolution 设置为 DNS 时，才能使用域名，并且必须是完全限定的，没有通配符。Unix 域套接字 endpoint 使用格式：unix:///absolute/path/to/socket。 |
| ports          | 与 endpoint 关联的 endpoint 集。如果指定了端口映射，则它必须是 servicePortName 到此 endpoint 端口的映射，这样到服务端口的流量将被转发到映射到服务端口名称的 endpoint 端口。如果省略，并且 targetPort 被指定为服务端口规范的一部分，则到服务端口的流量将被转发到指定 targetPort 上的 endpoint 之一。如果未指定 targetPort 和 endpoint 的端口映射，则到服务端口的流量将被转发到同一端口上的 endpoint 之一 |
| labels         | 与 endpoint 关联的一个或多个标签                             |
| network        | 使 Istio 能够对驻留在同一 L3 域/网络中的 endpoint 进行分组。假定同一网络中的所有 endpoint 彼此可直接访问。当不同网络中的 endpoint 无法直接相互访问时，可以使用 Istio Gateway 建立连接（通常在 Gateway Server 中使用 AUTO_PASSTHROUGH 模式）。这是一种高级配置，通常用于跨越多个集群的 Istio 网格 |
| locality       | 与 endpoint 关联的位置信息。位置对应于故障域（例如，国家/地区/区域）。任意故障域层次结构可以通过用 / 分隔每个封装故障域来表示。例如，endpoint 在 US、US-East-1 区域、可用区 az-1 内、数据中心机架 r11 中的位置可以表示为 us/us-east-1/az-1/r11。 Istio 将配置 sidecar 以路由到与 sidecar 相同位置的 endpoint。如果本地没有可用的 endpoint，则将选择 endpoint 父本地（但在同一网络 ID 内）。例如，如果同一网络中有两个 endpoint（networkID “n1”），例如 e1 的位置为 us/us-east-1/az-1/r11 和 e2 的位置为 us/us-east-1/az-2 /r12，来自 us/us-east-1/az-1/r11 地区的 Sidecar 将更倾向于来自同一地区的 e1 而不是来自不同地区的 e2。endpoint e2 可以是与 Gateway 关联的 IP（桥接网络 n1 和 n2），也可以是与标准服务 endpoint 关联的 IP |
| weight         | 与 endpoint 关联的负载平衡权重。具有较高权重的 endpoint 将按比例获得较高的流量 |
| serviceAccount | 如果工作负载中存在 Sidecar，则与工作负载关联的 Service Account。Service Account 必须存在于与配置相同的命名空间中（WorkloadEntry 或 ServiceEntry） |
