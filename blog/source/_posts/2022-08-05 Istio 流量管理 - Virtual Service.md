---
title: "「 Istio 」流量管理 — VirtualService"
excerpt: "Istio 中流量管理组件 VirtualService 对象介绍"
cover: https://picsum.photos/0?sig=20220805
thumbnail: https://github.com/cncf/artwork/raw/master/projects/istio/stacked/color/istio-stacked-color.svg
date: 2022-08-05
toc: true
categories:
- Service Mesh
tag:
- Istio
---

<div align=center><img width="120" style="border: 0px" src="https://github.com/cncf/artwork/raw/master/projects/istio/horizontal/color/istio-horizontal-color.svg"></div>

------

> based on **1.15.0**

VirtualService 定义了一组流量路由规则，以在 host 被寻址时应用。每个路由规则都为特定协议的流量定义了匹配标准。如果流量匹配，则将其发送到注册表中定义的命名目标服务（或其子集/版本）

流量来源也可以在路由规则中匹配。这允许为特定的客户端上下文定制路由。

**Service**

绑定到 service registry 中唯一名称的应用程序行为单元。Service 由多个 endpoints 组成，这些 endpoints 也就是运行在 Pod、容器、VM 等上的工作负载实例。

**Service versions（subsets）**

在持续部署场景中，一个服务可以有不同的实例子集的不同变体，这些变体不一定是不同的 API 版本，它们可能是对同一服务的迭代更改，部署在不同的环境（prod, staging, dev 等）中。常见场景包括 A/B 测试、金丝雀发布等。不同版本的选择可以根据各种标准（headers, url 等）的权重来决定。每个服务都有一个由其所有实例组成的默认版本。

**Source**

调用服务的下游客户端。

**Host**

客户端尝试连接到服务时使用的地址。

**Access model**

应用程序仅处理目标服务（Host），而不知道各个服务版本（subsets）。版本的实际选择由代理或者 sidecar 决定，这样应用程序代码能够将自身与依赖服务的解耦。

例如，以下示例默认将所有 HTTP 流量路由到标签为 review：v1 的 review 服务的 Pod。此外，路径以 /wpcatalog/ 或 /consumercatalog/ 开头的 HTTP 请求将被重写为 /newcatalog 并发送到标签为 version: v2 的 Pod。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews-route
spec:
  hosts:
  - reviews.prod.svc.cluster.local
  http:
  - name: "reviews-v2-routes"
    match:
    - uri:
        prefix: "/wpcatalog"
    - uri:
        prefix: "/consumercatalog"
    rewrite:
      uri: "/newcatalog"
    route:
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v2
  - name: "reviews-v1-route"
    route:
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v1
```

路由目的地的子集或者版本通过对必须在相应 DestinationRule 中声明的命名服务子集的引用来标识。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: reviews-destination
spec:
  host: reviews.prod.svc.cluster.local
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
```

# VirtualService

| Field              | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| hosts              | 向其发送流量的目标主机（只有访问客户端的 Host 字段为 hosts 配置的地址才能路由到后端服务）。可以是带有通配符前缀的 DNS 名称或 IP 地址。根据平台的不同，也可以使用短名称代替 FQDN（即名称中没有点）。在这种情况下，主机的 FQDN 将基于底层平台派生<br />单个 VirtualService 可用于描述相应 host 的所有流量属性，包括多个 HTTP 和 TCP 端口的流量属性。或者，可以使用多个 VirtualService 定义 host 的流量属性，但有一些注意事项，参阅[操作指南](https://istio.io/latest/docs/ops/best-practices/traffic-management/#split-virtual-services)<br />Kubernetes 用户注意事项：当使用短名称时（例如 reviews 而不是 reviews.default.svc.cluster.local），Istio 将根据规则的命名空间而不是服务来解析短名称。名为 reviews 的 host 在 default 命名空间中的规则将被解析为 reviews.default.svc.cluster.local，而与 reviews 服务关联的实际命名空间无关。为避免潜在的错误配置，建议始终使用完全限定域名而不是短名称<br />hosts 字段适用于 HTTP 和 TCP 服务。网格内的服务，即服务注册表中的服务，必须始终使用它们的字母数字名称来引用。 IP 地址仅允许用于通过 Gateway 定义的服务<br />注意：对于 delegate VirtualService 而言，必须为空。 |
| gateways           | 应用这些路由规则的 Gateway 和 sidecar 的名称。 其他命名空间中的 Gateway 的引用方式为\<gateway namespace\>/\<gateway name\>；指定没有显式的指定命名空间则与 VirtualService 的命名空间相同。单个 VirtualService 用于网格内的 sidecar 以及一个或多个 Gateway。可以使用协议特定路由的匹配条件中的源字段覆盖该字段强加的选择条件。保留字 mesh 用于表述网格中的所有 sidecar。当省略该字段时，将使用默认 Gateway 即 mesh，这会将规则应用于网格中的所有 sidecar。如果提供了 Gateway 名称列表，则规则将仅适用该相关的 Gateway。要将规则应用于 Gateway 和 sidecar，请将 mesh 指定为 gateways 之一 |
| [http](#HTTPRoute) | HTTP 流量的路由规则的有序列表。 使用匹配传入请求的第一个规则 |
| [tls](#TLSRoute)   | An ordered list of route rule for non-terminated TLS & HTTPS traffic. Routing is typically performed using the SNI value presented by the ClientHello message. TLS routes will be applied to platform service ports named ‘https-*’, ‘tls-*’, unterminated gateway ports using HTTPS/TLS protocols (i.e. with “passthrough” TLS mode) and service entry ports using HTTPS/TLS protocols. The first rule matching an incoming request is used. NOTE: Traffic ‘https-*’ or ‘tls-*’ ports without associated virtual service will be treated as opaque TCP traffic. |
| [tcp](#TCPRoute)   | 不透明 TCP 流量的路由规则的有序列表。 TCP 路由将应用于不是 HTTP 或 TLS 端口的任何端口。使用匹配传入请求的第一个规则 |
| exportTo           | 此 VirtualService 暴露到的命名空间列表。暴露 VirtualService 允许它被其他命名空间中定义的 sidecar 和 Gateway 使用。此功能为服务所有者和网格管理员提供了一种机制来控制跨命名空间边界的 VirtualService 的可见性<br />如果未指定命名空间，则默认情况下将 VirtualService 暴露到所有命名空间<br /> . 为保留标识代表暴露到 VirtualService 的同一命名空间中。类似地，* 代表暴露到所有命名空间 |

# Destination

Destination 表示在处理路由规则后将请求（连接）发送到的网络可寻址服务。 destination.host 应该明确引用服务注册表中的服务。 Istio 的服务注册表由平台服务注册表中的所有服务（例如 Kubernetes 服务、Consul 服务）以及通过 ServiceEntry 资源声明的服务组成。

以下示例中默认将所有流量路由到带有标签 version：v1（即 subset v1）的 review 服务的 Pod，并将符合条件的路由到 subset v2。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews-route
  namespace: foo
spec:
  hosts:
  - reviews # interpreted as reviews.foo.svc.cluster.local
  http:
  - match:
    - uri:
        prefix: "/wpcatalog"
    - uri:
        prefix: "/consumercatalog"
    rewrite:
      uri: "/newcatalog"
    route:
    - destination:
        host: reviews # interpreted as reviews.foo.svc.cluster.local
        subset: v2
  - route:
    - destination:
        host: reviews # interpreted as reviews.foo.svc.cluster.local
        subset: v1
```

以下 VirtualService 为 productpage.prod.svc.cluster.local 服务的设置了 5s 的超时时间。此规则中没有定义子集，Istio 将从服务注册表中获取 productpage.prod.svc.cluster.local 服务的所有实例，并填充 sidecar 的负载均衡池。另外，此规则设置在 istio-system 命名空间中，但使用的是 productpage 服务的完全限定域名 productpage.prod.svc.cluster.local。因此，规则的命名空间对解析 productpage 服务的名称没有影响。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: my-productpage-rule
  namespace: istio-system
spec:
  hosts:
  - productpage.prod.svc.cluster.local # ignores rule namespace
  http:
  - timeout: 5s
    route:
    - destination:
        host: productpage.prod.svc.cluster.local
```

为了路由到网格外服务的流量，必须首先使用 ServiceEntry 资源将外部服务添加到 Istio 的内部服务注册表中。然后可以定义 VirtualServices 来控制绑定到这些外部服务的流量。例如，以下规则为 wikipedia.org 定义了一个 Service，并为 HTTP 请求设置了 5s 的超时时间。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-svc-wikipedia
spec:
  hosts:
  - wikipedia.org
  location: MESH_EXTERNAL
  ports:
  - number: 80
    name: example-http
    protocol: HTTP
  resolution: DNS
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: my-wiki-rule
spec:
  hosts:
  - wikipedia.org
  http:
  - timeout: 5s
    route:
    - destination:
        host: wikipedia.org
```

| Field                 | Description                                                  |
| --------------------- | ------------------------------------------------------------ |
| host                  | 服务注册表中的服务名称。服务名称从平台的服务注册表（例如，Kubernetes 服务、Consul 服务等）和 ServiceEntry 声明的 host 中查找。两者都找不到的流量将被丢弃。<br />*同样的，推荐使用完全限定域名* |
| subset                | 服务中子集的名称。仅适用于网格内的服务。子集必须在相应的 DestinationRule 中定义 |
| [port](#PortSelector) | 指定寻址目标主机上的端口。如果服务只公开一个端口，则不需要显式选择端口 |

# <a name="HTTPRoute">HTTPRoute</a>

描述路由 HTTP/1.1、HTTP2 和 gRPC 流量的匹配条件和行为。有关使用示例，请参阅 VirtualService。

| Field                          | Description                                                  |
| ------------------------------ | ------------------------------------------------------------ |
| name                           | 分配给路由的名称，用作调试。该名称将与匹配的路由结果一起记录在访问日志 |
| [match](#HTTPMatchRequest)     | 应用规则需要满足的匹配条件。单个匹配块内的所有条件都具有 AND 语义，而匹配块列表具有 OR 语义。如果任何一个匹配块匹配成功，则应用该规则 |
| [route](#HTTPRouteDestination) | HTTP 规则可以重定向或转发（默认）流量。转发目标可以是服务的多个版本之一（subset）。与服务版本相关的权重决定了它接收的流量比例 |
| [redirect](#HTTPRedirect)      | HTTP 规则可以重定向或转发（默认）流量。如果在规则中指定了流量直通选项，则重定向将被忽略。重定向可用于将 HTTP 301 重定向发送到不同的 URI |
| [delegate](#Delegate)          | 委托是指定可用于定义委托 HTTPRoute 的特定 VirtualService。只有Route和Redirect都为空时才可以设置，并且delegate VirtualService的路由规则会与当前的路由规则合并。<br />*注意：仅支持一级委派，delegate 的 HTTPMatchRequest 必须是 root 的严格子集，否则会发生冲突，HTTPRoute 不会生效* |
| [rewrite](#HTTPRewrite)        | 重写 HTTP URI 和 Authority header。 Rewrite 不能与 Redirect 原语一起使用。转发前会进行重写 |
| timeout                        | HTTP 请求超时，默认禁用                                      |
| [retries](#HTTPRetry)          | HTTP 请求的重试策略                                          |
| [fault](#HTTPFaultInjection)   | 故障注入策略应用于客户端的 HTTP 流量。<br />*注意，在客户端启用故障时，将不会启用超时或重试* |
| [mirror](#Destination)         | 除了将请求转发到预期目标之外，还将 HTTP 流量镜像到另一个目标。镜像流量是在尽力而为的基础上，sidercar / gateway 在从原始目的地返回响应之前不会等待镜像集群响应。会为镜像目的地生成统计信息。 |
| [mirrorPercentage](#Percent)   | 要镜像的流量百分比。默认为所有的流量（100%）都会被镜像。最大值为 100%。 |
| [corsPolicy](#CorsPolicy)      | 跨域资源共享策略 (CORS)。有关跨源资源共享的更多详细信息，参阅 [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) |
| [headers](#Headers)            | Header 操作规则                                              |

# <a name="Delegate">Delegate</a>

在 Istio 1.5 中，VirtualService 资源之间是无法进行转发的，在 Istio 1.6 版本中规划了 VirtualService Chain 机制，也就是说，我们可以通过 delegate 配置项将一个 VirtualService 代理到另外一个 VirtualService 中进行规则匹配。

例如，路由规则通过名为 productpage 的委托 VirtualService 将流量转发到 /productpage，通过名为 reviews 的委托 VirtualService 将流量转发到 /reviews。

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: bookinfo
spec:
  hosts:
  - "bookinfo.com"
  gateways:
  - mygateway
  http:
  - match:
    - uri:
        prefix: "/productpage"
    delegate:
       name: productpage
       namespace: nsA
  - match:
    - uri:
        prefix: "/reviews"
    delegate:
        name: reviews
        namespace: nsB
```

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: productpage
  namespace: nsA
spec:
  http:
  - match:
     - uri:
        prefix: "/productpage/v1/"
    route:
    - destination:
        host: productpage-v1.nsA.svc.cluster.local
  - route:
    - destination:
        host: productpage.nsA.svc.cluster.local
```

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: reviews
  namespace: nsB
spec:
  http:
  - route:
    - destination:
        host: reviews.nsB.svc.cluster.local
```

| Field     | Description                                                  |
| --------- | ------------------------------------------------------------ |
| name      | 委托 VirtualService 的名称                                   |
| namespace | 委托 VirtualService 所在的命名空间。默认情况下，它与根的相同 |

# <a name="Headers">Headers</a>

当 Envoy 将请求转发到目标服务或从目标服务转发响应时，可以操作 header 信息。可以为特定路由目的地或所有目的地指定 header 操作规则。以下 VirtualService 将值为 test: true 的请求头添加到路由到任何 reviews 服务目标的请求中。同时，删除了仅来自 reviews 服务的 v1 版本的响应头中的 foo。 

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews-route
spec:
  hosts:
  - reviews.prod.svc.cluster.local
  http:
  - headers:
      request:
        set:
          test: "true"
    route:
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v2
      weight: 25
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v1
      headers:
        response:
          remove:
          - foo
      weight: 75
```

| Field                         | Description                                  |
| ----------------------------- | -------------------------------------------- |
| [request](#HeaderOperations)  | 在将请求转发到目标服务之前应用的标头操作规则 |
| [response](#HeaderOperations) | 在向调用者返回响应之前应用的标头操作规则     |

# <a name="TLSRoute">TLSRoute</a>

描述路由未终止的 TLS 流量 (TLS/HTTPS) 的匹配条件和操作。

以下路由规则根据 SNI 值将到达名为 mygateway 的 Gateway 的端口 443 的未终止 TLS 流量转发到网格中的内部服务。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo-sni
spec:
  hosts:
  - "*.bookinfo.com"
  gateways:
  - mygateway
  tls:
  - match:
    - port: 443
      sniHosts:
      - login.bookinfo.com
    route:
    - destination:
        host: login.prod.svc.cluster.local
  - match:
    - port: 443
      sniHosts:
      - reviews.bookinfo.com
    route:
    - destination:
        host: reviews.prod.svc.cluster.local
```

| Field                          | Description                                                  |
| ------------------------------ | ------------------------------------------------------------ |
| [match](#TLSMatchAttributes[]) | 应用规则需要满足的匹配条件。单个匹配块内的所有条件都具有 AND 语义，而匹配块列表具有 OR 语义。如果任何一个匹配块匹配成功，则应用该规则 |
| [route](#RouteDestination)     | 连接应转发到的目的地                                         |

# <a name="TCPRoute">TCPRoute</a>

描述路由 TCP 流量的匹配条件和操作。

以下路由规则将到达端口 27017 的 mongo.prod.svc.cluster.local 的流量转发到端口 5555 上的另一个 Mongo 服务器。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo-mongo
spec:
  hosts:
  - mongo.prod.svc.cluster.local
  tcp:
  - match:
    - port: 27017
    route:
    - destination:
        host: mongo.backup.svc.cluster.local
        port:
          number: 5555
```

| Field                         | Description                                                  |
| ----------------------------- | ------------------------------------------------------------ |
| [match](#L4MatchAttributes[]) | 应用规则需要满足的匹配条件。单个匹配块内的所有条件都具有 AND 语义，而匹配块列表具有 OR 语义。如果任何一个匹配块匹配成功，则应用该规则 |
| [route](#RouteDestination[])  | 连接应转发到的目的地                                         |

# <a name="HTTPMatchRequest">HTTPMatchRequest</a>

HttpMatchRequest 指定要满足的一组标准，以便将规则应用于 HTTP 请求。

例如，以下内容将规则限制为仅匹配 URL 路径以 /ratings/v2/ 开头的请求，并且请求包含具有值 jason 的自定义最终用户标头。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings-route
spec:
  hosts:
  - ratings.prod.svc.cluster.local
  http:
  - match:
    - headers:
        end-user:
          exact: jason
      uri:
        prefix: "/ratings/v2/"
      ignoreUriCase: true
    route:
    - destination:
        host: ratings.prod.svc.cluster.local
```

HTTPMatchRequest 不能为空。注意：指定委托 VirtualService 时，不能设置正则表达式字符串匹配。

| Field                          | Description                                                  |
| ------------------------------ | ------------------------------------------------------------ |
| name                           | 分配给匹配项的名称。匹配的名称将与父路由的名称一同记录在与该路由匹配的请求的访问日志中 |
| [uri](#StringMatch)            | 区分大小写，格式包括 exact（精准匹配）、prefix（前缀匹配）和 regex（正则）<br />*注意：可以通过 ignore_uri_case 标志启用不区分大小写的匹配* |
| [scheme](#StringMatch)         | 区分大小写，格式包括 exact（精准匹配）、prefix（前缀匹配）和 regex（正则） |
| [method](#StringMatch)         | 区分大小写，格式包括 exact（精准匹配）、prefix（前缀匹配）和 regex（正则） |
| [authority](#StringMatch)      | 区分大小写，格式包括 exact（精准匹配）、prefix（前缀匹配）和 regex（正则） |
| [headers](#StringMatch)        | header 的 key 必须为小写并使用连字符作为分隔符，例如 x-request-id。区分大小写，格式包括 exact（精准匹配）、prefix（前缀匹配）和 regex（正则）。如果该值为空并且仅指定了 header 的名称，则检查标头的存在<br />*注意：键 uri、scheme、method 和 authority 将被忽略* |
| port                           | 指定正在寻址的 host 上的端口。许多服务只公开单个端口或使用它们支持的协议标记端口，在这些情况下，不需要显式选择端口 |
| sourceLabels                   | 一个或多个标签，用于限制规则对具有给定标签的源（客户端）工作负载的适用性。如果 VirtualService 在顶级 gateways 字段中指定了网关列表，仅当列表中包含保留的 gateway — mesh 时，该字段才生效 |
| gateways                       | 待应用规则的网关的名称。 VirtualService 的顶级 gateways 字段中的网关名称（如果有）被覆盖。Gateway 匹配独立于 sourceLabels |
| [queryParams](#StringMatch)    | 用于匹配的查询参数，例如 exact: "true"，extact: "" 和 regex: “\\d+$”<br />*注意：目前不支持前缀匹配* |
| ignoreUriCase                  | 用于指定 URI 匹配是否不区分大小写的标志。<br />*注意：只有在完全和前缀 URI 匹配的情况下才会忽略大小写* |
| [withoutHeaders](#StringMatch) | withoutHeaders 与 header 的语法相同，但含义相反。如果 header 与 withoutHeaders 中的匹配规则匹配，则流量变为不匹配 |
| sourceNamespace                | 源命名空间限制规则对该命名空间中工作负载的适用性。如果 VirtualService 在顶级 gateways 字段中指定了网关列表，仅当列表中包含保留的 gateway — mesh 时，该字段才生效 |

# <a name="HTTPRouteDestination">HTTPRouteDestination</a>

每个路由规则都与一个或多个服务版本相关联。与版本相关的权重决定了它接收的流量比例。

例如，以下规则会将 reviews 服务的 25% 流量路由到 v2 版本的实例中，而剩余流量（即 75%）将路由到 v1 版本。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews-route
spec:
  hosts:
  - reviews.prod.svc.cluster.local
  http:
  - route:
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v2
      weight: 25
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v1
      weight: 75
```

流量也可以分成两个完全不同的服务，而无需定义新的子集（也就是内部分流）。

例如，以下规则将 25% 的流量从 reviews.com 转发到 dev.reviews.com。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews-route-two-domains
spec:
  hosts:
  - reviews.com
  http:
  - route:
    - destination:
        host: dev.reviews.com
      weight: 25
    - destination:
        host: reviews.com
      weight: 75
```

| Field                       | Description                                                  |
| --------------------------- | ------------------------------------------------------------ |
| [destination](#Destination) | 请求/连接应转发到的服务实例                                  |
| weight                      | 要转发到目的地的流量的相对比例（即权重/所有权重的总和）。如果规则中只有一个目的地，它将接收所有流量。如果权重为 0，则目的地将不会收到任何流量 |
| [headers](#Headers)         | header 操作规则                                              |

# <a name="RouteDestination">RouteDestination</a>

| Field                       | Description                                                  |
| --------------------------- | ------------------------------------------------------------ |
| [destination](#Destination) | 请求/连接应转发到的服务实例                                  |
| weight                      | 要转发到目的地的流量的相对比例（即权重/所有权重的总和）。如果规则中只有一个目的地，它将接收所有流量。如果权重为 0，则目的地将不会收到任何流量 |

# <a name="L4MatchAttributes">L4MatchAttributes</a>

| Field              | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| destinationSubnets | 带有可选子网的目标 IPv4 或 IPv6 IP 地址。例如，a.b.c.d/xx  或只是 a.b.c.d |
| port               | 指定正在寻址的 host 上的端口。许多服务只公开单个端口或使用它们支持的协议标记端口，在这些情况下，不需要显式选择端口 |
| sourceLabels       | 一个或多个标签，用于限制规则对具有给定标签的源（客户端）工作负载的适用性。如果 VirtualService 在顶级 gateways 字段中指定了网关列表，仅当列表中包含保留的 gateway — mesh 时，该字段才生效 |
| gateways           | 待应用规则的网关的名称。 VirtualService 的顶级 gateways 字段中的网关名称（如果有）被覆盖。Gateway 匹配独立于 sourceLabels |
| sourceNamespace    | 源命名空间限制规则对该命名空间中工作负载的适用性。如果 VirtualService 在顶级 gateways 字段中指定了网关列表，仅当列表中包含保留的 gateway — mesh 时，该字段才生效 |

# <a name="TLSMatchAttributes">TLSMatchAttributes</a>

| Field              | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| sniHosts           | 要匹配的 SNI（server name indicator）。通配符前缀可用于 SNI 值，例如，*.com 将匹配 foo.example.com 以及 example.com。 SNI 值必须是相应虚拟服务主机的子集（即属于域内） |
| destinationSubnets | 带有可选子网的目标 IPv4 或 IPv6 IP 地址。例如，a.b.c.d/xx  或只是 a.b.c.d |
| port               | 指定正在寻址的 host 上的端口。许多服务只公开单个端口或使用它们支持的协议标记端口，在这些情况下，不需要显式选择端口 |
| sourceLabels       | 一个或多个标签，用于限制规则对具有给定标签的源（客户端）工作负载的适用性。如果 VirtualService 在顶级 gateways 字段中指定了网关列表，仅当列表中包含保留的 gateway — mesh 时，该字段才生效 |
| gateways           | 待应用规则的网关的名称。 VirtualService 的顶级 gateways 字段中的网关名称（如果有）被覆盖。Gateway 匹配独立于 sourceLabels |
| sourceNamespace    | 源命名空间限制规则对该命名空间中工作负载的适用性。如果 VirtualService 在顶级 gateways 字段中指定了网关列表，仅当列表中包含保留的 gateway — mesh 时，该字段才生效 |

# <a name="HTTPRedirect">HTTPRedirect</a>

HTTPRedirect 可用于向调用者发送 301 重定向响应。例如，以下规则将 review 服务上的 /v1/getProductRatings API 请求重定向到 bookratings 服务提供的 /v1/bookRatings。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings-route
spec:
  hosts:
  - ratings.prod.svc.cluster.local
  http:
  - match:
    - uri:
        exact: /v1/getProductRatings
    redirect:
      uri: /v1/bookRatings
      authority: newratings.default.svc.cluster.local
  ...
```

| Field                                             | Description                                                  |
| ------------------------------------------------- | ------------------------------------------------------------ |
| uri                                               | 在重定向时，使用此值覆盖 URL 的 path 部分。<br />*无论请求 URI 是否匹配为确切路径或前缀，都将替换整个路径* |
| authority                                         | 在重定向时，使用此值覆盖 URL 的 Authority/Host 部分          |
| port                                              | 在重定向时，使用此值覆盖 URL 的 Port 部分                    |
| [derivePort](#HTTPRedirect.RedirectPortSelection) | 在重定向时，动态设置端口                                     |
| scheme                                            | 在重定向时，使用此值覆盖 URL 的 Scheme 部分。例如，http 或 https。如果未设置，将使用原 Scheme。如果 derivePort 设置为 FROM_PROTOCOL_DEFAULT，这也会影响该端口 |
| redirectCode                                      | 在重定向时，指定要在重定向响应中使用的 HTTP 状态代码。默认响应代码为 MOVED_PERMANENTLY (301) |

# <a name="HTTPRewrite">HTTPRewrite</a>

HTTPRewrite 可用于在将请求转发到目标之前重写 HTTP 请求的特定部分。HTTPRewrite 只能与 HTTPRouteDestination 一起使用。

以下示例演示如何在实际调用之前将 /ratings 前缀重写为 rating 服务。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings-route
spec:
  hosts:
  - ratings.prod.svc.cluster.local
  http:
  - match:
    - uri:
        prefix: /ratings
    rewrite:
      uri: /v1/bookRatings
    route:
    - destination:
        host: ratings.prod.svc.cluster.local
        subset: v1
```

| Field     | Description                                                  |
| --------- | ------------------------------------------------------------ |
| uri       | 用这个值重写 URI 的 Path 或 Prefix 部分。如果原始 URI 是根据前缀匹配的，则此字段中提供的值将替换相应匹配的前缀 |
| authority | 用这个值重写  Authority/Host header                          |

# <a name="StringMatch">StringMatch</a>

匹配 HTTP header 中的特定字符串的策略。匹配区分大小写。

| Field  | Description                                                  |
| ------ | ------------------------------------------------------------ |
| exact  | 精准匹配                                                     |
| prefix | 前缀匹配                                                     |
| regex  | 正则匹配 （参阅：https://github.com/google/re2/wiki/Syntax） |

# <a name="HTTPRetry">HTTPRetry</a>

HTTP 请求失败时的重试策略。

例如，以下规则将调用 rating 服务 v1 版本时的最大重试次数设置为 3，每次重试超时为 2 秒。如果出现 gateway-error，connect-failure 和 refused-stream 的错误将重试。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings-route
spec:
  hosts:
  - ratings.prod.svc.cluster.local
  http:
  - route:
    - destination:
        host: ratings.prod.svc.cluster.local
        subset: v1
    retries:
      attempts: 3
      perTryTimeout: 2s
      retryOn: gateway-error,connect-failure,refused-stream
```

| Field                 | Description                                                  |
| --------------------- | ------------------------------------------------------------ |
| attempts              | 请求允许的重试次数。重试间隔将自动确定（25ms+）。当配置了 HTTP 路由的请求超时或 perTryTimeout 时，实际尝试的重试次数还取决于指定的请求超时和 perTryTimeout 值 |
| perTryTimeout         | 给定请求的每次尝试超时，包括初始调用和任何重试。格式：1h/1m/1s/1ms。必须 >=1 ms。默认值与 HTTP 路由的请求超时相同，即没有超时 |
| retryOn               | 指定重试发生的条件。可以使用 , 分隔列表指定一个或多个策略。如果 retryOn 指定了一个有效的 HTTP 状态，它将被添加到 retriable_status_codes 重试策略中。<br />*有关更多详细信息，参阅[重试策略](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#x-envoy-retry-on)和 [gRPC 重试策略](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#x-envoy-retry-grpc-on)* |
| retryRemoteLocalities | 是否应重试到其他位置。<br />*有关更多详细信息，参阅[重试插件配置](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_connection_management#retry-plugin-configuration)* |

# <a name="CorsPolicy">CorsPolicy</a>

描述给定服务的跨域资源共享（CORS）策略。有关跨源资源共享的更多详细信息，参阅 [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/Access_control_CORS)。

例如，以下规则使用 HTTP POST/GET 将跨源请求限制为来自 example.com 域的请求，并将 Access-Control-Allow-Credentials header 设置为 false。此外，它只暴露 X-Foo-bar header 并设置 1 天的有效期。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings-route
spec:
  hosts:
  - ratings.prod.svc.cluster.local
  http:
  - route:
    - destination:
        host: ratings.prod.svc.cluster.local
        subset: v1
    corsPolicy:
      allowOrigins:
      - exact: https://example.com
      allowMethods:
      - POST
      - GET
      allowCredentials: false
      allowHeaders:
      - X-Foo-Bar
      maxAge: "24h"
```

| Field                        | Description                                                  |
| ---------------------------- | ------------------------------------------------------------ |
| [allowOrigins](#StringMatch) | 匹配允许的来源的字符串模式。如果任何字符串匹配器匹配，则允许来源。如果找到匹配项，则传出的 Access-Control-Allow-Origin 将设置为客户端提供的源 |
| allowMethods                 | 允许访问资源的 HTTP 方法列表。内容将被序列化到 Access-Control-Allow-Methods header 中 |
| allowHeaders                 | 请求资源时可以使用的 HTTP header 列表。序列化为 Access-Control-Allow-Headers header |
| exposeHeaders                | 允许浏览器访问的 HTTP header 列表。序列化为 Access-Control-Expose-Headers header |
| maxAge                       | 指定预检请求的结果可以缓存多长时间。转换为 Access-Control-Max-Age header |
| allowCredentials             | 是否允许调用者使用凭据发送实际请求（而不是预检）。转换为 Access-Control-Allow-Credentials header |

# <a name="HTTPFaultInjection">HTTPFaultInjection</a>

HTTPFaultInjection 可用于在将 HTTP 请求转发到路由中注入故障。故障规范是 VirtualService 规则的一部分。错误包括从下游服务中止 HTTP 请求、延迟请求代理等。故障规则中至少有 delay 或者 abort。<br />*注意：delay 和 abort 故障相互独立，即使两者同时指定。*

| Field                              | Description                                                  |
| ---------------------------------- | ------------------------------------------------------------ |
| [delay](#HTTPFaultInjection.Delay) | 在转发之前延迟请求，模拟网络问题、上游服务过载等各种故障     |
| [abort](#HTTPFaultInjection.Abort) | 中止 HTTP 请求尝试并将错误代码返回给下游服务，模拟上游服务故障 |

# <a name="PortSelector">PortSelector</a>

PortSelector 指定用于匹配或选择最终路由的端口号。

| Field  | Description  |
| ------ | ------------ |
| number | 有效的端口号 |

# <a name="Percent">Percent</a>

百分比范围 0 ~ 100。

| Field | Description |
| ----- | ----------- |
| value | 百分比范围  |

# <a name="Headers.HeaderOperations">Headers.HeaderOperations</a>

header 操作方式。

| Field  | Description                         |
| ------ | ----------------------------------- |
| set    | 用给定的值覆盖原 header 中的 Key    |
| add    | 讲给定的值追加到 header 中的 Key 中 |
| remove | 移除指定的 header                   |

# <a name="HTTPFaultInjection.Delay">HTTPFaultInjection.Delay</a>

延迟类故障注入。

例如，在所有标签为 env: prod 的 Pod 中对 reviews 服务的 v1 版本的发起的每 1000 个请求中引入 1 个延迟 5 秒的故障。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews-route
spec:
  hosts:
  - reviews.prod.svc.cluster.local
  http:
  - match:
    - sourceLabels:
        env: prod
    route:
    - destination:
        host: reviews.prod.svc.cluster.local
        subset: v1
    fault:
      delay:
        percentage:
          value: 0.1
        fixedDelay: 5s
```

fixedDelay 字段用于指示延迟量（以秒为单位）。可选的百分比字段可用于仅延迟一定百分比的请求。如果未指定，所有请求都将被延迟。

| Field      | Description                                                  |
| ---------- | ------------------------------------------------------------ |
| fixedDelay | 在转发请求之前添加一个固定的延迟。格式：1h/1m/1s/1ms。必须 >= 1 ms |
| percentage | 注入延迟的请求的百分比                                       |
| percent    | 注入延迟的请求的百分比（0-100）。不推荐使用整数百分比值。请改用 percentage 字段 |

# <a name="HTTPFaultInjection.Abort">HTTPFaultInjection.Abort</a>

中止类故障注入。

例如，为 ratings 服务 v1 版本的每 1000 个请求中的 1 个返回 HTTP 400 错误代码。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings-route
spec:
  hosts:
  - ratings.prod.svc.cluster.local
  http:
  - route:
    - destination:
        host: ratings.prod.svc.cluster.local
        subset: v1
    fault:
      abort:
        percentage:
          value: 0.1
        httpStatus: 400
```

httpStatus 字段表示返回给调用者的 HTTP 状态码。可选的百分比字段只能用于中止一定百分比的请求。如果未指定，则中止所有请求。

| Field                  | Description                      |
| ---------------------- | -------------------------------- |
| httpStatus             | 用于中止 HTTP 请求的 HTTP 状态码 |
| [percentage](#Percent) | 中止请求的百分比                 |

# <a name="HTTPRedirect.RedirectPortSelection">HTTPRedirect.RedirectPortSelection</a>

| Name                  | Description                                        |
| --------------------- | -------------------------------------------------- |
| FROM_PROTOCOL_DEFAULT | 对于 HTTP 自动设置为 80，对于 HTTPS 自动设置为 443 |
| FROM_REQUEST_PORT     | 自动使用请求的端口                                 |
