package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/status/application"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// StatusHandler handles HTTP requests for status monitoring
type StatusHandler struct {
	getCurrentStatus      *application.GetCurrentStatusUseCase
	getStatusHistory      *application.GetStatusHistoryUseCase
	getUptimeStats        *application.GetUptimeStatsUseCase
	getDailyUptime        *application.GetDailyUptimeUseCase
	getAllServiceHistory  *application.GetAllServiceHistoryUseCase
}

// NewStatusHandler creates a new StatusHandler
func NewStatusHandler(
	getCurrentStatus *application.GetCurrentStatusUseCase,
	getStatusHistory *application.GetStatusHistoryUseCase,
	getUptimeStats *application.GetUptimeStatsUseCase,
	getDailyUptime *application.GetDailyUptimeUseCase,
	getAllServiceHistory *application.GetAllServiceHistoryUseCase,
) *StatusHandler {
	return &StatusHandler{
		getCurrentStatus:     getCurrentStatus,
		getStatusHistory:     getStatusHistory,
		getUptimeStats:       getUptimeStats,
		getDailyUptime:       getDailyUptime,
		getAllServiceHistory: getAllServiceHistory,
	}
}

// GetCurrentStatus handles GET /api/v1/status
func (h *StatusHandler) GetCurrentStatus(c *gin.Context) {
	checks, err := h.getCurrentStatus.Execute(c.Request.Context())
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Convert domain models to response DTOs
	statusList := make([]StatusCheckDTO, 0, len(checks))
	for _, check := range checks {
		statusList = append(statusList, ToStatusCheckDTO(check))
	}

	response.OK(c, gin.H{
		"services": statusList,
	})
}

// GetStatusHistory handles GET /api/v1/status/:serviceName/history
func (h *StatusHandler) GetStatusHistory(c *gin.Context) {
	serviceNameStr := c.Param("serviceName")
	serviceName, err := value.NewServiceName(serviceNameStr)
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Get hours from query parameter (default: 24 hours)
	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // Max 7 days
			hours = h
		}
	}

	checks, err := h.getStatusHistory.Execute(c.Request.Context(), serviceName, hours)
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Convert domain models to response DTOs
	history := make([]StatusCheckDTO, 0, len(checks))
	for _, check := range checks {
		history = append(history, ToStatusCheckDTO(check))
	}

	response.OK(c, gin.H{
		"service": serviceName.String(),
		"hours":   hours,
		"history": history,
	})
}

// GetUptimeStats handles GET /api/v1/status/:serviceName/uptime
func (h *StatusHandler) GetUptimeStats(c *gin.Context) {
	serviceNameStr := c.Param("serviceName")
	serviceName, err := value.NewServiceName(serviceNameStr)
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Get hours from query parameter (default: 24 hours)
	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // Max 7 days
			hours = h
		}
	}

	stats, err := h.getUptimeStats.Execute(c.Request.Context(), serviceName, hours)
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	response.OK(c, ToUptimeStatsDTO(stats))
}

// GetDailyUptime handles GET /api/v1/status/:serviceName/daily
func (h *StatusHandler) GetDailyUptime(c *gin.Context) {
	serviceNameStr := c.Param("serviceName")
	serviceName, err := value.NewServiceName(serviceNameStr)
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Get days from query parameter (default: 7 days)
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 30 { // Max 30 days
			days = d
		}
	}

	dailyData, err := h.getDailyUptime.Execute(c.Request.Context(), serviceName, days)
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Convert domain models to response DTOs
	daily := make([]DailyUptimeDTO, 0, len(dailyData))
	for _, data := range dailyData {
		daily = append(daily, ToDailyUptimeDTO(data))
	}

	response.OK(c, gin.H{
		"service": serviceName.String(),
		"days":    days,
		"data":    daily,
	})
}

// GetAllServiceHistory handles GET /api/v1/status/history/all
func (h *StatusHandler) GetAllServiceHistory(c *gin.Context) {
	// Get days from query parameter (default: 7 days)
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 30 { // Max 30 days
			days = d
		}
	}

	result, err := h.getAllServiceHistory.Execute(c.Request.Context(), application.GetAllServiceHistoryRequest{
		Days: days,
	})
	if err != nil {
		response.Error(c, err, nil)
		return
	}

	// Convert domain models to response DTOs
	services := make([]ServiceHistoryDTO, 0, len(result.Services))
	for _, svc := range result.Services {
		dailyUptime := make([]DailyUptimeDTO, 0, len(svc.DailyUptime))
		for _, data := range svc.DailyUptime {
			dailyUptime = append(dailyUptime, ToDailyUptimeDTO(data))
		}

		services = append(services, ServiceHistoryDTO{
			ServiceName: svc.ServiceName.String(),
			DailyUptime: dailyUptime,
		})
	}

	response.OK(c, AllServiceHistoryDTO{
		Services: services,
	})
}
