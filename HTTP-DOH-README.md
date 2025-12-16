# HTTP DoH (DNS-over-HTTP) 支持

CoreDNS现在支持在**局域网/内部网络环境**中使用HTTP（非TLS）运行DNS-over-HTTPS服务器。

## ⚠️ 重要安全提示

**HTTP DoH会导致DNS流量未加密传输！**

- ✅ **适用场景**：受信任的局域网、内部网络、开发/测试环境
- ❌ **不适用场景**：公网、不受信任的网络、生产环境（除非在受保护的内网）

## 配置方法

### 方式1：仅HTTP DoH

```corefile
https://.:8053 {
    tls {
        allow_http_doh
    }
    log
    errors
    cache 30
    forward . 8.8.8.8
}
```

### 方式2：同时提供普通DNS和HTTP DoH

```corefile
# 普通DNS服务
. :53 {
    log
    errors
    cache 30
    forward . 8.8.8.8
}

# HTTP DoH服务
https://.:8053 {
    tls {
        allow_http_doh
    }
    log
    errors
    cache 30
    forward . 8.8.8.8
}
```

### 方式3：标准HTTPS DoH (推荐生产环境)

```corefile
https://.:443 {
    tls cert.pem key.pem
    log
    errors
    cache 30
    forward . 8.8.8.8
}
```

## 使用方法

### 1. 启动服务器

```bash
# 使用示例配置
./coredns -conf Corefile.http-doh

# 或使用自定义配置
./coredns -conf your-config.conf
```

### 2. 测试连接

#### 使用curl

```bash
# 创建一个DNS查询文件（example.com A记录）
echo "AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=" | base64 -d > query.bin

# 发送查询
curl -H "Content-Type: application/dns-message" \
     --data-binary @query.bin \
     http://localhost:8053/dns-query
```

#### 使用Python测试脚本

```bash
# 安装依赖
pip install requests

# 运行测试
./test-http-doh.py

# 自定义服务器
./test-http-doh.py --server http://192.168.1.100:8053/dns-query
```

#### 使用DoH客户端

如果你有支持HTTP DoH的客户端（如修改过的dog、doh-client等）：

```bash
# 示例（具体命令取决于客户端）
dog example.com @http://localhost:8053/dns-query
```

## 技术实现

### 修改的文件

1. **core/dnsserver/config.go**
   - 添加 `AllowHTTP` 字段控制是否允许HTTP DoH

2. **core/dnsserver/server_https.go**
   - 修改服务器启动逻辑，仅在有TLS配置时才启用TLS

3. **plugin/tls/tls.go**
   - 添加 `allow_http_doh` 配置选项
   - 支持无证书的tls块配置

### 工作原理

当使用 `allow_http_doh` 时：
1. tls插件设置 `config.AllowHTTP = true`
2. 不设置 `config.TLSConfig`
3. ServerHTTPS检测到没有TLS配置，跳过TLS包装
4. HTTP服务器直接处理DoH请求，无加密

## 与HTTPS DoH的对比

| 特性 | HTTP DoH | HTTPS DoH |
|------|----------|-----------|
| 加密传输 | ❌ 否 | ✅ 是 |
| 需要证书 | ❌ 否 | ✅ 是 |
| 适用环境 | 局域网/内网 | 公网/生产环境 |
| 性能开销 | 低 | 稍高（TLS握手） |
| 安全性 | 低 | 高 |
| 端口 | 任意 | 通常443 |

## 故障排查

### 问题：配置解析错误

```
Error during parsing: Wrong argument count or unexpected line ending after 'tls'
```

**解决方案**：确保使用正确的语法：
```corefile
tls {
    allow_http_doh
}
```
而不是：
```corefile
tls allow_http_doh  # 错误！
```

### 问题：连接被拒绝

**检查项**：
1. CoreDNS是否正在运行？
2. 端口是否正确？
3. 防火墙是否阻止了连接？

### 问题：400 Bad Request

如果看到 "no 'dns' query parameter found"，说明请求格式不正确。

**确保**：
- 使用POST方法
- Content-Type设置为 `application/dns-message`
- 请求体包含有效的DNS消息（二进制格式）

## 示例场景

### 场景1：内网DNS服务器

在公司内网中，你有一个内部DNS服务器，所有客户端都在受信任的网络中：

```corefile
https://.:8053 {
    tls {
        allow_http_doh
    }
    kubernetes cluster.local
    forward . 10.0.0.1
}
```

### 场景2：开发环境

本地开发时测试DoH集成：

```corefile
https://127.0.0.1:8053 {
    tls {
        allow_http_doh
    }
    file db.example.com example.com
}
```

### 场景3：容器内部通信

在Kubernetes集群内部，Pod之间通过HTTP DoH通信（集群网络已加密）：

```corefile
https://.:8053 {
    tls {
        allow_http_doh
    }
    kubernetes cluster.local in-addr.arpa ip6.arpa
}
```

## 相关资源

- [RFC 8484 - DNS Queries over HTTPS (DoH)](https://tools.ietf.org/html/rfc8484)
- [CoreDNS Documentation](https://coredns.io/)
- [CoreDNS TLS Plugin](https://coredns.io/plugins/tls/)

## 贡献

如有问题或建议，请提交Issue或Pull Request。
