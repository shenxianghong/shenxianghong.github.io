---
layout: post
title:  "[ Kata Containers ] 3. Rsource Limitation"
date:   2021-04-11
excerpt: "Kata Containers 在 K8s 中资源限制问题"
project: true
tag:
- Cloud Native
- Kubernetes
- Kata Containers
- Container Runtime
comments: false
---

# Overhead

通过 `overhead.podFixed` 指定额外的 1C，2G 资源，这部分资源可以被 K8s 控制面感知，并体现在数据面，在 Pod 调度、ResourceQuota 以及 Pod 驱逐等场景下均会受到影响。但是，需要注意的是，overhead 的资源仅用作上层编排、调度等，并不会作用于底层 VM 的实际大小。

```yaml
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: kata-runtime
handler: kata
overhead:
  podFixed:
    memory: "2000Mi"
    cpu: "1000m"
```

被 overhead 注入的 pod，可以通过 `kubectl get pod <pod> -o jsonpath='{.spec.overhead}'` 查看额外注入的资源。

# Pod QoS

overhead 的注入不会影响到 Pod 的 QoS。overhead 中申请的额外资源，会追加到 Pod 的 request 值，从而影响到控制面的调度等场景，如果 Pod 声明了 limit，同样的也会追加到 limit 中。

*guaranteed*

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: guaranteed
spec:
  nodeName: archcnstcm5403
  runtimeClassName: kata-runtime
  containers:
    - name: uname
      image: busybox
      command: ["/bin/sh", "-c", "uname -r && tail -f /dev/null"]
      resources:
        requests:
          memory: "1000Mi"
          cpu: "1000m"
        limits:
          memory: "1000Mi"
          cpu: "1000m"
```

*burstable*

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: burstable
spec:
  nodeName: archcnstcm5403
  runtimeClassName: kata-runtime
  containers:
    - name: uname
      image: busybox
      command: ["/bin/sh", "-c", "uname -r && tail -f /dev/null"]
      resources:
        limits:
          cpu: "1000m"
        requests:
          cpu: "1000m"
```

*besteffect*

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: besteffort
spec:
  nodeName: archcnstcm5403
  runtimeClassName: kata-runtime
  containers:
    - name: uname
      image: busybox
      command: ["/bin/sh", "-c", "uname -r && tail -f /dev/null"]
```

查看 Pod 最终资源信息

```shell
  Namespace                   Name                                      CPU Requests  CPU Limits  Memory Requests  Memory Limits
  default                     besteffort                                1 (2%)        0 (0%)      2000Mi (1%)      0 (0%)       
  default                     burstable                                 2 (4%)        2 (4%)      2000Mi (1%)      0 (0%)      
  default                     guaranteed                                2 (4%)        2 (4%)      3000Mi (2%)      3000Mi (2%)   
```

# Kata VM

*/etc/kata-containers/configuration.toml*

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

Kata Pod 中额外对资源的限制是通过 `hotplug` 的方式实现。资源目前特指 CPU 和 Memory。Pod requests 影响到调度等控制层面的行为，不同于 Limits，它不会对 Kata VM 的资源造成影响。而**最终的 VM 资源大小为 limit + default，其中 limit 为 Pod 声明的 limit，而非包含 overhead 在内的最终限制**。

以上述的 guaranteed Pod 为例，可以看到，最终的 VM 是一个 2C，3G 的规格大小，是因为 Pod limit（1C，1G）+ Kata Config（1C，2G）

```shell
bash-5.0# cat /proc/cpuinfo | grep processor | wc -l
2
bash-5.0# free -m
              total        used        free      shared  buff/cache   available
Mem:           3009          38        2941          29          30        2913
Swap:             0           0           0
```

总结一下，Kubernetes 新增了 Kata Containers 作为底层 runtime 后，对于 runtime 运行环境的额外开销不容忽视，但是 K8s 角度又无法感知到这部分资源，而 overhead 的设计就弥补了这一缺陷，并且 overhead 对于资源的额外声明，是会统计在 Cgroup 中的

# Cgroup

从 Kubernetes 角度来讲，Cgroup 指的是 Pod Cgroup，由 Kubelet 创建，限制的是 Pod 的资源；
从 Container 角度来讲，Cgroup 指的是 Container Cgroup，由对应的 runtime 创建，限制的是 Container 的资源。

但是为了可以获取到更准确的容器资源，Kubelet 会根据 Container Cgroup 去调整 Pod Cgroup。在传统的 runtime 中，两者没有太大的区别。而 Kata Containers 引入 VM 的概念，所以针对这种情况有两种处理方式：

- 启用 SandboxCgroupOnly，Kubelet 在调整 Pod Cgroup 的大小时，会将 sandbox 的开销统计进去
- 禁用 SandboxCgroupOnly，sandbox 的开销和 Pod Cgroup 分开计算，独立存在

## Resource

| Location  | Kind               | runC             | Kata sandbox_cgroup_only = true | Kata  sandbox_cgroup_only = false |
| --------- | ------------------ | ---------------- | ------------------------------- | --------------------------------- |
| host      | Pod                | overhead + limit | overhead + limit                | overhead + limit                  |
| host      | Infra container    | -1               | -1                              | -1                                |
| host      | workload container | limit            | /                               | limit                             |
| Container | /                  | limit            | limit                           | limit                             |
| VM        | Pod                | /                | -1                              | -1                                |
| VM        | Infra container    | /                | -1                              | -1                                |
| VM        | workload container | /                | limit                           | limit                             |

[^Location]: host 代表

