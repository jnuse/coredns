#!/usr/bin/env python3
"""
测试HTTP DoH (DNS-over-HTTP) 功能

此脚本演示如何使用HTTP（非HTTPS）向CoreDNS发送DoH查询。
注意：这仅用于局域网/内部网络测试。
"""

import base64
import sys

def create_dns_query(domain, qtype='A'):
    """
    创建一个简单的DNS查询消息
    
    这里使用预编码的查询作为示例
    实际应用中建议使用dns.message库
    """
    # 为example.com创建A记录查询的标准DNS消息
    # 消息格式：Header(12字节) + Question(可变长度)
    if domain == 'example.com' and qtype == 'A':
        # 预编码的example.com A记录查询
        return base64.b64decode('AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=')
    else:
        print(f"错误：此示例脚本仅支持example.com A记录查询")
        sys.exit(1)

def test_http_doh(server_url='http://localhost:8053/dns-query', domain='example.com'):
    """
    测试HTTP DoH查询
    """
    try:
        import requests
    except ImportError:
        print("错误：需要安装requests库")
        print("运行: pip install requests")
        sys.exit(1)
    
    print(f"测试HTTP DoH服务器: {server_url}")
    print(f"查询域名: {domain}")
    print("-" * 60)
    
    # 创建DNS查询
    dns_query = create_dns_query(domain, 'A')
    print(f"DNS查询大小: {len(dns_query)} 字节")
    
    # 发送HTTP POST请求
    try:
        response = requests.post(
            server_url,
            headers={
                'Content-Type': 'application/dns-message',
                'Accept': 'application/dns-message'
            },
            data=dns_query,
            timeout=5
        )
        
        print(f"\n响应状态: {response.status_code}")
        print(f"Content-Type: {response.headers.get('Content-Type')}")
        print(f"Cache-Control: {response.headers.get('Cache-Control', 'N/A')}")
        print(f"响应大小: {len(response.content)} 字节")
        
        if response.status_code == 200:
            print("\n✓ HTTP DoH查询成功!")
            print(f"响应数据 (hex): {response.content.hex()[:100]}...")
            
            # 简单解析响应（仅用于演示）
            if len(response.content) > 12:
                # DNS header中的answer count (偏移6-7)
                answer_count = int.from_bytes(response.content[6:8], 'big')
                print(f"应答记录数: {answer_count}")
        else:
            print(f"\n✗ 请求失败: {response.text}")
            
    except requests.exceptions.ConnectionError:
        print("\n✗ 连接失败：无法连接到服务器")
        print("请确保CoreDNS正在运行:")
        print("  ./coredns -conf Corefile.http-doh")
    except Exception as e:
        print(f"\n✗ 错误: {e}")

if __name__ == '__main__':
    import argparse
    
    parser = argparse.ArgumentParser(description='测试HTTP DoH服务器')
    parser.add_argument('--server', default='http://localhost:8053/dns-query',
                        help='DoH服务器URL (默认: http://localhost:8053/dns-query)')
    parser.add_argument('--domain', default='example.com',
                        help='要查询的域名 (默认: example.com)')
    
    args = parser.parse_args()
    test_http_doh(args.server, args.domain)
