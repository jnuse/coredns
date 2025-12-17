#!/usr/bin/env python3
"""
DNS 查询工具 - 用于生成和发送 DNS 查询请求
支持标准 UDP/TCP 查询和 DoH (HTTP/HTTPS) 查询
"""

import socket
import struct
import base64
import argparse
from typing import Tuple, Optional

class DNSQuery:
    """DNS 查询构造器"""
    
    @staticmethod
    def build_query(domain: str, qtype: str = 'A') -> bytes:
        """
        构造 DNS 查询报文
        
        Args:
            domain: 查询的域名
            qtype: 查询类型 (A, AAAA, CNAME等)
        
        Returns:
            DNS 查询报文的二进制数据
        """
        # DNS Header (12 bytes)
        transaction_id = 0x1234
        flags = 0x0100  # Standard query with recursion desired
        qdcount = 1     # One question
        ancount = 0     # No answers
        nscount = 0     # No authority records
        arcount = 0     # No additional records
        
        header = struct.pack('!HHHHHH', 
                           transaction_id, flags, qdcount, 
                           ancount, nscount, arcount)
        
        # Question Section
        qname = b''
        for part in domain.split('.'):
            qname += struct.pack('B', len(part)) + part.encode('ascii')
        qname += b'\x00'  # End of domain name
        
        # Query Type and Class
        type_map = {
            'A': 1,
            'AAAA': 28,
            'CNAME': 5,
            'MX': 15,
            'TXT': 16
        }
        qtype_val = type_map.get(qtype.upper(), 1)
        qclass = 1  # IN (Internet)
        
        question = qname + struct.pack('!HH', qtype_val, qclass)
        
        return header + question
    
    @staticmethod
    def parse_response(response: bytes) -> dict:
        """
        解析 DNS 响应报文
        
        Args:
            response: DNS 响应的二进制数据
        
        Returns:
            解析后的响应字典
        """
        if len(response) < 12:
            return {'error': 'Response too short'}
        
        # Parse header
        (tid, flags, qdcount, ancount, nscount, arcount) = \
            struct.unpack('!HHHHHH', response[:12])
        
        result = {
            'transaction_id': tid,
            'flags': flags,
            'questions': qdcount,
            'answers': ancount,
            'authority': nscount,
            'additional': arcount,
            'rcode': flags & 0x000F,
            'records': []
        }
        
        # Skip question section (简化处理)
        offset = 12
        for _ in range(qdcount):
            while offset < len(response) and response[offset] != 0:
                label_len = response[offset]
                offset += label_len + 1
            offset += 5  # Skip null + type + class
        
        # Parse answer section
        for _ in range(ancount):
            if offset >= len(response):
                break
            
            # Skip name (简化处理，假设使用压缩)
            if response[offset] >= 0xC0:
                offset += 2
            else:
                while offset < len(response) and response[offset] != 0:
                    label_len = response[offset]
                    offset += label_len + 1
                offset += 1
            
            if offset + 10 > len(response):
                break
            
            rtype, rclass, ttl, rdlength = struct.unpack(
                '!HHIH', response[offset:offset+10]
            )
            offset += 10
            
            if offset + rdlength > len(response):
                break
            
            rdata = response[offset:offset+rdlength]
            offset += rdlength
            
            # Parse A record (IPv4)
            if rtype == 1 and rdlength == 4:
                ip = '.'.join(str(b) for b in rdata)
                result['records'].append({
                    'type': 'A',
                    'ttl': ttl,
                    'ip': ip
                })
            # Parse AAAA record (IPv6)
            elif rtype == 28 and rdlength == 16:
                ip = ':'.join(f'{rdata[i]:02x}{rdata[i+1]:02x}' 
                             for i in range(0, 16, 2))
                result['records'].append({
                    'type': 'AAAA',
                    'ttl': ttl,
                    'ip': ip
                })
        
        return result


def query_udp(domain: str, qtype: str, server: str, port: int = 53) -> dict:
    """通过 UDP 发送 DNS 查询"""
    query = DNSQuery.build_query(domain, qtype)
    
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(5)
    
    try:
        sock.sendto(query, (server, port))
        response, _ = sock.recvfrom(512)
        return DNSQuery.parse_response(response)
    finally:
        sock.close()


def query_doh(domain: str, qtype: str, url: str) -> dict:
    """通过 DoH (DNS over HTTP) 发送查询"""
    import urllib.request
    
    query = DNSQuery.build_query(domain, qtype)
    
    # 使用 POST 方法发送
    req = urllib.request.Request(
        url,
        data=query,
        headers={
            'Content-Type': 'application/dns-message',
            'Accept': 'application/dns-message'
        }
    )
    
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            if resp.status != 200:
                return {'error': f'HTTP {resp.status}'}
            response = resp.read()
            return DNSQuery.parse_response(response)
    except Exception as e:
        return {'error': str(e)}


def main():
    parser = argparse.ArgumentParser(description='DNS Query Tool')
    parser.add_argument('domain', help='Domain to query')
    parser.add_argument('-t', '--type', default='A', 
                       help='Query type (A, AAAA, etc.)')
    parser.add_argument('-s', '--server', default='127.0.0.1',
                       help='DNS server address')
    parser.add_argument('-p', '--port', type=int, default=53,
                       help='DNS server port')
    parser.add_argument('--doh', help='DoH URL (e.g., http://127.0.0.1:8053/dns-query)')
    
    args = parser.parse_args()
    
    if args.doh:
        print(f"Querying {args.domain} ({args.type}) via DoH: {args.doh}")
        result = query_doh(args.domain, args.type, args.doh)
    else:
        print(f"Querying {args.domain} ({args.type}) via UDP: {args.server}:{args.port}")
        result = query_udp(args.domain, args.type, args.server, args.port)
    
    print("\nResult:")
    if 'error' in result:
        print(f"Error: {result['error']}")
    else:
        print(f"Transaction ID: {result['transaction_id']:#x}")
        print(f"Response Code: {result['rcode']}")
        print(f"Answers: {result['answers']}")
        for record in result['records']:
            print(f"  {record['type']}: {record['ip']} (TTL: {record['ttl']})")


if __name__ == '__main__':
    main()
