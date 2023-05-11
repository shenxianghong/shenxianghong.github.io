---
title: "「 Istio 」流量管理 — WorkloadGroup"
excerpt: "Istio 流量管理场景下的 WorkloadGroup 资源对象使用范例与 API 结构概览"
cover: https://picsum.photos/0?sig=20220908
thumbnail: https://github.com/cncf/artwork/raw/master/projects/istio/stacked/color/istio-stacked-color.svg
date: 2022-09-08
toc: true
categories:
- Service Mesh
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="https://github.com/cncf/artwork/raw/master/projects/istio/horizontal/color/istio-horizontal-color.svg"></div>

------

> based on **1.15.0**

WorkloadGroup 描述工作负载实例的集合。提供了工作负载实例可用于引导其代理的规范，包括元数据和标识。它仅适用于虚拟机等非 Kubernetes 工作负载，旨在模仿用于 Kubernetes 工作负载的现有 Sidecar 注入和部署规范模型以引导 Istio 代理。

以下示例声明了一个 WorkloadGroup，代表了一组在 bookinfo 命名空间中 reviews 下注册的工作负载集合。标签集将在初始化过程中与每个工作负载实例相关联，端口 3550 和 8080 将与 WorkloadGroup 相关联并使用名为 default Service Account。

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: WorkloadGroup
metadata:
  name: reviews
  namespace: bookinfo
spec:
  metadata:
    labels:
      app.kubernetes.io/name: reviews
      app.kubernetes.io/version: "1.3.4"
  template:
    ports:
      grpc: 3550
      http: 8080
    serviceAccount: default
  probe:
    initialDelaySeconds: 5
    timeoutSeconds: 3
    periodSeconds: 4
    successThreshold: 3
    failureThreshold: 3
    httpGet:
     path: /foo/bar
     host: 127.0.0.1
     port: 3100
     scheme: HTTPS
     httpHeaders:
     - name: Lit-Header
       value: Im-The-Best
```

# WorkloadGroup

WorkloadGroup 可以为 bootstrap 指定单个工作负载的属性，并为 WorkloadEntry 提供模板，类似于 Deployment 通过 Pod 模板指定工作负载的属性。一个 WorkloadGroup 可以有多个 WorkloadEntry。 WorkloadGroup 与控制 ServiceEntry 等服务注册表的资源没有关系，因此不会为这些工作负载配置主机名。

| Field                                 | Description                                                  |
| ------------------------------------- | ------------------------------------------------------------ |
| [metadata](#WorkloadGroup.ObjectMeta) | metadata 会作用于对应的所有 WorkloadEntry 中，WorkloadGroup 的 label 设置应位于 metadata 中，而不是 template |
| template                              | 用于生成属于此 WorkloadGroup 的 WorkloadEntry 资源的模板。注意，不应在模板中设置地址和标签字段，空的 serviceAccount 应默认为 default。工作负载身份（mTLS 证书）将使用指定 Service Account 的令牌进行初始化。该组中的 WorkloadEntry 将与工作负载组位于同一命名空间中，并从上述 metadata 字段继承标签和注释 |
| [probe](#ReadinessProbe)              | ReadinessProbe 描述了用户对其工作负载进行健康检查提供的配置。具体参考 Kubernetes 的语法与逻辑 |

# <a name="ReadinessProbe">ReadinessProbe</a>

| Field                              | Description                                               |
| ---------------------------------- | --------------------------------------------------------- |
| initialDelaySeconds                | 容器启动后，进行就绪探测之前的秒数                        |
| timeoutSeconds                     | 探测超时的秒数。默认为 1 秒。最小值为 1 秒                |
| periodSeconds                      | 执行探测的频率（以秒为单位）。默认为 10 秒。最小值为 1 秒 |
| successThreshold                   | 探测失败后被视为成功的最小连续成功次数。默认为 1          |
| failureThreshold                   | 探测成功后被视为失败的最小连续失败次数。默认为 3          |
| [httpGet](#HTTPHealthCheckConfig)  | 基于 http get 的健康检查                                  |
| [tcpSocket](#TCPHealthCheckConfig) | 基于代理是否能够连接的健康检查                            |
| [exec](#ExecHealthCheckConfig)     | 基于执行的命令如何退出的健康检查                          |

# <a name="HTTPHealthCheckConfig">HTTPHealthCheckConfig</a>

| Field       | Description                     |
| ----------- | ------------------------------- |
| path        | HTTP 服务器上的访问路径         |
| port        | endpoint 所在的端口             |
| host        | 要连接的主机名，默认为 Pod IP   |
| scheme      | HTTP 或者 HTTPS，默认为 HTTP    |
| httpHeaders | 请求的 header。允许重复的请求头 |

# <a name="HTTPHeader">HTTPHeader</a>

| Field | Description |
| ----- | ----------- |
| name  | header 的键 |
| value | header 的值 |

# <a name="TCPHealthCheckConfig">TCPHealthCheckConfig</a>

| Field | Description                    |
| ----- | ------------------------------ |
| host  | 待连接的主机，默认为 localhost |
| port  | 主机端口                       |

# <a name="ExecHealthCheckConfig">ExecHealthCheckConfig</a>

| Field   | Description                                                |
| ------- | ---------------------------------------------------------- |
| command | 待运行的命令。退出状态为 0 被视为活动/健康，非零表示不健康 |

# <a name="WorkloadGroup.ObjectMeta">WorkloadGroup.ObjectMeta</a>

| Field       | Description |
| ----------- | ----------- |
| labels      | 标签信息    |
| annotations | 注解信息    |
