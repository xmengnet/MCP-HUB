// Package holiday 提供中国节假日和工作日计算功能。
// 使用 timor.tech API 获取节假日数据。
package holiday

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HolidayInfo 节假日信息
type HolidayInfo struct {
	Holiday bool   `json:"holiday"` // true=放假, false=调休上班
	Name    string `json:"name"`    // 节日名称
	Wage    int    `json:"wage"`    // 薪资倍数
	Date    string `json:"date"`    // 完整日期 "2026-02-14"
}

// HolidayResponse API响应结构
type HolidayResponse struct {
	Code    int                    `json:"code"`
	Holiday map[string]HolidayInfo `json:"holiday"`
	Type    map[string]DateType    `json:"type,omitempty"`
}

// DateType 日期类型
type DateType struct {
	Type int    `json:"type"` // 0工作日 1周末 2节日 3调休
	Name string `json:"name"`
	Week int    `json:"week"` // 1-7表示周一到周日
}

// DateInfo 日期信息（用于返回给调用者）
type DateInfo struct {
	Date      string `json:"date"`       // 日期 YYYY-MM-DD
	IsWorkday bool   `json:"is_workday"` // 是否为工作日
	IsWeekend bool   `json:"is_weekend"` // 是否为周末
	IsHoliday bool   `json:"is_holiday"` // 是否为法定节假日
	Name      string `json:"name"`       // 节日名称（如有）
}

// DateInfoAPIResponse /api/holiday/info 接口响应结构
type DateInfoAPIResponse struct {
	Code int `json:"code"`
	Type struct {
		Type int    `json:"type"` // 0工作日 1周末 2节日 3调休
		Name string `json:"name"` // 类型中文名
		Week int    `json:"week"` // 1-7 表示周一到周日
	} `json:"type"`
	Holiday *struct {
		Holiday bool   `json:"holiday"` // true节假日 false调休
		Name    string `json:"name"`    // 节假日/调休名称
		Wage    int    `json:"wage"`    // 薪资倍数
		After   bool   `json:"after"`   // 调休：放假后调休
		Target  string `json:"target"`  // 调休：对应的节假日
	} `json:"holiday"` // 非节假日时为 null
}

// DateDetailInfo 日期详情（包含完整信息）
type DateDetailInfo struct {
	Date          string `json:"date"`           // 日期
	TypeCode      int    `json:"type_code"`      // 类型代码: 0工作日 1周末 2节日 3调休
	TypeName      string `json:"type_name"`      // 类型名称
	WeekDay       int    `json:"week_day"`       // 星期几 1-7
	IsHoliday     bool   `json:"is_holiday"`     // 是否为节假日
	IsWorkday     bool   `json:"is_workday"`     // 是否需要上班
	Name          string `json:"name"`           // 节假日/调休名称
	Wage          int    `json:"wage"`           // 薪资倍数
	IsTransfer    bool   `json:"is_transfer"`    // 是否为调休日
	AfterHoliday  bool   `json:"after_holiday"`  // 是否为假后调休
	TargetHoliday string `json:"target_holiday"` // 调休对应的节假日
}

// WorkDayCalculator 工作日计算器
type WorkDayCalculator struct {
	client    *http.Client
	userAgent string
}

// NewWorkDayCalculator 创建新的工作日计算器
func NewWorkDayCalculator() *WorkDayCalculator {
	return &WorkDayCalculator{
		client:    &http.Client{Timeout: 30 * time.Second},
		userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0",
	}
}

// GetMonthHolidays 获取指定月份的节假日信息
func (calc *WorkDayCalculator) GetMonthHolidays(year, month int) (*HolidayResponse, error) {
	url := fmt.Sprintf("https://timor.tech/api/holiday/year/%d-%02d", year, month)
	return calc.getHolidayData(url)
}

// GetYearHolidays 获取整年节假日信息
func (calc *WorkDayCalculator) GetYearHolidays(year int) (*HolidayResponse, error) {
	url := fmt.Sprintf("https://timor.tech/api/holiday/year/%d/", year)
	return calc.getHolidayData(url)
}

// getHolidayData 获取API数据的内部方法
func (calc *WorkDayCalculator) getHolidayData(url string) (*HolidayResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", calc.userAgent)

	resp, err := calc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result HolidayResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误码: %d", result.Code)
	}

	return &result, nil
}

// IsHolidayOrWorkday 判断指定日期是否为节假日或调休
// 返回值: isHoliday(true表示放假), isWeekend(是否为周末)
func (calc *WorkDayCalculator) IsHolidayOrWorkday(year, month, day int, holidays map[string]HolidayInfo) (bool, bool) {
	dateStr := fmt.Sprintf("%02d-%02d", month, day)
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	isWeekend := date.Weekday() == time.Saturday || date.Weekday() == time.Sunday

	if holidayInfo, exists := holidays[dateStr]; exists {
		return holidayInfo.Holiday, isWeekend
	}

	return isWeekend, isWeekend
}

// CalculateMonthWorkdays 计算指定月份的工作日数
func (calc *WorkDayCalculator) CalculateMonthWorkdays(year, month int) (int, []time.Time, []time.Time, error) {
	holidays, err := calc.GetMonthHolidays(year, month)
	if err != nil {
		return 0, nil, nil, err
	}

	// 计算该月天数
	firstOfNextMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	daysInMonth := firstOfNextMonth.Day()

	workdaysCount := 0
	workdays := make([]time.Time, 0)
	holidaysList := make([]time.Time, 0)

	for day := 1; day <= daysInMonth; day++ {
		currentDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

		if isHoliday, _ := calc.IsHolidayOrWorkday(year, month, day, holidays.Holiday); !isHoliday {
			workdaysCount++
			workdays = append(workdays, currentDate)
		} else {
			holidaysList = append(holidaysList, currentDate)
		}
	}

	return workdaysCount, workdays, holidaysList, nil
}

// CalculateManDays 计算人日（工作日 × 人数）
func (calc *WorkDayCalculator) CalculateManDays(year, month, employeeCount int) (int, error) {
	workdays, _, _, err := calc.CalculateMonthWorkdays(year, month)
	if err != nil {
		return 0, err
	}
	return workdays * employeeCount, nil
}

// GetHolidayDetail 获取具体的节假日详情
func (calc *WorkDayCalculator) GetHolidayDetail(year, month int) ([]HolidayInfo, error) {
	holidays, err := calc.GetMonthHolidays(year, month)
	if err != nil {
		return nil, err
	}

	details := make([]HolidayInfo, 0, len(holidays.Holiday))
	for _, info := range holidays.Holiday {
		details = append(details, info)
	}

	return details, nil
}

// GetDateInfo 获取指定日期的详细信息
func (calc *WorkDayCalculator) GetDateInfo(dateString string) (*DateInfo, error) {
	date, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误，请使用 YYYY-MM-DD 格式: %v", err)
	}

	year, month, day := date.Year(), int(date.Month()), date.Day()
	holidays, err := calc.GetMonthHolidays(year, month)
	if err != nil {
		return nil, err
	}

	isHoliday, isWeekend := calc.IsHolidayOrWorkday(year, month, day, holidays.Holiday)

	info := &DateInfo{
		Date:      dateString,
		IsWorkday: !isHoliday,
		IsWeekend: isWeekend,
		IsHoliday: false,
		Name:      "",
	}

	// 检查是否为法定节假日
	dateKey := fmt.Sprintf("%02d-%02d", month, day)
	if holidayInfo, exists := holidays.Holiday[dateKey]; exists {
		info.IsHoliday = holidayInfo.Holiday
		info.Name = holidayInfo.Name
	}

	return info, nil
}

// NextWorkdayResponse 下一个工作日 API 响应
type NextWorkdayResponse struct {
	Code    int `json:"code"`
	Workday *struct {
		Type int    `json:"type"` // 0: 工作日, 3: 调休
		Name string `json:"name"` // 周一至周五 或 某某调休
		Week int    `json:"week"` // 1-7，周一至周日
		Date string `json:"date"` // 工作日日期
		Rest int    `json:"rest"` // 距离目标还有多少天
	} `json:"workday"`
}

// GetNextWorkday 查找下一个工作日（使用官方 API）
func (calc *WorkDayCalculator) GetNextWorkday(dateString string) (string, error) {
	url := fmt.Sprintf("https://timor.tech/api/holiday/workday/next/%s", dateString)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", calc.userAgent)

	resp, err := calc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result NextWorkdayResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("API返回错误")
	}

	if result.Workday == nil {
		return "", fmt.Errorf("未找到下一个工作日")
	}

	return result.Workday.Date, nil
}

// GetNextHoliday 查找下一个节假日（法定节日或周末）
func (calc *WorkDayCalculator) GetNextHoliday(dateString string) (string, string, error) {
	date, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return "", "", fmt.Errorf("日期格式错误: %v", err)
	}

	// 最多查找365天
	for i := 1; i <= 365; i++ {
		nextDate := date.AddDate(0, 0, i)
		year, month, day := nextDate.Year(), int(nextDate.Month()), nextDate.Day()

		holidays, err := calc.GetMonthHolidays(year, month)
		if err != nil {
			continue
		}

		isHoliday, _ := calc.IsHolidayOrWorkday(year, month, day, holidays.Holiday)
		if isHoliday {
			dateKey := fmt.Sprintf("%02d-%02d", month, day)
			name := "周末"
			if info, exists := holidays.Holiday[dateKey]; exists {
				name = info.Name
			}
			return nextDate.Format("2006-01-02"), name, nil
		}
	}

	return "", "", fmt.Errorf("未找到下一个节假日")
}

// GetPeriodWorkdays 计算两个日期之间的工作日数
func (calc *WorkDayCalculator) GetPeriodWorkdays(startDate, endDate string) (int, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, fmt.Errorf("开始日期格式错误: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, fmt.Errorf("结束日期格式错误: %v", err)
	}

	if end.Before(start) {
		return 0, fmt.Errorf("结束日期不能早于开始日期")
	}

	workdaysCount := 0
	currentDate := start

	for !currentDate.After(end) {
		year, month, day := currentDate.Year(), int(currentDate.Month()), currentDate.Day()

		holidays, err := calc.GetMonthHolidays(year, month)
		if err != nil {
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		isHoliday, _ := calc.IsHolidayOrWorkday(year, month, day, holidays.Holiday)
		if !isHoliday {
			workdaysCount++
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return workdaysCount, nil
}

// BatchCheckDates 批量检查多个日期
func (calc *WorkDayCalculator) BatchCheckDates(dates []string) (map[string]*DateInfo, error) {
	result := make(map[string]*DateInfo)

	for _, dateStr := range dates {
		info, err := calc.GetDateInfo(dateStr)
		if err != nil {
			continue
		}
		result[dateStr] = info
	}

	return result, nil
}

// GetDateDetail 获取指定日期的详细节假日信息
// 使用 /api/holiday/info 接口，返回更完整的日期类型和节假日信息
func (calc *WorkDayCalculator) GetDateDetail(dateString string) (*DateDetailInfo, error) {
	// 构建 URL，如果日期为空则使用服务器当前时间
	url := "https://timor.tech/api/holiday/info"
	if dateString != "" {
		url = fmt.Sprintf("https://timor.tech/api/holiday/info/%s", dateString)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", calc.userAgent)

	resp, err := calc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp DateInfoAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API返回错误码: %d", apiResp.Code)
	}

	// 构建返回结果
	detail := &DateDetailInfo{
		Date:      dateString,
		TypeCode:  apiResp.Type.Type,
		TypeName:  apiResp.Type.Name,
		WeekDay:   apiResp.Type.Week,
		IsWorkday: apiResp.Type.Type == 0 || apiResp.Type.Type == 3, // 工作日或调休上班
		IsHoliday: apiResp.Type.Type == 2,                           // 节日
		Wage:      1,
	}

	// 处理节假日/调休信息
	if apiResp.Holiday != nil {
		detail.Name = apiResp.Holiday.Name
		detail.Wage = apiResp.Holiday.Wage
		detail.IsTransfer = !apiResp.Holiday.Holiday // holiday=false 表示是调休
		detail.AfterHoliday = apiResp.Holiday.After
		detail.TargetHoliday = apiResp.Holiday.Target

		// 根据 holiday 字段判断是否真正放假
		if apiResp.Holiday.Holiday {
			detail.IsWorkday = false
		} else {
			detail.IsWorkday = true // 调休需要上班
		}
	}

	return detail, nil
}
