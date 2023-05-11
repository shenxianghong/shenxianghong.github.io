---
title: "「 Kata Containers 」资源限制场景总结"
excerpt: "Kata Containers 在 Kubernetes 集群场景中资源限制与实践验证"
cover: https://picsum.photos/0?sig=20210714
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2021-07-14
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **2.1.1**

# Overhead

通过 `overhead.podFixed` 指定额外的 1C，2G 资源，这部分资源可以被 K8s 控制面感知，并体现在数据面，具体包括在 Pod 调度、ResourceQuota 以及 Pod 驱逐等场景下均会受到影响。但是，需要注意的是，overhead 的资源仅用作上层（K8s 层面）编排、调度等，并不会作用于底层 VM 的实际大小。

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

overhead 的注入不会影响到 Pod 的 QoS。overhead 中申请的额外资源，会追加到 Pod 的 request 值（即使 Pod 没有设置 request 值），从而影响到控制面的调度等场景，如果 Pod 声明了 limit，同样的也会追加到 limit 中。需要注意的是：虽然 overhead 最终会影响到 pod limit 和 request，但是**不会影响到 Pod 绑核**，Pod 的绑核仍然依据 request 和 limit。

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

**默认情况，Kata VM 最小的资源是 1C 256M，内存低于 256M 时，会默认为 2G**

Kata Pod 中额外对资源的限制是通过 `hotplug` 的方式实现。资源目前特指 CPU 和 Memory。Pod requests 影响到调度等控制层面的行为，不同于 Limits，它不会对 Kata VM 的资源造成影响。而**最终的 VM 资源大小为 limit + default，其中 limit 为 Pod 声明的 limit，而不包含 overhead 在内**。

以上述的 guaranteed Pod 为例，可以看到，最终的 VM 是一个 2C，3G 的规格大小，是因为 Pod limit（1C，1G）+ Kata Config（1C，2G）

```shell
bash-5.0# cat /proc/cpuinfo | grep processor | wc -l
2
bash-5.0# free -m
              total        used        free      shared  buff/cache   available
Mem:           3009          38        2941          29          30        2913
Swap:             0           0           0
```

# Cgroup

从 Kubernetes 角度来讲，Cgroup 指的是 Pod Cgroup，由 Kubelet 创建，限制的是 Pod 的资源；
从 Container 角度来讲，Cgroup 指的是 container Cgroup，由对应的 runtime 创建，限制的是 container 的资源。

但是为了可以获取到更准确的容器资源，Kubelet 会根据 Container Cgroup 去调整 Pod Cgroup。在传统的 runtime 中，两者没有太大的区别。而 Kata Containers 引入 VM 的概念，所以针对这种情况有两种处理方式：

- 启用 SandboxCgroupOnly，Kubelet 在调整 Pod Cgroup 的大小时，会将 sandbox 的开销统计进去
- 禁用 SandboxCgroupOnly，sandbox 的开销和 Pod Cgroup 分开计算，独立存在

**host 上 Pod 的 Cgroup 限制，虽然会受到 overhead 的影响，但是仅限于 request 和 limit 本身就存在的情况。如果 request 不存在，那么 overhead 不会被追加到 Cgroup 中。**

## Resource

| Location  | Kind               | runC             | Kata (true)      | Kata (false)     |
| --------- | ------------------ | ---------------- | ---------------- | ---------------- |
| host      | Pod                | overhead + limit | overhead + limit | overhead + limit |
| host      | Infra container    | -1               | -1               | -1               |
| host      | workload container | limit            | /                | limit            |
| container | /                  | limit            | limit            | limit            |
| VM        | Pod                | /                | -1               | -1               |
| VM        | Infra container    | /                | -1               | -1               |
| VM        | workload container | /                | limit            | limit            |

## Task

| Location  | Kind               | runC     | Kata (true)             | Kata (false) |
| --------- | ------------------ | -------- | ----------------------- | ------------ |
| host      | Pod                | 无       | 无                      | 无           |
| host      | Infra container    | pause    | viriofsd, containerd 等 | 无           |
| host      | workload container | workload | workload                | 无           |
| Container | /                  | workload | workload                | workload     |
| VM        | Pod                | /        | 无                      | 无           |
| VM        | Infra container    | /        | pause                   | pause        |
| VM        | workload container | /        | workload                | workload     |

*Location 表示 Cgroup 文件的位置，分别为宿主机、容器、Kata VM*

*Kind 表示 Cgroup 中的层级信息，Pod 为 `/sys/fs/cgroup/cpu/kubepods/<pod id>`，Infra Container 为 `/sys/fs/cgroup/cpu/kubepods/<pod id>/<infra id>`，workload container 为 `/sys/fs/cgroup/cpu/kubepods/<pod id>/<workload id>`*

*/ 表示在该模式下，不存在此对象，无 表示文件是空*

**总结**

从 host 视角来看，在 Kata 没有开启 SandboxCgroupOnly 的时候，可以看到有两个容器（infra 和 workload）的 Cgroup 策略文件，结构模式类似于 runC，但是并没有找到有关限制进程信息的 task 文件。

从 container 视角看，三种情况表象一致，均为 Pod 工作负载的最大资源限量。

从 VM 视角看，无论是否开启 SandboxCgroupOnly，都可以看到有两个容器（infra 和 workload）的 Cgroup 策略文件，VM 中的 Cgroup 都是针对工作负载做的限制，而这个视图更像是 runC 中看到的一切。

**从进程服务来讲，overhead 提供了额外的资源消耗的空间，但是并不代表额外的资源消耗会严格遵守这部分空间，而是会与业务进程进行资源抢占；但是业务进程的 cgroup 做了更细粒度的划分，只能使用 Pod limit 中限制的资源量。而当没有任何资源使用约束条件时，Kata 容器使用的最大资源量就是 Kata 配置文件中的默认大小。**

# Example

那么举例做一个说明

**Kata Config**

```toml
default_vcpus = 5
default_memory = 2048
```

**RuntimeClass**

*声明了一个 1C 的额外资源*

```yaml
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: kata-runtime
handler: kata
overhead:
  podFixed:
    memory: "1000Mi"
    cpu: "1000m"
```

**Pod**

*声明了一个最大 2C 的业务容器*

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: guaranteed
spec:
  runtimeClassName: kata-runtime
  containers:
    - name: uname
      image: busybox
      command: ["/bin/sh", "-c", "uname -r && tail -f /dev/null"]
      resources:
        requests:
          memory: "1000Mi"
          cpu: "2"
        limits:
          memory: "1000Mi"
          cpu: "2"
```

**模拟高计算服务**

```bash
#! /bin/sh 
# filename killcpu.sh
if [ $# != 1 ] ; then
  echo "USAGE: $0 <CPUs>"
  exit 1;
fi
for i in `seq $1`
do
  echo -ne " 
i=0; 
while true
do
i=i+1; 
done" | /bin/sh &
  pid_array[$i]=$! ;
done
 
for i in "${pid_array[@]}"; do
  echo 'kill ' $i ';';
done
```

根据以上结论理应存在：

- 1C 的额外资源会作用于 Kata Containers 的额外开销，不会作用在业务负载容器中
- 2C 的容器资源为容器的最大使用上限
- overhead 的 1C + Pod 的 2C 一共作为 VM 的最大使用量
- VM 中的 CPU 个数为 7C

**查看 Pod 的资源限制**

可以看到，pod 的 cpu limit 最终为 1（overhead） + 2（limit）

```
[root@archcnstcm5403 kata]# kubectl describe node | grep guaranteed
  default       guaranteed                 3 (6%)        3 (6%)      2000Mi (1%)      2000Mi (1%)    26s
```

**Pod 中运行高负载应用**

在 host 上通过 top 可以看到，进程占用了 2C ，而并不是上步骤看到的 3C 的最大使用量

```shell
PID   USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND
28239 root      20   0 3948324 192908 177888 S 197.7  0.1   0:21.27 qemu-system-x86 
```

**VM 中运行高负载应用**

在 host 上通过 top 可以看到，进程占用了 3C

```shell
PID   USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND
28239 root      20   0 3948324 195244 180212 S 298.3  0.1   8:43.44 qemu-system-x86 
```

**总结**

也就是说，Pod limit 最终作用的对象就是业务容器的资源限制，而 overhead 作用的对象是附加在业务容器之上的，两者之和是最终 VM 的资源限制。

# Overhead.Podfixed

Kubernetes 新增了 Kata Containers 作为底层 runtime 后，对于 runtime 运行环境的额外开销不容忽视，但是 K8s 角度又无法感知到这部分资源，而 overhead 的设计就弥补了这一缺陷，并且 overhead 对于资源的额外声明，是会统计在 Cgroup 中的，所以即使底层 Kata Containers 的配置即使很高，也可以通过 limit 实现资源限额，这是因为 Kata 对于资源并不是完全占用，不同的 Kata VM 之间会存在资源抢占现象。

## Overhead.Podfixed 等于 Kata Config

两者相等的情况，Kata VM 的资源大小就是 VM 可以用到的资源大小。

但是默认的 1C 2G 在大多数场景下过于浪费，在 Kata Containers 和 runC 的比较下来看，资源大小难以统一管理。

## Overhead.Podfixed 小于 Kata Config

在 overhead 远小于 Kata Config 的时候，可以根据 Pod request 的值动态的调整 overhead 的大小，也就是随着业务负载请求资源的变大，可以理解成需要 Kata VM，OS，Kernel 做更多的工作来满足服务开销。

但是从 VM 视角来看的话，VM 的大小并不是真实的可用大小（比如一个 4C 8G 的 VM，其真实可用的大小或许只有 1C 2G）。不过，这种情况也不会浪费资源，因为 Kata 对于资源仍然是抢占而不是独占。
