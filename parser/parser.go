package parser

import (
	"regexp"
	"strings"
)

// LogRecord 代表解析后的结构化日志
type LogRecord struct {
	Time     string `json:"time"`
	SrcIP    string `json:"src_ip"`
	SrcPort  string `json:"src_port"`
	Protocol string `json:"protocol"` // tcp 或 udp
	DstIP    string `json:"dst_ip"`
	DstPort  string `json:"dst_port"`
	Inbound  string `json:"inbound"`
	Outbound string `json:"outbound"`
	Email    string `json:"email"`
}

// XrayRegex 完美兼容 IPv4 / IPv6 / 域名的高级健壮正则
// 1. (?:from )?(?:(?P<proto_from>\w+):)? 兼容 "from " 和 "from tcp:"
// 2. (?P<src_ip>\[[0-9a-fA-F:]+\]|[\d\.]+) 核心核心！同时兼容 [IPv6] 和 IPv4
// 3. (?P<dst_ip>\[[0-9a-fA-F:]+\]|[^:]+) 完美捕获 [IPv6目标]、IPv4目标 或 域名目标
var XrayRegex = regexp.MustCompile(`^(?P<time>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) (?:from )?(?:(?P<proto_from>\w+):)?(?P<src_ip>\[[0-9a-fA-F:]+\]|[\d\.]+):(?P<src_port>\d+) accepted (?P<proto>\w+):(?P<dst_ip>\[[0-9a-fA-F:]+\]|[^:]+):(?P<dst_port>\d+) \[(?P<inbound>\w+) -> (?P<outbound>\w+)\] email: (?P<email>.+)$`)

// ParseLine 解析单行日志
func ParseLine(line string) (*LogRecord, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}

	matches := XrayRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, false // 无法匹配的直接跳过
	}

	// 映射正则捕获组
	result := make(map[string]string)
	for i, name := range XrayRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = matches[i]
		}
	}

	srcIP := result["src_ip"]
	dstIP := result["dst_ip"]

	// 🧼 数据清洗：由于 IPv6 在日志里带方括号（如 [2604:...:7076]），
	// 为了让大屏和前端地图、下钻面板看起来更干净漂亮，我们自动把方括号剥离掉
	srcIP = strings.Trim(srcIP, "[]")
	dstIP = strings.Trim(dstIP, "[]")

	return &LogRecord{
		Time:     result["time"],
		SrcIP:    srcIP,
		SrcPort:  result["src_port"],
		Protocol: result["proto"],
		DstIP:    dstIP,
		DstPort:  result["dst_port"],
		Inbound:  result["inbound"],
		Outbound: result["outbound"],
		Email:    result["email"],
	}, true
}
