// Package workday 提供工作日计算 MCP 服务。
package workday

import (
	"context"
	"fmt"
	"strings"

	mcpinternal "mcp-hub/internal/mcp"
	"mcp-hub/pkg/holiday"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// init 自动注册服务到全局注册器
func init() {
	mcpinternal.Register(NewService())
}

// Service 工作日 MCP 服务
type Service struct {
	calc      *holiday.WorkDayCalculator
	mcpServer *server.MCPServer
}

// NewService 创建工作日 MCP 服务
func NewService() mcpinternal.MCPService {
	svc := &Service{
		calc: holiday.NewWorkDayCalculator(),
	}

	// 创建 MCP 服务器
	svc.mcpServer = server.NewMCPServer(
		"workday-service",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// 注册所有工具
	svc.registerTools()

	return svc
}

// Name 返回服务名称
func (s *Service) Name() string {
	return "工作日服务"
}

// Path 返回服务路径
func (s *Service) Path() string {
	return "/mcp/workday"
}

// Description 返回服务描述
func (s *Service) Description() string {
	return "中国节假日和工作日计算服务，支持查询日期类型、计算工作日、人日统计等功能"
}

// MCPServer 返回 MCP 服务器实例
func (s *Service) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// registerTools 注册所有工具
func (s *Service) registerTools() {
	// 1. 获取日期信息
	s.mcpServer.AddTool(
		mcp.NewTool("get_date_info",
			mcp.WithDescription("查询指定日期的详细信息，包括是否为工作日、周末、节假日等"),
			mcp.WithString("date",
				mcp.Required(),
				mcp.Description("日期，格式 YYYY-MM-DD，如 2026-02-14"),
			),
		),
		s.handleGetDateInfo,
	)

	// 2. 获取月份工作日数
	s.mcpServer.AddTool(
		mcp.NewTool("get_month_workdays",
			mcp.WithDescription("计算指定月份的工作日数量"),
			mcp.WithNumber("year",
				mcp.Required(),
				mcp.Description("年份，如 2026"),
			),
			mcp.WithNumber("month",
				mcp.Required(),
				mcp.Description("月份，1-12"),
			),
		),
		s.handleGetMonthWorkdays,
	)

	// 3. 计算人日
	s.mcpServer.AddTool(
		mcp.NewTool("calculate_man_days",
			mcp.WithDescription("计算指定月份的人日总数（工作日 × 员工数）"),
			mcp.WithNumber("year",
				mcp.Required(),
				mcp.Description("年份"),
			),
			mcp.WithNumber("month",
				mcp.Required(),
				mcp.Description("月份"),
			),
			mcp.WithNumber("employee_count",
				mcp.Required(),
				mcp.Description("员工数量"),
			),
		),
		s.handleCalculateManDays,
	)

	// 4. 获取节假日列表
	s.mcpServer.AddTool(
		mcp.NewTool("get_holiday_list",
			mcp.WithDescription("获取指定月份的节假日和调休安排列表"),
			mcp.WithNumber("year",
				mcp.Required(),
				mcp.Description("年份"),
			),
			mcp.WithNumber("month",
				mcp.Required(),
				mcp.Description("月份"),
			),
		),
		s.handleGetHolidayList,
	)

	// 5. 查找下一个工作日
	s.mcpServer.AddTool(
		mcp.NewTool("get_next_workday",
			mcp.WithDescription("查找指定日期之后的下一个工作日"),
			mcp.WithString("date",
				mcp.Required(),
				mcp.Description("起始日期，格式 YYYY-MM-DD"),
			),
		),
		s.handleGetNextWorkday,
	)

	// 6. 查找下一个节假日
	s.mcpServer.AddTool(
		mcp.NewTool("get_next_holiday",
			mcp.WithDescription("查找指定日期之后的下一个节假日或周末"),
			mcp.WithString("date",
				mcp.Required(),
				mcp.Description("起始日期，格式 YYYY-MM-DD"),
			),
		),
		s.handleGetNextHoliday,
	)

	// 7. 批量查询日期
	s.mcpServer.AddTool(
		mcp.NewTool("batch_check_dates",
			mcp.WithDescription("批量查询多个日期的类型"),
			mcp.WithString("dates",
				mcp.Required(),
				mcp.Description("多个日期，用逗号分隔，格式 YYYY-MM-DD,YYYY-MM-DD"),
			),
		),
		s.handleBatchCheckDates,
	)

	// 8. 计算时间段工作日
	s.mcpServer.AddTool(
		mcp.NewTool("get_period_workdays",
			mcp.WithDescription("计算两个日期之间的工作日数量"),
			mcp.WithString("start_date",
				mcp.Required(),
				mcp.Description("开始日期，格式 YYYY-MM-DD"),
			),
			mcp.WithString("end_date",
				mcp.Required(),
				mcp.Description("结束日期，格式 YYYY-MM-DD"),
			),
		),
		s.handleGetPeriodWorkdays,
	)

	// 9. 获取日期详细信息
	s.mcpServer.AddTool(
		mcp.NewTool("get_date_detail",
			mcp.WithDescription("获取指定日期的详细节假日信息，包括日期类型、调休信息、薪资倍数等完整信息"),
			mcp.WithString("date",
				mcp.Description("日期，格式 YYYY-MM-DD，如 2026-02-14。可省略则使用当前日期"),
			),
		),
		s.handleGetDateDetail,
	)
}

// --- 工具处理函数 ---

func (s *Service) handleGetDateInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	date, err := req.RequireString("date")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	info, err := s.calc.GetDateInfo(date)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := fmt.Sprintf("日期: %s\n是否工作日: %v\n是否周末: %v\n是否节假日: %v",
		info.Date, info.IsWorkday, info.IsWeekend, info.IsHoliday)
	if info.Name != "" {
		result += fmt.Sprintf("\n节日名称: %s", info.Name)
	}

	return mcp.NewToolResultText(result), nil
}

func (s *Service) handleGetMonthWorkdays(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	year, err := req.RequireFloat("year")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	month, err := req.RequireFloat("month")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workdays, _, holidays, err := s.calc.CalculateMonthWorkdays(int(year), int(month))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := fmt.Sprintf("%d年%d月工作日统计:\n工作日数: %d天\n休息日数: %d天",
		int(year), int(month), workdays, len(holidays))

	return mcp.NewToolResultText(result), nil
}

func (s *Service) handleCalculateManDays(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	year, _ := req.RequireFloat("year")
	month, _ := req.RequireFloat("month")
	employeeCount, _ := req.RequireFloat("employee_count")

	manDays, err := s.calc.CalculateManDays(int(year), int(month), int(employeeCount))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workdays, _, _, _ := s.calc.CalculateMonthWorkdays(int(year), int(month))

	result := fmt.Sprintf("%d年%d月人日统计:\n工作日数: %d天\n员工数: %d人\n人日总数: %d",
		int(year), int(month), workdays, int(employeeCount), manDays)

	return mcp.NewToolResultText(result), nil
}

func (s *Service) handleGetHolidayList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	year, _ := req.RequireFloat("year")
	month, _ := req.RequireFloat("month")

	holidays, err := s.calc.GetHolidayDetail(int(year), int(month))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(holidays) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("%d年%d月没有特殊节假日安排", int(year), int(month))), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d年%d月节假日安排:\n", int(year), int(month)))
	for _, h := range holidays {
		status := "放假"
		if !h.Holiday {
			status = "调休上班"
		}
		sb.WriteString(fmt.Sprintf("- %s: %s (%s)\n", h.Date, h.Name, status))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Service) handleGetNextWorkday(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	date, _ := req.RequireString("date")

	nextWorkday, err := s.calc.GetNextWorkday(date)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("从 %s 开始的下一个工作日是: %s", date, nextWorkday)), nil
}

func (s *Service) handleGetNextHoliday(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	date, _ := req.RequireString("date")

	nextHoliday, name, err := s.calc.GetNextHoliday(date)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("从 %s 开始的下一个节假日是: %s (%s)", date, nextHoliday, name)), nil
}

func (s *Service) handleBatchCheckDates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	datesStr, _ := req.RequireString("dates")
	dates := strings.Split(datesStr, ",")

	// 清理空格
	for i := range dates {
		dates[i] = strings.TrimSpace(dates[i])
	}

	results, err := s.calc.BatchCheckDates(dates)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var sb strings.Builder
	sb.WriteString("批量日期查询结果:\n")
	for date, info := range results {
		status := "工作日"
		if !info.IsWorkday {
			if info.Name != "" {
				status = info.Name
			} else {
				status = "休息日"
			}
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", date, status))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Service) handleGetPeriodWorkdays(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startDate, _ := req.RequireString("start_date")
	endDate, _ := req.RequireString("end_date")

	workdays, err := s.calc.GetPeriodWorkdays(startDate, endDate)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("从 %s 到 %s 期间共有 %d 个工作日", startDate, endDate, workdays)), nil
}

func (s *Service) handleGetDateDetail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	date, _ := req.RequireString("date")

	detail, err := s.calc.GetDateDetail(date)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 构建友好的输出
	weekDays := []string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	weekDay := ""
	if detail.WeekDay >= 1 && detail.WeekDay <= 7 {
		weekDay = weekDays[detail.WeekDay]
	}

	workStatus := "需要上班"
	if !detail.IsWorkday {
		workStatus = "休息日"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 日期: %s (%s)\n", detail.Date, weekDay))
	sb.WriteString(fmt.Sprintf("📌 类型: %s\n", detail.TypeName))
	sb.WriteString(fmt.Sprintf("💼 状态: %s\n", workStatus))

	if detail.Name != "" {
		sb.WriteString(fmt.Sprintf("🎉 名称: %s\n", detail.Name))
	}

	if detail.Wage > 1 {
		sb.WriteString(fmt.Sprintf("💰 薪资: %d倍工资\n", detail.Wage))
	}

	if detail.IsTransfer {
		sb.WriteString(fmt.Sprintf("🔄 调休类型: %s调休\n", func() string {
			if detail.AfterHoliday {
				return "假后"
			}
			return "假前"
		}()))
		if detail.TargetHoliday != "" {
			sb.WriteString(fmt.Sprintf("🎯 对应节日: %s\n", detail.TargetHoliday))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}
