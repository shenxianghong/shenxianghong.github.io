---
title: "「 Istio 」流量管理 — DestinationRule"
excerpt: "Istio 流量管理场景下的 DestinationRule 资源对象使用范例与 API 结构概览"
cover: https://picsum.photos/0?sig=20220809
thumbnail: https://github.com/cncf/artwork/raw/master/projects/istio/stacked/color/istio-stacked-color.svg
date: 2022-08-09
toc: true
categories:
- Service Mesh
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="https://github.com/cncf/artwork/raw/master/projects/istio/horizontal/color/istio-horizontal-color.svg"></div>

------

> based on **1.15.0**

DestinationRule 定义了在路由发生后应用于服务的流量的策略。这些规则指定负载均衡的配置、sidecar 的连接池大小以及异常值检测设置，用来检测、驱逐负载均衡池中不健康的后端。

例如，review 服务的简单负载均衡策略如下。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: bookinfo-ratings
spec:
  host: ratings.prod.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: LEAST_REQUEST
```

可以通过定义 subset 覆盖在服务级别指定的设置来指定版本特定策略。

以下规则对流向名为 testversion 的子集的所有流量使用 ROUND_ROBIN 负载均衡策略。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: bookinfo-ratings
spec:
  host: ratings.prod.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: LEAST_REQUEST
  subsets:
  - name: testversion
    labels:
      version: v3
    trafficPolicy:
      loadBalancer:
        simple: ROUND_ROBIN
```

*注意：为子集指定的策略在路由规则明确向该子集发送流量之前不会生效。*

流量策略也可以针对特定端口进行定制。以下规则对流向端口 80 的所有流量使用 LEAST_REQUEST 负载均衡策略，而对流向端口 9080 的流量使用 ROUND_ROBIN  负载均衡策略。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: bookinfo-ratings-port
spec:
  host: ratings.prod.svc.cluster.local
  trafficPolicy: # Apply to all ports
    portLevelSettings:
    - port:
        number: 80
      loadBalancer:
        simple: LEAST_REQUEST
    - port:
        number: 9080
      loadBalancer:
        simple: ROUND_ROBIN
```

目标规则也可以针对特定的工作负载进行定制。以下示例显示了如何使用工作负载选择器将目标规则应用于特定工作负载。

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: configure-client-mtls-dr-with-workloadselector
spec:
  workloadSelector:
    matchLabels:
      app: ratings
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
    portLevelSettings:
    - port:
        number: 31443
      tls:
        credentialName: client-credential
        mode: MUTUAL
```

# DestinationRule

| Field                           | Description                                                  |
| ------------------------------- | ------------------------------------------------------------ |
| host                            | 服务注册表中的服务名称。服务名称从平台的服务注册表（例如，Kubernetes 服务、Consul 服务等）和 ServiceEntry 声明的 host 中查找。两者都找不到的流量将被丢弃。<br />*同样的，推荐使用完全限定域名，host 字段适用于 HTTP 和 TCP 的服务* |
| [trafficPolicy](#TrafficPolicy) | 要应用的流量策略（负载均衡策略、连接池大小、异常值检测）     |
| [subsets](#Subset)              | 一个或多个命名集，代表服务的各个版本。流量策略可以在子集级别被覆盖 |
| exportTo                        | 此 VirtualService 暴露到的命名空间列表。暴露 VirtualService 允许它被其他命名空间中定义的 sidecar 和 Gateway 使用。此功能为服务所有者和网格管理员提供了一种机制来控制跨命名空间边界的 VirtualService 的可见性<br />如果未指定命名空间，则默认情况下将 VirtualService 暴露到所有命名空间<br /> . 为保留标识代表暴露到 VirtualService 的同一命名空间中。类似地，* 代表暴露到所有命名空间 |
| workloadSelector                | 用于选择应用此 DestinationRule 配置的特定 pod/VM 集的条件。如果指定，则 DestinationRule 配置将仅应用于与同一命名空间中的选择器标签匹配的工作负载实例。工作负载选择器不适用于跨命名空间。如果省略，DestinationRule 将回退到其默认行为。例如，如果特定的 sidecar 需要为网格外的服务设置出口 TLS 设置，而不是网格中的每个 sidecar 都需要配置（这是默认行为），则可以指定工作负载选择器 |

# <a name="TrafficPolicy">TrafficPolicy</a>

| Field                                                 | Description                                                  |
| ----------------------------------------------------- | ------------------------------------------------------------ |
| [loadBalancer](#LoadBalancerSettings)                 | 负载均衡器算法的设置                                         |
| [connectionPool](#ConnectionPoolSettings)             | 与上游服务的连接池的设置                                     |
| [outlierDetection](#OutlierDetection)                 | 负载均衡池中逐出不健康后端的设置                             |
| [tls](#ClientTLSSettings)                             | 与上游服务连接的 TLS 相关设置                                |
| [portLevelSettings](#TrafficPolicy.PortTrafficPolicy) | 各个端口的流量策略。<br />*注意，portLevelSettings 将覆盖 destinationLevel 设置。portLevelSettings 未设置的部分，会以 destinationLevel  级别为准* |
| [tunnel](#TrafficPolicy.TunnelSettings)               | 在 DestinationRule 中配置的 host 的其他传输层或应用层上的隧道 TCP 配置。隧道设置可以应用于 TCP 或 TLS 路由，但不能应用于 HTTP 路由 |

# <a name="Subset">Subset</a>

Service 的 endpoint 的子集。子集可用于 A/B 测试或路由到特定服务版本等场景。此外，在 VirtualService 级别定义的流量策略可以被子集级别覆盖。

通常需要一个或多个标签来标识子集目的地，但是，当相应的 DestinationRule 表示支持多个 SNI 主机的主机（例如，出口网关）时，没有标签的子集可能是有意义的。在这种情况下，可以使用带有 ClientTLSSettings 的流量策略来识别与命名子集相对应的特定 SNI 主机。

| Field                           | Description                                                  |
| ------------------------------- | ------------------------------------------------------------ |
| name                            | 子集的名称。服务名称和子集名称可用于路由规则中的流量拆分     |
| labels                          | 服务注册表中的服务 endpoint 上应用过滤器                     |
| [trafficPolicy](#TrafficPolicy) | 子集的流量策略。子集会继承在 DestinationRule 级别指定的流量策略，同时会被子集级别指定的设置覆盖 |

# <a name="LoadBalancerSettings">LoadBalancerSettings</a>

应用到特定目的地的负载均衡策略。有关更多详细信息，参阅 [Envoy 的负载均衡文档](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/load_balancing/load_balancing)。

例如，以下规则对流向 ratings 服务的所有流量使用 ROUND_ROBIN 负载均衡策略。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: bookinfo-ratings
spec:
  host: ratings.prod.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
```

以下示例对于使用 User cookie 作为哈希键访问 rating 服务设置粘性会话。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: bookinfo-ratings
spec:
  host: ratings.prod.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      consistentHash:
        httpCookie:
          name: user
          ttl: 0s
```

| Field                                                    | Description                                                  |
| -------------------------------------------------------- | ------------------------------------------------------------ |
| [simple](#LoadBalancerSettings.SimpleLB)                 | 参考 LoadBalancerSettings.SimpleLB 说明                      |
| [consistentHash](#LoadBalancerSettings.ConsistentHashLB) | 参考 LoadBalancerSettings.ConsistentHashLB 说明              |
| [localityLbSetting](#LocalityLoadBalancerSetting)        | 本地负载均衡器设置，这将完全覆盖网格范围的设置，这意味着不会在此对象和 MeshConfig 之间执行合并 |
| warmupDurationSecs                                       | 表示 Service 的预热持续时间。如果设置，则新创建的 service endpoint 在此窗口期间从其创建时间开始保持预热模式，并且 Istio 逐渐增加该 endpoint 的流量，而不是发送成比例的流量。应该为需要预热时间的合理延迟服务完整生产负载的服务启用此功能。目前只支持 ROUND_ROBIN 和 LEAST_CONN 负载均衡器 |

# <a name="ConnectionPoolSettings">ConnectionPoolSettings</a>

上游主机的连接池设置。这些设置适用于上游服务中的每个单独的主机。有关更多详细信息，参阅 [Envoy 的断路器](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/circuit_breaking)。连接池设置可以应用于 TCP 级别以及 HTTP 级别。

例如，以下规则设置了 100 个连接到名为 myredissrv 的 redis 服务的限制，连接超时为 30 毫秒。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: bookinfo-redis
spec:
  host: myredissrv.prod.svc.cluster.local
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
        connectTimeout: 30ms
        tcpKeepalive:
          time: 7200s
          interval: 75s
```

| Field                                        | Description                    |
| -------------------------------------------- | ------------------------------ |
| [tcp](#ConnectionPoolSettings.TCPSettings)   | HTTP 和 TCP 上游连接通用的设置 |
| [http](#ConnectionPoolSettings.HTTPSettings) | HTTP 连接池设置                |

# <a name="OutlierDetection">OutlierDetection</a>

一种断路器实现，用于跟踪上游服务中每个单独主机的状态。

适用于 HTTP 和 TCP 服务。

- 对于 HTTP 服务，对于 API 调用持续返回 5xx 错误的主机将在预定义的时间段内从池中弹出
- 对于 TCP 服务，在测量连续错误指标时，与给定主机的连接超时或连接失败将计为错误

有关更多详细信息，参阅 [Envoy 的异常值检测](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/outlier)。

以下规则将连接池大小设置为 100 个 HTTP1 连接，其中 reviews 服务的单连接的请求最大不超过 10 个。此外，它设置了 1000 个并发 HTTP2 请求的限制，并将上游 host 配置为每 5 分钟扫描一次，任何连续 7 次失败并出现 502、503 或 504 错误代码的 host 将被弹出 15 分钟。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: reviews-cb-policy
spec:
  host: reviews.prod.svc.cluster.local
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http2MaxRequests: 1000
        maxRequestsPerConnection: 10
    outlierDetection:
      consecutive5xxErrors: 7
      interval: 5m
      baseEjectionTime: 15m
```

| Field                          | Description                                                  |
| ------------------------------ | ------------------------------------------------------------ |
| splitExternalLocalOriginErrors | 确定是否区分本地源故障和外部错误。如果设置为 true，则连续_local_origin_failure 被考虑用于异常值检测计算。当您希望根据本地看到的错误（例如连接失败、连接超时等）而不是上游服务重新调整的状态码来推导异常检测状态时，应该使用此选项。当上游服务为某些请求显式返回 5xx 并且您希望在确定主机的异常检测状态时忽略来自上游服务的这些响应时，这尤其有用。默认为假 |
| consecutiveLocalOriginFailures | 在弹出发生之前连续发生的本地故障数。默认为 5。参数仅在 splitExternalLocalOriginErrors 设置为 true 时生效 |
| consecutiveGatewayErrors       | 主机从连接池中弹出之前的网关错误数。当通过 HTTP 访问上游主机时，502、503 或 504 返回码被视为网关错误。当通过不透明的 TCP 连接访问上游主机时，连接超时和连接错误/失败事件符合网关错误。默认为 0，即禁用此功能 |
| consecutive5xxErrors           | 主机从连接池中弹出之前的 5xx 错误数。当通过不透明的 TCP 连接访问上游主机时，连接超时、连接错误/失败和请求失败事件被视为 5xx 错误。默认为 5，可以通过将值设置为 0 来禁用<br />*注意， consecutiveGatewayErrors 和  consecutive5xxErrors 可以单独使用，也可以一起使用。因为 consecutiveGatewayErrors 统计的错误也包含在 consecutive5xxErrors 中，如果 consecutiveGatewayErrors 的值大于或等于 consecutive5xxErrors 的值，则 consecutiveGatewayErrors 将不起作用* |
| interval                       | 弹出分析的时间间隔。格式：1h/1m/1s/1ms。必须 >=1 ms。默认为 10 s |
| baseEjectionTime               | 最短弹出时间。被弹出的时间等于最小弹出持续时间与主机被弹出次数的乘积。这种方式下系统将自动增加不健康的上游服务器的弹出周期。格式：1h/1m/1s/1ms。必须 >=1 ms。默认为 30 s |
| maxEjectionPercent             | 上游服务的负载均衡池中可以弹出的后端的最大百分比。默认为 10% |
| minHealthPercent               | 只要关联的负载均衡池至少有 minHealthPercent 主机处于健康模式，就会启用异常值检测。当负载均衡池中健康主机的百分比低于此阈值时，将禁用异常值检测，并且代理将在池中的所有主机（健康和不健康）之间进行负载均衡。可以通过将阈值设置为 0% 来禁用该阈值。默认值为 0%，因为它通常不适用于每个服务的 pod 很少的 k8s 环境 |

# <a name="ClientTLSSettings">ClientTLSSettings</a>

上游连接的 SSL/TLS 相关设置。有关更多详细信息，参阅 [Envoy 的 TLS 上下文](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/transport_sockets/tls/v3/common.proto.html#common-tls-configuration)。对 HTTP 和 TCP 上游都是通用的。

例如，以下规则将客户端配置为使用双向 TLS（MUTUAL）连接到上游数据库集群。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: db-mtls
spec:
  host: mydbserver.prod.svc.cluster.local
  trafficPolicy:
    tls:
      mode: MUTUAL
      clientCertificate: /etc/certs/myclientcert.pem
      privateKey: /etc/certs/client_private_key.pem
      caCertificates: /etc/certs/rootcacerts.pem
```

以下规则将客户端配置为在与 rating 服务通信时使用 Istio 双向 TLS。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: ratings-istio-mtls
spec:
  host: ratings.prod.svc.cluster.local
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

| Field                              | Description                                                  |
| ---------------------------------- | ------------------------------------------------------------ |
| [mode](#ClientTLSSettings.TLSmode) | 是否应使用 TLS 保护与此端口的连接。该字段的值决定了 TLS 的行为方式 |
| clientCertificate                  | 如果 mode 为 MUTUAL 时，则为必需。表示使用的客户端 TLS 证书的文件的路径。如果模式为 ISTIO_MUTUAL，则应为空 |
| privateKey                         | 如果 mode 为 MUTUAL 时，则为必需。保存客户端私钥的文件的路径。如果模式为 ISTIO_MUTUAL，则应为空 |
| caCertificates                     | 用于验证提供的服务器证书的证书颁发机构证书的文件的路径。如果省略，代理将不会验证服务器的证书。如果模式为 ISTIO_MUTUAL，则应为空 |
| credentialName                     | 保存客户端 TLS 证书的密钥的名称，包括 CA 证书。 Secret 必须与使用证书的代理存在于同一命名空间中。密钥（通用类型）应包含以下键和值：key：\<privateKey\>、cert：\<clientCert\>、cacert：\<CACertificate\>。这里 CACertificate 用于验证服务器证书。还支持用于客户端证书的 tls 类型的密钥以及用于 CA 证书的 ca.crt 密钥。只能指定客户端证书和 CA 证书或 credentialName 中的一个<br />注意：仅当 DestinationRule 指定了工作负载选择器时，此字段才适用于 sidecar。否则该字段将仅适用于网关，sidecar 将继续使用证书路径 |
| subjectAltNames                    | 用于验证证书中主体身份的备用名称列表。如果指定，代理将验证服务器证书的主题 alt 名称是否与指定值之一匹配。如果指定，此列表将覆盖 ServiceEntry 中的 subject_alt_names 的值。如果未指定，则将根据下游 HTTP 主机/授权标头自动验证新上游连接的上游证书，前提是 VERIFY_CERT_AT_CLIENT 和 ENABLE_AUTO_SNI 环境变量设置为 true |
| sni                                | 在 TLS 握手期间呈现给服务器的 SNI 字符串。如果未指定，则 SNI 将根据 SIMPLE 和 MUTUAL TLS 模式的下游 HTTP host/authority header 自动设置，前提是 ENABLE_AUTO_SNI 环境变量设置为 true |
| insecureSkipVerify                 | InsecureSkipVerify 指定代理是否应该跳过验证主机对应的服务器证书的 CA 签名和 SAN。仅当启用全局 CA 签名验证，VerifyCertAtClient 环境变量设置为 true，但不需要对特定主机进行验证时，才应设置此标志。无论是否启用了 VerifyCertAtClient，都将跳过 CA 签名和 SAN 的验证<br />InsecureSkipVerify 默认为 false。在 Istio 1.9 版本中，VerifyCertAtClient 默认为 false，但在以后的版本中默认为 true |

# <a name="LocalityLoadBalancerSetting">LocalityLoadBalancerSetting</a>

位置加权负载均衡允许根据流量的来源和终止位置来控制到 endpoint 的流量分配。这些地区是使用任意标签指定的，这些标签以 {region}/{zone}/{sub-zone} 形式指定地区层次结构。有关更多详细信息，参阅[局部权重](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/load_balancing/locality_weight)。

以下示例显示如何在网格范围内设置局部权重：在一个网格中，其工作负载中的服务部署到了 us-west/zone1/ 和 us-west/zone2/ 中。当访问服务的流量源自 us-west/zone1/ 中时，80% 的流量将发送到 us-west/zone1/ 的 endpoint，即同一区域，其余的 20% 将进入 us-west/zone2/ 的 endpoint。此设置旨在将流量路由到同一位置的 endpoint。为源自 us-west/zone2/ 的流量指定了类似的设置。

```yaml
  distribute:
    - from: us-west/zone1/*
      to:
        "us-west/zone1/*": 80
        "us-west/zone2/*": 20
    - from: us-west/zone2/*
      to:
        "us-west/zone1/*": 20
        "us-west/zone2/*": 80
```

如果运营商的目标不是跨区域和区域分配负载，而是限制故障转移的区域性以满足其他运营要求，则运营商可以设置“故障转移”策略而不是“分布”策略。

以下示例为区域设置本地故障转移策略。假设服务驻留在 us-east、us-west 和 eu-west 内的区域中，此示例指定当 us-east 内的 endpoint 变得不健康时，流量应故障转移到 eu-west 内的 endpoint，类似地 us-west 应该故障转移到 us-east。

```yaml
 failover:
   - from: us-east
     to: eu-west
   - from: us-west
     to: us-east
```

| Field                                                 | Description                                                  |
| ----------------------------------------------------- | ------------------------------------------------------------ |
| [distribute](#LocalityLoadBalancerSetting.Distribute) | 只能设置 distribute、failover 或 failoverPriority 中的其一。指定跨不同区域和地理位置的负载平衡权重。参考 [Locality weighted load balancing](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/load_balancing/locality_weight)。如果为空，则根据其中的 endpoints 数量设置 locality 权重 |
| [failover](#LocalityLoadBalancerSetting.Failover)     | 只能设置 distribute、failover 或 failoverPriority 中的其一。当本地区域的 endpoint 变得不健康时，显式指定待转移流量的区域。应与 OutlierDetection 一起使用以检测不健康的 endpoint<br />*注意：生效的前提是指定了 OutlierDetection* |
| failoverPriority                                      | failoverPriority 是标签的有序列表，用于对 endpoint 进行排序以进行基于优先级的负载平衡。这是为了支持跨不同 endpoint 组的流量故障转移。假设总共指定了 N 个标签：<br />- 与客户端代理匹配所有 N 个标签的 endpoint 具有优先级 P(0)，即最高优先级<br />- 将前 N-1 个标签与客户端代理匹配的 endpoint 具有优先级 P(1)，即第二高优先级<br />- 通过扩展此逻辑，仅与客户端代理匹配第一个标签的 endpoint 具有优先级 P(N-1)，即第二低优先级<br />- 所有其他 endpoint 具有优先级 P(N)，即最低优先级<br />注意：对于要考虑匹配的标签，之前的标签必须匹配，即仅当前 n-1 个标签匹配时，才会认为第 n 个标签匹配<br />它可以是在客户端和服务器工作负载上指定的任何标签。还支持以下具有特殊语义含义的标签<br />- topology.istio.io/network 用于匹配 endpoint 的网络元数据，可以通过 pod/namespace 标签 topology.istio.io/network、sidecar env ISTIO_META_NETWORK 或 MeshNetworks 指定<br />- topology.istio.io/cluster 用于匹配一个 endpoint 的clusterID，可以通过 pod label topology.istio.io/cluster 或pod env ISTIO_META_CLUSTER_ID 指定<br />- topology.kubernetes.io/region 用于匹配 endpoint 的区域元数据，映射到 Kubernetes 节点标签 topology.kubernetes.io/region 或 failure-domain.beta.kubernetes.io/region（已弃用）<br />- topology.kubernetes.io/zone 用于匹配 endpoint 的 zone 元数据，映射到 Kubernetes 节点标签 topology.kubernetes.io/zone 或 failure-domain.beta.kubernetes.io/zone（已弃用）<br />- topology.istio.io/subzone 用于匹配 endpoint 的子区域元数据，映射到 Istio 节点标签 topology.istio.io/subzone<br />拓扑配置遵循以下优先级：<br />failoverPriority<br/>- topology.istio.io/network<br/>- topology.kubernetes.io/region<br/>- topology.kubernetes.io/zone<br/>- topology.istio.io/subzone<br />即 endpoint 和客户端代理的 [network, region, zone, subzone] label 的匹配度越高，则优先级越高<br />只能设置 distribute、failover 或 failoverPriority 中的其一。并且应该和 OutlierDetection 一起使用来检测不健康的 endpoint，否则没有效果 |
| enabled                                               | 是否启用局部负载平衡，为 DestinationRule 级别，将完全覆盖网格范围的设置<br />*例如 true 表示无论网格范围的设置是什么，都为此 DestinationRule 打开局部负载平衡* |

# <a name="TrafficPolicy.PortTrafficPolicy">TrafficPolicy.PortTrafficPolicy</a>

| Field                                     | Description                            |
| ----------------------------------------- | -------------------------------------- |
| port                                      | 指定应用此策略的目标服务上的端口号     |
| [loadBalancer](#LoadBalancerSettings)     | 控制负载均衡器算法的设置               |
| [connectionPool](#ConnectionPoolSettings) | 控制与上游服务的连接量的设置           |
| [outlierDetection](#OutlierDetection)     | 控制从负载均衡池中逐出不健康主机的设置 |
| [tls](#ClientTLSSettings)                 | 与上游服务连接的 TLS 相关设置          |

# <a name="TrafficPolicy.TunnelSettings">TrafficPolicy.TunnelSettings</a>

| Field      | Description                                                  |
| ---------- | ------------------------------------------------------------ |
| protocol   | 指定用于隧道下行连接的协议。支持的协议有：connect - 使用 HTTP CONNECT； post - 使用 HTTP POST。上游请求的 HTTP 版本由为代理定义的服务协议确定 |
| targetHost | 指定下游连接通过隧道连接到的主机。目标主机必须是 FQDN 或 IP 地址 |
| targetPort | 指定下游连接通过隧道连接到的端口                             |

# <a name="LoadBalancerSettings.ConsistentHashLB">LoadBalancerSettings.ConsistentHashLB</a>

一致的基于哈希的负载均衡可用于提供基于 HTTP 标头、cookie 或其他属性的软会话亲和性。当从目标服务中添加/删除一个或多个主机时，与特定目标主机的关联将被移除。

| Field                                                        | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| httpHeaderName                                               | 基于特定 HTTP 标头的 hash                                    |
| [httpCookie](#LoadBalancerSettings.ConsistentHashLB.HTTPCookie) | 基于 HTTP cookie 的 hash                                     |
| useSourceIp                                                  | 基于源 IP 地址的 hash。适用于 TCP 和 HTTP 连接               |
| httpQueryParameterName                                       | 基于特定 HTTP query 参数的 hash                              |
| minimumRingSize                                              | 用于哈希环的最小虚拟节点数。默认为 1024。较大的环尺寸会产生更精细的负载分布。如果负载均衡池中的主机数量大于环大小，则每个主机将被分配一个虚拟节点 |

# <a name="LoadBalancerSettings.ConsistentHashLB.HTTPCookie">LoadBalancerSettings.ConsistentHashLB.HTTPCookie</a>

描述将用作一致哈希负载均衡器的哈希键的 HTTP cookie。如果 cookie 不存在，则会自动生成

| Field | Description          |
| ----- | -------------------- |
| name  | cookie 的名称        |
| path  | 为 cookie 设置的路径 |
| ttl   | cookie 的生命周期    |

# <a name="ConnectionPoolSettings.TCPSettings">ConnectionPoolSettings.TCPSettings</a>

HTTP 和 TCP 上游连接通用的设置。

| Field                                                        | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| maxConnections                                               | 到目标主机的最大 HTTP1 /TCP 连接数。默认 2^32-1              |
| connectTimeout                                               | TCP 连接超时。格式：1h/1m/1s/1ms。必须 >=1ms。默认为 10s     |
| [tcpKeepalive](#ConnectionPoolSettings.TCPSettings.TcpKeepalive) | 如果设置，则在套接字上设置 SO_KEEPALIVE 以启用 TCP Keepalive |

# <a name="ConnectionPoolSettings.HTTPSettings">ConnectionPoolSettings.HTTPSettings</a>

适用于 HTTP1.1/HTTP2/GRPC 连接的设置。

| Field                                                        | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| http1MaxPendingRequests                                      | 目标的最大挂起 HTTP 请求数。默认 2^32-1                      |
| http2MaxRequests                                             | 后端的最大请求数。默认 2^32-1                                |
| maxRequestsPerConnection                                     | 每个连接到后端的最大请求数。将此参数设置为 1 将禁用 alive。默认为 0，表示无限制，最大为 2^29 |
| maxRetries                                                   | 在给定时间可以对集群中的所有主机进行的最大重试次数。默认为 2^32-1 |
| idleTimeout                                                  | 上游连接池连接的空闲超时。空闲超时定义为没有 alive 请求的时间段。如果未设置，则默认为 1 小时。当达到空闲超时时，连接将被关闭。如果连接是 HTTP/2 连接，则会在关闭连接之前发生耗尽序列。请注意，基于请求的超时意味着 HTTP/2 PING 不会使连接保持活动状态。适用于 HTTP1.1 和 HTTP2 连接 |
| [h2UpgradePolicy](#ConnectionPoolSettings.HTTPSettings.H2UpgradePolicy) | 指定是否应将关联目标的 http1.1 连接升级到 http2              |
| useClientProtocol                                            | 如果设置为 true，则启动与后端的连接时将保留客户端协议。请注意，当设置为 true 时，h2UpgradePolicy 将无效，即客户端连接不会升级到 http2 |

# <a name="ConnectionPoolSettings.TCPSettings.TcpKeepalive">ConnectionPoolSettings.TCPSettings.TcpKeepalive</a>

| Field    | Description                                                  |
| -------- | ------------------------------------------------------------ |
| probes   | 在确定连接已死之前，要发送且无响应的最大保活探测数。默认是使用操作系统级别的配置（除非被覆盖，Linux 默认为 9) |
| time     | 在开始发送 keep-alive 探测之前，连接需要空闲的持续时间。默认是使用操作系统级别的配置（除非被覆盖，Linux 默认为 7200s） |
| interval | 保持活动探测之间的持续时间。默认是使用操作系统级别的配置（除非被覆盖，Linux 默认为 75s） |

# <a name="LocalityLoadBalancerSetting.Distribute">LocalityLoadBalancerSetting.Distribute</a>

描述源自 from 区域或子区域的流量如何分布在一组 to 区域上。指定区域的语法是 {region}/{zone}/{sub-zone} 并且规范的任何部分都允许使用终端通配符。例如：

* \* 匹配所有的地区
* us-west/* 匹配 us-west 区域内的所有区域和子区域
* us-west/zone-1/* 匹配 us-west/zone-1 中的所有子区域

| Field | Description                                                  |
| ----- | ------------------------------------------------------------ |
| from  | 源位置，用 / 分隔，例如 region/zone/sub_zone                 |
| to    | 上游地区到流量分布权重的地图。所有权重的总和应为 100。任何不存在的位置都不会收到流量 |

# <a name="LocalityLoadBalancerSetting.Failover">LocalityLoadBalancerSetting.Failover</a>

指定跨区域的流量故障转移策略。由于默认情况下支持区域和子区域故障转移，因此仅当运营商需要限制流量故障转移时才需要为区域指定，以便全局故障转移到任何 endpoint 的默认行为不适用。这在跨区域故障转移流量不会改善服务健康或可能需要因监管控制等其他原因而受到限制时非常有用。

| Field | Description                                                  |
| ----- | ------------------------------------------------------------ |
| from  | 源区域                                                       |
| to    | 当 from 区域中的 endpoint 变得不健康时，流量将故障转移到目标区域 |

# <a name="SimpleLB">LoadBalancerSettings.SimpleLB</a>

无需调整的标准负载均衡算法。

| Name          | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| UNSPECIFIED   | 用户未指定负载均衡算法。 Istio 将选择适当的默认值            |
| RANDOM        | 随机负载均衡器选择一个随机的健康主机。如果没有配置健康检查策略，随机负载均衡器的性能通常比轮询更好 |
| PASSTHROUGH   | 此选项会将连接转发到调用者请求的原始 IP 地址，而不进行任何形式的负载平衡。慎重使用。它适用于高级用例。有关更多详细信息，参阅 Envoy 中的原始目标负载均衡器 |
| ROUND_ROBIN   | 一个基本的循环负载均衡策略。这对于许多场景（例如，使用 endpoint 加权时）通常是不安全的，因为它可能会使 endpoint 负担过重。一般来说，倾向于使用 LEAST_REQUEST 替代 ROUND_ROBIN |
| LEAST_REQUEST | 最少请求负载均衡器将负载分散到各个 endpoint，倾向于具有最少未完成请求的 endpoint。这通常更安全，并且几乎在所有情况下都优于 ROUND_ROBIN。倾向于使用 LEAST_REQUEST 替代 ROUND_ROBIN |
| LEAST_CONN    | 已弃用。改用 LEAST_REQUEST                                   |

# <a name="ConnectionPoolSettings.HTTPSettings.H2UpgradePolicy">ConnectionPoolSettings.HTTPSettings.H2UpgradePolicy</a>

将 http1.1 连接升级到 http2 的策略。

| Name           | Description                        |
| -------------- | ---------------------------------- |
| DEFAULT        | 使用全局默认值                     |
| DO_NOT_UPGRADE | 不将连接升级到 http2。会覆盖默认值 |
| UPGRADE        | 将连接升级到 http2。会覆盖默认值   |

# <a name="ClientTLSSettings.TLSmode">ClientTLSSettings.TLSmode</a>

| Name         | Description                                                  |
| ------------ | ------------------------------------------------------------ |
| DISABLE      | 不设置到上游 endpoint 的 TLS 连接                            |
| SIMPLE       | 发起到上游 endpoint 的 TLS 连接                              |
| MUTUAL       | 通过提供客户端证书进行身份验证，使用双向 TLS 保护与上游的连接 |
| ISTIO_MUTUAL | 通过提供客户端证书进行身份验证，使用双向 TLS 保护与上游的连接。与 MUTUAL 模式相比，该模式使用 Istio 自动生成的证书进行 mTLS 身份验证。使用此模式时，ClientTLSSettings 中的所有其他字段都应为空 |
