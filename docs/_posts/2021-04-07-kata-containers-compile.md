---
layout: post
title:  "[ Kata Containers ] 2. Build from source"
date:   2021-04-07
excerpt: "Kata Containers 源码编译"
project: true
tag:
- Cloud Native
- Kubernetes
- Kata Containers
- Container Runtime
comments: false
---

Kata Containers 目前在 centos8 支持 yum 直接安装，除此之外还支持 snap，kata-manager 等多种安装方式，源码编译仅做记录与流程参考。

# 环境要求

Kata Containers 需要 CPU 支持以下之一的虚拟化技术

- Intel VT-x technology.
- ARM Hyp mode (virtualization extension).
- IBM Power Systems.
- IBM Z mainframes.

# Kata 组件编译

## 版本依赖

各依赖版本参考[版本说明](https://github.com/kata-containers/kata-containers/blob/main/versions.yaml)

- `golang`（最小支持版本：1.11.10  最新支持版本：1.14.4）
- `rust`（最小支持版本：1.38.0   最新支持版本：1.47.0，注：在手动编译 `kata-agent`时需要）
- `make`
- `gcc`（版本需要至少升级到 4.9）

## rust 安装（可选）

```shell
$ 自动识别 OS 并安装最新稳定版本的 rust 工具链，包括 rustup，rustc, cargo 等
# curl https://sh.rustup.rs -sSf | sh
# source $HOME/.cargo/env
 
$ 下载并安装指定版本
# rustup override set 1.47.0
 
$ 必须安装 musl 版本，kata-agent 构建使用
# arch=$(uname -m)
# rustup target add "${arch}-unknown-linux-musl"
# sudo ln -s /usr/bin/g++ /bin/musl-g++
 
$ 确认版本
# rustc --version
rustc 1.47.0 (18bf6b4f0 2020-10-07)
```

## 编译组件

编译结果一共有

> Summary:
>   ``destination ``install` `path (DESTDIR) : /
>   ``binary installation path (BINDIR) : ``/usr/local/bin
>   ``binaries to ``install` `:
>    ``- ``/usr/local/bin/kata-runtime
>    ``- ``/usr/local/bin/containerd-shim-kata-v2
>    ``- ``/usr/local/bin/kata-monitor
>    ``- ``/usr/libexec/kata-containers/kata-netmon
>    ``- ``/usr/local/bin/data/kata-collect-data``.sh
>   ``configs to ``install` `(CONFIGS) :
>    ``- cli``/config/configuration-acrn``.toml
>    ``- cli``/config/configuration-clh``.toml
>    ``- cli``/config/configuration-fc``.toml
>    ``- cli``/config/configuration-qemu``.toml
>   ``install` `paths (CONFIG_PATHS) :
>    ``- ``/usr/share/defaults/kata-containers/configuration-acrn``.toml
>    ``- ``/usr/share/defaults/kata-containers/configuration-clh``.toml
>    ``- ``/usr/share/defaults/kata-containers/configuration-fc``.toml
>    ``- ``/usr/share/defaults/kata-containers/configuration-qemu``.toml
>   ``alternate config paths (SYSCONFIG_PATHS) :
>    ``- ``/etc/kata-containers/configuration-acrn``.toml
>    ``- ``/etc/kata-containers/configuration-clh``.toml
>    ``- ``/etc/kata-containers/configuration-fc``.toml
>    ``- ``/etc/kata-containers/configuration-qemu``.toml
>   ``default ``install` `path ``for` `qemu (CONFIG_PATH) : ``/usr/share/defaults/kata-containers/configuration``.toml
>   ``default alternate config path (SYSCONFIG) : ``/etc/kata-containers/configuration``.toml
>   ``qemu hypervisor path (QEMUPATH) : ``/usr/bin/qemu-system-x86_64
>   ``cloud-hypervisor hypervisor path (CLHPATH) : ``/usr/bin/cloud-hypervisor
>   ``firecracker hypervisor path (FCPATH) : ``/usr/bin/firecracker
>   ``acrn hypervisor path (ACRNPATH) : ``/usr/bin/acrn-dm
>   ``assets path (PKGDATADIR) : ``/usr/share/kata-containers
>   ``shim path (PKGLIBEXECDIR) : ``/usr/libexec/kata-containers

```shell
$ go get -d -u github.com /kata-containers/kata-containers
$ cd $GOPATH/src/github.com /kata-containers/kata-containers/src/runtime
$ git checkout stable-2.1
$ make && sudo -E PATH=$PATH make install
<skip ...>
	 INSTALL  install-scripts
     INSTALL  install-completions
     INSTALL  install-configs
     INSTALL  install-configs
     INSTALL  install-bin
     INSTALL  install-containerd-shim-v2
     INSTALL  install-monitor
     INSTALL  install-bin-libexec
```

```shell
$ kata-runtime -version
 ``kata-runtime : 2.1.0
  ``commit  : 645e950b8e0e238886adbff695a793126afb584f
  ``OCI specs: 1.0.1-dev
```

# 其余组件编译

## 构建引导镜像

镜像的构建采用官方提供的脚本，构建过程可以借助于 docker，要确保 docker 服务正在运行并且 docker runtime 是 runC。

### 构建 rootfs 镜像

#### 构建 osbuilder

支持的 distro 有 alpine，centos，clearlinux，debian，euleros，fedora，suse，ubuntu

```shell
$ export ROOTFS_DIR=${GOPATH}/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder/rootfs
$ sudo rm -rf ${ROOTFS_DIR}
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder
$ export distro=centos
$ script -fec 'sudo -E GOPATH=$GOPATH USE_DOCKER=true SECCOMP=no ./rootfs.sh ${distro}'
<skip ...>
[OK] Agent installed
INFO: Check init is installed
[OK] init is installed
INFO: Create /etc/resolv.conf file in rootfs if not exist
INFO: Creating summary file
INFO: Created summary file '/var/lib/osbuilder/osbuilder.yaml' inside rootfs
Script done, file is typescript

$ docker images
REPOSITORY                   TAG       IMAGE ID       CREATED          SIZE
centos-rootfs-osbuilder      latest    18bdd0cb48ea   17 minutes ago   2.1GB
registry.centos.org/centos   7         a1bb412b2847   5 months ago     202MB
```

#### 构建目标镜像

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/image-builder
$ script -fec 'sudo -E USE_DOCKER=true ./image_builder.sh ${ROOTFS_DIR}'
>>>
<skip ...>
OK!
1+0 records in
1+0 records out
2097152 bytes (2.1 MB, 2.0 MiB) copied, 0.00793108 s, 264 MB/s
1044480+0 records in
1044480+0 records out
534773760 bytes (535 MB, 510 MiB) copied, 3.28126 s, 163 MB/s
Script done, file is typescript
 
$ docker images
>>>
REPOSITORY                          TAG       IMAGE ID       CREATED          SIZE
image-builder-osbuilder             latest    87910915e7c4   34 seconds ago   568MB
centos-rootfs-osbuilder             latest    18bdd0cb48ea   26 minutes ago   2.1GB
registry.fedoraproject.org/fedora   latest    5f05951e2065   12 days ago      180MB
registry.centos.org/centos          7         a1bb412b2847   5 months ago     202MB
```

#### 安装镜像

```shell
$ commit=$(git log --format=%h -1 HEAD)
$ date=$(date +%Y-%m-%d-%T.%N%z)
$ image="kata-containers-${date}-${commit}"
$ sudo install -o root -g root -m 0640 -D kata-containers.img "/usr/share/kata-containers/${image}"
$ (cd /usr/share/kata-containers && sudo ln -sf "$image" kata-containers.img)
 
$ ll /usr/share/kata-containers/
-rw-r----- 1 root root 536870912 May  7 10:57 kata-containers-2021-05-07-10:57:42.903708982+0800-3e81373
lrwxrwxrwx 1 root root        58 May  7 10:57 kata-containers.img -> kata-containers-2021-05-07-10:57:42.903708982+0800-3e81373
```

#### 构建 initrd 镜像

#### 构建 osbuilder

支持的 distro 有 alpine，centos，clearlinux，euleros，fedora；
AGENT_INIT 参数表示是否使用 kata-agent 作为 guest kernel 的 init 进程，**创建 initrd 镜像时，AGENT_INIT 必须为 yes**

```shell
$ export ROOTFS_DIR="${GOPATH}/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder/rootfs"
$ sudo rm -rf ${ROOTFS_DIR}
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder
$ export distro=centos
$ script -fec 'sudo -E GOPATH=$GOPATH AGENT_INIT=yes USE_DOCKER=true SECCOMP=no ./rootfs.sh ${distro}'
[OK] Agent installed
INFO: Install /root/go/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder/rootfs/usr/bin/kata-agent as init process
[OK] Agent is installed as init process
INFO: Check init is installed
[OK] init is installed
INFO: Create /etc/resolv.conf file in rootfs if not exist
INFO: Creating summary file
INFO: Created summary file '/var/lib/osbuilder/osbuilder.yaml' inside rootfs
Script done, file is typescript
 
$ docker images
REPOSITORY                   TAG       IMAGE ID       CREATED          SIZE
centos-rootfs-osbuilder      latest    5f323e966873   22 minutes ago   2.1GB
registry.centos.org/centos   7         a1bb412b2847   5 months ago     202MB
```

#### 构建镜像

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/initrd-builder
$ script -fec 'sudo -E AGENT_INIT=yes USE_DOCKER=true ./initrd_builder.sh ${ROOTFS_DIR}'
Script started, file is typescript
[OK] init is installed
[OK] Agent is installed
INFO: Creating /root/go/src/github.com/kata-containers/kata-containers/tools/osbuilder/initrd-builder/kata-containers-initrd.img based on rootfs at /root/go/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder/rootfs
645047 blocks
Script done, file is typescript
 
$ docker images
REPOSITORY                   TAG       IMAGE ID       CREATED          SIZE
centos-rootfs-osbuilder      latest    5f323e966873   22 minutes ago   2.1GB
registry.centos.org/centos   7         a1bb412b2847   5 months ago     202MB
```

#### 安装镜像

```shell
$ commit=$(git log --format=%h -1 HEAD)
$ date=$(date +%Y-%m-%d-%T.%N%z)
$ image="kata-containers-initrd-${date}-${commit}"
$ sudo install -o root -g root -m 0640 -D kata-containers-initrd.img "/usr/share/kata-containers/${image}"
$ (cd /usr/share/kata-containers && sudo ln -sf "$image" kata-containers-initrd.img)
 
$ ll /usr/share/kata-containers/
-rw-r----- 1 root root 109075781 May  7 12:19 kata-containers-initrd-2021-05-07-12:18:52.216347708+0800-3e81373
lrwxrwxrwx 1 root root        65 May  7 12:19 kata-containers-initrd.img -> kata-containers-initrd-2021-05-07-12:18:52.216347708+0800-3e81373
```

### 配置引导镜像

Kata Containers 必须要指定系统引导镜像，可选有 initrd（10MB+）和 rootfs（100MB+）

rootfs：`image = "/usr/share/kata-containers/kata-containers.img"`
initrd：`initrd = "/usr/share/kata-containers/kata-containers-initrd.img"`

```toml
# /usr/share/defaults/kata-containers/configuration.toml
[hypervisor.qemu]
path = "/usr/bin/qemu-system-x86_64"
kernel = "/usr/share/kata-containers/vmlinux.container"
image = "/usr/share/kata-containers/kata-containers.img"
machine_type = "pc"
```

## 编译 Kata 容器内核

### 安装依赖

```shell
$ yum -y install flex bison bc elfutils-libelf-devel patch
```

### 准备配置

生成的配置文件位于 $GOPATH/src/github.com/kata-containers/kata-containers/tools/packaging/kernel/configs/fragments/x86_64/.config

```
$ go get -d -u github.com/kata-containers/kata-containers
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/packaging/kernel
$ ./build-kernel.sh setup
>>>
~/go/src/github.com/kata-containers/tests ~/go/src/github.com/kata-containers/kata-containers/tools/packaging/kernel
~/go/src/github.com/kata-containers/kata-containers/tools/packaging/kernel
INFO: Config version: 85
INFO: Kernel version: 5.10.25
<skip ...>
Kernel source ready: /root/go/src/github.com/kata-containers/kata-containers/tools/packaging/kernel/kata-linux-5.10.25-85
```

### 编译内核

*如果 gcc 版本低于 4.9，则要升级，升级方法仅供参考*

```shell
$ yum -y install centos-release-scl
$ yum -y install devtoolset-9-gcc*
# 只在当前的 terminal 生效
$ scl enable devtoolset-9 bash
 
$ gcc --version
gcc (GCC) 9.3.1 20200408 (Red Hat 9.3.1-2)
Copyright (C) 2019 Free Software Foundation, Inc.
This is free software; see the source for copying conditions.  There is NO
warranty; not even for MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE
```

```shell
$ ./build-kernel.sh build
<skip ...>
  LD      arch/x86/boot/setup.elf
  OBJCOPY arch/x86/boot/setup.bin
  BUILD   arch/x86/boot/bzImage
Kernel: arch/x86/boot/bzImage is ready  (#1)
```

### 安装内核

```shell
$ ./build-kernel.sh install
~/go/src/github.com/kata-containers/tests ~/go/src/github.com/kata-containers/kata-containers/tools/packaging/kernel
~/go/src/github.com/kata-containers/kata-containers/tools/packaging/kernel
INFO: Config version: 84
INFO: Kernel version: 5.10.25
  DESCEND  objtool
  CALL    scripts/atomic/check-atomics.sh
  CALL    scripts/checksyscalls.sh
  CHK     include/generated/compile.h
Kernel: arch/x86/boot/bzImage is ready  (#1)
lrwxrwxrwx 1 root root 18 May  7 09:39 /usr/share/kata-containers/vmlinux.container -> vmlinux-5.10.25-84
lrwxrwxrwx 1 root root 18 May  7 09:39 /usr/share/kata-containers/vmlinuz.container -> vmlinuz-5.10.25-84
```

## Qemu 编译

按照[版本说明](https://github.com/kata-containers/kata-containers/blob/main/versions.yaml)中关于 QEMU 的版本信息，Kata Containers 2.1 版本采用 QEMU 5.2.0 版本。Kata Containers 社区基于上游 QEMU 代码进行了定制化 patch，补丁位于 https://github.com/kata-containers/kata-containers/tree/main/tools/packaging/qemu/patches。

### 安装依赖

配置 ceph 源（非必须）

```shell

$ cat <<EOF > /etc/yum.repos.d/ceph.repo
[ceph]
name=Ceph packages for \$basearch
baseurl=https://download.ceph.com/rpm-nautilus/el7/\$basearch
enabled=1
priority=2
gpgcheck=1
gpgkey=https://download.ceph.com/keys/release.asc
 
[ceph-noarch]
name=Ceph noarch packages
baseurl=https://download.ceph.com/rpm-nautilus/el7//noarch
enabled=1
priority=2
gpgcheck=1
gpgkey=https://download.ceph.com/keys/release.asc
 
[ceph-source]
name=Ceph source packages
baseurl=https://download.ceph.com/rpm-nautilus/el7/SRPMS
enabled=0
priority=2
gpgcheck=1
gpgkey=https://download.ceph.com/keys/release.asc
EOF
```

```shell
# 注意 python 要高于 python 3.5
$ yum -y install bc python3 libseccomp-devel libcap-ng-devel glib2-devel librbd-devel libpmem-devel pixman-devel bzip2 zlib-devel ninja-build

# ubuntu 环境下
apt-get install -y librbd-dev
```

### Qemu 代码准备

```shell
$ wget https://download.qemu.org/qemu-5.2.0.tar.xz
$ tar xvf qemu-5.2.0.tar.xz
```

### 编译

生成配置参数 kata.cfg，加载 patch 补丁

```shell
$ cd qemu-5.2.0
$ ${GOPATH}/src/github.com/kata-containers/kata-containers/tools/packaging/scripts/configure-hypervisor.sh kata-qemu > kata.cfg
$ packaging_dir=$GOPATH/src/github.com/kata-containers/kata-containers/tools/packaging
$ $packaging_dir/scripts/apply_patches.sh $packaging_dir/qemu/patches/5.2.x/
$ eval ./configure "$(cat kata.cfg)"
$ make -j $(nproc)
$ sudo -E make install
```

kata.cfg 示例

```
--disable-sheepdog --disable-live-block-migration --disable-brlapi --disable-docs --disable-curses --disable-gtk --disable-opengl --disable-sdl --disable-spice --disable-vte --disable-vnc --disable-vnc-jpeg --disable-vnc-png --disable-vnc-sasl --disable-auth-pam --disable-fdt --disable-glusterfs --disable-libiscsi --disable-libnfs --disable-libssh --disable-bzip2 --disable-lzo --disable-snappy --disable-tpm --disable-slirp --disable-libusb --disable-usb-redir --disable-tcg --disable-debug-tcg --disable-tcg-interpreter --disable-qom-cast-debug --disable-tcmalloc --disable-curl --disable-rdma --disable-tools --disable-bsd-user --disable-linux-user --disable-sparse --disable-vde --disable-xfsctl --disable-libxml2 --disable-nettle --disable-xen --disable-linux-aio --disable-capstone --disable-virglrenderer --disable-replication --disable-smartcard --disable-guest-agent --disable-guest-agent-msi --disable-vvfat --disable-vdi --disable-qed --disable-qcow1 --disable-bochs --disable-cloop --disable-dmg --disable-parallels --enable-kvm --enable-vhost-net --enable-rbd --enable-virtfs --enable-attr --enable-cap-ng --enable-seccomp --enable-avx2 --enable-avx512f --enable-libpmem --enable-malloc-trim --target-list=x86_64-softmmu --extra-cflags=" -O3 -falign-functions=32 -D_FORTIFY_SOURCE=2 -fPIE" --extra-ldflags=" -pie -z noexecstack -z relro -z now" --prefix=/usr --libdir=/usr/lib/kata-qemu --libexecdir=/usr/libexec/kata-qemu --datadir=/usr/share/kata-qemu
```

# 自定义配置

默认情况下，Kata Containers 会从 `/etc/kata-containers/configuration.toml` 和 `/usr/share/defaults/kata-containers/configuration.toml` 两处获取配置文件，并且前者的优先级高于后者。

```shell
$ kata-runtime --kata-show-default-config-paths
>>>
/etc/kata-containers/configuration.toml
/usr/share/defaults/kata-containers/configuration.toml
```

也可以通过手动指定配置文件的方式。如果指定多个配置文件的路径，Kata Containers 会依次查找，直到找到第一个存在的配置为止

```shell
$ kata-runtime --kata-config=/some/where/configuration.toml ...
```

# 检查工作

## check

通过 `kata-runtime check` 的输出结果可以看到当前环境是否可以运行 Kata Containers

```shell
$ kata-runtime check
WARN[0000] Not running network checks as super user      arch=arm64 name=kata-runtime pid=3815453 source=runtime
System is capable of running Kata Containers
System can currently create Kata Containers
```

## Env 

通过 `kata-runtime kata-env` 的输出信息可以判断版本是否符合预期等

```shell
$ kata-runtime kata-env
[Meta]
  Version = "1.0.25"

[Runtime]
  Debug = false
  Trace = false
  DisableGuestSeccomp = true
  DisableNewNetNs = false
  SandboxCgroupOnly = true
  Path = "/usr/bin/kata-runtime"
  [Runtime.Version]
    OCI = "1.0.1-dev"
    [Runtime.Version.Version]
      Semver = "2.1.1"
      Major = 2
      Minor = 1
      Patch = 1
      Commit = "0e2be438bdd6d213ac4a3d7d300a5757c4137799"
  [Runtime.Config]
    Path = "/etc/kata-containers/configuration.toml"

[Hypervisor]
  MachineType = "pc"
  Version = "QEMU emulator version 5.2.0\nCopyright (c) 2003-2020 Fabrice Bellard and the QEMU Project developers"
  Path = "/opt/kata/bin/qemu-system-x86_64"
  BlockDeviceDriver = "virtio-scsi"
  EntropySource = "/dev/urandom"
  SharedFS = "virtio-fs"
  VirtioFSDaemon = "/opt/kata/libexec/kata-qemu/virtiofsd"
  Msize9p = 8192
  MemorySlots = 10
  PCIeRootPort = 0
  HotplugVFIOOnRootBus = false
  Debug = false

[Image]
  Path = ""

[Kernel]
  Path = "/opt/kata/share/kata-containers/vmlinux.container"
  Parameters = "scsi_mod.scan=none agent.debug_console agent.debug_console_vport=1026"

[Initrd]
  Path = "/opt/kata/share/kata-containers/kata-containers-initrd.img"

[Agent]
  Debug = false
  Trace = false
  TraceMode = ""
  TraceType = ""

[Host]
  Kernel = "3.10.0-957.10.4.el7.x86_64"
  Architecture = "amd64"
  VMContainerCapable = true
  SupportVSocks = true
  [Host.Distro]
    Name = "ArcherOS HCI"
    Version = "5.0"
  [Host.CPU]
    Vendor = "GenuineIntel"
    Model = "Intel(R) Xeon(R) CPU E5-2650 v4 @ 2.20GHz"
    CPUs = 48
  [Host.Memory]
    Total = 131798792
    Free = 88588112
    Available = 91340708

[Netmon]
  Path = "/opt/kata/libexec/kata-containers/kata-netmon"
  Debug = false
  Enable = false
  [Netmon.Version]
    Semver = "2.1.1"
    Major = 2
    Minor = 1
    Patch = 1
    Commit = "<<unknown>>"
```

# FAQ

> $ kata-runtime kata-check
> /usr/share/defaults/kata-containers/configuration-qemu.toml: file /usr/bin/qemu-system-x86_64 does not exist

host 中未安装 Qemu

> $ kata-runtime kata-check
> /usr/share/defaults/kata-containers/configuration-qemu.toml: file /usr/share/kata-containers/vmlinux.container does not exist

未安装 Kata 容器内核

> $ kata-runtime kata-check
> /usr/share/defaults/kata-containers/configuration-qemu.toml: Either initrd or image must be set to a valid path (initrd: initrd is not set) (image: file /usr/share/kata-containers/kata-containers.img does not exist)

host 未安装引导镜像

> $ kata-runtime kata-check
> /usr/share/defaults/kata-containers/configuration-qemu.toml: host system doesn't support vsock: stat /dev/vhost-vsock: no such file or directory

host 没有加载 `vhost-vsock` 内核模块，执行 `modprobe vhost_vsock`

> $ ./build-kernel.sh install
> cc1: error: -Werror=date-time: no option -Wdate-time

gcc 4.9 之后加入 -Wate-time 参数，[gcc 4.9 版本说明](https://gcc.gnu.org/gcc-4.9/changes.html)，需要升级 gcc

