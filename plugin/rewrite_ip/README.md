# rewrite_ip

## Name

*rewrite_ip* - 根据 hosts 文件智能重写 DNS 响应中的 IP 地址。

## Description

`rewrite_ip` 插件允许根据域名匹配规则，将 DNS 响应中的 IP 地址替换为本地 hosts 文件中定义的地址。支持通配符匹配和域名映射功能。

## Syntax

```corefile
rewrite_ip {
    match PATTERN [PATTERN...]
    [map_to DOMAIN]
    hosts HOSTSFILE
}
```

- **match**: 一个或多个通配符模式（支持 `*`），如 `*.example.com`
- **map_to**: （可选）将匹配的域名映射到另一个域名再查询 hosts 文件
- **hosts**: hosts 文件路径（标准 `/etc/hosts` 格式）

## Examples

### 直接重写模式

将所有 `*.test.com` 的解析结果替换为 hosts 文件中的定义：

```corefile
.:53 {
    rewrite_ip {
        match *.test.com
        hosts /etc/coredns/hosts_test.txt
    }
    forward . 8.8.8.8
}
```

`hosts_test.txt`:
```
10.0.0.1  api.test.com
10.0.0.2  web.test.com
::1       api.test.com
```

查询 `api.test.com` 时，原本的公网 IP 会被替换为 `10.0.0.1` (IPv4) 或 `::1` (IPv6)。

### 映射重写模式

将多个子域名聚合到一个网关地址：

```corefile
.:53 {
    rewrite_ip {
        match *.prod.com *.api.com
        map_to gateway.local
        hosts /etc/coredns/hosts_gateway.txt
    }
    forward . 8.8.8.8
}
```

`hosts_gateway.txt`:
```
192.168.1.100  gateway.local
2001:db8::1    gateway.local
```

查询 `service.prod.com` 或 `v2.api.com` 时，IP 都会被替换为 `192.168.1.100`。

## 特性

- ✅ **通配符匹配**: 支持 `*.domain.com` 模式
- ✅ **类型严格匹配**: A 记录只替换为 IPv4，AAAA 记录只替换为 IPv6
- ✅ **Fallback 机制**: 如果 hosts 文件中无匹配记录，保留原始 IP
- ✅ **多规则支持**: 可配置多个 `rewrite_ip` 块
- ✅ **域名映射**: 支持将多个域名映射到一个共同的后端

## Implementation Notes

该插件使用 ResponseWriter wrapper 拦截后端返回的 DNS 响应，在 `WriteMsg` 阶段修改 Answer section 中的 A/AAAA 记录。
