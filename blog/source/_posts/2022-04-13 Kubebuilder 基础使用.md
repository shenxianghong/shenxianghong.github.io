---
title: "「 Kubebuilder 」基础使用"
excerpt: "使用 Kubebuilder 和 Code Generator 生成自定义的 K8s Operator 框架"
cover: https://picsum.photos/0?sig=20220413
thumbnail: https://storage.googleapis.com/kubebuilder-logos/icon-no-text.png
date: 2022-04-13
toc: true
categories:
- Scheduling & Orchestration
- Kubernetes
tag:
- Kubebuilder
---

<div align=center><img width="150" style="border: 0px" src="https://storage.googleapis.com/kubebuilder-logos/logo-single-line-01.png"></div>

------

> based on **v3.3.0**

# Overview

[Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) 是一个基于 CRD 来构建 Kubernetes API 的框架，可以使用 CRD 来构建 API、Controller 和 Admission Webhook。类似于 Ruby on Rails 和 SpringBoot 之类的 Web 开发框架，Kubebuilder 可以提高速度并降低开发人员管理的复杂性，以便在 Golang 中快速构建和发布 Kubernetes API。它建立在用于构建核心 Kubernetes API 的规范技术的基础之上，以提供减少样板和麻烦的简单抽象。

与 Kubebuilder 类似的项目还有 [Operator SDK](https://github.com/operator-framework/operator-sdk)，两者的区别可以参考 https://github.com/operator-framework/operator-sdk/issues/1758，目前两个社区的在做功能整合。

Where they differ is:

- Operator SDK also has support for Ansible and Helm operators, which make it easy to write operators without having to learn Go and if you already have experience with Ansible or Helm
- Operator SDK includes integrations with the Operator Lifecycle Manager (OLM), which is a key component of the Operator Framework that is important to Day 2 cluster operations, like managing a live upgrade of your operator.
- Operator SDK includes a scorecard subcommand that helps you understand if your operator follows best practices.
- Operator SDK includes an e2e testing framework that simplifies testing your operator against an actual cluster.
- Kubebuilder includes an envtest package that allows operator developers to run simple tests with a standalone etcd and apiserver.
- Kubebuilder scaffolds a Makefile to assist users in operator tasks (build, test, run, code generation, etc.); Operator SDK is currently using built-in subcommands. Each has pros and cons. The SDK team will likely be migrating to a Makefile-based approach in the future.
- Kubebuilder uses Kustomize to build deployment manifests; Operator SDK uses static files with placeholders.
- Kubebuilder has recently improved its support for admission and CRD conversion webhooks, which has not yet made it into SDK.

[Code Generator](https://github.com/kubernetes/code-generator) 是用于实现 Kubernetes 风格 API 类型的 Golang 代码生成器。可以利用该工程自动生成指定K8s 资源的 clientset、informers 和 listers API 接口，本身是位于 Kubernetes 项目中的工具，内部包含了大量的生成器，例如 client-gen，deepcopy-gen，informer-gen，lister-gen 等等。

- deepcopy-gen

  生成深度拷贝对象方法，例如 DeepCopy，DeepCopyObject，DeepCopyInto 等

- client-gen

  生成 clientSet 对象，为资源生成标准的操作方法，如 get，list，watch，create，update，patch 和 delete

- informer-gen

  生成 informer 对象，提供事件监听机制，如 AddFunc，UpdateFunc，DeleteFunc

- lister-gen

  生成 lister 对象，为 get 和 list 方法提供只读缓存层

Kubebuilder 与 Code Generator 都可以为 CRD 生成 Kubernetes API 相关代码，从代码生成层面来讲， 两者的区别在于：

- Kubebuilder 不会生成 informers、listers、clientsets，而 Code Generator 会
- Kubebuilder 会生成 Controller、Admission Webhooks，而 Code Generator 不会
- Kubebuilder 会生成 manifests yaml，而 Code Generator 不会
- Kubebuilder 还带有一些其他便利性设施

# Kubebuilder

## 环境要求

- [go](https://golang.org/dl/) version v1.15+ (kubebuilder v3.0 < v3.1).
- [go](https://golang.org/dl/) version v1.16+ (kubebuilder v3.1 < v3.3).
- [go](https://golang.org/dl/) version v1.17+ (kubebuilder v3.3+).
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

## 安装

```shell
$ curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
$ chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```

## 初始化

首先借助 Kubebuilder 初始化一个项目。

```shell
$ kubebuilder init --domain huayun.io --repo huayun.io/ake/android-operator --owner AKE --project-name android-operator

$ tree
.
├── config
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_auth_proxy_patch.yaml
│   │   └── manager_config_patch.yaml
│   ├── manager
│   │   ├── controller_manager_config.yaml
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   └── rbac
│       ├── auth_proxy_client_clusterrole.yaml
│       ├── auth_proxy_role_binding.yaml
│       ├── auth_proxy_role.yaml
│       ├── auth_proxy_service.yaml
│       ├── kustomization.yaml
│       ├── leader_election_role_binding.yaml
│       ├── leader_election_role.yaml
│       ├── role_binding.yaml
│       └── service_account.yaml
├── Dockerfile
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
├── main.go
├── Makefile
└── PROJECT

6 directories, 24 files
```

可选的 flags 包括

| 名称                    | 含义                                                         |
| ----------------------- | ------------------------------------------------------------ |
| --component-config      |                                                              |
| --domain                | group 的 domain 信息，默认为 my.domain                       |
| --fetch-deps            | 确保依赖已下载，默认为 true                                  |
| --license               | boilerplate.txt 中使用的 license，可选的有 apache2 和 none，默认为 apache2 |
| --owner                 | copyright 中的所有者名称                                     |
| --project-name          | 项目名称                                                     |
| --project-version       | 项目版本，默认为 3                                           |
| --repo                  | go module 中的 module 名称                                   |
| --skip-go-version-check | 如果指定，则跳过 Golang 版本检查                             |

## 生成 API

设置 Kubebuilder 开启 multigroup，即 api 支持多 group，例如 apps/policy，apps/admission 等。

```shell
$ kubebuilder edit --multigroup=true
```

可选的 flags 包括

| 名称         | 含义                    |
| ------------ | ----------------------- |
| --multigroup | 是否启用多 api group 组 |

生成 CRD API 对象。

```shell
$ kubebuilder create api --group android --version v1 --kind AnFile --controller --resource --namespaced --plural anfiles
$ kubebuilder create api --group android --version v1 --kind AnImage --controller --resource --namespaced --plural animages
```

可选的 flags 包括

| 名称         | 含义                                                         |
| ------------ | ------------------------------------------------------------ |
| --controller | 是否不询问默认生成 controller，默认为 true                   |
| --force      | 强制生成资源                                                 |
| --group      | 资源 group 组，例如 storage，events 等，最终会和 domain 一起作为 api group 信息 |
| --kind       | 资源 kind 信息，即资源名称，如 Pod，Service 等               |
| --make       | 生成文件后，是否执行一次 make generate，默认为 true          |
| --namespaced | 是否是命名空间级别的资源                                     |
| --plural     | 资源的复数信息                                               |
| --resource   | 是否不询问默认生成 resource，默认为 true                     |
| --version    | 资源版本信息，如 v1，v1beta1                                 |

最终的资源结构为

```yaml
apiVersion: android.huayun.io/v1
kind: AnImage
metadata:
  name: animage-sample
spec:
  # TODO(user): Add fields here
```

make manifest 会在 ./config/crd/bases 下根据 API 声明信息生成 CRD 的基础模板，在 API 变动后需要更新。

## 生成 webhook

TODO

## 目录结构

```shell
$ tree

.
├── apis
│   └── android
│       └── v1
│           ├── anfile_types.go
│           ├── animage_types.go
│           ├── groupversion_info.go
│           └── zz_generated.deepcopy.go
├── bin
│   └── controller-gen
├── config
│   ├── crd
│   │   ├── kustomization.yaml
│   │   ├── kustomizeconfig.yaml
│   │   └── patches
│   │       ├── cainjection_in_anfiles.yaml
│   │       ├── cainjection_in_animages.yaml
│   │       ├── webhook_in_anfiles.yaml
│   │       └── webhook_in_animages.yaml
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_auth_proxy_patch.yaml
│   │   └── manager_config_patch.yaml
│   ├── manager
│   │   ├── controller_manager_config.yaml
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── anfile_editor_role.yaml
│   │   ├── anfile_viewer_role.yaml
│   │   ├── animage_editor_role.yaml
│   │   ├── animage_viewer_role.yaml
│   │   ├── auth_proxy_client_clusterrole.yaml
│   │   ├── auth_proxy_role_binding.yaml
│   │   ├── auth_proxy_role.yaml
│   │   ├── auth_proxy_service.yaml
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   └── samples
│       ├── android_v1_anfile.yaml
│       └── android_v1_animage.yaml
├── controllers
│   └── android
│       ├── anfile_controller.go
│       ├── animage_controller.go
│       └── suite_test.go
├── Dockerfile
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
├── main.go
├── Makefile
└── PROJECT

15 directories, 44 files
```

## Markers

Kubebuilder 提供了众多 Markers，支持对 CRD 的校验，生成等操作，具体可以参考[官方文档](https://book.kubebuilder.io/reference/markers.html)。

# Code Generator

参考了 Kubernetes，Velero 等开源项目风格，将 apis 目录移至 pkg 层级下。

## 文件准备

### doc.go

在 pkg/apis/android/v1 目录下创建 doc.go 文件，参考内容如下

```go
// +k8s:deepcopy-gen=package

// Package v1 is the v1 version of the API.
// +groupName=android.huayun.io
package v1
```

- groupName 的名称为 \<group\>.\<domain\>

### register.go

在 pkg/apis/android/v1 目录下创建 register.go 文件，参考内容如下

```golang
package v1

import (
    "k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects.
var SchemeGroupVersion = GroupVersion

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
    return SchemeGroupVersion.WithResource(resource).GroupResource()
}
```

### \<CRD\>_types.go

以 pkg/apis/android/v1/animage_types.go 为例，新增以下 tag 信息

```go
/*
Copyright 2022 AKE.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AnImageSpec defines the desired state of AnImage
type AnImageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of AnImage. Edit animage_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// AnImageStatus defines the observed state of AnImage
type AnImageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AnImage is the Schema for the animages API
type AnImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnImageSpec   `json:"spec,omitempty"`
	Status AnImageStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// AnImageList contains a list of AnImage
type AnImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AnImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AnImage{}, &AnImageList{})
}
```

- 在 AnImage struct 上，新增 `+genclient`
- 在 AnImage 和 AnImageList struct 上，新增 `+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object`

### tools.go

该文件主要用于追踪构建阶段所需的依赖包，而非运行阶段，因此，文件位置、名称和内容等信息根据具体情况而定。

在 hack 目录下创建 tools.go 文件，参考内容如下

```go
//go:build tools
// +build tools

package tools

import _ "k8s.io/code-generator"
```

### update-codegen.sh

该文件用于调用 Code Generator 的 generate-groups.sh 脚本，生成代码。参考内容如下，逻辑参考脚本注释

```shell
#!/usr/bin/env bash
#
#Copyright 2022 AKE.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# this script expects to be run from the root of the android-operator repo.

# Corresponding to go mod init <module>.
MODULE=huayun.io/ake/android-operator
# Corresponding to kubebuilder create api --group <group> --version <version>.
GROUP_VERSION=android:v1

# Generated output package.
OUTPUT_PKG=pkg/generated
# API package.
APIS_PKG=pkg/apis

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

# Sync k8s.io/code-generator to go.mod
go mod tidy 

# Grab code-generator version from go.mod.
CODEGEN_VERSION=$(grep 'k8s.io/code-generator' go.mod | awk '{print $2}')
CODEGEN_PKG=$(go env GOPATH)/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION}

# Prepare code-generator.
if [[ ! -d ${CODEGEN_PKG} ]]; then
    echo "${CODEGEN_PKG} is missing. Running 'go mod download'."
    go mod download
fi

echo ">> Using ${CODEGEN_PKG}"

# code-generator does work with go.mod but makes assumptions about
# the project living in `$GOPATH/src`. To work around this and support
# any location; create a temporary directory, use this as an output
# base, and copy everything back once generated.
TEMP_DIR=$(mktemp -d)
cleanup() {
    echo ">> Removing ${TEMP_DIR}"
    rm -rf "${TEMP_DIR}"
}
trap "cleanup" EXIT SIGINT

echo ">> Temporary output directory ${TEMP_DIR}"

chmod +x "${CODEGEN_PKG}"/generate-groups.sh

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
#cd "${SCRIPT_ROOT}"
"${CODEGEN_PKG}"/generate-groups.sh \
  all \
  ${MODULE}/${OUTPUT_PKG} \
  ${MODULE}/${APIS_PKG} \
  ${GROUP_VERSION} \
  --output-base "${TEMP_DIR}" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

# Copy everything back.
cp -a "${TEMP_DIR}/${MODULE}/." "${SCRIPT_ROOT}/"
```

### Makefile

Makefile 中新增 .PHONY，集成构建流程。

```makefile
.PHONY: update-codegen
update-codegen: ## Generate code by code-generator
	bash ./hack/update-codegen.sh
```

## 代码生成

```shell
$ make update-codegen
```

## 文件结构

```shell
$ tree pkg/generated/

pkg/generated/
├── clientset
│   └── versioned
│       ├── clientset.go
│       ├── doc.go
│       ├── fake
│       │   ├── clientset_generated.go
│       │   ├── doc.go
│       │   └── register.go
│       ├── scheme
│       │   ├── doc.go
│       │   └── register.go
│       └── typed
│           └── android
│               └── v1
│                   ├── android_client.go
│                   ├── anfile.go
│                   ├── animage.go
│                   ├── doc.go
│                   ├── fake
│                   │   ├── doc.go
│                   │   ├── fake_android_client.go
│                   │   ├── fake_anfile.go
│                   │   └── fake_animage.go
│                   └── generated_expansion.go
├── informers
│   └── externalversions
│       ├── android
│       │   ├── interface.go
│       │   └── v1
│       │       ├── anfile.go
│       │       ├── animage.go
│       │       └── interface.go
│       ├── factory.go
│       ├── generic.go
│       └── internalinterfaces
│           └── factory_interfaces.go
└── listers
    └── android
        └── v1
            ├── anfile.go
            ├── animage.go
            └── expansion_generated.go

16 directories, 26 files
```

