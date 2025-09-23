# practive1 - Kubernetes Deployment & Service (NodePort)

## 目標
- 撰寫 YAML 建立 **Deployment** 與 **Service (NodePort)**
- 使用 web service 類型的 image（nginx）
- 能從本地透過 `curl` 測試存取 Deployment 的 web service

---

## 內容
1. **Deployment (`deploy.yaml`)**
   - 建立一個 Deployment（replicas=3）
   - 使用 `nginx:1.25` image
   - Pod 標籤：`app: hello`

2. **Service (`svc-nodeport.yaml`)**
   - 建立一個 NodePort Service（nodePort: 30080）
   - selector 對應 `app: hello`
   - 將外部流量導向 Deployment Pod 的 port 80

3. **驗證**
   - `kubectl get deployment` 確認 replicas 數量與狀態
   - `kubectl get svc` 取得 NodePort
   - `curl 127.0.0.1:30080` 成功回應 nginx 預設頁面

---

## 總結
- 瞭解 **Deployment** 在 K8s 中的角色（維護 Pod 數量與滾動更新）
- 熟悉 **Service (NodePort)** 如何將 cluster 外部流量導向內部 Pod
- 能用 `kubectl` 與 `curl` 驗證應用是否可用
