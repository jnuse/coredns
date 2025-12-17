# CoreDNS 插件测试套件

本目录包含 CoreDNS 新功能的完整测试用例，基于 TODO.MD 中的设计文档。

## 📁 目录结构

```
new_plugin_test/
├── README.md                    # 本文件
├── run_all_tests.sh            # 主测试运行脚本
├── configs/                    # 测试配置文件
│   ├── Corefile.http_doh       # HTTP DoH 测试配置
│   ├── Corefile.mixed_protocol # 混合协议测试配置
│   └── Corefile.rewrite_ip     # IP 重写测试配置
├── hostfiles/                  # hosts 文件（用于 IP 重写测试）
│   ├── hosts_direct.txt        # 直接重写测试
│   ├── hosts_mapped.txt        # 映射重写测试
│   └── hosts_mapped_no_v6.txt  # IPv6 缺失测试
├── scripts/                    # 测试脚本
│   ├── test_http_doh.sh        # HTTP DoH 测试套件
│   └── test_rewrite_ip.sh      # IP Rewrite 测试套件
└── utils/                      # 工具函数
    └── dns_query.py            # DNS 查询工具
```

## 🧪 测试用例覆盖

### Part 1: HTTP DoH 连通性测试

| 测试ID | 测试名称 | 脚本 |
|--------|----------|------|
| TC-01 | HTTP DoH 基础查询 | test_http_doh.sh |
| TC-02 | 混合协议共存（HTTP + HTTPS） | test_http_doh.sh |

### Part 2: IP Rewrite 功能测试

| 测试ID | 测试名称 | 脚本 |
|--------|----------|------|
| TC-03 | 直接重写 IPv4 | test_rewrite_ip.sh |
| TC-04 | 直接重写 IPv6 | test_rewrite_ip.sh |
| TC-05 | 映射重写 IPv4 | test_rewrite_ip.sh |
| TC-06 | 类型严格匹配（缺失 IPv6） | test_rewrite_ip.sh |
| TC-07 | 无匹配规则 | test_rewrite_ip.sh |
| TC-08 | Host 文件中无记录 | test_rewrite_ip.sh |
| TC-09 | 多条记录混合 | test_rewrite_ip.sh |

## 🚀 快速开始

### 前置要求

1. **Python 3** - 用于 DNS 查询工具
   ```bash
   python3 --version
   ```

2. **netcat (nc)** - 用于服务健康检查
   ```bash
   nc -h
   ```

3. **CoreDNS 二进制** - 已构建的 CoreDNS
   ```bash
   # 构建 CoreDNS
   cd /workspaces/coredns
   make
   ```

### 运行测试

#### 1. 运行所有测试
```bash
cd /workspaces/coredns/new_plugin_test
chmod +x run_all_tests.sh
./run_all_tests.sh --all
```

#### 2. 仅运行 HTTP DoH 测试
```bash
./run_all_tests.sh --http-doh
```

#### 3. 仅运行 IP Rewrite 测试
```bash
./run_all_tests.sh --rewrite-ip
```

## 📝 测试步骤详解

### HTTP DoH 测试

**启动 CoreDNS (HTTP DoH 模式):**
```bash
# 在终端 1 中
cd /workspaces/coredns
./coredns -conf new_plugin_test/configs/Corefile.http_doh
```

**运行测试:**
```bash
# 在终端 2 中
cd /workspaces/coredns/new_plugin_test
./scripts/test_http_doh.sh
```

**预期结果:**
- TC-01: HTTP DoH 查询成功返回 200 OK
- TC-02: 多端口监听正常工作

### IP Rewrite 测试

**启动 CoreDNS (IP Rewrite 模式):**
```bash
# 在终端 1 中
cd /workspaces/coredns
./coredns -conf new_plugin_test/configs/Corefile.rewrite_ip
```

**运行测试:**
```bash
# 在终端 2 中
cd /workspaces/coredns/new_plugin_test
./scripts/test_rewrite_ip.sh
```

**预期结果:**
- TC-03: `api.test.com` (A) 重写为 `10.0.0.1`
- TC-04: `api.test.com` (AAAA) 重写为 `::1`
- TC-05: `service.prod.com` (A) 映射到 `gateway.local` -> `192.168.1.100`
- TC-06: 类型严格匹配，IPv6 缺失时保留原值
- TC-07: 不匹配规则的域名保持原样
- TC-08: hosts 文件缺失记录时回退到上游
- TC-09: 混合记录选择性重写

## 🔧 手动测试工具

### DNS 查询工具使用

```bash
# 标准 UDP 查询
python3 utils/dns_query.py example.com -t A -s 127.0.0.1 -p 8053

# DoH 查询
python3 utils/dns_query.py example.com --doh http://127.0.0.1:8053/dns-query

# IPv6 查询
python3 utils/dns_query.py example.com -t AAAA -s 127.0.0.1 -p 8053
```

### 使用 curl 测试 DoH

```bash
# 生成 DNS 查询
python3 -c "
import sys
sys.path.insert(0, 'utils')
from dns_query import DNSQuery
sys.stdout.buffer.write(DNSQuery.build_query('example.com', 'A'))
" > /tmp/query.bin

# 发送 DoH 请求
curl -v \
  -H "Content-Type: application/dns-message" \
  -H "Accept: application/dns-message" \
  --data-binary @/tmp/query.bin \
  http://127.0.0.1:8053/dns-query
```

## 📊 测试报告

测试脚本会自动生成彩色输出报告：

```
==========================================
Test Summary
==========================================
Total Tests: 7
Passed: 7
Failed: 0
All tests passed! ✓
```

## 🐛 故障排查

### CoreDNS 启动失败
```bash
# 检查端口占用
netstat -tuln | grep 8053

# 查看 CoreDNS 日志
./coredns -conf new_plugin_test/configs/Corefile.http_doh -log
```

### Python 模块导入错误
```bash
# 确保 Python 路径正确
export PYTHONPATH=/workspaces/coredns/new_plugin_test:$PYTHONPATH
```

### DNS 查询超时
```bash
# 检查 CoreDNS 是否运行
ps aux | grep coredns

# 检查防火墙规则
sudo iptables -L
```

## 📚 参考文档

- [TODO.MD](../TODO.MD) - 功能设计文档
- [CoreDNS Plugin 开发文档](https://coredns.io/manual/plugins/)
- [DNS over HTTPS (RFC 8484)](https://datatracker.ietf.org/doc/html/rfc8484)

## 🤝 贡献

如需添加新的测试用例：

1. 在对应的测试脚本中添加测试函数 `test_tcXX_description()`
2. 在主测试流程中调用新函数
3. 更新本 README 的测试用例表格
4. 确保所有测试可以独立运行

## 📄 许可证

遵循 CoreDNS 项目的许可证要求。
