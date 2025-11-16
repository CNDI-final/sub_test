package subtest

// ...existing code...

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type s1Message struct {
	From string `json:"from"`
	Body string `json:"body"`
}

// StartSystem1 啟動 System1，監聽 :8081，對等端 System2 在 :8082
func StartSystem1(ctx context.Context) error {
	addr := ":8081"
	peer := "http://localhost:8082"
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var msg s1Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		log.Printf("[System1] 收到來自 %s 的訊息: %s", msg.From, msg.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "received": msg.Body})
	})

	// 觸發主動對 System2 發送訊息
	mux.HandleFunc("/ping-peer", func(w http.ResponseWriter, r *http.Request) {
		body := s1Message{From: "system1", Body: "你好，System2！"}
		if err := s1SendJSON(peer+"/message", body); err != nil {
			http.Error(w, fmt.Sprintf("send failed: %v", err), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("sent"))
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		ctxShut, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctxShut)
	}()

	log.Printf("[System1] 服務啟動於 %s，對等端 %s", addr, peer)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func s1SendJSON(url string, v any) error {
	b, _ := json.Marshal(v)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %s", resp.Status)
	}
	return nil
}
