package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wallet-transfer/internal/models"
	"wallet-transfer/internal/service"
)

type TransferHandler struct {
	service *service.TransferService
}

func NewTransferHandler(service *service.TransferService) *TransferHandler {
	return &TransferHandler{service: service}
}

func (h *TransferHandler) CreateTransfer(c *gin.Context) {
	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.service.CreateTransfer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(result.HTTPCode, result.Body)
}
