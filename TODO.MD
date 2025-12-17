为了确保开发过程顺畅，我为你整理了一份详细的 **《功能开发设计文档》** 和配套的 **《测试用例集》**。

我们分为两个模块来设计：
1.  **CoreDNS Server 核心修改**（支持 HTTP DoH）
2.  **新插件 `rewrite_ip` 开发**（支持智能 IP 重写）

---

## 📋 开发进度跟踪

| 任务 | 状态 | 完成日期 | 说明 |
|------|------|----------|------|
| 恢复官方 rewrite 插件 | ✅ 完成 | 2025-12-17 | 从 upstream 恢复，删除污染代码（ip.go） |
| 创建测试目录结构 | ✅ 完成 | 2025-12-17 | new_plugin_test/ 目录及子目录 |
| 编写测试工具 | ✅ 完成 | 2025-12-17 | dns_query.py DNS查询工具 |
| 编写测试配置文件 | ✅ 完成 | 2025-12-17 | Corefile配置（HTTP DoH, 混合协议, IP重写） |
| 编写测试脚本 | ✅ 完成 | 2025-12-17 | test_http_doh.sh, test_rewrite_ip.sh |
| 编写主测试运行脚本 | ✅ 完成 | 2025-12-17 | run_all_tests.sh |
| 更新配置语法 | ✅ 完成 | 2025-12-17 | 使用新的 doh {} 块语法 |
| **实现 HTTP DoH 支持** | ✅ 完成 | 2025-12-17 | **创建 plugin/doh，支持 HTTP 和 HTTPS** |
| **实现 rewrite_ip 插件** | ✅ 完成 | 2025-12-17 | **新建 plugin/rewrite_ip，支持智能IP重写** |
| 运行完整测试 | ✅ 完成 | 2025-12-17 | 基础功能验证通过 |

### 🎉 DoH 插件实现完成！

**已完成功能**:
- ✅ 创建 `plugin/doh` 插件
- ✅ 支持 `doh /dns-query {}` 块级配置
- ✅ HTTP DoH (无 TLS) - 用于本地/内网环境
- ✅ HTTPS DoH with `tls selfsigned` - 自动生成自签名证书
- ✅ HTTPS DoH with `tls cert.pem key.pem` - 使用真实证书
- ✅ 混合协议支持（同时监听 HTTP 和 HTTPS）
- ✅ 完整文档和示例

**测试结果**:
```bash
# HTTP DoH (端口 8053)
✓ 成功返回 DNS 响应: example.com -> 104.20.34.220, 172.66.144.113

# HTTPS DoH (端口 8443, 自签名证书)
✓ TLS 连接成功
✓ 成功返回 DNS 响应: example.com -> 104.20.34.220, 172.66.144.113

# 混合协议
✓ HTTP (8053) 和 HTTPS (8443) 同时工作
```

**配置示例**:
```corefile
# HTTP DoH
.:8053 {
    doh /dns-query
    forward . 8.8.8.8
}

# HTTPS DoH
.:8443 {
    doh /dns-query {
        tls selfsigned
    }
    forward . 8.8.8.8
}
```

### 🎉 rewrite_ip 插件实现完成！

**已完成功能**:
- ✅ 创建 `plugin/rewrite_ip` 插件
- ✅ 支持通配符模式匹配 (`*.example.com`)
- ✅ 支持域名映射 (`map_to gateway.local`)
- ✅ 类型严格匹配（A 记录 → IPv4，AAAA 记录 → IPv6）
- ✅ Fallback 机制（hosts 文件中无记录时保留原始 IP）
- ✅ ResponseWriter wrapper 拦截响应并修改
- ✅ 完整文档和配置示例

**测试结果**:
```bash
# 测试 1: example.com (重写)
原始 IP: 104.20.34.220, 172.66.144.113
重写后: 10.0.0.1, 10.0.0.1 ✓

# 测试 2: github.com (不重写)
保留原始 IP: 20.205.243.166 ✓
```

**配置示例**:
```corefile
.:8053 {
    rewrite_ip {
        match *.example.com
        hosts /path/to/hosts_file.txt
    }
    forward . 8.8.8.8
}
```

**关键技术点**:
- 插件必须放在 `forward` 之前才能拦截响应
- 使用 `dns.ResponseWriter` wrapper 模式
- 支持 IPv4/IPv6 分离存储和查询

### 📦 已完成的测试套件

测试套件位于 `new_plugin_test/` 目录，包含：
- **测试脚本**: 9个完整测试用例（TC-01 到 TC-09）
- **配置文件**: 多个 Corefile 配置示例
  - [Corefile.http_doh](new_plugin_test/configs/Corefile.http_doh) - HTTP DoH 测试
  - [Corefile.https_doh](new_plugin_test/configs/Corefile.https_doh) - HTTPS DoH 测试  
  - [Corefile.rewrite_ip](new_plugin_test/configs/Corefile.rewrite_ip) - IP 重写测试
  - **[Corefile.all](new_plugin_test/configs/Corefile.all) - 大一统配置** ✨
- **工具函数**: Python DNS 查询工具
- **文档**: 完整的 README 和使用说明

**大一统配置测试结果**:
```bash
✅ 标准 DNS (8053):  example.com -> 10.0.0.1
✅ HTTP DoH (8080):  example.com -> 10.0.0.1  
✅ HTTPS DoH (8443): 正常工作（需跳过SSL验证）
```

**快速运行测试**:
```bash
# 大一统测试
cd /workspaces/coredns
./coredns -conf new_plugin_test/configs/Corefile.all

# 测试所有端口
dig @127.0.0.1 -p 8053 example.com  # 标准 DNS
python3 new_plugin_test/utils/dns_query.py example.com \
    --doh http://127.0.0.1:8080/dns-query  # HTTP DoH
```

详见: [new_plugin_test/README.md](new_plugin_test/README.md)

---

# 📘 CoreDNS 功能扩展开发文档

## 模块一：HTTP DoH 支持 (No-TLS)

### 1. 设计目标
修改 CoreDNS 的启动逻辑和 DoH 服务端实现，使其支持非加密的 HTTP 协议，以便在本地受信任环境（如 Service Mesh sidecar 或本地代理）中使用。

### 2. 代码修改点

#### A. 协议识别 (`core/dnsserver`)
目前 CoreDNS 通常根据前缀（如 `tls://`, `https://`）或端口来判断 Server 类型。
*   **修改目标**：在解析 Corefile Server Block 时，支持识别 `http://` 前缀。
*   **逻辑**：
    *   如果监测到 `http://` 前缀，标记该 Server 为 DoH 类型，但 `TLSConfig` 为 nil。
    *   端口监听逻辑改为标准的 `net.Listen` 而非 `tls.Listen`。

#### B. DoH Server 实现 (`plugin/doh` 或 `server/doh`)
*   **现有的逻辑**：强依赖 `http.ListenAndServeTLS`。
*   **修改建议**：
    *   在 Server 结构体中增加 `PlainHTTP bool` 字段。
    *   启动时判断：
        *   若 `PlainHTTP` 为 `true`，使用 `http.Server{}` 并调用 `ListenAndServe()`。
        *   同时需注意 HTTP/2 的支持（H2C），或者仅支持 HTTP/1.1（视 Go 的 `net/http` 默认行为而定，通常建议在 Cleartext 下也尽量支持 H2）。

### 3. 配置示例

**✅ 更新: 已采用新的配置语法** (2025-12-17)

```corefile
# HTTP DoH (不使用 TLS)
.:8053 {
    doh /dns-query {
        # 不写tls就是http
    }
}

# HTTPS DoH (使用自签名证书)
.:8443 {
    doh /dns-query {
        tls selfsigned
    }
}

# HTTPS DoH (使用真实证书)
.:8443 {
    doh /dns-query {
        tls cert.pem key.pem
    }
}
```

**说明**:
- `doh` 指令采用块语法，而非顶层协议前缀
- 不指定 `tls` 时默认使用 HTTP（无加密）
- `tls selfsigned` 自动生成自签名证书用于测试
- `tls <cert> <key>` 使用指定的证书文件

---

## 模块二：Rewrite IP 插件开发

### 1. 插件结构
新建插件目录 `plugin/rewrite_ip`。

**核心结构体 `RewriteIP`**:
```go
type RewriteIP struct {
    Next  plugin.Handler
    Rules []Rule
}

type Rule struct {
    Patterns []string    // 通配符模式，如 *.example.com
    MapTo    string      // 映射目标域名（可选），如 backend.local
    HostFile *HostFile   // 解析后的 hosts 文件数据
}
```

### 2. 核心逻辑 (`ServeDNS`)

由于我们需要修改 **响应内容**，所以不能直接在请求阶段处理。必须使用 `ResponseWriter` 包装器（Wrapper）。

**流程图：**
1.  **Intercept**: 使用 `ResponsePrinter` 包装 `dns.ResponseWriter`。
2.  **Forward**: 调用 `plugin.Next.ServeDNS` 让后端插件（如 `forward`）去获取真实 IP。
3.  **Process Response** (在 WriteMsg 阶段):
    *   遍历响应消息中的 `Msg.Answer` 列表。
    *   对于每一条 `A` 或 `AAAA` 记录：
        *   **Step 1**: 遍历 `Rules` 列表。
        *   **Step 2**: 检查记录的 `Header().Name` 是否匹配 `Rule.Patterns`（支持通配符）。
        *   **Step 3**: 匹配成功后：
            *   **Direct Mode**: 如果 `MapTo` 为空，用原始域名去查 `HostFile`。
            *   **Mapped Mode**: 如果 `MapTo` 不为空，用 `MapTo` 的域名去查 `HostFile`。
        *   **Step 4 (Strict Match)**:
            *   如果是 `A` 记录，只从 HostFile 拿 IPv4。
            *   如果是 `AAAA` 记录，只从 HostFile 拿 IPv6。
        *   **Step 5**: 如果查到了 IP，修改 `RR.A` 或 `RR.AAAA` 字段；没查到则跳过。
    *   发送修改后的消息给客户端。

### 3. Hosts 文件解析器
*   需要实现一个轻量级的解析器，或者复用 CoreDNS 现有的 `plugin/hosts` 中的逻辑。
*   **数据结构**：
    ```go
    map[string]struct{
        IPv4 []net.IP
        IPv6 []net.IP
    }
    ```
*   支持热加载（可选，利用 `fsnotify` 监控文件变化）。

---

# 🧪 测试用例集 (Test Cases)

为了保证功能稳健，我们需要覆盖以下场景。

## Part 1: HTTP DoH 连通性测试

| ID | 测试名称 | 前置条件 | 操作步骤 | 预期结果 |
| :--- | :--- | :--- | :--- | :--- |
| **TC-01** | **HTTP DoH 基础查询** | 启动 CoreDNS 监听 `:8053` (HTTP) | 使用 `curl` 发送 DoH 请求：<br>`curl -v -H "Content-Type: application/dns-message" --data-binary @query.bin http://127.0.0.1:8053/dns-query` | 1. 响应状态码 200 OK<br>2. Body 返回有效的 DNS 响应二进制流<br>3. 无 TLS 握手过程 |
| **TC-02** | **混合协议共存** | 监听 `:8053` (HTTP) 和 `:8443` (HTTPS) | 分别向两个端口发送请求 | 两个端口均能正常解析 DNS，互不干扰 |

## Part 2: IP Rewrite 功能测试

**环境准备：**
*   **HostFile A (`hosts_direct.txt`)**:
    ```
    10.0.0.1  api.test.com
    ::1       api.test.com
    ```
*   **HostFile B (`hosts_mapped.txt`)**:
    ```
    192.168.1.100  gateway.local
    2001:db8::1    gateway.local
    ```

**配置片段：**
```corefile
.:8053 {
    # 规则1: 直接重写
    rewrite_ip {
        match *.test.com
        hosts /path/to/hosts_direct.txt
    }
    
    # 规则2: 映射重写
    rewrite_ip {
        match *.prod.com
        map_to gateway.local
        hosts /path/to/hosts_mapped.txt
    }
    
    forward . 8.8.8.8
}
```

**✅ 测试配置已就绪**: 参见 `new_plugin_test/configs/Corefile.rewrite_ip`

| ID | 测试名称 | 输入 (DNS 响应原值) | 匹配逻辑 | 预期输出 (客户端收到的值) | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **TC-03** | **直接重写 IPv4** | `api.test.com` -> `1.1.1.1` (A) | 匹配规则1 (`*.test.com`)<br>查 `hosts_direct` 找 `api.test.com` | `api.test.com` -> `10.0.0.1` | 成功替换 |
| **TC-04** | **直接重写 IPv6** | `api.test.com` -> `2606::1` (AAAA) | 匹配规则1<br>查 IPv6 | `api.test.com` -> `::1` | 成功替换 |
| **TC-05** | **映射重写 IPv4** | `service.prod.com` -> `8.8.8.8` (A) | 匹配规则2 (`*.prod.com`)<br>映射为 `gateway.local`<br>查 `hosts_mapped` | `service.prod.com` -> `192.168.1.100` | 实现了多子域聚合到一个IP |
| **TC-06** | **类型严格匹配 (Missing V6)** | `service.prod.com` -> `2001::1` (AAAA) | 假设 `hosts_mapped` 里删掉了 IPv6 记录 | `service.prod.com` -> `2001::1` | **保持原值** (不回退到IPv4，也不返回空) |
| **TC-07** | **无匹配规则** | `www.google.com` -> `142.250.x.x` | 无规则匹配 | `www.google.com` -> `142.250.x.x` | 原封不动 |
| **TC-08** | **Host文件中无记录** | `unknown.test.com` -> `1.2.3.4` | 匹配规则1<br>但 `hosts_direct` 中无此域名 | `unknown.test.com` -> `1.2.3.4` | **保持原值** (Fallback) |
| **TC-09** | **多条记录混合** | `api.test.com` (A) + `other.com` (A) | 只有 `api` 匹配 | `api` 变更为 `10.0.0.1`<br>`other` 保持原值 | 仅重写匹配的记录 |
