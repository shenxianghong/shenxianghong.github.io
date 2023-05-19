---
title: "「 Kata Containers 」源码走读 — virtcontainers/network"
excerpt: "virtcontainers 中与 Endpoint、Network 等网络管理相关的流程梳理"
cover: https://picsum.photos/0?sig=20230415
thumbnail: https://camo.githubusercontent.com/fc2b272df13c770b08a779c5f96690946039c45998b1bb439eb193b3fcd829ab/68747470733a2f2f7777772e6f70656e737461636b2e6f72672f6173736574732f6b6174612f6b6174612d766572746963616c2d6f6e2d77686974652e706e67
date: 2023-04-15
toc: true
categories:
- Container Runtime
tag:
- Kata Containers

---

<div align=center><img width="200" style="border: 0px" src="https://katacontainers.io/static/logo-a1e2d09ad097b3fc8536cb77aa615c42.svg"></div>

------

> based on **3.0.0**