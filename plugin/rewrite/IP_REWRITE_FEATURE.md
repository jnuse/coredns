# CoreDNS Rewrite 插件 IP 地址重写功能

## 功能概述

为 CoreDNS 的 rewrite 插件添加了基于域名匹配的 IP 地址重写功能。此功能允许根据 DNS 响应记录中的域名来替换 A 记录和 AAAA 记录中的 IP 地址。

## 实现内容

### 1. 核心功能实现

#### 文件: `plugin/rewrite/ip.go`
- 实现 `ipRewriterResponseRule` 结构体
- 支持域名到 IPv4/IPv6 的映射
- 实现 `RewriteResponse()` 方法处理 A 和 AAAA 记录
- 支持从标准 hosts 文件格式加载映射

**关键特性:**
- 基于域名匹配（不是 IP 到 IP 的映射）
- 无论原始 IP 是什么，只要域名匹配就替换
- 保留所有其他 DNS 记录字段（TTL、Class 等）
- 自动区分 IPv4 和 IPv6

### 2. 配置解析

#### 文件: `plugin/rewrite/name.go`
修改 `parseAnswerRules()` 函数，添加对 `ip` 类型的支持：
- `answer ip IP DOMAIN` - 单个域名到 IP 的映射
- `answer ip file PATH` - 从 hosts 文件批量加载

### 3. 文档更新

#### 文件: `plugin/rewrite/README.md`
添加了完整的 IP 地址重写功能说明，包括：
- 语法说明
- 使用场景
- 多个实际示例
- 注意事项

### 4. 测试覆盖

#### 文件: `plugin/rewrite/ip_test.go`
实现了全面的单元测试：
- 基本的 IPv4/IPv6 重写测试
- 域名匹配/不匹配测试
- 从文件加载测试
- 保留其他字段测试
- 错误处理测试

#### 文件: `plugin/rewrite/testdata/ip-mappings.txt`
测试数据文件，包含多种映射格式

### 5. 示例配置

#### 目录: `plugin/rewrite/examples/`
- `Corefile.ip-rewrite` - 多种使用场景的完整配置
- `ip-rewrites.txt` - 示例映射文件
- `README.md` - 详细使用说明

## 使用方法

### 基本语法

```corefile
rewrite [continue|stop] {
    name [exact|prefix|suffix|substring|regex] STRING STRING
    answer ip IP DOMAIN
}
```

或使用 hosts 文件：

```corefile
rewrite [continue|stop] {
    name [exact|prefix|suffix|substring|regex] STRING STRING
    answer ip file /path/to/hosts
}
```

### Hosts 文件格式

```
# 标准 hosts 格式
IP地址 域名 [别名...]

# 示例
10.0.0.1 api.example.com api-v1.example.com
2001:db8::1 ipv6.example.com
```

### 配置示例

#### 1. 单个域名 IP 重写
```corefile
. {
    rewrite stop {
        name suffix .example.com. .example.com.
        answer auto
        answer ip 203.0.113.10 api.example.com
    }
    forward . 8.8.8.8
}
```

#### 2. 从文件加载多个映射
```corefile
. {
    rewrite stop {
        name suffix .internal.com. .internal.com.
        answer auto
        answer ip file /etc/coredns/ip-mappings.txt
    }
    forward . 192.168.1.1
}
```

#### 3. IPv6 支持
```corefile
. {
    rewrite stop {
        name suffix .example.com. .example.com.
        answer auto
        answer ip 2001:db8:1::100 ipv6.example.com
    }
    forward . 2001:4860:4860::8888
}
```

## 使用场景

1. **NAT 穿透**: 将内网服务的公网域名映射到内网 IP
2. **测试环境**: 将生产域名映射到测试 IP
3. **IP 迁移**: 透明地迁移服务到新 IP
4. **Split-Horizon DNS**: 内外网返回不同 IP
5. **负载均衡**: 配合其他插件实现智能 IP 分配

## 技术细节

### 工作原理

1. DNS 请求到达 CoreDNS
2. Rewrite 插件根据配置重写请求（可选）
3. 请求转发到后端 DNS 服务器
4. 后端返回响应
5. Rewrite 插件检查响应中的 A/AAAA 记录
6. 如果记录的域名在映射表中，替换 IP 地址
7. 返回修改后的响应给客户端

### 关键实现点

- **域名规范化**: 使用 `plugin.Name().Normalize()` 确保一致性
- **IPv4/IPv6 分离**: 分别维护 v4 和 v6 的映射表
- **保留原始字段**: 只修改 IP 地址，其他字段不变
- **性能**: 使用 map 实现 O(1) 查找

## 测试结果

所有测试通过：
```
=== RUN   TestIPRewriterResponseRule
--- PASS: TestIPRewriterResponseRule (0.00s)
=== RUN   TestIPRewriterInvalidMappings  
--- PASS: TestIPRewriterInvalidMappings (0.00s)
=== RUN   TestIPRewriterMultipleMappings
--- PASS: TestIPRewriterMultipleMappings (0.00s)
=== RUN   TestIPRewriterLoadFromFile
--- PASS: TestIPRewriterLoadFromFile (0.00s)
=== RUN   TestIPRewriterPreservesOtherFields
--- PASS: TestIPRewriterPreservesOtherFields (0.00s)
PASS
```

## 注意事项

1. **启动时加载**: hosts 文件在 CoreDNS 启动时加载，修改后需要重启
2. **精确匹配**: 域名必须完全匹配（包括尾部的点）
3. **不影响其他记录**: 只处理 A 和 AAAA 记录，其他类型不受影响
4. **IP 版本**: IPv4 映射只影响 A 记录，IPv6 映射只影响 AAAA 记录

## 兼容性

- 完全兼容现有的 rewrite 插件功能
- 可以与其他 rewrite 规则（name、value、ttl 等）组合使用
- 所有现有测试通过，无破坏性变更

## 文件清单

新增/修改的文件：
- `plugin/rewrite/ip.go` - 核心实现（新增）
- `plugin/rewrite/ip_test.go` - 单元测试（新增）
- `plugin/rewrite/name.go` - 解析逻辑（修改）
- `plugin/rewrite/README.md` - 文档（修改）
- `plugin/rewrite/testdata/ip-mappings.txt` - 测试数据（新增）
- `plugin/rewrite/examples/` - 示例文件（新增目录）
  - `Corefile.ip-rewrite`
  - `ip-rewrites.txt`
  - `README.md`
