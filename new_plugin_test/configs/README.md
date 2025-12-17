# Corefile 配置示例

本目录包含不同测试场景的 Corefile 配置示例。

## DoH 配置语法

### HTTP DoH (不使用 TLS)

```corefile
.:8053 {
    doh /dns-query {
        # 不写tls就是http
    }
    log
}
```

### HTTPS DoH (使用自签名证书)

```corefile
.:8443 {
    doh /dns-query {
        tls selfsigned
    }
    log
}
```

### HTTPS DoH (使用真实证书)

```corefile
.:8443 {
    doh /dns-query {
        tls /path/to/cert.pem /path/to/key.pem
    }
    log
}
```

## 配置文件说明

### Corefile.http_doh
用于 HTTP DoH 基础测试（TC-01）。
- 监听端口: 8053
- 协议: HTTP (无加密)
- 路径: /dns-query

### Corefile.mixed_protocol
用于混合协议测试（TC-02）。
- HTTP DoH: 端口 8053
- HTTPS DoH: 端口 8443 (使用 selfsigned 证书)

### Corefile.rewrite_ip
用于 IP 重写插件测试（TC-03 到 TC-09）。
- 监听端口: 8053
- 包含 rewrite_ip 插件配置
- 可选启用 HTTP DoH 支持

## 启动示例

```bash
# HTTP DoH 测试
coredns -conf configs/Corefile.http_doh

# 混合协议测试
coredns -conf configs/Corefile.mixed_protocol

# IP Rewrite 测试
coredns -conf configs/Corefile.rewrite_ip
```

## 注意事项

1. **端口冲突**: 确保测试端口（8053, 8443）未被占用
2. **证书配置**: 
   - 测试环境推荐使用 `tls selfsigned`
   - 生产环境应使用有效的 TLS 证书
3. **权限要求**: 监听 1024 以下端口需要 root 权限
4. **防火墙**: 确保测试端口在防火墙中开放
