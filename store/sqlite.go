package store

import (
	"database/sql"
	_ "github.com/glebarez/go-sqlite" // 隐式引入纯 Go 版本的 SQLite 驱动
	"log"
	"os"
	"path/filepath"
	"xray-monitor/parser" // 注意：此处保持和你项目 go.mod 匹配的模块名
)

var DB *sql.DB

// InitDB 初始化本地 SQLite 数据库
func InitDB(dbPath string) {
	// 确保数据库存放的目录存在
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, os.ModePerm)
	}

	var err error
	// 连接 SQLite（如果文件不存在，会自动创建）
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("❌ 无法连接到 SQLite 数据库: %v", err)
	}

	// 🛠️ 极致性能优化：开启 WAL 模式、关闭同步盘，榨干 SQLite 写入性能
	_, _ = DB.Exec("PRAGMA journal_mode=WAL;")
	_, _ = DB.Exec("PRAGMA synchronous=NORMAL;")

	// 创建结构化日志存储表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS xray_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		time TEXT,
		src_ip TEXT,
		src_port TEXT,
		protocol TEXT,
		dst_ip TEXT,
		dst_port TEXT,
		inbound TEXT,
		outbound TEXT,
		email TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_email ON xray_logs(email);
	CREATE INDEX IF NOT EXISTS idx_time ON xray_logs(time);
	`
	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("❌ 创建数据表失败: %v", err)
	}

	log.Println("📦 SQLite 数据库初始化成功，已开启高性能 WAL 模式！")
}

// SaveRecord 将解析后的结构化数据写入数据库
func SaveRecord(r *parser.LogRecord) error {
	insertSQL := `INSERT INTO xray_logs (time, src_ip, src_port, protocol, dst_ip, dst_port, inbound, outbound, email) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := DB.Exec(insertSQL, r.Time, r.SrcIP, r.SrcPort, r.Protocol, r.DstIP, r.DstPort, r.Inbound, r.Outbound, r.Email)
	return err
}

// GetOverviewStats 获取基础大屏总览数据
func GetOverviewStats() (map[string]interface{}, error) {
	var totalRequests int
	var activeUsers int
	var activeOutbounds int

	// 1. 统计总请求数
	_ = DB.QueryRow("SELECT COUNT(*) FROM xray_logs").Scan(&totalRequests)

	// 2. 统计独立用户数 (Email 数量)
	_ = DB.QueryRow("SELECT COUNT(DISTINCT email) FROM xray_logs").Scan(&activeUsers)

	// 3. 统计使用的独立出站线路数
	_ = DB.QueryRow("SELECT COUNT(DISTINCT outbound) FROM xray_logs").Scan(&activeOutbounds)

	return map[string]interface{}{
		"total_requests":   totalRequests,
		"active_users":     activeUsers,
		"active_outbounds": activeOutbounds,
	}, nil
}

// UserTrafficRank 代表用户请求排行的结构体
type UserTrafficRank struct {
	Email string `json:"email"`
	Count int    `json:"count"`
}

// OutboundRank 代表出站线路负载的结构体
type OutboundRank struct {
	Outbound string `json:"outbound"`
	Count    int    `json:"count"`
}

type InboundRank struct {
	SrcIP string `json:"src_ip"`
	Count int    `json:"count"`
}

// TargetMap 代表 1对N 映射关系的结构体
type TargetMap struct {
	SrcIP  string `json:"src_ip"`
	Target string `json:"target"` // 可能是域名，也可能是IP
	Count  int    `json:"count"`
}

// GetTopStats 获取用于图表渲染的排行数据
func GetTopStats() ([]UserTrafficRank, []OutboundRank, []InboundRank, []TargetMap, error) {
	// 1. 查询活跃用户 Top 10
	userRows, _ := DB.Query("SELECT email, COUNT(*) as c FROM xray_logs GROUP BY email ORDER BY c DESC LIMIT 10")
	var userRanks []UserTrafficRank
	if userRows != nil {
		for userRows.Next() {
			var r UserTrafficRank
			_ = userRows.Scan(&r.Email, &r.Count)
			userRanks = append(userRanks, r)
		}
		userRows.Close()
	}

	// 2. 查询出站线路负载排行
	outRows, _ := DB.Query("SELECT outbound, COUNT(*) as c FROM xray_logs GROUP BY outbound ORDER BY c DESC")
	var outboundRanks []OutboundRank
	if outRows != nil {
		for outRows.Next() {
			var o OutboundRank
			_ = outRows.Scan(&o.Outbound, &o.Count)
			outboundRanks = append(outboundRanks, o)
		}
		outRows.Close()
	}

	// 3. 🆕 新增：查询入站 IP 贡献排行 (用于新增的饼图)
	inRows, _ := DB.Query("SELECT src_ip, COUNT(*) as c FROM xray_logs GROUP BY src_ip ORDER BY c DESC LIMIT 10")
	var inboundRanks []InboundRank
	if inRows != nil {
		for inRows.Next() {
			var i InboundRank
			_ = inRows.Scan(&i.SrcIP, &i.Count)
			inboundRanks = append(inboundRanks, i)
		}
		inRows.Close()
	}

	// 4. 🆕 新增：查询 1对N 映射（入站IP 访问 目标域名/IP 的频次统计，展示最近活跃的30条映射）
	targetRows, _ := DB.Query("SELECT src_ip, dst_ip, COUNT(*) as c FROM xray_logs GROUP BY src_ip, dst_ip ORDER BY c DESC LIMIT 30")
	var targetMaps []TargetMap
	if targetRows != nil {
		for targetRows.Next() {
			var t TargetMap
			_ = targetRows.Scan(&t.SrcIP, &t.Target, &t.Count)
			targetMaps = append(targetMaps, t)
		}
		targetRows.Close()
	}

	return userRanks, outboundRanks, inboundRanks, targetMaps, nil
}

// StreamLink 代表一条从入站到出站的链路
type StreamLink struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Email string `json:"email"`
}

// GetLatestStreams 获取最新的 N 条链路用于前端拓扑连线
func GetLatestStreams(limit int) ([]StreamLink, error) {
	rows, err := DB.Query("SELECT src_ip, outbound, email FROM xray_logs ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []StreamLink
	for rows.Next() {
		var s StreamLink
		_ = rows.Scan(&s.From, &s.To, &s.Email)
		streams = append(streams, s)
	}
	return streams, nil
}

// TargetDetail 代表某个特定 IP 访问的单个目标统计
type TargetDetail struct {
	Target string `json:"target"`
	Count  int    `json:"count"`
}

// GetTargetsByIP 根据源 IP 统计其访问的目标排行 Top 20
func GetTargetsByIP(srcIP string) ([]TargetDetail, error) {
	querySQL := `
		SELECT dst_ip, COUNT(*) as c 
		FROM xray_logs 
		WHERE src_ip = ? 
		GROUP BY dst_ip 
		ORDER BY c DESC 
		LIMIT 20`

	rows, err := DB.Query(querySQL, srcIP)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []TargetDetail
	for rows.Next() {
		var d TargetDetail
		_ = rows.Scan(&d.Target, &d.Count)
		details = append(details, d)
	}
	return details, nil
}

// UserIPRelation 用于表示 Email 和它关联的 IP 以及请求数
type UserIPRelation struct {
	Email string   `json:"email"`
	IPs   []string `json:"ips"`
}

// GetUserIPHierarchy 获取所有活跃 Email 及其关联的所有独立 SrcIP 列表
func GetUserIPHierarchy() ([]UserIPRelation, error) {
	// 先查出所有的 Email 和 IP 组合
	querySQL := `
		SELECT email, src_ip 
		FROM xray_logs 
		WHERE email != '' AND src_ip != ''
		GROUP BY email, src_ip
		ORDER BY email ASC`

	rows, err := DB.Query(querySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 内存组装层级树
	mapping := make(map[string][]string)
	for rows.Next() {
		var email, srcIP string
		if err := rows.Scan(&email, &srcIP); err == nil {
			mapping[email] = append(mapping[email], srcIP)
		}
	}

	var result []UserIPRelation
	for email, ips := range mapping {
		result = append(result, UserIPRelation{
			Email: email,
			IPs:   ips,
		})
	}
	return result, nil
}

// GetTargetsByEmailAndIP 升级版：根据 Email 和特定 IP 联合查询目标排行 Top 20
func GetTargetsByEmailAndIP(email, srcIP string) ([]TargetDetail, error) {
	querySQL := `
		SELECT dst_ip, COUNT(*) as c 
		FROM xray_logs 
		WHERE email = ? AND src_ip = ? 
		GROUP BY dst_ip 
		ORDER BY c DESC 
		LIMIT 20`

	rows, err := DB.Query(querySQL, email, srcIP)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []TargetDetail
	for rows.Next() {
		var d TargetDetail
		_ = rows.Scan(&d.Target, &d.Count)
		details = append(details, d)
	}
	return details, nil
}
