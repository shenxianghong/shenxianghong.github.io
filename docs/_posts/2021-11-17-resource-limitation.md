---
layout: post
title: Kata Containers Resource Limitation
date: 2021-11-17
excerpt: "Demo post displaying the various ways of highlighting code in Markdown."
tags: [kata-containers]
comments: true
---

## Kata Configuration

```toml
# Default number of vCPUs per SB/VM:
# unspecified or 0                --> will be set to 1
# < 0                             --> will be set to the actual number of physical cores
# > 0 <= number of physical cores --> will be set to the specified number
# > number of physical cores      --> will be set to the actual number of physical cores
default_vcpus = 1

# Default maximum number of vCPUs per SB/VM:
# unspecified or == 0             --> will be set to the actual number of physical cores or to the maximum number
#                                     of vCPUs supported by KVM if that number is exceeded
# > 0 <= number of physical cores --> will be set to the specified number
# > number of physical cores      --> will be set to the actual number of physical cores or to the maximum number
#                                     of vCPUs supported by KVM if that number is exceeded
# WARNING: Depending of the architecture, the maximum number of vCPUs supported by KVM is used when
# the actual number of physical cores is greater than it.
# WARNING: Be aware that this value impacts the virtual machine's memory footprint and CPU
# the hotplug functionality. For example, `default_maxvcpus = 240` specifies that until 240 vCPUs
# can be added to a SB/VM, but the memory footprint will be big. Another example, with
# `default_maxvcpus = 8` the memory footprint will be small, but 8 will be the maximum number of
# vCPUs supported by the SB/VM. In general, we recommend that you do not edit this variable,
# unless you know what are you doing.
# NOTICE: on arm platform with gicv2 interrupt controller, set it to 8.
default_maxvcpus = 0

# Default memory size in MiB for SB/VM.
# If unspecified then it will be set 2048 MiB.
default_memory = 2048
```

- default_vcpus  表示 Kata VM 中的 CPU 个数，默认为 1
- default_maxvcpus 表示 Kata VM 中的最多的 CPU 个数，默认为 0，即 host 上的所有 CPU
- default_memory 表示 Kata VM 中的内存大小，默认为 2G

**需要注意的是，Kata VM 中的资源量，不仅仅受 Kata 配置影响，还会受到 Pod Resource 的影响。**

## Pod Resource

Kata Pod 中额外对资源的限制是通过 hotplug 的方式实现。资源目前特指 CPU 和 Memory

### Requests

Pod requests 影响到调度等控制层面的行为，不同于 Limits，它不会对 Kata VM 的资源造成影响。

### Limits

- 没有设置，Kata VM 中的资源最终为 default_vcpus（default_memory）
- 设置了，Kata VM 中的资源最终为 limits + default_vcpus（default_memory）

## Runtimeclass Overhead

```yaml
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: kata-containers
handler: kata
overhead:
  podFixed:
    memory: "500Mi"
    cpu: "500m
```

如果 Pod 额外指定了资源 limits 或 requests，则 overhead 中定义的资源限制会追加到其中，如果没有指定，则不会影响到 Pod。overhead 影响的方向是控制层面，不会影响到创建出来的 VM 的资源大小。

## Cgroup

```toml
# if enabled, the runtime will add all the kata processes inside one dedicated cgroup.
# The container cgroups in the host are not created, just one single cgroup per sandbox.
# The runtime caller is free to restrict or collect cgroup stats of the overall Kata sandbox.
# The sandbox cgroup path is the parent cgroup of a container with the PodSandbox annotation.
# The sandbox cgroup is constrained if there is no container type annotation.
# See: https://godoc.org/github.com/kata-containers/runtime/virtcontainers#ContainerType
sandbox_cgroup_only=true
```

pod 级别的 cgroup 是由 kubelet 创建的；container 级别的 cgroup 是由 CRI 创建的。

### resource limit

| Location  | Kind               | runC             | Kata sandbox_cgroup_only = true | Kata  sandbox_cgroup_only = false |
| --------- | ------------------ | ---------------- | ------------------------------- | --------------------------------- |
| host      | Pod                | overhead + limit | overhead + limit                | overhead + limit                  |
| host      | Infra container    | -1               | -1                              | -1                                |
| host      | workload container | limit            | /                               | limit                             |
| Container | /                  | limit            | limit                           | limit                             |
| VM        | Pod                | /                | -1                              | -1                                |
| VM        | Infra container    | /                | -1                              | -1                                |
| VM        | workload container | /                | limit                           | limit                             |

### tasks

| Location  | Kind               | runC     | Kata sandbox_cgroup_only = true | Kata  sandbox_cgroup_only = false |
| --------- | ------------------ | -------- | ------------------------------- | --------------------------------- |
| host      | Pod                | 无       | 无                              | 无                                |
| host      | Infra container    | pause    | viriofsd, containerd 等         | 无                                |
| host      | workload container | workload | workload                        | 无                                |
| Container | /                  | workload | workload                        | workload                          |
| VM        | Pod                | /        | 无                              | 无                                |
| VM        | Infra container    | /        | pause                           | pause                             |
| VM        | workload container | /        | workload                        | workload                          |