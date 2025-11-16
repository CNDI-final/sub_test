package subtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestStartSystem1(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 啟動 System1
	go func() {
		_ = StartSystem1(ctx)
	}()

	// 等待服務啟動
	time.Sleep(500 * time.Millisecond)

	t.Run("health check", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/health")
		if err != nil {
			t.Fatalf("health check 失敗: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("health check 狀態碼: got %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "ok" {
			t.Errorf("health check 回應: got %q, want %q", string(body), "ok")
		}
	})

	t.Run("receive message", func(t *testing.T) {
		msg := s1Message{From: "test", Body: "測試訊息"}
		b, _ := json.Marshal(msg)
		resp, err := http.Post("http://localhost:8081/message", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("POST /message 失敗: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("POST /message 狀態碼: got %d, want 200", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("解析回應失敗: %v", err)
		}

		if result["status"] != "ok" {
			t.Errorf("status: got %q, want %q", result["status"], "ok")
		}
		if result["received"] != "測試訊息" {
			t.Errorf("received: got %q, want %q", result["received"], "測試訊息")
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/message")
		if err != nil {
			t.Fatalf("GET /message 失敗: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("狀態碼: got %d, want 405", resp.StatusCode)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		resp, err := http.Post("http://localhost:8081/message", "application/json", bytes.NewReader([]byte("invalid")))
		if err != nil {
			t.Fatalf("POST invalid json 失敗: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("狀態碼: got %d, want 400", resp.StatusCode)
		}
	})
}

func TestS1SendJSON(t *testing.T) {
	// 啟動一個簡單的測試伺服器
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		var msg s1Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"echo": msg.Body})
	})

	srv := &http.Server{Addr: ":9998", Handler: mux}
	go func() {
		_ = srv.ListenAndServe()
	}()
	defer srv.Close()

	time.Sleep(200 * time.Millisecond)

	t.Run("successful send", func(t *testing.T) {
		msg := s1Message{From: "test", Body: "hello"}
		err := s1SendJSON("http://localhost:9998/test", msg)
		if err != nil {
			t.Errorf("s1SendJSON 失敗: %v", err)
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		msg := s1Message{From: "test", Body: "hello"}
		err := s1SendJSON("http://localhost:1235/test", msg)
		if err == nil {
			t.Error("預期有錯誤但沒有發生")
		}
	})
}
