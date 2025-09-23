package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	// K8s 的 API 型別套件（Deployment/Service/Metadata）
	appv1 "k8s.io/api/apps/v1"         // Deployment/ReplicaSet 等資源的型別
	corev1 "k8s.io/api/core/v1"        // Service/Pod/ConfigMap 等核心資源型別
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // 共用的 ObjectMeta/時間戳等

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"      // clientset 入口（typed clients）
	"k8s.io/client-go/rest"             // in-cluster 連線設定
	"k8s.io/client-go/tools/clientcmd"  // out-of-cluster（~/.kube/config）連線設定
)

var (
	// 預設 Namespace；所有 Create/Get/Delete 都會指向這個 namespace
	namespace = "default"
)

func main() {
	// 用"叢集外"或"叢集內"的連線方式 
	// outside-cluster = true：本機開發/CI 透過 ~/.kube/config 連 API Server
	// outside-cluster = false：程式跑在 K8s Pod 內，透過 ServiceAccount 連 API Server
	outsideCluster := flag.Bool("outside-cluster", false, "set to true when run out of cluster. (default: false)")
	flag.Parse()

	// 建立 clientset
	var clientset *kubernetes.Clientset
	if *outsideCluster {
		// 叢集外：讀使用者家目錄ㄉ kubeconfig
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		config, err := clientcmd.BuildConfigFromFlags("", path.Join(home, ".kube/config"))
		if err != nil {
			panic(err.Error())
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	} else {
		// 叢集內：Pod 內用 ServiceAccount/CA/Token 自動連 API Server（再搭配 RBAC 權限）
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	}

	// 建立 resources
	// createDeployment：建立一個 Nginx Deployment（會產生帶有 label 的 Pod）
	// createService：建立 NodePort Service，selector 會對到 Pod 的 label
	dm := createDeployment(clientset)
	sm := createService(clientset)

	// 讓 client 執行期間持續讀 Deployment 的名稱並印出
	// 用 goroutine + for 來每秒 Get 一次
	go func() {
		for {
			read, err := clientset.
				AppsV1().
				Deployments(namespace).
				Get(
					context.Background(),
					dm.GetName(),      // 讀剛剛建立的那個 Deployment
					metav1.GetOptions{},
				)
			if err != nil { // 沒抓到
				panic(err.Error())
			}

			fmt.Printf("Read Deployment %s/%s\n", namespace, read.GetName())
			time.Sleep(time.Second)
		}
	}()

	// 等終止訊號做清理，也就是刪掉剛剛建立的 Deployment 與 Service
	fmt.Println("Waiting for Kill Signal...")
	var stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	fmt.Printf("Delete Deployment %s/%s ", namespace, dm.GetName())
	deleteDeployment(clientset, dm)
	fmt.Printf("Delete Service %s/%s ", namespace, sm.GetName())
	deleteService(clientset, sm)
}

// 很多規格要 *int32
func int32Ptr(i int32) *int32 { return &i }

// 這邊用 Go struct 定義 Deployment（程式碼版ㄉ YAML），再用 clientset.Create() 丟給 API Server
func createDeployment(client kubernetes.Interface) *appv1.Deployment {
	dm := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-app1",   // Deployment 名稱
			Labels: map[string]string{
				"fy20047-k8s": "practice2",
			},
		},
		Spec: appv1.DeploymentSpec{
			Replicas: int32Ptr(1),           // 副本數
			Selector: &metav1.LabelSelector{
				// 定義 Deployment 管理哪些 Pod（跟 PodTemplate 的 labels 一致）
				MatchLabels: map[string]string{
					"fy20047-k8s": "practice2",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// PodTemplate 的 labels（Service selector 會用這個來導流量）
					Labels: map[string]string{
						"fy20047-k8s": "practice2",
					},
				},
				Spec: corev1.PodSpec{
					// 跑一個 Nginx 容器，對外開 80 port（內部容器埠）
					Containers: []corev1.Container{
						{
							Name:  "nginx-container",
							Image: "nginx:1.14.2",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,                    // Pod 內部埠
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	dm.Namespace = namespace

	// 送到 API Server 建立
	dm, err := client.
		AppsV1().
		Deployments(namespace).
		Create(
			context.Background(),
			dm,
			metav1.CreateOptions{},
		)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Created Deployment %s/%s\n", dm.GetNamespace(), dm.GetName())
	return dm
}

// 刪除 Deployment
func deleteDeployment(client kubernetes.Interface, dm *appv1.Deployment) {
	err := client.
		AppsV1().
		Deployments(dm.GetNamespace()).
		Delete(
			context.Background(),
			dm.GetName(),
			metav1.DeleteOptions{},
		)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Deleted Deployment %s/%s\n", dm.GetNamespace(), dm.GetName())
}

var portnum int32 = 80

// 建立 Service
// 建一個 NodePort Service，selector 對到 Pod 的 labels，把外部 nodeIP:nodePort → Pod:80
func createService(client kubernetes.Interface) *corev1.Service {
	sm := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-service",
			Labels: map[string]string{
				// 統一貼上同一組 label（方便檢查/分類）
				"fy20047-k8s": "practice2",
			},
		},
		Spec: corev1.ServiceSpec{
			// selector 要對到 Pod 的 labels
			Selector: map[string]string{
				"fy20047-k8s": "practice2",
			},
			Type: corev1.ServiceTypeNodePort, 
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,                               // Service 對外的 port
					TargetPort: intstr.IntOrString{IntVal: portnum}, // 導向 Pod 的 target port（80）
					NodePort:   30080,                            // 節點上的 NodePort
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	sm.Namespace = namespace

	// 送到 API Server 建立
	sm, err := client.
		CoreV1().
		Services(namespace).
		Create(
			context.Background(),
			sm,
			metav1.CreateOptions{},
		)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Created Service %s/%s\n", sm.GetNamespace(), sm.GetName())
	return sm
}

// 刪除 Service
func deleteService(client kubernetes.Interface, sm *corev1.Service) {
	err := client.
		CoreV1().
		Services(sm.GetNamespace()).
		Delete(
			context.Background(),
			sm.GetName(),
			metav1.DeleteOptions{},
		)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Deleted Service %s/%s\n", sm.GetNamespace(), sm.GetName())
}
