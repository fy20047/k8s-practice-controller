# 建置階段
FROM golang:1.22 as builder
WORKDIR /workspace

# 先複製 go.mod/go.sum，下載依賴，用快取加速
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# 再複製原始碼（只複製 cmd/，因為要 build ./cmd/incluster）
COPY cmd/ cmd/

# 編譯出 Linux/amd64 的靜態執行檔 main
# CGO_ENABLED=0 → 不依賴系統動態庫（好搬運）
# GOOS/GOARCH → 即使你在 Windows/Mac 也能交叉編譯給 Linux container 跑
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o main ./cmd/incluster

# 執行階段
# distroless 小且安全
FROM gcr.io/distroless/static:nonroot 
WORKDIR /
# 只把剛剛編好的二進位 COPY 進來
COPY --from=builder /workspace/main .

# 以非 root 身份執行（distroless 的內建 nonroot 使用者）
USER 65532:65532     

# container 啟動就跑 /main
ENTRYPOINT ["/main"]
