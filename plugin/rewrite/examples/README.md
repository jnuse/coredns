# CoreDNS Rewrite 插件 IP 地址重写示例

本目录包含 CoreDNS rewrite 插件 IP 地址重写功能的示例配置。

## 文件说明

- **Corefile.ip-rewrite**: 包含多个使用场景的完整配置示例
- **ip-rewrites.txt**: hosts 格式的 IP 映射文件示例

## 功能特性

IP 地址重写功能允许你根据域名匹配来替换 DNS 响应中的 IP 地址，而不管后端 DNS 服务器返回什么 IP。

### 主要特点

1. **基于域名匹配**: 根据响应记录的域名来决定是否重写 IP
2. **保留其他字段**: 只修改 IP 地址，保留 TTL、Class 等其他字段
3. **支持 IPv4 和 IPv6**: 同时支持 A 记录和 AAAA 记录的重写
4. **多种配置方式**:
   - 单个映射: `answer ip IP DOMAIN`
   - 文件批量映射: `answer ip file /path/to/hosts`

## 使用场景

### 1. NAT 穿透
将内网服务的公网域名映射到内网 IP：

```
rewrite stop {
    name suffix .internal.example.com. .internal.example.com.
    answer auto
    answer ip 192.168.1.10 web.internal.example.com
}
```

### 2. 测试环境
将生产域名映射到测试环境的 IP：

```
rewrite stop {
    name suffix .example.com. .example.com.
    answer ip file /etc/coredns/test-ips.txt
}
```

### 3. IP 迁移
透明地将服务迁移到新的 IP 地址：

```
rewrite stop {
    name suffix .services.com. .services.com.
    answer ip 10.0.0.100 old-service.services.com
}
```

### 4. Split-Horizon DNS
根据查询路径返回不同的 IP：

```
# 内网查询
internal.example.com:53 {
    rewrite stop {
        name suffix .example.com. .example.com.
        answer ip file /etc/coredns/internal-ips.txt
    }
    forward . 192.168.1.1
}

# 外网查询
.:53 {
    forward . 8.8.8.8
}
```

## 测试方法

### 1. 启动 CoreDNS
```bash
coredns -conf Corefile.ip-rewrite
```

### 2. 测试查询
```bash
# 测试单个域名
dig @localhost api.example.com A

# 测试 IPv6
dig @localhost server.ipv6.example.com AAAA

# 测试从文件加载的映射
dig @localhost web.internal.example.com A
```

### 3. 验证结果
检查返回的 IP 地址是否是你在配置中指定的值。

## 注意事项

1. **文件格式**: IP 映射文件使用标准的 hosts 格式（IP domain [domain...]）
2. **启动加载**: 文件在 CoreDNS 启动时加载，修改后需要重启
3. **域名必须匹配**: 只有当响应记录的域名与配置的域名完全匹配时才会重写
4. **不影响原始响应**: 其他 DNS 记录（如 MX、TXT）不受影响

## 高级用法

### 组合多个 IP 重写规则

```
rewrite stop {
    name suffix .example.com. .example.com.
    answer auto
    answer ip 10.0.0.1 api.example.com
    answer ip 10.0.0.2 web.example.com
    answer ip file /etc/coredns/additional-ips.txt
}
```

### 结合其他 rewrite 规则

```
rewrite stop {
    name regex (.+)\.old\.com\. {1}.new.com.
    answer name (.+)\.new\.com\. {1}.old.com.
    answer ip 192.168.1.100 service.old.com
}
```

这样可以同时重写域名和 IP 地址。
