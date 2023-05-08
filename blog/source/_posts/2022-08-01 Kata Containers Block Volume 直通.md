---
title: "「 Kata Containers 」Block Volume 直通"
excerpt: "Kata Containers stable-2.4 中的 Block Volume 直通特性与实践验证"
cover: https://picsum.photos/0?sig=20220801
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2022-08-01
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> Based on **v2.4.3**

# 背景

Kubernetes 设计之初对基于 VM 的容器运行时考虑较少，很多场景都默认容器运行时能直接访问宿主机资源，这也使得 Kata Containers 在和 Kubernetes 集成时对某些 Kubernetes 特性的支持存在一些不足或限制，尤其是在存储方面。

Kubernetes 提供了 PV （persistent volume）资源来管理存储卷，制定了 CSI （Container Storage Interface）规范在存储提供者和容器运行时之间来管理存储设备。通常来说，CSI 会将不同类型的存储设备，比如云盘、本地存储、网络文件系统等，以文件系统的方式挂载到宿主机，然后再从宿主机将此文件系统挂载到容器中。在 Kata Containers 中，这个挂载是通过 virtiofs 协议，在宿主机和 guest OS 中实现了该存储卷的文件共享。虽然 virtiofs 在性能上比之前的 9p 有很大提升，但是和直接在宿主机上使用相比，性能损耗成为在生产环境中使用 Kata Containers 的阻碍因素之一。

其次，使用 Kata Containers 在线调整 PV 的大小是很困难的。虽然 PV 可以在 host 上扩展，但更新后的元数据需要传递到 Guest OS 中，以便应用程序容器使用扩展的卷。目前，没有办法在不重新启动 Pod Sandbox 的情况下将 PV 元数据从 Host OS 传递到 Guest OS。

一个理想的长期解决方案是 Kubelet 协调 CSI Driver 和 Container Runtime 之间的通信，如 [KEP-2857](https://github.com/kubernetes/enhancements/pull/2893/files) 讨论，但是目前而言，KEP 仍在审查中，并且提议的解决方案有两个弊端：

- 将 csiPlugin.json 文件写入卷的根路径会带来安全隐患。恶意用户可以将自己的 csiPlugin.json 写入上述位置，从而获得对块设备的未经授权的访问
- 提案中并没有描述如何在卷和 Kata Containers 中如何建立映射关系，然而这是 CSI 调整卷大小和信息所需的必备 API

对此 Kata Containers 社区提出一个短期/中期的解决方案 — Block Volume 直通。

**当前 CSI 挂载方式**

<div align=center><img width="800" style="border: 0px" src="https://raw.githubusercontent.com/kubernetes/enhancements/8202b8a7e4f1c19d8f32b40288cc73060828fc34/keps/sig-storage/2857-runtime-assisted-pv-mounts/images/CurrentMounts.png"></div>

**CSI 与 Runtime 协调挂载**

<div align=center><img width="800" style="border: 0px" src="https://raw.githubusercontent.com/kubernetes/enhancements/8202b8a7e4f1c19d8f32b40288cc73060828fc34/keps/sig-storage/2857-runtime-assisted-pv-mounts/images/RuntimeAssistedMounts.png"></div>

# 现阶段缺陷

- 一个块设备卷一次只能由一个节点上的一个 Pod 使用，其实这是 Kata Containers 用例中最常见的模式。将同一个块设备连接到多个 Kata Pod 也是不安全的。在 Kubernetes 中，需要将 PersistentVolumeClaim (PVC) 的 accessMode 设置为 ReadWriteOncePod
- 不支持更高级的 Kubernetes 卷功能，例如 fsGroup、fsGroupChangePolicy 和 subPath

# 实现方案

传统 CSI 都会将存储设备挂载到宿主机上，在 Kata Containers 中，由于 VM 的存在，挂载操作需要移动到 Guest 中，由 Kata Agent 来完成存储卷的挂载。如下所示：

**原挂载方案**

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/kata-mount-current.png"></div>

**直通挂载方案**

<div align=center><img width="600" style="border: 0px" src="/gallery/kata-containers/kata-mount-direct.png"></div>

因此，需要 CSI 具备直通卷的挂载能力，Kata Containers 社区提供了一些参考方案

- StorageClass 参数中指定直通卷的相关标识，这样可以免去 CSI 查询 PVC 或者 Pod 的信息，但是基于该 StorageClass 供应的 PV 均会视为直通卷
- PVC annotation 中注明，需要 CSI Plugin 支持 --extra-create-metadata 
- RuntimeClass 中注明，CSI Driver 在 node publish 阶段通过 Runtime 来获得 Volume 是否需要直接挂载到 Guest 中，参考[阿里云实现](https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/master/pkg/disk/nodeserver.go#L248)

当 CSI Driver 并不会直接将直通卷挂载给 Kata Containers 使用，而是需要在 CSI 的不同阶段调用 Kata Containers 在 2.4 新增的 direct-volume 命令向 Kata Containers 运行时传递并收集卷信息。

- NodePublishVolume 
  调用 kata-runtime direct-volume add --volume-path [volumePath] --mount-info [mountInfo] 将卷挂载信息传递到 Kata Containers 用来执行文件系统挂载操作。 volumePath 是 CSI NodePublishVolumeRequest 中的 target_path。 mountInfo 是一个序列化的 JSON 字符串
- NodeGetVolumeStats
  调用 kata-runtime direct-volume stats --volume-path [volumePath] 获取直通卷的信息
- NodeExpandVolume 
  调用 kata-runtime direct-volume resize --volume-path [volumePath] --size [size] 请求 Kata Containers 调整直通卷的大小
- NodeStageVolume/NodeUnStageVolume
  调用 kata-runtime direct-volume remove --volume-path [volumePath] 删除直通卷的元数据信息

# 源码分析

## kata-runtime direct-volume

*<u>src/runtime/cmd/kata-runtime/kata-volume.go</u>*

Kata Containers 在 2.4 版本时，新增了 kata-runtime direct-volume 的命令，用于管理 Kata Containers 所使用的直通卷。

```go
// MountInfo contains the information needed by Kata to consume a host block device and mount it as a filesystem inside the guest VM.
type MountInfo struct {
	// The type of the volume (ie. block)
	VolumeType string `json:"volume-type"`
	// The device backing the volume.
	Device string `json:"device"`
	// The filesystem type to be mounted on the volume.
	FsType string `json:"fstype"`
	// Additional metadata to pass to the agent regarding this volume.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Additional mount options.
	Options []string `json:"options,omitempty"`
}
```

### add

```shell
$ kata-runtime direct-volume add
```

可选的 flags 包括

| 名称          | 含义                 |
| ------------- | -------------------- |
| --volume-path | 待操作的目标卷路径   |
| --mount-info  | 管理卷挂载的详情信息 |

**主体流程**

1. 校验合法性，创建 /run/kata-containers/shared/direct-volumes/\<base64 volume path\> 目录
2. 将 mount info 序列化为 json，以名为 mountInfo.json 的文件形式保存在该目录下

### remove

```SHELL	
$ kata-runtime direct-volume remove
```

可选的 flags 包括

| 名称          | 含义               |
| ------------- | ------------------ |
| --volume-path | 待操作的目标卷路径 |

**主体流程**

1. 移除 /run/kata-containers/shared/direct-volumes/\<base64 volume path\> 目录

### stats

```SHELL	
$ kata-runtime direct-volume stats
```

可选的 flags 包括

| 名称          | 含义               |
| ------------- | ------------------ |
| --volume-path | 待操作的目标卷路径 |

**主体流程**

1. 获取 /run/kata-containers/shared/direct-volumes/\<base64 volume path\> 目录下的 sandbox id 名称<br>*预期是一个直通卷仅有一个相关联的 sanbox，因此，该目录下，仅有两个文件，一个名为 sandbox id，一个名为 mountInfo.json*
2. 解析 /run/kata-containers/shared/direct-volumes/\<base64 volume path\>/mountInfo.json 文件，构建 mountInfo 对象，获取 volume 的源 device 信息
3. 向 shim 的 /direct-volume/stats 接口发起 http Get 请求
4. 判断 host 上的该 device 是否存在，根据 sandbox 中的 container 卷挂载的映射信息，获取到位于 Guest OS 中的对应的挂载点
5. 向 Kata Agent 发起 rpc 请求，获取 Guest OS 中的卷信息

### resize

```SHELL	
$ kata-runtime direct-volume resize
```

可选的 flags 包括

| 名称          | 含义                     |
| ------------- | ------------------------ |
| --volume-path | 待操作的目标卷路径       |
| --size        | 调整后的卷大小，默认为 0 |

**主体流程**

1. 获取 /run/kata-containers/shared/direct-volumes/\<base64 volume path\> 目录下的 sandbox id 名称<br>*预期是一个直通卷仅有一个相关联的 sandbox，因此，该目录下，仅有两个文件，一个名为 sandbox id，一个名为 mountInfo.json*
2. 解析 /run/kata-containers/shared/direct-volumes/\<base64 volume path\>/mountInfo.json 文件，构建 mountInfo 对象
3. 向 shim 的 /direct-volume/resize 接口发起 http Post 请求
4. 判断 host 上的该 device 是否存在，根据 sandbox 中的 container 卷挂载的映射信息，获取到位于 Guest OS 中的对应的挂载点
5. 向 Kata Agent 发起 rpc 请求，对 Guest OS 中指定的卷进行大小调整

## containerd-shim-kata-v2

### createBlockDevices

*<u>src/runtime/virtcontainers/container.go</u>*

**部分流程**

1. 如果不支持块设备的挂载特性，则直接返回错误信息
2. 针对每一个 container spec 中记录的挂载点 mount，进行以下操作
   1. 判断是否已经有 BlockDeviceID 信息，如果有代表已经有设备关联挂载点，后续不需要为其创建新的设备
   2. 判断 mount 类型是否是 bind
   3. 根据 mount source 获取到 mount info 信息，如果获取不到，则不是直通卷
   4. 在 /run/kata-containers/shared/direct-volumes/\<base64 volume path\> 目录下，写入以 sandbox ID 为名的文件，用于后续 CSI 与 runtime 通信
   5. 根据 mount info 信息，设置 container mount 的所需信息
3. ......

```go
// Mount describes a container mount.
// nolint: govet
type Mount struct {
	// Source is the source of the mount.
	Source string
	// Destination is the destination of the mount (within the container).
	Destination string

	// Type specifies the type of filesystem to mount.
	Type string

	// HostPath used to store host side bind mount path
	HostPath string

	// GuestDeviceMount represents the path within the VM that the device
	// is mounted. Only relevant for block devices. This is tracked in the event
	// runtime wants to query the agent for mount stats.
	GuestDeviceMount string

	// BlockDeviceID represents block device that is attached to the
	// VM in case this mount is a block device file or a directory
	// backed by a block device.
	BlockDeviceID string

	// Options list all the mount options of the filesystem.
	Options []string

	// ReadOnly specifies if the mount should be read only or not
	ReadOnly bool

	// FSGroup a group ID that the group ownership of the files for the mounted volume
	// will need to be changed when set.
	FSGroup *int

	// FSGroupChangePolicy specifies the policy that will be used when applying
	// group id ownership change for a volume.
	FSGroupChangePolicy volume.FSGroupChangePolicy
}
```

# 实践操作

## 准备操作

```shell
# 准备两个 runtime 为 Kata Containers 的 Pod，分别为 direct 和 Virtiofs 的模式
$ kubectl get pod
NAME                READY   STATUS    RESTARTS   AGE
local-kata-direct   1/1     Running   0          2m7s
local-kata-virt     1/1     Running   0          2m3s

$ crictl pods --no-trunc
POD ID                                                             CREATED             STATE               NAME                                                      NAMESPACE           ATTEMPT
3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9   11 minutes ago      Ready               local-kata-virt                                           default             0
3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7   11 minutes ago      Ready               local-kata-direct                                         default             0

$ kubectl get pvc
NAME                STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS        AGE
local-kata-direct   Bound    pvc-23dc3ecf-37fb-44ee-b8a6-7667a98aeb05   20Mi       RWO            local-kata-direct   20m
local-kata-virt     Bound    pvc-0878818c-5d93-4ed4-b034-5bf37a657a6e   20Mi       RWO            local-kata-virt     20m

# 分别在持久卷种写入测试数据
$ kubectl exec -it local-kata-direct touch /datadir/direct-data
$ kubectl exec -it local-kata-virt touch /datadir/virt-data
```

## host 端进程

```shell
# virtiofs 类型的 Pod
$ ps -ef | grep 3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9
root       889 10374  0 17:23 pts/34   00:00:00 grep --color=auto 3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9
root     42721     1  0 17:10 ?        00:00:00 /usr/local/bin/containerd-shim-kata-v2 -namespace k8s.io -address /run/containerd/containerd.sock -publish-binary /usr/bin/containerd -id 3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9 -debug
root     42744 42721  0 17:10 ?        00:00:00 /opt/kata/libexec/kata-qemu/virtiofsd --syslog -o cache=auto -o no_posix_lock -o source=/run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared --fd=3 -f --thread-pool-size=1 -o announce_submounts
root     42751     1  0 17:10 ?        00:00:01 /opt/kata/bin/qemu-system-x86_64 -name sandbox-3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9 -uuid 26864a9b-ad50-43ab-9d41-2b6f4c7e34c3 -machine q35,accel=kvm,kernel_irqchip=on,nvdimm=on -cpu host,pmu=off -qmp unix:/run/vc/vm/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/qmp.sock,server=on,wait=off -m 256M,slots=10,maxmem=129389M -device pci-bridge,bus=pcie.0,id=pci-bridge-0,chassis_nr=1,shpc=off,addr=2,io-reserve=4k,mem-reserve=1m,pref64-reserve=1m -device virtio-serial-pci,disable-modern=false,id=serial0 -device virtconsole,chardev=charconsole0,id=console0 -chardev socket,id=charconsole0,path=/run/vc/vm/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/console.sock,server=on,wait=off -device nvdimm,id=nv0,memdev=mem0,unarmed=on -object memory-backend-file,id=mem0,mem-path=/opt/kata/share/kata-containers/kata-containers.img,size=134217728,readonly=on -device virtio-scsi-pci,id=scsi0,disable-modern=false -object rng-random,id=rng0,filename=/dev/urandom -device virtio-rng-pci,rng=rng0 -device vhost-vsock-pci,disable-modern=false,vhostfd=3,id=vsock-3588796381,guest-cid=3588796381 -chardev socket,id=char-597c9c629356cf41,path=/run/vc/vm/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/vhost-fs.sock -device vhost-user-fs-pci,chardev=char-597c9c629356cf41,tag=kataShared -netdev tap,id=network-0,vhost=on,vhostfds=4,fds=5 -device driver=virtio-net-pci,netdev=network-0,mac=6e:ae:f6:c6:82:0a,disable-modern=false,mq=on,vectors=4 -rtc base=utc,driftfix=slew,clock=host -global kvm-pit.lost_tick_policy=discard -vga none -no-user-config -nodefaults -nographic --no-reboot -daemonize -object memory-backend-file,id=dimm1,size=256M,mem-path=/dev/shm,share=on -numa node,memdev=dimm1 -kernel /opt/kata/share/kata-containers/vmlinux.container -append tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k console=hvc0 console=hvc1 cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/pmem0p1 rootflags=dax,data=ordered,errors=remount-ro ro rootfstype=ext4 quiet systemd.show_status=false panic=1 nr_cpus=48 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.debug_console agent.debug_console_vport=1026 -pidfile /run/vc/vm/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/pid -smp 1,cores=1,threads=1,sockets=48,maxcpus=48
root     42758 42744  0 17:10 ?        00:00:00 /opt/kata/libexec/kata-qemu/virtiofsd --syslog -o cache=auto -o no_posix_lock -o source=/run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared --fd=3 -f --thread-pool-size=1 -o announce_submounts

# direct 类型的 Pod
$ ps -ef | grep 3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7
root     25155 10374  0 17:26 pts/34   00:00:00 grep --color=auto 3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7
root     38975     1  0 17:10 ?        00:00:00 /usr/local/bin/containerd-shim-kata-v2 -namespace k8s.io -address /run/containerd/containerd.sock -publish-binary /usr/bin/containerd -id 3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7 -debug
root     39008 38975  0 17:10 ?        00:00:00 /opt/kata/libexec/kata-qemu/virtiofsd --syslog -o cache=auto -o no_posix_lock -o source=/run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared --fd=3 -f --thread-pool-size=1 -o announce_submounts
root     39312     1  0 17:10 ?        00:00:01 /opt/kata/bin/qemu-system-x86_64 -name sandbox-3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7 -uuid c3fccd7b-f344-4819-a894-80f5e1c5606d -machine q35,accel=kvm,kernel_irqchip=on,nvdimm=on -cpu host,pmu=off -qmp unix:/run/vc/vm/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/qmp.sock,server=on,wait=off -m 256M,slots=10,maxmem=129389M -device pci-bridge,bus=pcie.0,id=pci-bridge-0,chassis_nr=1,shpc=off,addr=2,io-reserve=4k,mem-reserve=1m,pref64-reserve=1m -device virtio-serial-pci,disable-modern=false,id=serial0 -device virtconsole,chardev=charconsole0,id=console0 -chardev socket,id=charconsole0,path=/run/vc/vm/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/console.sock,server=on,wait=off -device nvdimm,id=nv0,memdev=mem0,unarmed=on -object memory-backend-file,id=mem0,mem-path=/opt/kata/share/kata-containers/kata-containers.img,size=134217728,readonly=on -device virtio-scsi-pci,id=scsi0,disable-modern=false -object rng-random,id=rng0,filename=/dev/urandom -device virtio-rng-pci,rng=rng0 -device vhost-vsock-pci,disable-modern=false,vhostfd=3,id=vsock-2091051760,guest-cid=2091051760 -chardev socket,id=char-2c14db67d9a8d23c,path=/run/vc/vm/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/vhost-fs.sock -device vhost-user-fs-pci,chardev=char-2c14db67d9a8d23c,tag=kataShared -netdev tap,id=network-0,vhost=on,vhostfds=4,fds=5 -device driver=virtio-net-pci,netdev=network-0,mac=86:3f:52:0e:93:29,disable-modern=false,mq=on,vectors=4 -rtc base=utc,driftfix=slew,clock=host -global kvm-pit.lost_tick_policy=discard -vga none -no-user-config -nodefaults -nographic --no-reboot -daemonize -object memory-backend-file,id=dimm1,size=256M,mem-path=/dev/shm,share=on -numa node,memdev=dimm1 -kernel /opt/kata/share/kata-containers/vmlinux.container -append tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k console=hvc0 console=hvc1 cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/pmem0p1 rootflags=dax,data=ordered,errors=remount-ro ro rootfstype=ext4 quiet systemd.show_status=false panic=1 nr_cpus=48 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.debug_console agent.debug_console_vport=1026 -pidfile /run/vc/vm/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/pid -smp 1,cores=1,threads=1,sockets=48,maxcpus=48
root     39547 39008  0 17:10 ?        00:00:00 /opt/kata/libexec/kata-qemu/virtiofsd --syslog -o cache=auto -o no_posix_lock -o source=/run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared --fd=3 -f --thread-pool-size=1 -o announce_submounts
You have mail in /var/spool/mail/root

# 可以看到无论是哪种持久卷挂载方式，host 上的进程信息均为：两个 virtiofsd 进程，一个 qemu-system 进程，一个 containerd-shim-kata-v2 进程
# 之所以卷直通模式也会有 virtiofsd 进程启动是因为卷直通仅限于持久卷的部分，对于 sandbox rootfs 仍以 virtiofs 协议挂载至 Guest 中
```

## host 端卷目录结构

```shell
# virtiofs 类型的 Pod
$ ls /var/lib/kubelet/pods/6cccbed3-45eb-4925-92af-aa2313f1e3f8/volumes/kubernetes.io~csi/pvc-0878818c-5d93-4ed4-b034-5bf37a657a6e/mount
lost+found  virt-data

# direct 类型的 Pod
$ ls /var/lib/kubelet/pods/7c974c3d-865a-4684-87ce-25d01a48d89e/volumes/kubernetes.io~csi/pvc-23dc3ecf-37fb-44ee-b8a6-7667a98aeb05/mount/

# direct 类型的卷目录存在，但是没有相应的数据
```

## host 端挂载点

```shell
$ lsblk
...
sdt                                                            65:48   0   200G  0 disk  
└─mpathb                                                      253:11   0   200G  0 mpath 
  ├─i1666574565-lvmlock                                       253:13   0    10G  0 lvm   
  ├─i1666574565-pvc--23dc3ecf--37fb--44ee--b8a6--7667a98aeb05 253:15   0    20M  0 lvm   
  └─i1666574565-pvc--0878818c--5d93--4ed4--b034--5bf37a657a6e 253:16   0    20M  0 lvm   /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-b00bec5dd6f4958e-datadir

# virtiofs 类型的 Pod
$ mount | grep 3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9
shm on /run/containerd/io.containerd.grpc.v1.cri/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shm type tmpfs (rw,nosuid,nodev,noexec,relatime,size=65536k)
overlay on /run/containerd/io.containerd.runtime.v2.task/k8s.io/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254839/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254839/work)
tmpfs on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared type tmpfs (ro,mode=755)
overlay on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254839/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254839/work)
overlay on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254839/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254839/work)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9-2c4e4dce04926bea-resolv.conf type xfs (ro,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9-2c4e4dce04926bea-resolv.conf type xfs (ro,relatime,attr2,inode64,noquota)
overlay on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/251248/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254840/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254840/work)
overlay on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/251248/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254840/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254840/work)
/dev/mapper/i1666574565-pvc--0878818c--5d93--4ed4--b034--5bf37a657a6e on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-b00bec5dd6f4958e-datadir type ext4 (rw,relatime,data=ordered)
/dev/mapper/i1666574565-pvc--0878818c--5d93--4ed4--b034--5bf37a657a6e on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-b00bec5dd6f4958e-datadir type ext4 (rw,relatime,data=ordered)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-d10c2427b7cfd382-hosts type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-d10c2427b7cfd382-hosts type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-aec3c5c428ab0e89-termination-log type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-aec3c5c428ab0e89-termination-log type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-0a63282c5f8f4163-hostname type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-0a63282c5f8f4163-hostname type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-a9d3c3e132ce299b-resolv.conf type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-a9d3c3e132ce299b-resolv.conf type xfs (rw,relatime,attr2,inode64,noquota)
tmpfs on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/mounts/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-30db53e3c43cb09f-serviceaccount type tmpfs (ro,relatime,size=32675592k)
tmpfs on /run/kata-containers/shared/sandboxes/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/shared/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-30db53e3c43cb09f-serviceaccount type tmpfs (ro,relatime,size=32675592k)

# direct 类型的 Pod
$ mount | grep 3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7
shm on /run/containerd/io.containerd.grpc.v1.cri/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shm type tmpfs (rw,nosuid,nodev,noexec,relatime,size=65536k)
overlay on /run/containerd/io.containerd.runtime.v2.task/k8s.io/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254837/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254837/work)
tmpfs on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared type tmpfs (ro,mode=755)
overlay on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254837/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254837/work)
overlay on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254837/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254837/work)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7-27977320e8b95e05-resolv.conf type xfs (ro,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7-27977320e8b95e05-resolv.conf type xfs (ro,relatime,attr2,inode64,noquota)
overlay on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/251248/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254838/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254838/work)
overlay on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379/rootfs type overlay (rw,relatime,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/251248/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254838/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/254838/work)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-b03e8081dab8db44-hosts type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-b03e8081dab8db44-hosts type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-fc067734b16d5aec-termination-log type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-fc067734b16d5aec-termination-log type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-9578923c5ca9be73-hostname type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-9578923c5ca9be73-hostname type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-19d449c66d9d623d-resolv.conf type xfs (rw,relatime,attr2,inode64,noquota)
/dev/sda2 on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-19d449c66d9d623d-resolv.conf type xfs (rw,relatime,attr2,inode64,noquota)
tmpfs on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/mounts/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-fd37e76da5e2d86d-serviceaccount type tmpfs (ro,relatime,size=32675592k)
tmpfs on /run/kata-containers/shared/sandboxes/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/shared/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-fd37e76da5e2d86d-serviceaccount type tmpfs (ro,relatime,size=32675592k)

# 卷直通场景下，持久卷的块设备由 CSI 格式化后 attach 到节点后，不会执行 mount 到节点的操作，而是由 Kata Agent 触发 mount 操作，挂载到 VM 中，因此 host 端看不到挂载点信息，所以写在直通卷中的数据不会出现在 host 中
# 两种模式下，rootfs 的挂载点一致
```

## container 端挂载点

```shell
# virtiofs 类型的 Pod
$ kubectl exec -it local-kata-virt sh
/ # df -Th
Filesystem           Type            Size      Used Available Use% Mounted on
none                 virtiofs       80.0G     65.3G     14.7G  82% /
tmpfs                tmpfs          64.0M         0     64.0M   0% /dev
tmpfs                tmpfs         111.9M         0    111.9M   0% /sys/fs/cgroup
none                 virtiofs       18.4M    332.0K     17.6M   2% /datadir
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /etc/hosts
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /dev/termination-log
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /etc/hostname
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /etc/resolv.conf
shm                  tmpfs         111.9M         0    111.9M   0% /dev/shm
none                 virtiofs       31.2G     12.0K     31.2G   0% /var/run/secrets/kubernetes.io/serviceaccount
tmpfs                tmpfs          64.0M         0     64.0M   0% /proc/timer_list

# direct 类型的 Pod
$ kubectl exec -it local-kata-direct sh
/ # df -Th
Filesystem           Type            Size      Used Available Use% Mounted on
none                 virtiofs       80.0G     65.3G     14.7G  82% /
tmpfs                tmpfs          64.0M         0     64.0M   0% /dev
tmpfs                tmpfs         111.9M         0    111.9M   0% /sys/fs/cgroup
/dev/sda             ext4           18.4M    332.0K     17.6M   2% /datadir
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /etc/hosts
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /dev/termination-log
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /etc/hostname
kataShared           virtiofs       80.0G     65.3G     14.7G  82% /etc/resolv.conf
shm                  tmpfs         111.9M         0    111.9M   0% /dev/shm
none                 virtiofs       31.2G     12.0K     31.2G   0% /var/run/secrets/kubernetes.io/serviceaccount
tmpfs                tmpfs          64.0M         0     64.0M   0% /proc/timer_list

# virtiofs 的场景下的持久卷以 virtiofs 类型挂载到容器中
# direct 的场景下的持久卷以 ext4 类型挂载到容器中，其中 ext4 和 /dev/sda 均可通过直通时指定
```

## VM 端挂载点

```shell
# virtiofs 类型的 Pod
$ kata-runtime exec 3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9
root@clr-bab03a69f32d46fa9bfa8f735f0a4036 / $ df -Th
Filesystem     Type      Size  Used Avail Use% Mounted on
/dev/root      ext4      117M   95M   16M  87% /
devtmpfs       devtmpfs  111M     0  111M   0% /dev
tmpfs          tmpfs     112M     0  112M   0% /dev/shm
tmpfs          tmpfs      45M   28K   45M   1% /run
tmpfs          tmpfs     4.0M     0  4.0M   0% /sys/fs/cgroup
tmpfs          tmpfs     112M     0  112M   0% /tmp
kataShared     virtiofs   63G  147M   63G   1% /run/kata-containers/shared/containers
shm            tmpfs     112M     0  112M   0% /run/kata-containers/sandbox/shm
none           virtiofs   80G   66G   15G  83% /run/kata-containers/3f1316b887ba0cf7b0144a095fb543a262ae0c2b32a94c7658e1dbb3707c5ea9/rootfs
none           virtiofs   80G   66G   15G  83% /run/kata-containers/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce/rootfs
none           virtiofs   32G   12K   32G   1% /run/kata-containers/shared/containers/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-30db53e3c43cb09f-serviceaccount
none           virtiofs   19M  332K   18M   2% /run/kata-containers/shared/containers/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-b00bec5dd6f4958e-datadir
root@clr-bab03a69f32d46fa9bfa8f735f0a4036 / $ lsblk
NAME      MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
pmem0     259:0    0  126M  1 disk 
`-pmem0p1 259:1    0  124M  1 part /

# direct 类型的 Pod
$ kata-runtime exec 3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7
root@clr-e5fe15d519854877912b7c4556e71299 / $ df -Th
Filesystem     Type      Size  Used Avail Use% Mounted on
/dev/root      ext4      117M   95M   16M  87% /
devtmpfs       devtmpfs  111M     0  111M   0% /dev
tmpfs          tmpfs     112M     0  112M   0% /dev/shm
tmpfs          tmpfs      45M   28K   45M   1% /run
tmpfs          tmpfs     4.0M     0  4.0M   0% /sys/fs/cgroup
tmpfs          tmpfs     112M     0  112M   0% /tmp
kataShared     virtiofs   63G  147M   63G   1% /run/kata-containers/shared/containers
shm            tmpfs     112M     0  112M   0% /run/kata-containers/sandbox/shm
none           virtiofs   80G   66G   15G  83% /run/kata-containers/3763faeb5fc0aa25265b751123b63fbbae1c8aa35bec202c42a4d569c0ef63a7/rootfs
/dev/sda       ext4       19M  332K   18M   2% /run/kata-containers/sandbox/storage/MDow
none           virtiofs   80G   66G   15G  83% /run/kata-containers/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379/rootfs
none           virtiofs   32G   12K   32G   1% /run/kata-containers/shared/containers/6da9c84109d97ce2895b3e5b38631ebdf7185fc7b915cf4766572803ce4df379-fd37e76da5e2d86d-serviceaccount
root@clr-e5fe15d519854877912b7c4556e71299 / $ lsblk                            
NAME      MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
sda         8:0    0   20M  0 disk /run/kata-containers/sandbox/storage/MDow
pmem0     259:0    0  126M  1 disk 
`-pmem0p1 259:1    0  124M  1 part /

# direct 类型的 Pod 会较 virtiofs 多一个挂载点，/dev/sda -> /run/kata-containers/sandbox/storage/MDow
# virtiofs 类型的 Pod 会较 direct 多一个挂载点，/run/kata-containers/shared/containers/cc3782d5f7aef968f640a0c161ee027efec8054c9860cae9d7254f8fe0659cce-b00bec5dd6f4958e-datadir
```

