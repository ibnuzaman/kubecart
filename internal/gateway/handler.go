// Package gateway translates REST requests into gRPC calls to OrderService.
package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/ibnuzaman/kubecart/gen/go/order/v1"
)

type handler struct {
	orders  orderv1.OrderServiceClient
	timeout time.Duration
}

// NewHandler returns the gateway's HTTP routes. timeout is the deadline for
// the whole downstream call chain of one request.
func NewHandler(orders orderv1.OrderServiceClient, timeout time.Duration) http.Handler {
	h := &handler{orders: orders, timeout: timeout}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.createOrder)
	// Liveness of the gateway process only. It deliberately doesn't call
	// order-service: if it did, an order outage would also mark every gateway
	// unhealthy and restart them for nothing.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

type createOrderRequest struct {
	SKU      string `json:"sku"`
	Quantity int32  `json:"quantity"`
}

type orderResponse struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Quantity int32  `json:"quantity"`
}

func (h *handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Field validation lives in order-service, the owner of the rules. The
	// gateway only checks that the body is well-formed JSON.
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	resp, err := h.orders.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		Sku:      req.SKU,
		Quantity: req.Quantity,
	})
	if err != nil {
		st := status.Convert(err)
		code := httpStatus(st.Code())
		msg := st.Message()
		if code == http.StatusInternalServerError {
			msg = "internal error"
		}
		writeError(w, code, msg)
		return
	}

	o := resp.GetOrder()
	writeJSON(w, http.StatusCreated, orderResponse{ID: o.GetId(), SKU: o.GetSku(), Quantity: o.GetQuantity()})
}

// httpStatus maps a gRPC status code to the closest HTTP status.
func httpStatus(c codes.Code) int {
	switch c {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.FailedPrecondition:
		return http.StatusConflict
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
