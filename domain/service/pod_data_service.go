package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Pass-baci/common"
	"github.com/Pass-baci/pod/domain/model"
	"github.com/Pass-baci/pod/domain/repository"
	"github.com/Pass-baci/pod/proto/pod"
	v1 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strconv"
)

type IPodDataService interface {
	AddPod(pod *model.Pod) (int64, error)
	DeletePod(id int64) error
	UpdatePod(pod *model.Pod) error
	FindPodByID(id int64) (*model.Pod, error)
	FindAllPod() ([]model.Pod, error)
	CreateToK8s(pod *pod.PodInfo) error
	DeleteFromK8s(pod *model.Pod) error
	UpdateToK8s(pod *pod.PodInfo) error
}

func NewPodDataService(podRepository repository.IPodRepository,
	k8sClient *kubernetes.Clientset) IPodDataService {
	return &PodDataService{
		PodRepository: podRepository,
		K8sClient:     k8sClient,
		deployment:    &v1.Deployment{},
	}
}

type PodDataService struct {
	PodRepository repository.IPodRepository
	K8sClient     *kubernetes.Clientset
	deployment    *v1.Deployment
}

func (p *PodDataService) AddPod(pod *model.Pod) (int64, error) {
	return p.PodRepository.CreatePod(pod)
}

func (p *PodDataService) DeletePod(id int64) error {
	return p.PodRepository.DeletePodByID(id)
}

func (p *PodDataService) UpdatePod(pod *model.Pod) error {
	return p.PodRepository.UpdatePod(pod)
}

func (p *PodDataService) FindPodByID(id int64) (*model.Pod, error) {
	return p.PodRepository.FindPodByID(id)
}

func (p *PodDataService) FindAllPod() ([]model.Pod, error) {
	return p.PodRepository.FindAll()
}

func (p *PodDataService) SetDeployment(pod *pod.PodInfo) {
	deployment := &v1.Deployment{
		Spec: v1.DeploymentSpec{
			Replicas: &pod.PodReplicas,
			Selector: &v12.LabelSelector{
				MatchLabels: map[string]string{
					"app-name": pod.PodName,
				},
				MatchExpressions: nil,
			},
			Template: v13.PodTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Labels: map[string]string{
						"app-name": pod.PodName,
					},
				},
				Spec: v13.PodSpec{
					Containers: []v13.Container{
						{
							Name:            pod.PodName,
							Image:           pod.PodImage,
							Ports:           p.getContainerPort(pod),
							Env:             p.getEnv(pod),
							Resources:       p.getResources(pod),
							ImagePullPolicy: p.getImagePullPolicy(pod.PodPullPolicy),
						},
					},
				},
			},
			Strategy:                v1.DeploymentStrategy{},
			MinReadySeconds:         0,
			RevisionHistoryLimit:    nil,
			Paused:                  false,
			ProgressDeadlineSeconds: nil,
		},
	}
	deployment.TypeMeta = v12.TypeMeta{
		Kind:       "deployment",
		APIVersion: "v1",
	}
	deployment.ObjectMeta = v12.ObjectMeta{
		Name:      pod.PodName,
		Namespace: pod.PodNamespace,
		Labels: map[string]string{
			"app-name": pod.PodName,
			"author":   "baci",
		},
	}

	p.deployment = deployment
}

func (p *PodDataService) getContainerPort(pod *pod.PodInfo) []v13.ContainerPort {
	var res = make([]v13.ContainerPort, 0, len(pod.PodPort))
	for _, podPort := range pod.PodPort {
		res = append(res, v13.ContainerPort{
			Name:          fmt.Sprintf("port-%d", podPort.ContainerPort),
			ContainerPort: podPort.ContainerPort,
			Protocol:      p.getProtocol(podPort.Protocol),
		})
	}
	return res
}

func (p *PodDataService) getProtocol(protocol string) v13.Protocol {
	switch protocol {
	case "TCP":
		return v13.ProtocolTCP
	case "UDP":
		return v13.ProtocolUDP
	case "SCTP":
		return v13.ProtocolSCTP
	default:
		return v13.ProtocolTCP
	}
}

func (p *PodDataService) getEnv(pod *pod.PodInfo) []v13.EnvVar {
	var res = make([]v13.EnvVar, 0, len(pod.PodEnv))
	for _, portEnv := range pod.PodEnv {
		res = append(res, v13.EnvVar{
			Name:      portEnv.EnvKey,
			Value:     portEnv.EnvValue,
			ValueFrom: nil,
		})
	}
	return res
}

func (p *PodDataService) getResources(pod *pod.PodInfo) v13.ResourceRequirements {
	return v13.ResourceRequirements{
		Limits: v13.ResourceList{
			"cpu":    resource.MustParse(strconv.FormatFloat(float64(pod.PodCpuMax), 'f', 6, 64)),
			"memory": resource.MustParse(strconv.FormatFloat(float64(pod.PodMemoryMax), 'f', 6, 64)),
		},
		Requests: v13.ResourceList{
			"cpu":    resource.MustParse(strconv.FormatFloat(float64(pod.PodCpuMax), 'f', 6, 64)),
			"memory": resource.MustParse(strconv.FormatFloat(float64(pod.PodMemoryMax), 'f', 6, 64)),
		},
	}
}

func (p *PodDataService) getImagePullPolicy(imagePullPolicy string) v13.PullPolicy {
	switch imagePullPolicy {
	case "Always":
		return v13.PullAlways
	case "IfNotPresent":
		return v13.PullIfNotPresent
	case "Never":
		return v13.PullNever
	default:
		return v13.PullIfNotPresent
	}
}

func (p *PodDataService) CreateToK8s(pod *pod.PodInfo) error {
	p.SetDeployment(pod)
	if _, err := p.K8sClient.AppsV1().Deployments(pod.PodNamespace).Get(context.TODO(), pod.PodName, v12.GetOptions{}); err != nil {
		if _, err = p.K8sClient.AppsV1().Deployments(pod.PodNamespace).Create(context.TODO(), p.deployment, v12.CreateOptions{}); err != nil {
			common.Error(err)
			return err
		}
		common.Info("创建pod成功")
		return nil
	}
	common.Errorf("Pod %s 已经存在 \n", pod.PodName)
	return errors.New(fmt.Sprintf("Pod %s 已经存在", pod.PodName))
}

func (p *PodDataService) DeleteFromK8s(pod *model.Pod) error {
	if err := p.K8sClient.AppsV1().Deployments(pod.PodNamespace).Delete(context.TODO(), pod.PodName, v12.DeleteOptions{}); err != nil {
		common.Error(err)
		return err
	}
	if err := p.DeletePod(pod.ID); err != nil {
		common.Error(err)
		return err
	}
	common.Info("删除pod成功")
	return nil
}

func (p *PodDataService) UpdateToK8s(pod *pod.PodInfo) error {
	p.SetDeployment(pod)
	if _, err := p.K8sClient.AppsV1().Deployments(pod.PodNamespace).Get(context.TODO(), pod.PodName, v12.GetOptions{}); err != nil {
		common.Errorf("Pod %s 不存在 \n", pod.PodName)
		return errors.New(fmt.Sprintf("Pod %s 不存在", pod.PodName))
	}
	if _, err := p.K8sClient.AppsV1().Deployments(pod.PodNamespace).Update(context.TODO(), p.deployment, v12.UpdateOptions{}); err != nil {
		common.Error(err)
		return err
	}
	common.Info("更新pod成功")
	return nil
}
