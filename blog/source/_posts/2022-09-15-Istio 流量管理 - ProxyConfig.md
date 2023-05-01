---
title: "「 Istio 」 3.7 流量管理 — ProxyConfig"
excerpt: "Istio 中流量管理组件 ProxyConfig 对象介绍"
cover: https://picsum.photos/0?sig=20220915
thumbnail: https://istio.io/v1.8/img/istio-bluelogo-whitebackground-framed.svg
date: 2022-09-15
toc: true
categories:
- Overview
tag:
- Istio
---

<div align=center><img width="150" style="border: 0px" src="https://www.vectorlogo.zone/logos/istioio/istioio-ar21.svg"></div>

------

> Based on **v1.15.0**

ProxyConfig 暴露代理级别的配置选项。 ProxyConfig 可以基于每个工作负载、每个命名空间或网格范围进行配置。 ProxyConfig 不是必需的资源。

*注意：ProxyConfig 中的字段不是动态配置的，更改配置需要重启工作负载才能生效。*

对于任何命名空间，包括根配置命名空间，仅对只有一个无 workloadSelector 的 ProxyConfig 资源生效。

对于具有 workloadSelector 的资源，仅对只有一个资源选择任何给定工作负载生效。

对于网格级别配置，ProxyConfig 需部署在 Istio 安装的根配置命名空间 istio-system 中，并且无需设置 workloadSelector。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ProxyConfig
metadata:
  name: my-proxyconfig
  namespace: istio-system
spec:
  concurrency: 0
  image:
    imageType: distroless
```

对于命名空间级别的配置，ProxyConfig 需部署在该命名空间 user-namespace 中，并且无需设置 workloadSelector。

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ProxyConfig
metadata:
  name: my-ns-proxyconfig
  namespace: user-namespace
spec:
  concurrency: 0
```

对于工作负载级别配置，在 ProxyConfig 资源上设置 selector 字段：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ProxyConfig
metadata:
  name: per-workload-proxyconfig
  namespace: example
spec:
  selector:
    matchLabels:
      app: ratings
  concurrency: 0
  image:
    imageType: debug
```

如果定义了与工作负载匹配的 ProxyConfig CR，它将与其 proxy.istio.io/config 注释（如果存在）合并，重复字段内容以 ProxyConfig CR 为准。同样，如果定义了网格范围的 ProxyConfig CR 并设置了 meshConfig.DefaultConfig，则两个资源将合并，重复字段内容以 ProxyConfig CR 为准。

# ProxyConfig

| Field                | Description                                                  |
| -------------------- | ------------------------------------------------------------ |
| selector             | selector 指定待应用此 ProxyConfig 的一组 Pod/VM。如果未设置，ProxyConfig 资源将应用于其所在命名空间中的所有工作负载 |
| concurrency          | 要运行的工作线程数。如果未设置，则默认为 2。如果设置为 0，这将被配置为使用机器上的所有内核使用 CPU 请求和限制来选择一个值，限制优先于请求 |
| environmentVariables | 代理的其他环境变量。以 ISTIO_META_ 开头的名称将包含在生成的引导配置中并发送到 XDS 服务器 |
| [image](#ProxyImage) | 指定代理的镜像                                               |

# <a name="ProxyImage">ProxyImage</a>

用于构造代理镜像 url。格式：${hub}/${image_name}/${tag}-${image_type}，例如：docker.io/istio/proxyv2:1.11.1 或 docker.io/istio/proxyv2:1.11.1-distroless。

| Field     | Description                                                  |
| --------- | ------------------------------------------------------------ |
| imageType | 镜像的类型。可选的有：default、debug、distroless。如果 image 类型已经（例如：centos）发布到指定的 hub，则允许使用其他值。 |
