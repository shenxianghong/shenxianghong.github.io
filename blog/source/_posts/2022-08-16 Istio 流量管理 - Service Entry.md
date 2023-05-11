---
title: "「 Istio 」流量管理 — ServiceEntry"
excerpt: "Istio 流量管理场景下的 ServiceEntry 资源对象使用范例与 API 结构概览"
cover: https://picsum.photos/0?sig=20220816
thumbnail: https://github.com/cncf/artwork/raw/master/projects/istio/stacked/color/istio-stacked-color.svg
date: 2022-08-16
toc: true
categories:
- Service Mesh
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="https://github.com/cncf/artwork/raw/master/projects/istio/horizontal/color/istio-horizontal-color.svg"></div>

------

> based on **1.15.0**

ServiceEntry 允许将外部的服务添加到 Istio 的内部服务注册表中，以便网格中的服务可以访问/路由到这些手动指定的服务。ServiceEntry 描述了服务的属性（DNS 名称、VIP、端口、协议、端点）。这些服务可能在网格外部（例如，Web API）或网格内部服务，它们不属于平台的服务注册表（例如，一组与 Kubernetes 中的服务通信的 VM）。此外，还可以使用 workloadSelector 字段动态选择 ServiceEntry  的 endpoint。这些 endpoint 可以是使用 WorkloadEntry 对象或 Kubernetes Pod 声明的 VM 工作负载。在单个服务下同时选择 Pod 和 VM 的能力允许将服务从 VM 迁移到 Kubernetes，而无需更改与服务关联的现有 DNS 名称。

以下示例中声明了一些由内部应用程序通过 HTTPS 访问的外部 API。 Sidecar 检查 ClientHello 消息中的 SNI 值以路由到适当的外部服务。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-https
spec:
  hosts:
  - api.dropboxapi.com
  - www.googleapis.com
  - api.facebook.com
  location: MESH_EXTERNAL
  ports:
  - number: 443
    name: https
    protocol: TLS
  resolution: DNS
```

以下配置将一组运行在非托管 VM 上的 MongoDB 实例添加到 Istio 的注册表中，以便可以将这些服务视为网格中的服务。关联的 DestinationRule 用于启动与数据库实例的 mTLS 连接。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-mongocluster
spec:
  hosts:
  - mymongodb.somedomain # not used
  addresses:
  - 192.192.192.192/24 # VIPs
  ports:
  - number: 27018
    name: mongodb
    protocol: MONGO
  location: MESH_INTERNAL
  resolution: STATIC
  endpoints:
  - address: 2.2.2.2
  - address: 3.3.3.3
```

相关联的 DestinationRule。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: mtls-mongocluster
spec:
  host: mymongodb.somedomain
  trafficPolicy:
    tls:
      mode: MUTUAL
      clientCertificate: /etc/certs/myclientcert.pem
      privateKey: /etc/certs/client_private_key.pem
      caCertificates: /etc/certs/rootcacerts.pem
```

以下示例结合使用 VirtualService 中的 ServiceEntry 和 TLS 路由，根据 SNI 值将流量引导至内部出口防火墙。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-redirect
spec:
  hosts:
  - wikipedia.org
  - "*.wikipedia.org"
  location: MESH_EXTERNAL
  ports:
  - number: 443
    name: https
    protocol: TLS
  resolution: NONE
```

基于 SNI 值路由的相关联的 VirtualService。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: tls-routing
spec:
  hosts:
  - wikipedia.org
  - "*.wikipedia.org"
  tls:
  - match:
    - sniHosts:
      - wikipedia.org
      - "*.wikipedia.org"
    route:
    - destination:
        host: internal-egress-firewall.ns1.svc.cluster.local
```

具有 TLS 匹配的 Virtual Service 用于覆盖默认的 SNI 匹配。在没有 Virtual Service 的情况下，流量将被转发到 wikipedia 。

以下示例中演示了使用专用出口网关，通过该网关转发所有外部服务流量。 “exportTo” 字段允许控制服务声明对网格中其他命名空间的可见性。默认情况下，服务会导出到所有命名空间。下面的例子限制了当前命名空间的可见性，用 “.” 表示，所以它不能被其他命名空间使用。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-httpbin
  namespace : egress
spec:
  hosts:
  - example.com
  exportTo:
  - "."
  location: MESH_EXTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
  resolution: DNS
```

定义一个 Gateway 来处理所有出口流量。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
 name: istio-egressgateway
 namespace: istio-system
spec:
 selector:
   istio: egressgateway
 servers:
 - port:
     number: 80
     name: http
     protocol: HTTP
   hosts:
   - "*"
```

关联的 VirtualService 从 Sidecar 路由到网关服务（istio-egressgateway.istio-system.svc.cluster.local），同样的从 Gateway 路由到外部服务。VirtualService 被导出到所有命名空间，使它们能够通过 Gateway 将流量路由到外部服务。像这样强制流量通过托管中间代理是一种常见做法。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: gateway-routing
  namespace: egress
spec:
  hosts:
  - example.com
  exportTo:
  - "*"
  gateways:
  - mesh
  - istio-egressgateway
  http:
  - match:
    - port: 80
      gateways:
      - mesh
    route:
    - destination:
        host: istio-egressgateway.istio-system.svc.cluster.local
  - match:
    - port: 80
      gateways:
      - istio-egressgateway
    route:
    - destination:
        host: example.com
```

以下示例演示了在主机中为外部服务使用通配符。如果必须将连接路由到应用程序请求的 IP 地址（即应用程序解析 DNS 并尝试连接到特定 IP），则必须将 resolution 设置为 NONE。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-wildcard-example
spec:
  hosts:
  - "*.bar.com"
  location: MESH_EXTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
  resolution: NONE
```

以下示例演示了可通过客户端主机上的 Unix 域套接字获得的服务。resolution 必须设置为 STATIC 才能使用 Unix 地址 endpoint。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: unix-domain-socket-example
spec:
  hosts:
  - "example.unix.local"
  location: MESH_EXTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
  resolution: STATIC
  endpoints:
  - address: unix:///var/run/example/socket
```

对于基于 HTTP 的服务，可以创建一个由多个 DNS 可寻址 endpoint 支持的 VirtualService。在这种情况下，应用程序可以使用 HTTP_PROXY 环境变量透明地将 VirtualService 的 API 调用重新路由到选定的后端。

例如，以下配置创建了一个名为 foo.bar.com 的不存在的外部服务，后端：us.foo.bar.com:8080、uk.foo.bar.com:9080 和 in.foo.bar.com:7080。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-dns
spec:
  hosts:
  - foo.bar.com
  location: MESH_EXTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
  resolution: DNS
  endpoints:
  - address: us.foo.bar.com
    ports:
      http: 8080
  - address: uk.foo.bar.com
    ports:
      http: 9080
  - address: in.foo.bar.com
    ports:
      http: 7080
```

使用 HTTP_PROXY=http://localhost/，从应用程序到 http://foo.bar.com 的调用将在上面指定的三个域之间进行负载平衡。换句话说，对 http://foo.bar.com/baz 的调用将被转换为 http://uk.foo.bar.com/baz。

以下示例说明了包含 subjectAltNames 的 ServiceEntry 的用法，该名称的格式符合 SPIFFE 标准。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: httpbin
  namespace : httpbin-ns
spec:
  hosts:
  - example.com
  location: MESH_INTERNAL
  ports:
  - number: 80
    name: http
    protocol: HTTP
  resolution: STATIC
  endpoints:
  - address: 2.2.2.2
  - address: 3.3.3.3
  subjectAltNames:
  - "spiffe://cluster.local/ns/httpbin-ns/sa/httpbin-service-account"
```

以下示例演示了使用 ServiceEntry 和 workloadSelector 来处理服务 details.bookinfo.com 从 VM 到 Kubernetes 的迁移。该服务有两个基于 VM 的实例和 Sidecar，以及一组由标准部署对象管理的 Kubernetes Pod。网格中此服务的使用者将在 VM 和 Kubernetes 之间自动进行负载平衡。用于 details.bookinfo.com 服务的 VM。此 VM 已使用 details-legacy Service Account 安装和引导 Sidecar。 Sidecar 在端口 80 上接收 HTTP 流量（包装在 istio 双向 TLS 中）并将其转发到同一端口上 localhost 上的应用程序。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: WorkloadEntry
metadata:
  name: details-vm-1
spec:
  serviceAccount: details
  address: 2.2.2.2
  labels:
    app: details
    instance-id: vm1
---
apiVersion: networking.istio.io/v1beta1
kind: WorkloadEntry
metadata:
  name: details-vm-2
spec:
  serviceAccount: details
  address: 3.3.3.3
  labels:
    app: details
    instance-id: vm2
```

假设还有一个 Deployment 带有 Pod 标签 app: details 使用相同的 ServiceAccount（即 details），以下 ServiceEntry 声明了一个跨 VM 和 Kubernetes 的服务：

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
  resolution: STATIC
  workloadSelector:
    labels:
      app: details
```

# ServiceEntry

| Field                                  | Description                                                  |
| -------------------------------------- | ------------------------------------------------------------ |
| hosts                                  | 与 ServiceEntry 关联的主机。可以是带有通配符前缀的 DNS 名称。<br />- hosts 字段用于在 VirtualServices 和 DestinationRules 中选择匹配的主机<br />- 对于 HTTP 流量，HTTP Host/Authority 标头将与 hosts 字段匹配<br />- 对于包含服务器名称指示（SNI） 的 HTTPs 或 TLS 流量，SNI 值将与 hosts 字段匹配<br /><br />NOTE：<br />- 当解析设置为 DNS 类型且未指定 endpoint 时，主机字段将用作将流量路由到的 endpoint 的 DNS 名称<br />- 如果主机名与另一个服务注册中心的服务名称匹配，例如 Kubernetes，它也提供自己的一组 endpoint，则 ServiceEntry 将被视为现有 Kubernetes 服务的装饰器。如果适用，Service Entry 中的属性将添加到 Kubernetes 服务中。目前，istiod 只会考虑以下附加属性：<br />- subjectAltNames：除了验证与服务的 pod 关联的服务帐户的 SAN 之外，还将验证此处指定的 SAN |
| addresses                              | 与服务关联的虚拟 IP 地址。可能是 CIDR 前缀。对于 HTTP 流量，生成的路由配置将包括地址和主机字段值的 http 路由域，并且将根据 HTTP Host/Authority 标头识别目标。如果指定了一个或多个 IP 地址，如果目标 IP 与地址字段中指定的 IP/CIDR 匹配，则传入流量将被标识为属于此服务。如果地址字段为空，则将仅根据目标端口识别流量。在这种情况下，服务被访问的端口不能被网格中的任何其他服务共享。换句话说，Sidecar 将充当简单的 TCP 代理，将指定端口上的传入流量转发到指定的目标 endpoint IP/主机。此字段不支持 Unix 域套接字地址 |
| ports                                  | 与外部服务关联的端口。如果 endpoint 是 Unix 域套接字地址，则必须只有一个端口 |
| [location](#ServiceEntry.Location)     | 指定是否应将服务视为网格外部或网格的一部分                   |
| [resolution](#ServiceEntry.Resolution) | 主机的服务发现模式。为没有附带 IP 地址的 TCP 端口设置解析模式为 NONE 时需要注意，在这种情况下，将允许到所述端口上的任何 IP 的流量（即 0.0.0.0:\<port\>） |
| endpoints                              | 与服务关联的一个或多个 endpoint。只能指定 endpoints 或 workloadSelector 之一 |
| workloadSelector                       | 仅适用于 MESH_INTERNAL 服务。只能指定 endpoint 或工作负载选择器之一。根据标签选择一个或多个 Kubernetes Pod 或 VM 工作负载（使用 WorkloadEntry 指定）。表示 VM 的 WorkloadEntry 对象应与 ServiceEntry 定义在相同的命名空间中 |
| exportTo                               | 此服务导出到的命名空间列表。导出服务允许它被其他命名空间中定义的 Sidecar、Gateway 和 VirtualService 使用。此功能为服务所有者和网格管理员提供了一种机制来控制跨命名空间边界的服务的可见性。<br />如果未指定命名空间，则默认将服务导出到所有命名空间。<br /> ”.” 保留语义表示导出到声明服务的同一命名空间。类似地，“*” 保留语义定义导出到所有命名空间。<br />对于 Kubernetes Service，可以通过将注解 networking.istio.io/exportTo 设置为以逗号分隔的命名空间名称列表来实现等效效果 |
| subjectAltNames                        | 如果指定，代理将验证服务器证书的 subject alternate name 是否与指定值之一匹配。<br />注意：将 workloadEntry  与 workloadSelector 一起使用时，workloadEntry 中指定的 ServiceAccount 也将用于派生应验证的其他 subject alternate name |

# <a name="ServiceEntry.Location">ServiceEntry.Location</a>

location 指定服务是 Istio 网格的一部分还是网格之外。location 决定了几个特性的行为，例如服务到服务的 mTLS 身份验证、策略执行等。当与网格外的服务通信时，Istio 的 mTLS 身份验证被禁用，并且策略执行在客户端执行，而不是在服务端。

| Name          | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| MESH_EXTERNAL | 表示服务在网格外部。通常用于指示通过 API 使用的外部服务      |
| MESH_INTERNAL | 表示服务是网格的一部分。通常用于指示作为扩展服务网格以包括非托管基础设施的一部分而显式添加的服务（例如，添加到基于 Kubernetes 的服务网格的虚拟机） |

# <a name="ServiceEntry.Resolution">ServiceEntry.Resolution</a>

resolution 决定代理将如何解析与服务关联的网络 endpoint 的 IP 地址，以便它可以路由到其中一个。此处指定的解析模式对应用程序如何解析与服务关联的 IP 地址没有影响。应用程序可能仍需要使用 DNS 将服务解析为 IP，以便代理可以捕获出站流量。或者，对于 HTTP 服务，应用程序可以直接与代理通信（例如，通过设置 HTTP_PROXY）来与这些服务通信。

| Name            | Description                                                  |
| --------------- | ------------------------------------------------------------ |
| NONE            | 假设传入的连接已经被解析（到特定的目标 IP 地址）。此类连接通常使用 IP 表 REDIRECT/eBPF 等机制通过代理进行路由。在执行任何与路由相关的转换后，代理会将连接转发到连接绑定的 IP 地址 |
| STATIC          | 使用 endpoint 中指定的静态 IP 地址（见下文）作为与服务关联的支持实例 |
| DNS             | 尝试通过异步查询环境 DNS 来解析 IP 地址。如果未指定 endpoint，则代理将解析主机字段中指定的 DNS 地址（如果未使用通配符）。如果指定了 endpoint，则将解析 endpoint 中指定的 DNS 地址以确定目标 IP 地址。 DNS 解析不能与 Unix 域套接字 endpoint 一起使用 |
| DNS_ROUND_ROBIN | 尝试通过异步查询环境 DNS 来解析 IP 地址。与 DNS 不同，DNS_ROUND_ROBIN 仅在需要启动新连接时使用返回的第一个 IP 地址，而不依赖于 DNS 解析的完整结果，即使 DNS 记录频繁更改，与主机建立的连接也将被保留，从而消除了耗尽连接池和连接循环。这最适合必须通过 DNS 访问的大型 Web 规模服务。如果不使用通配符，代理将解析主机字段中指定的 DNS 地址。 DNS 解析不能与 Unix 域套接字 endpoint 一起使用 |

