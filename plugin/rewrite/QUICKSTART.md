# IP 地址重写功能快速开始指南

## 5 分钟快速上手

### 1. 创建 IP 映射文件

创建文件 `/tmp/test-ips.txt`:
```bash
cat > /tmp/test-ips.txt << 'EOF'
# 测试 IP 映射
10.0.0.100 test.example.com
10.0.0.200 api.example.com
2001:db8::1 ipv6.example.com
EOF
```

### 2. 创建 Corefile

创建文件 `/tmp/Corefile`:
```bash
cat > /tmp/Corefile << 'EOF'
.:1053 {
    # 日志输出
    log
    
    # IP 地址重写
    rewrite stop {
        name suffix .example.com. .example.com.
        answer auto
        answer ip file /tmp/test-ips.txt
    }
    
    # 转发到 Google DNS
    forward . 8.8.8.8
    
    # 缓存
    cache 30
}
EOF
```

### 3. 启动 CoreDNS

```bash
cd /workspaces/coredns
./coredns -conf /tmp/Corefile
```

### 4. 测试功能

在另一个终端中测试：

```bash
# 测试 IPv4 地址重写
dig @127.0.0.1 -p 1053 test.example.com A

# 期望输出包含:
# test.example.com.       300     IN      A       10.0.0.100

# 测试另一个域名
dig @127.0.0.1 -p 1053 api.example.com A

# 期望输出包含:
# api.example.com.        300     IN      A       10.0.0.200

# 测试 IPv6
dig @127.0.0.1 -p 1053 ipv6.example.com AAAA

# 期望输出包含:
# ipv6.example.com.       300     IN      AAAA    2001:db8::1

# 测试未配置的域名（应该返回真实 IP）
dig @127.0.0.1 -p 1053 google.com A
```

### 5. 验证结果

观察 CoreDNS 日志，你应该能看到：
- DNS 查询请求
- IP 地址被成功重写
- 响应返回给客户端

## 单个映射示例

如果只需要重写一个域名，可以不使用文件：

```bash
cat > /tmp/Corefile-simple << 'EOF'
.:1053 {
    log
    
    rewrite stop {
        name suffix .example.com. .example.com.
        answer auto
        answer ip 192.168.1.100 myserver.example.com
    }
    
    forward . 8.8.8.8
}
EOF

./coredns -conf /tmp/Corefile-simple
```

测试：
```bash
dig @127.0.0.1 -p 1053 myserver.example.com A
# 应该返回 192.168.1.100
```

## 常见问题

### Q: 为什么我的 IP 没有被重写？
A: 检查以下几点：
1. 域名必须完全匹配（包括尾部的点）
2. 确认 name 规则能匹配该域名
3. 检查 IP 映射文件格式是否正确

### Q: 可以同时使用多个映射文件吗？
A: 不能在一个 `answer ip file` 中使用多个文件，但可以使用多个 `answer ip file` 行：
```
answer ip file /path/to/file1.txt
answer ip file /path/to/file2.txt
```

### Q: IPv4 和 IPv6 可以混用吗？
A: 可以，IPv4 映射只影响 A 记录，IPv6 映射只影响 AAAA 记录。

### Q: 修改映射文件后需要重启吗？
A: 是的，映射文件在启动时加载，修改后需要重启 CoreDNS。

## 下一步

- 查看 [README.md](README.md) 了解详细文档
- 查看 [examples/](examples/) 目录了解更多使用场景
- 查看 [IP_REWRITE_FEATURE.md](IP_REWRITE_FEATURE.md) 了解技术实现
