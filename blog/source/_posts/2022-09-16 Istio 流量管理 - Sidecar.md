---
title: "「 Istio 」流量管理 — Sidecar"
excerpt: "Istio 中流量管理组件 Sidecar 对象介绍"
cover: https://picsum.photos/0?sig=20220916
thumbnail: /gallery/istio/istio-thumbnail.png
date: 2022-09-16
toc: true
categories:
- Overview
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="https://landscape.cncf.io/logos/istio.svg"></div>

------

> Based on **v1.15.0**

Sidecar 描述了 sidecar 代理的配置，该代理将入站和出站通信调解到它所连接的工作负载实例。默认情况下，Istio 将对网格中的所有代理进行编程，并使用必要的配置来访问网格中的每个工作负载实例，并接管与工作负载关联的所有端口上的流量。 Sidecar 配置提供了一种微调端口集的方法，代理在转发流量进出工作负载时将接受的协议。此外，可以限制代理在转发来自工作负载实例的出站流量时可以访问的服务集。

网格中的服务和配置被划分成一个或多个命名空间（例如，Kubernetes 命名空间或 CF org/space）。命名空间中的 Sidecar 配置将应用于同一命名空间中的一个或多个工作负载实例，使用 workloadSelector 字段选择。在没有 workloadSelector 的情况下，它将应用于同一命名空间中的所有工作负载实例。在确定要应用于工作负载实例的 Sidecar 配置时，将优先考虑具有选择此工作负载实例的 workloadSelector 的资源，而不是没有任何 workloadSelector 的 Sidecar 配置。

注意点

- 每个命名空间只能有一个没有 workloadSelector 的  Sidecar 配置，该配置为该命名空间中的所有 Pod 指定默认值。建议对命名空间范围的 sidecar 使用名称 default。如果给定命名空间中存在多个无选择器的 Sidecar 配置，则系统的行为是未定义的。如果具有 workloadSelector 的两个或多个 Sidecar 配置选择相同的工作负载实例，则系统的行为是未定义的
- 默认情况下，MeshConfig 根命名空间中的 Sidecar 配置将应用于所有没有 Sidecar 配置的命名空间。这个全局默认 Sidecar 配置不应该有任何 workloadSelector 

下面的示例在名为 istio-config 的命名空间中声明了一个全局默认 Sidecar 配置，该配置将所有命名空间中的 Sidecar 配置为仅允许出口流量到同一命名空间中的其他工作负载以及 istio-system 命名空间中的服务：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: default
  namespace: istio-config
spec:
  egress:
  - hosts:
    - "./*"
    - "istio-system/*"
```

下面的示例在 prod-us1 命名空间中声明了一个 Sidecar 配置，它覆盖了上面定义的全局默认值，并在命名空间中配置了 Sidecar 以允许出口流量到 prod-us1、prod-apis 和 istio-system 中的命名空间。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: default
  namespace: prod-us1
spec:
  egress:
  - hosts:
    - "prod-us1/*"
    - "prod-apis/*"
    - "istio-system/*"
```

以下示例在 prod-us1 命名空间中为所有带有标签 app: rating 的 Pod 声明了 Sidecar 配置，属于 rating.prod-us1 服务。工作负载在端口 9080 上接受入站 HTTP 流量。然后将流量转发到在 Unix 域套接字上侦听的附加工作负载实例。在出口方向，除了 istio-system 命名空间，Sidecar 仅代理 prod-us1 命名空间中服务的 9080 端口的 HTTP 流量。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: ratings
  namespace: prod-us1
spec:
  workloadSelector:
    labels:
      app: ratings
  ingress:
  - port:
      number: 9080
      protocol: HTTP
      name: somename
    defaultEndpoint: unix:///var/run/someuds.sock
  egress:
  - port:
      number: 9080
      protocol: HTTP
      name: egresshttp
    hosts:
    - "prod-us1/*"
  - hosts:
    - "istio-system/*"
```

如果在没有基于 IPTables 的流量捕获的情况下部署工作负载，则 Sidecar 配置是连接到工作负载实例的代理上的端口的唯一方法。

以下示例在 prod-us1 命名空间中为所有带有 app: productpage 标签的 Pod 声明了 Sidecar 配置，属于 productpage.prod-us1 服务。假设这些 Pod 部署时没有 IPtable 规则（即 istio-init 容器）并且代理中 metadata 的 ISTIO_META_INTERCEPTION_MODE 设置为 NONE，下面的规范允许这些 Pod 在端口 9080 上接收 HTTP 流量（包裹在 Istio 双向 TLS 中）和将其转发到监听 127.0.0.1:8080 的应用程序。它还允许应用程序与 127.0.0.1:3306 上的支持 MySQL 数据库通信，然后将其代理到 mysql.foo.com:3306 上的外部托管 MySQL 服务。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: no-ip-tables
  namespace: prod-us1
spec:
  workloadSelector:
    labels:
      app: productpage
  ingress:
  - port:
      number: 9080 # binds to proxy_instance_ip:9080 (0.0.0.0:9080, if no unicast IP is available for the instance)
      protocol: HTTP
      name: somename
    defaultEndpoint: 127.0.0.1:8080
    captureMode: NONE # not needed if metadata is set for entire proxy
  egress:
  - port:
      number: 3306
      protocol: MYSQL
      name: egressmysql
    captureMode: NONE # not needed if metadata is set for entire proxy
    bind: 127.0.0.1
    hosts:
    - "*/mysql.foo.com"
```

以及用于路由到 mysql.foo.com:3306 的关联 ServiceEntry：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-mysql
  namespace: ns1
spec:
  hosts:
  - mysql.foo.com
  ports:
  - number: 3306
    name: mysql
    protocol: MYSQL
  location: MESH_EXTERNAL
  resolution: DNS
```

还可以在单个代理中混合和匹配流量捕获模式。例如，考虑内部服务位于 192.168.0.0/16 子网上的设置。因此，在 VM 上设置 IP 表以捕获 192.168.0.0/16 子网上的所有出站流量。假设 VM 在 172.16.0.0/16 子网上有一个额外的网络接口用于入站流量。以下 Sidecar 配置允许 VM 在 172.16.1.32:80（VM 的 IP）上公开一个侦听器，以接收来自 172.16.0.0/16 子网的流量。

注意：VM 中代理上的 ISTIO_META_INTERCEPTION_MODE 元数据可选值有 REDIRECT 或 TPROXY，这意味着当前是基于 IP 表的流量捕获。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: partial-ip-tables
  namespace: prod-us1
spec:
  workloadSelector:
    labels:
      app: productpage
  ingress:
  - bind: 172.16.1.32
    port:
      number: 80 # binds to 172.16.1.32:80
      protocol: HTTP
      name: somename
    defaultEndpoint: 127.0.0.1:8080
    captureMode: NONE
  egress:
    # use the system detected defaults
    # sets up configuration to handle outbound traffic to services
    # in 192.168.0.0/16 subnet, based on information provided by the
    # service registry
  - captureMode: IPTABLES
    hosts:
    - "*/*"
```

以下示例在 prod-us1 命名空间中为所有带有标签 app: rating 的 Pod 声明了 Sidecar 配置，属于 rating.prod-us1 服务。该服务在端口 8443 上接受入站 HTTPS 流量，并且 sidecar 代理使用给定的服务器证书以一种方式终止 TLS。然后将流量转发到在 Unix 域套接字上侦听的附加工作负载实例。预计将配置 PeerAuthentication 策略，以便在特定端口上将 mTLS 模式设置为“禁用”。在此示例中，在 PORT 80 上禁用了 mTLS 模式。此功能目前是实验性的。

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: ratings-peer-auth
  namespace: prod-us1
spec:
  selector:
    matchLabels:
      app: ratings
  mtls:
    mode: STRICT
  portLevelMtls:
    80:
      mode: DISABLE
```

# Sidecar

| Field                                 | Description                                                  |
| ------------------------------------- | ------------------------------------------------------------ |
| [workloadSelector](#WorkloadSelector) | 用于选择应用此 Sidecar 配置的特定 Pod/VM 集的标准。如果省略，Sidecar 配置将应用于同一命名空间中的所有工作负载实例 |
| [ingress](#IstioIngressListener)      | Ingress 指定 Sidecar 的配置，用于处理附加工作负载实例的入站流量。如果省略，Istio 将根据从编排平台获得的工作负载信息（例如，暴露的端口、服务等）自动配置 sidecar。如果指定，当且仅当工作负载实例与服务关联时，才会配置入站端口 |
| [egress](#IstioEgressListener)        | Egress 指定 sidecar 的配置，用于处理从附加工作负载实例到网格中其他服务的出站流量。如果未指定，则从命名空间范围或全局默认 Sidecar 继承系统检测到的默认值 |
| outboundTrafficPolicy                 | 出方向流量策略的配置。如果应用程序使用一个或多个未知的外部服务，将策略设置为 ALLOW_ANY 将导致 sidecar 将来自应用程序的任何未知流量路由到其请求的目的地。如果未指定，则从命名空间范围或全局默认 Sidecar 继承系统检测到的默认值 |

# <a name="IstioIngressListener">IstioIngressListener</a>

| Field                       | Description                                                  |
| --------------------------- | ------------------------------------------------------------ |
| port                        | 与 listener 关联的端口                                       |
| bind                        | listener 应绑定到的 IP（IPv4 或 IPv6）。入口 listener 的绑定字段中不允许使用 Unix 域套接字地址。如果省略，Istio 将根据导入的服务和应用此配置的工作负载实例自动配置默认值 |
| [captureMode](#CaptureMode) | 表示如何捕获（或不捕获）到 listener 的流量                   |
| defaultEndpoint             | 应将流量转发到的 IP 端点或 Unix 域套接字。此配置可用于将到达 sidecar 上的绑定 IP:Port 的流量重定向到应用程序工作负载实例正在侦听连接的 localhost:port 或 Unix 域套接字。不支持任意 IP。格式应该是 127.0.0.1:PORT、[::1]:PORT（转发到 localhost）、0.0.0.0:PORT、[::]:PORT（转发到实例 IP）或 unix:/// 之一path/to/socket（转发到 Unix 域套接字） |
| tls                         | 一组 TLS 相关选项，将在 sidecar 上为来自网格外部的请求启用 TLS 终止。目前仅支持 SIMPLE 和 MUTUAL TLS 模式 |

# <a name="IstioEgressListener">IstioEgressListener</a>

| Field                       | Description                                                  |
| --------------------------- | ------------------------------------------------------------ |
| port                        | 与 listener 关联的端口。如果使用 Unix 域套接字，请使用 0 作为端口号，并使用有效的协议。如果指定了端口，将用作与导入主机关联的默认目标端口。如果省略端口，Istio 将根据导入的主机推断 listener 端口。请注意，当指定多个出口 listener 时，其中一个或多个侦听器具有特定端口而其他 listener 没有端口，则在 listener 端口上暴露的主机将基于具有最特定端口的 listener |
| bind                        | listener 应绑定到的 IP（IPv4 或 IPv6）或 Unix 域套接字。如果 bind 不为空，则必须指定端口。格式：IPv4 或 IPv6 地址格式或 unix:///path/to/uds 或 unix://@foobar（Linux 抽象命名空间）。如果省略，Istio 将根据导入的服务、应用此配置的工作负载实例和 captureMode 自动配置默认值。如果 captureMode 为 NONE，bind 将默认为 127.0.0.1 |
| [captureMode](#CaptureMode) | 当绑定地址是 IP 时，captureMode 选项指示如何捕获（或不捕获）到 listener 的流量。对于 Unix 域套接字绑定，captureMode 必须为 DEFAULT 或 NONE |
| hosts                       | listener 以 namespace/dnsName 格式暴露一个或多个服务主机。与 dnsName 匹配的指定命名空间中的服务将被暴露。相应的服务可以是服务注册表中的服务（例如，Kubernetes 或云服务）或使用 ServiceEntry 或 VirtualService 配置指定的服务。还将使用同一命名空间中的任何关联 DestinationRule。<br />应使用 FQDN 格式指定 dnsName，可选择在最左侧的组件中包含通配符（例如 prod/*.example.com）。将 dnsName 设置为 * 以选择指定命名空间中的所有服务（例如 prod/*）。<br />命名空间可以设置为 *、. 或 ~，分别表示任何命名空间、当前命名空间或无命名空间。例如，*/foo.example.com 从任何可用的命名空间中选择服务，而 ./foo.example.com 仅从 sidecar 的命名空间中选择服务。如果主机设置为 */*，Istio 将配置 sidecar 以便能够访问网格中导出到 sidecar 命名空间的每个服务。值 ~/* 可用于完全修剪 Sidecar 的配置，这些 Sidecar 仅接收流量并响应，但不建立自己的出站连接。<br />只能引用导出到 sidecar 命名空间的服务和配置工件（例如，exportTo 的 * 值）。私有配置（例如，exportTo 设置为 .） |

# <a name="WorkloadSelector">WorkloadSelector</a>

WorkloadSelector 指定用于确定是否可以将 Gateway、Sidecar、EnvoyFilter、ServiceEntry 或 DestinationRule 配置应用于代理的标准。匹配条件包括与代理关联的元数据、工作负载实例信息（例如附加到 pod/VM 的标签）或代理在初始握手期间提供给 Istio 的任何其他信息。如果指定了多个条件，则需要匹配所有条件才能选择工作负载实例。目前，仅支持基于标签的选择机制。

| Field  | Description                                                  |
| ------ | ------------------------------------------------------------ |
| labels | 一个或多个标签，指示应用配置的一组特定 Pod/VM。标签搜索的范围仅限于资源所在的配置命名空间 |

# <a name="OutboundTrafficPolicy">OutboundTrafficPolicy</a>

| Field                               | Description                                                  |
| ----------------------------------- | ------------------------------------------------------------ |
| [mode](#OutboundTrafficPolicy.Mode) | 设置 sidecar 的默认行为以处理来自应用程序的出站流量。如果应用程序使用一个或多个先验未知的外部服务，将策略设置为 ALLOW_ANY 将导致边车将来自应用程序的任何未知流量路由到其请求的目的地。强烈建议用户使用 ServiceEntry 配置来显式声明任何外部依赖项，而不是使用 ALLOW_ANY，以便可以监控到这些服务的流量 |

# <a name="OutboundTrafficPolicy.Mode">OutboundTrafficPolicy.Mode</a>

| Name          | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| REGISTRY_ONLY | 出站流量将仅限于服务注册表中定义的服务以及通过 ServiceEntry 配置定义的服务 |
| ALLOW_ANY     | 如果目标端口没有服务或 ServiceEntry 配置，则将允许到未知目标的出站流量 |

# <a name="CaptureMode">CaptureMode</a>

| Name     | Description                                                  |
| -------- | ------------------------------------------------------------ |
| DEFAULT  | 环境定义的默认捕获模式                                       |
| IPTABLES | 使用 IPtables 重定向捕获流量                                 |
| NONE     | 没有流量捕获。当在出口 listener 中使用时，应用程序应与 listener 端口或 Unix 域套接字显式通信。在入口 listener 中使用时，需要注意确保 listener 端口没有被主机上的其他进程使用 |
