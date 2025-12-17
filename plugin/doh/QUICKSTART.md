# DoH 插件快速开始

## 🚀 5分钟快速开始

### 1. 编译 CoreDNS

```bash
cd /workspaces/coredns
go generate && go build
```

### 2. 创建配置文件

创建 `Corefile`:

```corefile
.:8053 {
    doh /dns-query
    forward . 8.8.8.8
    log
}
```

### 3. 启动服务器

```bash
./coredns -conf Corefile
```

### 4. 测试查询

**使用 Python 工具**:
```bash
python3 new_plugin_test/utils/dns_query.py example.com --doh http://127.0.0.1:8053/dns-query
```

**使用 curl**:
```bash
# 生成 DNS 查询
echo "00 00 01 00 00 01 00 00 00 00 00 00 07 65 78 61 6d 70 6c 65 03 63 6f 6d 00 00 01 00 01" | \
  xxd -r -p > query.bin

# 发送 DoH 请求
curl -H "Content-Type: application/dns-message" \
     --data-binary @query.bin \
     http://localhost:8053/dns-query | \
     xxd
```

## 📖 常用配置

### HTTP DoH (本地/内网)
```corefile
.:8053 {
    doh /dns-query
    forward . 8.8.8.8 1.1.1.1
    cache 30
    log
}
```

### HTTPS DoH (测试环境)
```corefile
.:8443 {
    doh /dns-query {
        tls selfsigned
    }
    forward . 8.8.8.8
    log
}
```

### HTTPS DoH (生产环境)
```corefile
.:443 {
    doh /dns-query {
        tls /etc/ssl/cert.pem /etc/ssl/key.pem
    }
    forward . 1.1.1.1 8.8.8.8
    cache 300
    log
}
```

### 混合协议
```corefile
# 内网 HTTP
.:8053 {
    doh /dns-query
    forward . 192.168.1.1
}

# 公网 HTTPS
.:443 {
    doh /dns-query {
        tls /etc/ssl/cert.pem /etc/ssl/key.pem
    }
    forward . 1.1.1.1
}
```

## 🧪 验证配置

```bash
# 检查配置语法
./coredns -conf Corefile -plugins

# 测试启动（5秒后自动停止）
timeout 5 ./coredns -conf Corefile || true
```

## 📝 完整文档

- [plugin/doh/README.md](README.md) - 详细配置文档
- [plugin/doh/IMPLEMENTATION.md](IMPLEMENTATION.md) - 实现细节
