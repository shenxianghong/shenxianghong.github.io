---
title: "「 Istio 」流量管理 — Gateway"
excerpt: "Istio 流量管理场景下的 Gateway 资源对象使用范例与 API 结构概览"
cover: https://picsum.photos/0?sig=20220710
thumbnail: /gallery/istio/thumbnail.svg
date: 2022-07-10
toc: true
categories:
- Service Mesh
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="/gallery/istio/logo.svg"></div>

------

> based on **1.15.0**

Gateway 描述了在网格边缘运行的负载均衡器，用于接收传入或传出的 HTTP/TCP 连接。该规范描述了一组应该公开的端口、要使用的协议类型、负载均衡器的 SNI 配置等。

例如，以下 Gateway 配置设置代理以充当负载均衡器，将端口 80 和 9080 (http)、443 (https)、9443 (https) 和端口 2379 (TCP) 用于入口。Gateway 会应用在带有标签 app: my-gateway-controller 的 Pod 上。<br>*虽然 Istio 配置代理侦听这些端口，但用户有责任确保允许到这些端口的外部流量进入网格。*

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: my-gateway
  namespace: some-config-namespace
spec:
  selector:
    app: my-gateway-controller
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - uk.bookinfo.com
    - eu.bookinfo.com
    tls:
      httpsRedirect: true # sends 301 redirect for http requests
  - port:
      number: 443
      name: https-443
      protocol: HTTPS
    hosts:
    - uk.bookinfo.com
    - eu.bookinfo.com
    tls:
      mode: SIMPLE # enables HTTPS on this port
      serverCertificate: /etc/certs/servercert.pem
      privateKey: /etc/certs/privatekey.pem
  - port:
      number: 9443
      name: https-9443
      protocol: HTTPS
    hosts:
    - "bookinfo-namespace/*.bookinfo.com"
    tls:
      mode: SIMPLE # enables HTTPS on this port
      credentialName: bookinfo-secret # fetches certs from Kubernetes secret
  - port:
      number: 9080
      name: http-wildcard
      protocol: HTTP
    hosts:
    - "*"
  - port:
      number: 2379 # to expose internal service via external port 2379
      name: mongo
      protocol: MONGO
    hosts:
    - "*"
```

Gateway 描述了负载均衡器的 L4 - L6 属性。然后，可以将 VirtualService 绑定到 Gateway 控制到达特定 host 或 Gateway 端口的流量的转发。

例如，下面的 VirtualService 把流量路径  https://uk.bookinfo.com/reviews、https://eu.bookinfo.com/reviews、http://uk.bookinfo.com:9080/reviews、http://eu.bookinfo.com:9080/reviews 分为了两个版本（prod 和 qa）。另外，包含 user: dev-123 cookie 的请求将发送到 7777 端口的 qa 版本。The same rule is also applicable inside the mesh for requests to the “reviews.prod.svc.cluster.local” service. This rule is applicable across ports 443, 9080. Note that http://uk.bookinfo.com gets redirected to https://uk.bookinfo.com (i.e. 80 redirects to 443).

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo-rule
  namespace: bookinfo-namespace
spec:
  hosts:
  - reviews.prod.svc.cluster.local
  - uk.bookinfo.com
  - eu.bookinfo.com
  gateways:
  - some-config-namespace/my-gateway
  - mesh # applies to all the sidecars in the mesh
  http:
  - match:
    - headers:
        cookie:
          exact: "user=dev-123"
    route:
    - destination:
        port:
          number: 7777
        host: reviews.qa.svc.cluster.local
  - match:
    - uri:
        prefix: /reviews/
    route:
    - destination:
        port:
          number: 9080 # can be omitted if it's the only port for reviews
        host: reviews.prod.svc.cluster.local
      weight: 80
    - destination:
        host: reviews.qa.svc.cluster.local
      weight: 20
```

以下 VirtualService 将到达（外部）端口 27017 的流量转发到端口 5555 上的内部 Mongo 服务器。此规则在网格内部不适用，因为 gateways 中省略了保留名称网格（mesh）。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo-mongo
  namespace: bookinfo-namespace
spec:
  hosts:
  - mongosvr.prod.svc.cluster.local # name of internal Mongo service
  gateways:
  - some-config-namespace/my-gateway # can omit the namespace if gateway is in same namespace as virtual service.
  tcp:
  - match:
    - port: 27017
    route:
    - destination:
        host: mongo.prod.svc.cluster.local
        port:
          number: 5555
```

可以使用 hosts 字段中的 namespace/host 语法来限制可以绑定到 Gateway 服务器的 VirtualService。例如，下面的 Gateway 允许 ns1 命名空间中的任何 VirtualService 绑定到它，同时限制只有 ns2 命名空间中具有 foo.bar.com host 的 VirtualService 绑定。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: my-gateway
  namespace: some-config-namespace
spec:
  selector:
    app: my-gateway-controller
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "ns1/*"
    - "ns2/foo.bar.com"
```

# Gateway

Gateway 描述了在网格边缘运行的负载均衡器，用于接收传入或传出的 HTTP/TCP 连接。

| Field              | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| [servers](#Server) | Server 集合                                                  |
| selector           | 一个或多个标签，用以匹配一组特定 pod/VM 上应用此 Gateway。默认情况下，工作负载会根据标签选择器在所有命名空间中进行搜索。这意味着 Namespace foo 中的 Gateway 资源可以根据标签选择 Namespace bar 中的 Pod。这种行为可以通过 Istiod 中的 PILOT_SCOPE_GATEWAY_TO_NAMESPACE 环境变量来控制。如果此变量设置为 true，则标签搜索的范围仅限于 Gateway 资源所在的 Namespace。换言之，Gateway 资源必须与 Gateway 工作负载实例位于相同的 Namespace 中。如果选择器为 nil，则 Gateway 将应用于所有工作负载 |

# <a name="Server">Server</a>

Server 描述特定负载均衡端口上的代理属性。

| Field                     | Description                                                  |
| ------------------------- | ------------------------------------------------------------ |
| [port](#Port)             | 代理监听的请求链接端口信息                                   |
| bind                      | 绑定到的 IP 或 Unix 域套接字。格式为 x.x.x.x、unix:///path/to/uds 或 unix://@foobar（Linux 抽象命名空间）。使用 Unix 域套接字时，端口号应为 0。用于将此 server 的可达性限制为仅限 Gateway 内部。这通常在 Gateway 需要与另一个网格 server 通信时使用，例如发布指标。在这种情况下，使用指定绑定创建的 server 将不可用于外部 Gateway 客户端 |
| hosts                     | Gateway 暴露的 host 信息。虽然通常适用于 HTTP 服务，但它也表述带有 SNI 的 TLS 的 TCP 服务。host 为带有 \<namespace\>/ 可选前缀的 dnsName（FQDN 格式，也可以类似如 prod/*.example.com）。将 dnsName 设置为 * 表示从指定的命名空间（例如 prod/*）中选择所有 VirtualService host。<br />namespace 可以设置为 * 或 .，分别代表任何或当前命名空间。例如，\*/foo.example.com 从任何可用的命名空间中选择 server，而 ./foo.example.com 仅从 sidecar 的命名空间中选择服务。如果未指定 namespace/ 前缀部分，则默认为 \*/。所选命名空间中的任何关联 DestinationRule 也将被使用。<br />VirtualService 必须绑定到 Gateway，并且必须有一个或多个与 server 中匹配的 host。匹配可以是完全匹配或后缀匹配。<br />*例如，如果 server 的 host 指定 .example.com，则具有 host dev.example.com 或 prod.example.com 的 VirtualService 将匹配。但是，host example.com 或 newexample.com 的 VirtualService 将不匹配* |
| [tls](#ServerTLSSettings) | 一组控制 server 行为的 TLS 相关选项。使用这些选项来控制是否应将所有 HTTP 请求重定向到 HTTPS，以及要使用的 TLS 模式 |
| name                      | server 的可选名称，设置后在所有 server 中必须是唯一的。可用于例如为使用此名称生成的统计信息添加前缀等 |

# <a name="Port">Port</a>

Port 描述了 server 的特定端口的属性。

| Field      | Description                                                  |
| ---------- | ------------------------------------------------------------ |
| number     | 端口号                                                       |
| protocol   | 端口协议。可选值 HTTP、HTTPS、GRPC、HTTP2、MONGO、TCP、TLS。TLS 意味着连接将根据 SNI 标头路由到目的地，而不会终止 TLS 连接 |
| name       | 分配给端口的标签                                             |
| targetPort | 接收流量的 endpoint 上的端口号。仅在与 ServiceEntries 一起使用时适用 |

# <a name="ServerTLSSettings">ServerTLSSettings</a>

| Field                                                | Description                                                  |
| ---------------------------------------------------- | ------------------------------------------------------------ |
| httpsRedirect                                        | 如果设置为 true，负载均衡器将为所有 HTTP 连接发送 301 重定向，要求客户端使用 HTTPS |
| [mode](#ServerTLSSettings.TLSmode)                   | 可选：表明是否应使用 TLS 保护与此端口的连接。该字段的值决定了 TLS 的实施方式 |
| serverCertificate                                    | 如果 mode 是 SIMPLE 或 MUTUAL，则为必需字段。保存使用的服务器端 TLS 证书的文件的路径 |
| privateKey                                           | 如果 mode 是 SIMPLE 或 MUTUAL，则为必需字段。保存服务器私钥的文件的路径 |
| caCertificates                                       | 如果 mode 是 MUTUAL，则为必需字段。包含证书颁发机构证书的文件的路径，用于验证提供的客户端证书 |
| credentialName                                       | 对于在 Kubernetes 上运行的网关，包含 TLS 证书（包括 CA 证书）的密钥的名称。仅适用于 Kubernetes。密钥（通用类型）应包含以下键和值：键：\<privateKey\> 和证书：\<serverCert\>。对于双向 TLS，cacert: \<CACertificate\> 可以在同一个密钥或名为 \<secret\>-cacert 的单独密钥中提供。还支持用于服务器证书的 tls 类型的密钥以及用于 CA 证书的 ca.crt 密钥。只能指定服务器证书和 CA 证书或 credentialName 之一 |
| subjectAltNames                                      | 用于验证客户端提供的证书中的主体身份的备用名称列表           |
| verifyCertificateSpki                                | 授权客户端证书的 SKPI 的 base64 编码 SHA-256 哈希的可选列表。注意：当同时指定 verify_certificate_hash 和 verify_certificate_spki 时，匹配任一值的哈希将导致证书被接受 |
| verifyCertificateHash                                | 授权客户端证书的十六进制编码 SHA-256 哈希的可选列表。简单格式和冒号分隔格式都可以接受。注意：当同时指定 verify_certificate_hash 和 verify_certificate_spki 时，匹配任一值的哈希将导致证书被接受 |
| [minProtocolVersion](#ServerTLSSettings.TLSProtocol) | 最低 TLS 协议版本                                            |
| [maxProtocolVersion](#ServerTLSSettings.TLSProtocol) | 最高 TLS 协议版本                                            |
| cipherSuites                                         | 如果指定，只支持指定的密码列表。否则默认为 Envoy 支持的默认密码列表 |

# <a name="ServerTLSSettings.TLSmode">ServerTLSSettings.TLSmode</a>

| Name             | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| PASSTHROUGH      | 客户端提供的 SNI 字符串将用作 VirtualService TLS 路由中的匹配标准，以确定服务注册表中的目标服务 |
| SIMPLE           | 使用标准 TLS 语义的安全连接                                  |
| MUTUAL           | 通过提供服务器证书进行身份验证，使用双向 TLS 保护与下游的连接 |
| AUTO_PASSTHROUGH | 与直通模式类似，除了具有此 TLS 模式的服务器不需要关联的 VirtualService 从 SNI 值映射到注册表中的服务。诸如服务/子集/端口之类的目标详细信息在 SNI 值中进行编码。代理将转发到 SNI 值指定的上游（Envoy）集群（一组端点）。此服务器通常用于在不同的 L3 网络中提供服务之间的连接，否则它们各自的端点之间没有直接连接。使用此模式假定源和目标都使用 Istio mTLS 来保护流量 |
| ISTIO_MUTUAL     | 通过提供服务器证书进行身份验证，使用双向 TLS 保护来自下游的连接。与 Mutual 模式相比，该模式使用 Istio 自动生成的代表网关工作负载身份的证书进行 mTLS 身份验证。使用此模式时，TLSOptions 中的所有其他字段都应为空 |

# <a name="ServerTLSSettings.TLSProtocol">ServerTLSSettings.TLSProtocol</a>

| Name     | Description           |
| -------- | --------------------- |
| TLS_AUTO | 自动选择最佳 TLS 版本 |
| TLSV1_0  | TLS 1.0 版本          |
| TLSV1_1  | TLS 1.1 版本          |
| TLSV1_2  | TLS 1.2 版本          |
| TLSV1_3  | TLS 1.3 版本          |
