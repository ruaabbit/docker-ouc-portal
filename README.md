# OUC 网络认证 Docker 镜像

## 概述

本项目旨在创建一个 Docker 镜像，用于中国海洋大学（OUC）校园网的自动认证。通过运行一个 Golang 程序，该镜像能够自动检测网络连接状态，并在需要时发送认证请求，从而实现校园网的自动登录。

## 主要功能

- **自动认证**：通过 Golang 脚本定期检测网络状态，并在断网时自动发送 GET 请求进行认证。
- **环境变量配置**：支持通过环境变量传入校园网的用户名和密码，确保敏感信息的安全。
- **Docker 化部署**：方便在各种支持 Docker 的环境中部署和运行。

## 工作原理

1.  **初始化**：启动容器时，从环境变量中读取用户配置的校园网用户名和密码。
2.  **网络状态监测**：Golang 程序会定期尝试访问一个或多个外部网站（例如 `baidu.com` 或 `bing.com`）来判断当前设备是否已经连接到互联网。
3.  **认证请求**：
    - 如果检测到无法访问外部网络（表明当前可能未登录校园网或登录已失效），程序将使用预设的认证 URL 和从环境变量中获取的用户名、密码构造并发送一个 GET 请求到 OUC 的网络认证服务器。
    - 如果网络已连接，则程序会等待一段时间后再次进行检测。
4.  **循环执行**：上述监测和认证过程会持续循环执行，以确保网络连接的持续性。

## 如何使用

### 前提条件

- 已安装 Docker。

### 构建镜像

如果您有 Dockerfile 和相关的 Golang 代码，可以使用以下命令构建镜像（假设 Dockerfile 在当前目录）：

```bash
docker build -t docker-ouc-portal .
```

### 运行镜像

使用以下命令运行 Docker 容器。请将 `YOUR_USERNAME` 和 `YOUR_PASSWORD`替换为您的实际校园网用户名和密码。

```bash
docker run -d \
  -e WLJF_USERNAME=YOUR_USERNAME \
  -e WLJF_PASSWORD=YOUR_PASSWORD \
  --name ouc-auth \
  --restart always \
  docker-ouc-portal
```

参数说明：

- `-d`: 后台运行容器。
- `-e WLJF_USERNAME`: 设置校园网用户名的环境变量。
- `-e WLJF_PASSWORD`: 设置校园网密码的环境变量。
- `--name ouc-auth`: 为容器指定一个名称。
- `--restart always`: 容器退出时总是自动重启，确保认证服务持续运行。
- `docker-ouc-portal`: 您构建的镜像名称。

## 环境变量

在运行 Docker 容器时，需要配置以下环境变量：

- `WLJF_USERNAME`: 您的 OUC 校园网用户名。
- `WLJF_PASSWORD`: 您的 OUC 校园网密码。
- `WLJF_MODE` (可选): 认证请求的模式，例如：XHA,WXRZ,YXRZ。默认为 XHA。
- `CHECK_INTERVAL_SECONDS` (可选): 网络状态检测的时间间隔（单位：秒）。默认为 600 秒。
- `CHECK_TARGET_HOST` (可选): 用于检测网络连通性的目标主机。默认为 `https://www.baidu.com/`。
- `TZ`(可选): 时区。

## 待办事项 (TODO)

- [ ] 更完善的错误处理和日志记录。
- [ ] 支持多种网络状态检测策略。
- [ ] 考虑添加 Web UI 进行状态查看和配置（可选）。

## 校园网认证相关 cURL 命令

### 登录请求（XHA）

```bash
curl 'https://xha.ouc.edu.cn:802/eportal/portal/login?callback=dr1003&login_method=1&user_account=********&user_password=********&wlan_user_ip=0.0.0.0&wlan_user_ipv6=&wlan_user_mac=000000000000&wlan_ac_ip=&wlan_ac_name=&jsVersion=4.1&terminal_type=1&lang=zh-cn&lang=zh'
```

### 注销请求（XHA）

```bash
curl 'https://xha.ouc.edu.cn:802/eportal/portal/logout?callback=dr1006&login_method=1&user_account=drcom&user_password=123&ac_logout=0&register_mode=1&wlan_user_ip=&wlan_user_ipv6=&wlan_vlan_id=1&wlan_user_mac=000000000000&wlan_ac_ip=&wlan_ac_name=&jsVersion=4.1&bas_ip=xha.ouc.edu.cn&type=1&lang=zh'
```
