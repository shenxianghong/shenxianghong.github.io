---
title: "「 Kata Containers 」源码走读 — virtcontainers/storage"
excerpt: "virtcontainers 中与 PersistDriver、FilesystemSharer 等文件存储相关的流程梳理"
cover: https://picsum.photos/0?sig=20230305
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-03-05
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**

# PersistDriver

*<u>src/runtime/virtcontainers/persist/api/interface.go</u>*

PersistDriver（也称 store）的实现有两类：fs 和 rootless。

```go
type FS struct {
	sandboxState    *persistapi.SandboxState
	containerState  map[string]persistapi.ContainerState
	storageRootPath string
	driverName      string
}
```

在 fs driver 中，storageRootPath 为 /run/vc，用于保存 sandbox（sbs，其中容器信息以子目录形式保存）和 VM（vm）相关状态信息。

```go
type RootlessFS struct {
	*FS
}
```

rootlessfs driver 完全继承 fs driver，唯一的区别在于 rootlessfs driver 的 storageRootPath 为 \<XDG_RUNTIME_DIR\>/run/vc（当环境变量 XDG_RUNTIME_DIR 缺省时为 /run/user/\<UID\>）。

**工厂函数**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/manager.go#L49)

1. 根据当前是否为 root 用户权限，返回对应的 fs 实现

*以下接口 fs 和 rootlessfs 实现方式完全一样。*

## ToDisk

**保存 sandbox 和容器状态信息到文件中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L77)

1. 以当前用户组信息创建 \<RunStoragePath\>/\<sandboxID\> 目录（如果不存在）
2. 创建 \<RunStoragePath\>/\<sandboxID\>/persist.json 文件（如果不存在），写入 sandbox 状态信息
3. 遍历所有的容器状态信息
   1. 以当前用户组信息创建 \<RunStoragePath\>/\<sandboxID\>/\<containerID\> 目录（如果不存在）
   2. 创建 \<RunStoragePath\>/\<sandboxID\>/\<containerID\>/persist.json 文件（如果不存在），写入容器状态信息
4. 遍历 \<RunStoragePath\>/\<sandboxID\> 目录（目录下的所有子目录名称均为 containerID），由于步骤 3 中为当前全量的容器状态信息，以此为准移除不存在的容器目录

## FromDisk

**读取写入文件中的 sandbox 和容器状态信息**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L170)

1. 读取 \<RunStoragePath\>/\<sandboxID\>/persist.json 文件内容，获取 sandbox 状态信息
2. 遍历 \<RunStoragePath\>/\<sandboxID\> 目录（目录下的所有子目录名称均为 containerID），读取 \<RunStoragePath\>/\<sandboxID\>/\<containerID\>/persist.json 文件内容，获取容器状态信息

## Destroy

**删除 sandbox 状态信息目录**

*因为 sandbox 和容器状态信息目录之间为父子目录关系，删除父目录即可*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L228)

1. 删除 \<RunStoragePath\>/\<sandboxID\> 目录

## Lock

**对 sandbox 状态信息目录上锁**

*因为 sandbox 和容器状态信息目录之间为父子目录关系，对父目录上锁即可*

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L244)

1. 调用 syscall.Flock，对 \<RunStoragePath\>/\<sandboxID\> 目录上共享锁或排他锁（根据函数传参而定）
2. 返回一个调用 syscall.Flock 的函数体，对 \<RunStoragePath\>/\<sandboxID\> 目录释放锁（即 syscall.LOCK_UN）

## GlobalWrite

**在状态信息目录中的文件写入内容**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L282)

1. 以当前用户组信息创建 \<storageRootPath\>/\<relativePath\> （relativePath 为函数传参中的待写入内容的相对路径）所在目录（如果不存在）
2. 创建 \<storageRootPath\>/\<relativePath\> 文件，写入数据

## GlobalRead

**读取状态信息目录中的文件内容**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L315)

1. 读取 \<storageRootPath\>/\<relativePath\> 文件内容

## RunStoragePath

**获取 sandbox 状态信息目录**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L337)

1. 返回 \<storageRootPath\>/sbs 路径

## RunVMStoragePath

**获取 vm 状态信息目录**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/persist/fs/fs.go#L341)

1. 返回 \<storageRootPath\>/vm 路径

# FilesystemSharer

*<u>src/runtime/virtcontainers/fs_share.go</u>*

FilesystemSharer（也称 fsSharer）仅有 linux 操作系统下的实现。

```go
type FilesystemShare struct {
	sandbox *Sandbox
	sync.Mutex
	prepared bool
}
```

*工厂函数为参数赋值初始化，无复杂逻辑，不作详述。*

## bindMount

**将 src 绑定挂载到 dst**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/mount.go#L257)

1. 处理 src 中的符号链接，获得绝对路径，校验其是否存在
2. 创建 dst 的父目录（如果不存在）
3. 根据 src 的格式（目录或文件），创建 dst
4. 将 src 以 MS_BIND 的属性绑定挂载到 dst 下<br>*等价于 mount --bind foo bar，也就是把 foo 目录绑定挂载到 bar 目录，bar 目录为 foo 目录的镜像挂载点。绑定后的两个目录类似于硬链接，无论读写 bar 还是读写 foo，都会反应在另一方，内核在底层所操作的都是同一个物理位置。将 bar 卸载后，bar 目录回归原始空目录状态，期间所执行的修改都保留在 foo 目录下*
5. 更改 dst 目录挂载属性（即 MS_SHARED、MS_PRIVATE、MS_SLAVE 和 MS_UNBINDABLE）<br>*等价于 mount --make-slave bar，也可以和步骤 4 合并操作 mount --make-slave --bind foo bar。单向传播模式下，在 foo 下添加或移除子挂载点，会同步到 bar 挂载点，而在 bar 下添加或移除子挂载点，不会影响 foo*
6. 如果挂载为只读属性，则追加至绑定挂载属性中<br>*等价于 mount --read-only --bind foo bar*

## Prepare

**准备 host/guest 的共享文件系统目录**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/fs_share_linux.go#L127)

1. 创建 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/shared 目录（用于 9p/virtiofs 在 host 和 guest 之间共享数据）
2. 创建 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts 目录（用于维护所有 host 和 guest 之间的挂载点）
3. 调用 **bindMount**，将 mounts 目录以只读和 MS_SLAVE 的属性绑定挂载到 shared 目录下（为了后面 mounts 挂载点下的子挂载也能出现在 shared 中）
4. 处理 [runtime]. sandbox_bind_mounts 挂载点
   1. 创建 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/sandbox-mounts 目录
   2. 针对 sandbox_bind_mounts 目录中的每一个挂载点，调用 **bindMount**，以只读和 MS_PRIVATE 的属性绑定挂载到 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/sandbox-mounts/\<sandbox_bind_mounts\> 中，并追加 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/shared/sandbox-mounts/\<sandbox_bind_mounts\> 绑定挂载的只读属性
5. 设置 prepared 为 true，表示共享文件系统已就绪（标识位也用于保证 Prepare 和 Cleanup 操作的幂等性）

## Cleanup

**清理 host/guest 的共享文件系统目录**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/fs_share_linux.go#L180)

1. 处理 [runtime]. sandbox_bind_mounts 挂载点
   1. 针对 sandbox_bind_mounts 目录中的每一个挂载点，移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/sandbox-mounts/\<sandbox_bind_mounts\> 挂载点
   2. 删除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/sandbox-mounts 目录
2. 移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/shared 挂载点
3. 移除 sandbox 中所有容器的挂载点。如果容器 rootfs 的类型为 fuse.nydus-overlayfs，则移除 /rafs/\<containerID\>/lowerdir virtiofs 挂载点，移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/snapshotdir 挂载点并删除此目录，删除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs 目录（该目录和 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/rootfs 数据同步，借助 9pfs/virtiofs 实现容器文件系统共享，在 host 上以 overlay 挂载点形式存在）；否则，移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs 挂载点并删除此目录
4. 删除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\> 目录

## ShareFile

**共享 host 文件至 guest 中**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/fs_share_linux.go#L227)

1. 共享文件（shareFile）名称格式为 \<containerID\>-\<random bytes\>-\<dst\><br>*例如：\<containerID\>-47dcc9007bca8805-hostname*
2. 调用 hypervisor 的 **Capabilities**，判断是否支持 host 文件系统共享特性（QEMU 场景下支持）
   - 如果不支持，则通过文件拷贝实现共享
     1. 校验 src 是否存在。如果 src 非常规文件，则不做处理（这里并未视为错误，而是作为一种局限性将其忽略）
     2. 调用 agent 的 **copyFile**，将 src 文件拷贝至 sandbox 的 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/containers/\<shareFile\> 文件位置
   - 如果支持，则通过文件挂载实现共享
     1. 如果挂载为读写属性，则调用 **bindMount**，将 src 以读写和 MS_PRIVATE 的属性绑定挂载到 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<shareFile\>
     2. 否则，调用 **bindMount**，将 src 以只读和 MS_PRIVATE 的属性绑定挂载到 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/private/\<shareFile\>，进而调用 **bindMount**，将 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/private/\<shareFile\> 以读写和 MS_PRIVATE 的属性绑定挂载到 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<shareFile\><br>*对于只读挂载，bindMount 重新挂载事件不会传播到挂载子树，并且它也不会出现在 virtiofsd 独立挂载命名空间中*
     3. 设置挂载信息的 host 侧路径为 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<shareFile\>

## UnshareFile

**移除 host/guest 共享文件的挂载点**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/fs_share_linux.go#L296)

1. 移除挂载信息的 host 侧挂载点
2. 如果挂载类型为 bind，校验 host 侧挂载点文件是否存在。如果为常规文件且为空，则直接删除；如果为目录，则移除目录

## ShareRootFilesystem

**创建 guest 中容器 rootfs 共享挂载**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/fs_share_linux.go#L373)

1. 如果 rootfs 类型为 fuse.nydus-overlayfs
   1. 通过 virtiofsd 挂载 /rafs/\<containerID\>/lowerdir 目录至 guest 中
   2. 创建 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs 目录
   3. 调用 **bindMount**，将 rootfs 挂载参数中的 snapshotdir 目录以只读和 MS_SLAVE 的属性绑定挂载到 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/snapshotdir
   4. 挂载类型仍为联合文件系统的 overlay 形式
2. 如果 rootfs 类型不是 fuse.nydus-overlayfs，并且是基于块设备的 rootfs
   1. 调用 devManager 的 **GetDeviceByID**，根据容器状态中的信息获取到设备信息
   2. 挂载类型视 [hypervisor].block_device_driver 而定
   3. 创建 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs 目录
3. 对于传统的 rootfs（也就是非 fuse.nydus-overlayfs 类型，并且不是基于块设备的），调用 **bindMount**，以读写和 MS_PRIVATE 的属性将 rootfs（例如 /run/containerd/io.containerd.runtime.v2.task/k8s.io/\<containerID\>/rootfs）绑定挂载到 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs<br>*这个目录本身为共享目录，因此无需告知 agent 去挂载，在 host 侧挂载后会自动在 guest 中出现*
4. 无论以上哪种挂载形式，最终 guest 中容器 rootfs 的挂载点均为 <XDG_RUNTIME_DIR>/run/kata-containers/shared/containers/\<containerID\>/rootfs

## UnshareRootFilesystem

**移除 guest 中容器 rootfs 共享挂载**

[source code](https://github.com/kata-containers/kata-containers/blob/3.0.0/src/runtime/virtcontainers/fs_share_linux.go#L455)

1. 如果 rootfs 类型为 fuse.nydus-overlayfs
   1. 通过 virtiofsd 移除 guest 中的 /rafs/\<containerID\>/lowerdir 挂载点
   2. 移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/snapshotdir 挂载点，并删除目录
   3. 移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs 目录
2. 如果 rootfs 类型不是 fuse.nydus-overlayfs，则移除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\>/rootfs 挂载点，并删除目录
3. 删除 \<XDG_RUNTIME_DIR\>/run/kata-containers/shared/sandboxes/\<sandboxID\>/mounts/\<containerID\> 目录
