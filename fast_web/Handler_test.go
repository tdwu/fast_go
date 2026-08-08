package fast_web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type createWidgetRequest struct {
	ID   int64  `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type createWidgetResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func TestJSONHandlerBindsAndSerializesWithoutReflection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/widgets", JSONHandler(func(_ *gin.Context, request *createWidgetRequest) (createWidgetResponse, error) {
		return createWidgetResponse{ID: request.ID, Name: request.Name}, nil
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"id":"9007199254740993","name":"widget"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body: %s", response.Code, response.Body.String())
	}
	if got, want := response.Body.String(), `{"code":200,"message":"成功","data":{"id":"9007199254740993","name":"widget"}}`+"\n"; got != want {
		t.Fatalf("unexpected body:\n got: %s\nwant: %s", got, want)
	}
}

func TestJSONHandlerReturnsBadRequestForInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/widgets", JSONHandler(func(_ *gin.Context, request *createWidgetRequest) (createWidgetResponse, error) {
		return createWidgetResponse{ID: request.ID}, nil
	}))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"id":"1"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d, body: %s", response.Code, response.Body.String())
	}
}
