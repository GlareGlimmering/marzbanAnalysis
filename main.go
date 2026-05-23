package main

import (
	"io"
	"log"
	"xray-monitor/parser"
	"xray-monitor/store"

	fiber "github.com/gofiber/fiber/v2"
	cors "github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/hpcloud/tail"
)

func main() {
	log.Println("🚀 Xray 日志可视化监视器正在启动...")

	// 1. 初始化本地数据库
	store.InitDB("data.db")

	// 2. 利用 Goroutine 将日志异步监听移至后台，从头扫描并持续追更
	go func() {
		log.Println("📥 正在执行历史日志全量扫描初始化...")

		config := tail.Config{
			Follow:    true,
			ReOpen:    true,
			MustExist: true,
			// 💡 核心注入：指定从文件绝对头部 (0 字节位置) 开始向下狂飙
			Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekStart},
		}

		t, err := tail.TailFile("/var/lib/marzban/xray_access.log", config)
		if err != nil {
			log.Fatalf("❌ 无法监听日志文件: %v", err)
		}

		// 开启事务优化批量写入性能（可选，但对于几十M的日志初始入库速度极快）
		for line := range t.Lines {
			record, ok := parser.ParseLine(line.Text)
			if ok {
				_ = store.SaveRecord(record)
			}
		}
	}()

	// 3. 启动高性能 Fiber Web 服务
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
		AppName:               "Xray-Monitor Dashboard API v1.0",
	})

	// 允许跨域（方便本地前后端分离联调前端）
	app.Use(cors.New())

	// 🛠️ 接口 1: 获取大屏总览数字
	app.Get("/api/overview", func(c *fiber.Ctx) error {
		stats, err := store.GetOverviewStats()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(stats)
	})

	// 🛠️ 接口 2: 获取图表所需的排行数据
	app.Get("/api/charts", func(c *fiber.Ctx) error {
		userRanks, outboundRanks, inboundRanks, targetMaps, err := store.GetTopStats()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"user_rank":     userRanks,
			"outbound_rank": outboundRanks,
			"inbound_rank":  inboundRanks,
			"target_map":    targetMaps,
		})
	})

	// 🛠️ 接口 4: 传入用户和IP，获取双重过滤下的目标排行
	app.Get("/api/ip-targets", func(c *fiber.Ctx) error {
		email := c.Query("email")
		ip := c.Query("ip")
		if email == "" || ip == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Missing 'email' or 'ip' parameter"})
		}

		details, err := store.GetTargetsByEmailAndIP(email, ip)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(details)
	})

	// 🛠️ 接口 5: 获取 Email -> IPs 的层级联动树
	app.Get("/api/user-hierarchy", func(c *fiber.Ctx) error {
		hierarchy, err := store.GetUserIPHierarchy()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(hierarchy)
	})

	// 4. 监听本地 8080 端口
	log.Fatal(app.Listen(":8080"))
}
