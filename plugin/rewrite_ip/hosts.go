package rewrite_ip

import (
	"bufio"
	"net"
	"os"
	"strings"
	"sync"
)

// HostFile 存储从 hosts 文件解析的域名到 IP 映射
type HostFile struct {
	mu   sync.RWMutex
	data map[string]*HostEntry
}

// HostEntry 存储一个域名对应的 IPv4 和 IPv6 地址
type HostEntry struct {
	IPv4 []net.IP
	IPv6 []net.IP
}

// NewHostFile 创建并解析 hosts 文件
func NewHostFile(path string) (*HostFile, error) {
	hf := &HostFile{
		data: make(map[string]*HostEntry),
	}
	if err := hf.Load(path); err != nil {
		return nil, err
	}
	return hf, nil
}

// Load 从文件加载 hosts 条目
func (hf *HostFile) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hf.mu.Lock()
	defer hf.mu.Unlock()

	// 清空旧数据
	hf.data = make(map[string]*HostEntry)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析格式: IP hostname [hostname...]
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}

		// 判断 IP 类型
		isIPv6 := ip.To4() == nil

		// 为每个域名添加记录
		for _, hostname := range fields[1:] {
			hostname = strings.ToLower(hostname)
			
			if _, exists := hf.data[hostname]; !exists {
				hf.data[hostname] = &HostEntry{
					IPv4: []net.IP{},
					IPv6: []net.IP{},
				}
			}

			if isIPv6 {
				hf.data[hostname].IPv6 = append(hf.data[hostname].IPv6, ip)
			} else {
				hf.data[hostname].IPv4 = append(hf.data[hostname].IPv4, ip)
			}
		}
	}

	return scanner.Err()
}

// LookupIPv4 查询域名的 IPv4 地址
func (hf *HostFile) LookupIPv4(domain string) []net.IP {
	hf.mu.RLock()
	defer hf.mu.RUnlock()

	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	
	if entry, ok := hf.data[domain]; ok {
		return entry.IPv4
	}
	return nil
}

// LookupIPv6 查询域名的 IPv6 地址
func (hf *HostFile) LookupIPv6(domain string) []net.IP {
	hf.mu.RLock()
	defer hf.mu.RUnlock()

	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	
	if entry, ok := hf.data[domain]; ok {
		return entry.IPv6
	}
	return nil
}
