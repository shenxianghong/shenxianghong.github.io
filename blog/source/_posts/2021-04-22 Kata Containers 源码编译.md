---
title: "「 Kata Containers 」源码编译"
excerpt: "基于源码在 x86 和 arm64 架构下容器化编译 Kata Containers 的参考流程"
cover: https://picsum.photos/0?sig=20210422
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2021-04-22
toc: true
categories:
- Container Runtime
tag:
- Kata Containers
---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **2.4.3**

# Requirement

这里采用 ubuntu:18.04 容器化编译，各依赖版本参考[版本说明](https://github.com/kata-containers/kata-containers/blob/2.4.3/versions.yaml)

```shell
$ docker run --privileged -dit -v /sys/fs/cgroup:/sys/fs/cgroup:ro -v /dev:/dev --name kata-build ubuntu:18.04
$ docker exec -it kata-build bash

# 可选的 TARGET_ARCH 有 amd64 和 arm64
$ export TARGET_ARCH=amd64
$ export GOPATH=/root/go
$ export GOPROXY=https://proxy.golang.com.cn,direct
$ export https_proxy=http://10.52.17.42:7890

$ mkdir -p $GOPATH/bin
$ mkdir -p /etc/docker
```

## Dependence

- 软件包

  ```shell
  $ apt-get update && apt-get -y install git wget make curl gcc xz-utils sudo flex bison bc python3 ninja-build pkg-config libglib2.0-dev librbd-dev libseccomp-dev libpixman-1-dev apt-utils libcap-ng-dev cpio libpmem-dev libelf-dev
  ```

- Golang 1.16.10 - 1.17.3

  ```shell
  $ wget https://go.dev/dl/go1.16.10.linux-$TARGET_ARCH.tar.gz && tar -C /usr/local -zxvf go1.16.10.linux-$TARGET_ARCH.tar.gz && cp /usr/local/go/bin/go /usr/bin/go
  ```

- Rust （1.58.1，仅在手动编译 kata-agent 组件时需要）

  ```shell
  $ curl https://sh.rustup.rs -sSf | sh
  $ source $HOME/.cargo/env
  $ rustup override set 1.58.1
  $ export ARCH=$(uname -m)
  $ if [ "$ARCH" = "ppc64le" -o "$ARCH" = "s390x" ]; then export LIBC=gnu; else export LIBC=musl; fi
  $ [ ${ARCH} == "ppc64le" ] && export ARCH=powerpc64le
  $ rustup target add ${ARCH}-unknown-linux-${LIBC}
  ```

- yq 3.4.1

  ```shell
  $ wget https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_$TARGET_ARCH && chmod +x yq_linux_$TARGET_ARCH && mv yq_linux_$TARGET_ARCH $GOPATH/bin/yq && cp $GOPATH/bin/yq /usr/bin/
  ```

- docker

  ```shell
  $ curl -sSL https://get.docker.com/ | sh
  $ cat > /etc/docker/daemon.json << EOF
  {
    "storage-driver": "vfs"
  }
  EOF
  $ service docker start
  ```

## Source Code

- kata-containers 2.4.3

  ```shell
  $ GO111MODULE=off go get -d -u github.com/kata-containers/kata-containers
  $ cd $GOPATH/src/github.com/kata-containers/kata-containers
  $ git checkout 2.4.3
  ```
  
- tests 2.4.3（仅在编译 UEFI ROM 时需要）

  ```shell
  $ GO111MODULE=off go get -d github.com/kata-containers/tests
  $ cd $GOPATH/src/github.com/kata-containers/tests
  $ git checkout 2.4.3
  ```

- qemu（x86 下为 v6.2.0，arm64 下为 v6.1.0，仅在编译 QEMU 时需要）

  ```shell
  $ GO111MODULE=off go get -d github.com/qemu/qemu
  $ cd ${GOPATH}/src/github.com/qemu/qemu
  $ git checkout v6.2.0
  ```

# Kata Containers

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/src/runtime
$ make && sudo -E PATH=$PATH make install
```

**编译结果**

- /usr/local/bin/containerd-shim-kata-v2
- /usr/local/bin/kata-collect-data.sh
- /usr/local/bin/kata-monitor
- /usr/local/bin/kata-runtime
- /usr/share/defaults/kata-containers/configuration.toml

# Image

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder

# 根据社区 release 中所推荐对应架构所使用的 image 发行版，分别设置 rootfs 和 initrd 镜像，这里以 x86 架构为例
$ ./rootfs-builder/rootfs.sh -l
alpine
centos
clearlinux
debian
ubuntu

# x86 下推荐 clearlinux，arm64 下推荐 ubuntu
$ export rootfsdistro=clearlinux
# x86 和 arm64 下均推荐 alpine
$ export initrddistro=alpine
```

**编译 Kata Agent（可选）**

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/src/agent && make

# 默认情况下，Kata Agent 是使用 seccomp 功能构建的。如果要构建没有 seccomp 功能的 Kata Agent，则需要使用 SECCOMP=no 运行 make
$ make -C $GOPATH/src/github.com/kata-containers/kata-containers/src/agent SECCOMP=no

# 如果在配置文件中启用了 seccomp 但构建了没有 seccomp 功能的 Kata Agent，则 runtime 会保守地退出并显示一条错误消息
```

## rootfs

**创建镜像文件系统**

```shell
$ export ROOTFS_DIR=${GOPATH}/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder/rootfs
$ sudo rm -rf ${ROOTFS_DIR}
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder
$ script -fec 'sudo -E GOPATH=$GOPATH USE_DOCKER=true ./rootfs.sh ${rootfsdistro}'
```

**添加 Kata Agent**

*仅在 Kata Agent 定制化后添加*

```shell
$ sudo install -o root -g root -m 0550 -t ${ROOTFS_DIR}/usr/bin ../../../src/agent/target/x86_64-unknown-linux-musl/release/kata-agent
$ sudo install -o root -g root -m 0440 ../../../src/agent/kata-agent.service ${ROOTFS_DIR}/usr/lib/systemd/system/
$ sudo install -o root -g root -m 0440 ../../../src/agent/kata-containers.target ${ROOTFS_DIR}/usr/lib/systemd/system/
```

**构建镜像**

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/image-builder
$ script -fec 'sudo -E USE_DOCKER=true ./image_builder.sh ${ROOTFS_DIR}'
```

**安装镜像**

```shell
$ commit=$(git log --format=%h -1 HEAD)
$ date=$(date +%Y-%m-%d-%T.%N%z)
$ image="kata-containers-${date}-${commit}"
$ sudo install -o root -g root -m 0640 -D kata-containers.img "/usr/share/kata-containers/${image}"
$ (cd /usr/share/kata-containers && sudo ln -sf "$image" kata-containers.img)
```

**编译结果**

- /usr/share/kata-containers/kata-containers-\<date\>
- /usr/share/kata-containers/kata-containers.img

## initrd

**创建镜像文件系统**

```shell
$ export ROOTFS_DIR="${GOPATH}/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder/rootfs"
$ sudo rm -rf ${ROOTFS_DIR}
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/rootfs-builder
$ script -fec 'sudo -E GOPATH=$GOPATH AGENT_INIT=yes USE_DOCKER=true ./rootfs.sh ${initrddistro}'
```

**添加 Kata Agent**

*仅在 Kata Agent 定制化后添加*

```shell
$ sudo install -o root -g root -m 0550 -T ../../../src/agent/target/${ARCH}-unknown-linux-${LIBC}/release/kata-agent ${ROOTFS_DIR}/sbin/init
```

**构建镜像**

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/osbuilder/initrd-builder
$ script -fec 'sudo -E AGENT_INIT=yes USE_DOCKER=true ./initrd_builder.sh ${ROOTFS_DIR}'
```

**安装镜像**

```shell
$ commit=$(git log --format=%h -1 HEAD)
$ date=$(date +%Y-%m-%d-%T.%N%z)
$ image="kata-containers-initrd-${date}-${commit}"
$ sudo install -o root -g root -m 0640 -D kata-containers-initrd.img "/usr/share/kata-containers/${image}"
$ (cd /usr/share/kata-containers && sudo ln -sf "$image" kata-containers-initrd.img)
```

**编译结果**

- /usr/share/kata-containers/kata-containers-initrd-\<date\>
- /usr/share/kata-containers/kata-containers-initrd.img

# Hypervisor

## QEMU

```shell
$ qemu_directory=${GOPATH}/src/github.com/qemu/qemu
$ packaging_dir="${GOPATH}/src/github.com/kata-containers/kata-containers/tools/packaging"
$ cd $qemu_directory
# 根据架构的 QEMU，应用对应版本的 patch
$ $packaging_dir/scripts/apply_patches.sh $packaging_dir/qemu/patches/6.2.x/
# 本地 commit 去除 dirty
$ git config --global user.email kata@kata.com
$ git config --global user.name kata
$ git commit -am "update"

$ $packaging_dir/scripts/configure-hypervisor.sh kata-qemu > kata.cfg
$ eval ./configure "$(cat kata.cfg)"
$ make -j $(nproc) 
$ sudo -E make install
```

**编译结果**

- /usr/bin/qemu-system-\<arch\>
- /usr/libexec/kata-qemu/virtiofsd
- /usr/share/kata-qemu/qemu/*

# Kernel

```shell
$ cd $GOPATH/src/github.com/kata-containers/kata-containers/tools/packaging/kernel
```

**x86 操作**

```shell
# x86 环境下删除 arm-experimental 中的 patch 文件，避免误 patch
$ rm -rf patches/5.15.x/arm-experimental/

$ ./build-kernel.sh setup
$ ./build-kernel.sh build
$ ./build-kernel.sh install
```

**arm64 操作**

```shell
# 重复 patch 导致流程异常，注释即可
$ sed -i "377s/^/#/" build-kernel.sh
$ ./build-kernel.sh -a aarch64 -E -d setup
$ ./build-kernel.sh -a aarch64 -E -d build
$ ./build-kernel.sh -a aarch64 -E -d install
```

**编译结果**

- /usr/share/kata-containers/config-5.15.26
- /usr/share/kata-containers/vmlinux.container
- /usr/share/kata-containers/vmlinux-5.15.26-90
- /usr/share/kata-containers/vmlinuz.container
- /usr/share/kata-containers/vmlinuz-5.15.26-90

# UEFI ROM

*UEFI ROM 仅在 arm64 环境下需要，用于设备热插拔*

```shell
$ cd $GOPATH/src/github.com/kata-containers/tests
$ .ci/aarch64/install_rom_aarch64.sh
```

**编译结果**

- /usr/share/kata-containers/kata-flash0.img
- /usr/share/kata-containers/kata-flash1.img
