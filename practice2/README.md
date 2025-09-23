# Practice2 - Kubernetes Client with Go

## 目標
1. 採用 Go 語言，開發 client 程式，用來建立 Deployment 與 Service。
   - Deployment：使用 web service 類型的 image，例如 **nginx**
   - Service：使用 **NodePort**
   - 可從本地利用 `curl` 透過 NodePort 存取
2. client 執行期間，會持續讀取建立的 Deployment 名字並 print 出來
3. 刪除 client 時，會同時刪除所建立的 Deployment 與 Service
4. 撰寫 **Multi-stage Dockerfile** 建置 image
5. 部署 client 至 Kubernetes，需包含 Deployment 與 ServiceAccount（以及 RBAC 設定）

---

## 流程

### 1. 套用 manifest
```bash
kubectl apply -f manifest/
```
### 2. 確認 Deployment 與 Service 建立成功
```bash
kubectl get deploy
kubectl get svc
```
範例輸出：
```bash
NAME         READY   UP-TO-DATE   AVAILABLE   AGE
hello-app1   1/1     1            1           15s  # main.go 執行後，由 client-go 程式動態建立的 Deployment
hello-apps   1/1     1            1           16s  # manifest 裡的 Deployment，用來跑 client (image 裡有 main.go)

NAME            TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
hello-service   NodePort   10.103.11.247   <none>        80:30080/TCP   8s # main.go 執行後，由 client-go 程式動態建立的 Service
```
### 3. 刪除 Deployment 與 Service
```bash
kubectl delete deploy hello-apps
kubectl delete svc hello-service
```
驗證刪除：
```bash
kubectl get deploy
kubectl get svc
```
輸出：
```bash
No resources found in default namespace.
```

## 資源角色對照表

| 名稱           | 類型       | 建立方式        | 角色說明               |
| ------------- | ---------- | -------------- | ------------------------------ |
| hello-apps    | Deployment | manifest       | 用來執行 client (Go 程式)            |
| hello-app1    | Deployment | client-go 動態建立 | 由 client 建立的 nginx Deployment  |
| hello-service | Service    | client-go 動態建立 | 對外暴露 hello-app1 Pod 的 NodePort |
